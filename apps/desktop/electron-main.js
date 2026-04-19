import { app, BrowserWindow, dialog, ipcMain, nativeImage } from "electron";
import { mkdir, readFile, stat, writeFile } from "fs/promises";
import crypto from "crypto";
import path from "path";
import { fileURLToPath } from "url";
import grpc from "@grpc/grpc-js";
import protoLoader from "@grpc/proto-loader";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const protoPath = path.resolve(__dirname, "..", "..", "proto", "photo_engine.proto");
const packageDefinition = protoLoader.loadSync(protoPath, {
  defaults: true,
  oneofs: true,
});
const photoengine = grpc.loadPackageDefinition(packageDefinition).photoengine.v1;
const heifLikeExtensions = new Set([".heic", ".heif", ".hif"]);
const heifThumbnailHalfScale = 0.5;
let heifThumbnailCacheRoot = "";
const heifThumbnailInFlight = new Map();

function isHeifLikePath(filePath) {
  return heifLikeExtensions.has(path.extname(String(filePath || "")).toLowerCase());
}

function heifThumbnailCacheKey(photoPath, variant, modTimeMs, fileSize) {
  return crypto
    .createHash("sha1")
    .update(`${photoPath}|thumb:${variant}|${modTimeMs}|${fileSize}|v2`)
    .digest("hex");
}

function heifThumbnailCachePath(key) {
  return path.join(heifThumbnailCacheRoot, `${key}.jpg`);
}

function heifHalfSize(width, height) {
  return {
    width: Math.max(1, Math.round(Number(width || 0) * heifThumbnailHalfScale)),
    height: Math.max(1, Math.round(Number(height || 0) * heifThumbnailHalfScale)),
  };
}

async function createHeifThumbnailBytes(photoPath, width, height) {
  const thumb = await nativeImage.createThumbnailFromPath(photoPath, { width, height });
  if (thumb.isEmpty()) {
    throw new Error(`empty thumbnail for ${photoPath}`);
  }
  return thumb.toJPEG(90);
}

async function loadOrCreateHeifCachedBytes(photoPath, variant, createBytes) {
  const info = await stat(photoPath);
  const key = heifThumbnailCacheKey(photoPath, variant, info.mtimeMs, info.size);
  const existing = heifThumbnailInFlight.get(key);
  if (existing) {
    return existing;
  }

  const job = (async () => {
    if (heifThumbnailCacheRoot) {
      try {
        const cached = await readFile(heifThumbnailCachePath(key));
        return { data: cached, fromCache: true };
      } catch (error) {
        if (error?.code !== "ENOENT") {
          throw error;
        }
      }
    }

    const data = await createBytes(info);
    if (heifThumbnailCacheRoot) {
      await mkdir(heifThumbnailCacheRoot, { recursive: true });
      await writeFile(heifThumbnailCachePath(key), data);
    }

    return { data, fromCache: false };
  })();

  heifThumbnailInFlight.set(key, job);
  try {
    return await job;
  } finally {
    heifThumbnailInFlight.delete(key);
  }
}

async function loadOrCreateHeifHalfThumbnail(photoPath, sourceWidth, sourceHeight) {
  const { width, height } = heifHalfSize(sourceWidth, sourceHeight);
  return loadOrCreateHeifCachedBytes(photoPath, "half", async () => {
    if (width <= 0 || height <= 0) {
      throw new Error(`missing source dimensions for ${photoPath}`);
    }
    return createHeifThumbnailBytes(photoPath, width, height);
  });
}

async function loadOrCreateHeifSizedThumbnail(photoPath, size) {
  const targetSize = Math.max(1, Number(size) || 256);
  return loadOrCreateHeifCachedBytes(photoPath, `size:${targetSize}`, async () => {
    return createHeifThumbnailBytes(photoPath, targetSize, targetSize);
  });
}

async function downscaleHeifThumbnail(data, size) {
  const targetSize = Math.max(1, Number(size) || 0);
  if (!targetSize || !data?.length) {
    return data;
  }

  const image = nativeImage.createFromBuffer(data);
  if (image.isEmpty()) {
    return data;
  }

  const current = image.getSize();
  if (current.width <= targetSize && current.height <= targetSize) {
    return data;
  }

  const resized = image.resize({ width: targetSize, height: targetSize });
  if (resized.isEmpty()) {
    return data;
  }

  return resized.toJPEG(90);
}

