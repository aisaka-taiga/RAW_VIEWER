import { app, BrowserWindow, dialog, ipcMain } from "electron";
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

function invokeListPhotos(client, request) {
  return new Promise((resolve, reject) => {
    client.ListPhotos(request, (err, res) => {
      if (err) {
        reject(err);
        return;
      }
      resolve(res);
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
      const res = await invokeListPhotos(client, {
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

  ipcMain.handle("app:abort-thumbnails", async () => {
    activeThumbnailCalls.forEach((call, id) => {
      if (parseThumbnailSize(id) >= 1024) {
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

    try {
      const res = await new Promise((resolve, reject) => {
        const call = client.GetThumbnail(
          {
            photoId: request.photoId ?? "",
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