function warmHeifThumbnailCache(photo) {
  const photoPath = photo?.path ?? "";
  if (!isHeifLikePath(photoPath)) return;
  const width = Number(photo?.width ?? 0) || 0;
  const height = Number(photo?.height ?? 0) || 0;
  if (width <= 0 || height <= 0) {
    console.warn(
      `[thumbnail] HEIF cache warm skipped for ${photoPath}: missing source dimensions`
    );
    return;
  }
  void loadOrCreateHeifHalfThumbnail(photoPath, width, height).catch((error) => {
    console.warn(
      `[thumbnail] HEIF cache warm failed for ${photoPath} (50%):`,
      String(error?.message ?? error)
    );
  });
}

function createClient() {
  return new photoengine.PhotoEngine(
    "127.0.0.1:50051",
    grpc.credentials.createInsecure(),
    {
      "grpc.max_receive_message_length": 32 * 1024 * 1024,
      "grpc.max_send_message_length": 32 * 1024 * 1024,
    }
  );
}

function invokeHealth(client) {
  return new Promise((resolve, reject) => {
    client.Health({}, (err, res) => {
      if (err) {
        reject(err);
        return;
      }
      resolve(res);
    });
  });
}

function streamListPhotos(client, request) {
  return new Promise((resolve, reject) => {
    const items = [];
    const call = client.ListPhotosStream(request);
    call.on("data", (msg) => {
      items.push(msg);
    });
    call.on("error", reject);
    call.on("end", () => {
      resolve({
        items,
        total: items.length,
      });
    });
  });
}

function invokeScanFolder(client, request) {
  return new Promise((resolve, reject) => {
    client.ScanFolder(request, (err, res) => {
      if (err) {
        reject(err);
        return;
      }
      resolve(res);
    });
  });
}

function streamScanFolder(client, request, onProgress) {
  return new Promise((resolve, reject) => {
    const call = client.ScanFolderStream(request);
    call.on("data", (msg) => onProgress?.(msg));
    call.on("error", reject);
    call.on("end", resolve);
  });
}

function invokeDeletePhoto(client, request) {
  return new Promise((resolve, reject) => {
    client.DeletePhoto(request, (err, res) => {
      if (err) {
        reject(err);
        return;
      }
      resolve(res);
    });
  });
}
function invokeMoveToRejected(client, request) {
  return new Promise((resolve, reject) => {
    client.MoveToRejected(request, (err, res) => {
      if (err) {
        reject(err);
        return;
      }
      resolve(res);
    });
  });
}

function invokeGetMetadata(client, request) {
  return new Promise((resolve, reject) => {
    client.GetMetadata(request, (err, res) => {
      if (err) {
        reject(err);
        return;
      }
      resolve(res);
    });
  });
}

function createRpcBridge() {
  heifThumbnailCacheRoot = path.join(app.getPath("userData"), "heif-thumbnail-cache");
  void mkdir(heifThumbnailCacheRoot, { recursive: true }).catch((error) => {
    console.warn(
      `[thumbnail] failed to prepare HEIF cache directory ${heifThumbnailCacheRoot}:`,
      String(error?.message ?? error)
    );
  });
  const client = createClient();
  ipcMain.handle("app:choose-folder", async () => {
    const result = await dialog.showOpenDialog({
      properties: ["openDirectory"],
    });
    if (result.canceled || !result.filePaths.length) {
      return { ok: false, canceled: true };
    }
    return { ok: true, data: { path: result.filePaths[0] } };
  });
  ipcMain.handle("app:health", async () => {
    try {
      const res = await invokeHealth(client);
      return { ok: true, data: res };
    } catch (error) {
      return { ok: false, error: String(error?.message ?? error) };
    }
  });
  ipcMain.handle("app:scan-folder", async (_event, request = {}) => {
    try {
      const res = await invokeScanFolder(client, {
        folderPath: request.folderPath ?? "",
      });
      return { ok: true, data: res };
    } catch (error) {
      return { ok: false, error: String(error?.message ?? error) };
    }
  });
  ipcMain.handle("app:scan-folder-stream", async (event, request = {}) => {
    try {
      await streamScanFolder(client, {
        folderPath: request.folderPath ?? "",
      }, (msg) => {
        warmHeifThumbnailCache(msg.photo ?? null);
        event.sender.send("app:scan-folder-progress", {
          scanned: msg.scanned ?? 0,
          total: msg.total ?? 0,
          photo: msg.photo ?? null,
        });
      });
      return { ok: true };
    } catch (error) {
      return { ok: false, error: String(error?.message ?? error) };
    }
  });
  ipcMain.handle("app:list-photos", async (_event, request = {}) => {
    try {
      const res = await streamListPhotos(client, {
        folderPath: request.folderPath ?? "",
        limit: request.limit ?? 200,
        offset: request.offset ?? 0,
        sortBy: request.sortBy ?? "captured_at",
        descending: request.descending ?? true,
      });
      return { ok: true, data: res };
    } catch (error) {
      return { ok: false, error: String(error?.message ?? error) };
    }
  });
  const activeThumbnailCalls = new Map();

  function parseThumbnailSize(callId) {
    const idx = String(callId).lastIndexOf("_");
    if (idx === -1) return 0;
    const size = Number(String(callId).slice(idx + 1));
    return Number.isFinite(size) ? size : 0;
  }

  function parseThumbnailPhotoId(callId) {
    const idx = String(callId).lastIndexOf("_");
    if (idx === -1) return "";
    return String(callId).slice(0, idx);
  }

  ipcMain.handle("app:abort-thumbnails", async (_event, request = {}) => {
    const keepPhotoId = request.photoId ?? "";
    activeThumbnailCalls.forEach((call, id) => {
      if (parseThumbnailSize(id) >= 1024 && parseThumbnailPhotoId(id) !== keepPhotoId) {
        try { call.cancel(); } catch(e) {}
        activeThumbnailCalls.delete(id);
      }
    });
    return { ok: true };
  });

  ipcMain.handle("app:get-thumbnail", async (_event, request = {}) => {
    const callId = request.photoId + "_" + (request.size ?? 256);
    // If it's a large preview, cancel other large previews (keep it simple: only 1 large preview processing at a time)
    if ((request.size ?? 256) > 1024) {
      activeThumbnailCalls.forEach((call, id) => {
         if (id.includes("_4096")) {
            try { call.cancel(); } catch(e) {}
            activeThumbnailCalls.delete(id);
         }
      });
    }

    const photoPath = String(request.photoId ?? "");
    if (isHeifLikePath(photoPath)) {
      try {
        const size = Math.max(1, Number(request.size ?? 256) || 256);
        const sourceWidth = Math.max(0, Number(request.width ?? 0) || 0);
        const sourceHeight = Math.max(0, Number(request.height ?? 0) || 0);
        let cached = null;
        if (sourceWidth > 0 && sourceHeight > 0) {
          cached = await loadOrCreateHeifHalfThumbnail(photoPath, sourceWidth, sourceHeight);
        }
        if (!cached) {
          cached = await loadOrCreateHeifSizedThumbnail(photoPath, size);
        }
        if (cached?.data?.length) {
          const data = await downscaleHeifThumbnail(cached.data, size);
          return {
            ok: true,
            data: {
              mimeType: "image/jpeg",
              fromCache: Boolean(cached.fromCache),
              base64: Buffer.from(data).toString("base64"),
            },
          };
        }
      } catch (error) {
        console.warn(`[thumbnail] HEIF thumbnail fallback failed for ${photoPath}:`, String(error?.message ?? error));
      }
    }

    try {
      const res = await new Promise((resolve, reject) => {
        const call = client.GetThumbnail(
          {
            photoId: photoPath,
            size: request.size ?? 256,
          },
          (err, data) => {
            activeThumbnailCalls.delete(callId);
            if (err) {
              reject(err);
              return;
            }
            resolve(data);
          }
        );
        activeThumbnailCalls.set(callId, call);
      });
      return {
        ok: true,
        data: {
          mimeType: res.mimeType ?? "image/jpeg",
          fromCache: Boolean(res.fromCache),
          base64: Buffer.from(res.data ?? []).toString("base64"),
        },
      };
    } catch (error) {
      return { ok: false, error: String(error?.message ?? error) };
    }
  });
  ipcMain.handle("app:delete-photo", async (_event, request = {}) => {
    try {
      const res = await invokeDeletePhoto(client, {
        photoId: request.photoId ?? "",
      });
      return { ok: true, data: res };
    } catch (error) {
      return { ok: false, error: String(error?.message ?? error) };
    }
  });
  ipcMain.handle("app:move-to-rejected", async (_event, request = {}) => {
    try {
      const res = await invokeMoveToRejected(client, {
        photoId: request.photoId ?? "",
      });
      return { ok: true, data: res };
    } catch (error) {
      return { ok: false, error: String(error?.message ?? error) };
    }
  });
  ipcMain.handle("app:get-metadata", async (_event, request = {}) => {
    try {
      const res = await invokeGetMetadata(client, {
        photoId: request.photoId ?? "",
      });
      return { ok: true, data: res };
    } catch (error) {
      return { ok: false, error: String(error?.message ?? error) };
    }
  });
}

async function createWindow() {
  const win = new BrowserWindow({
    width: 1400,
    height: 900,
    webPreferences: {
      contextIsolation: true,
      nodeIntegration: false,
      preload: path.resolve(__dirname, "preload.cjs"),
    },
  });

  await win.loadFile(path.resolve(__dirname, "index.html"));
}

app.whenReady().then(() => {
  createRpcBridge();
  createWindow();
});
