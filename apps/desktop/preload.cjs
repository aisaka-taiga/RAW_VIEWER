const { contextBridge, ipcRenderer } = require("electron");

contextBridge.exposeInMainWorld("photoViewer", {
  chooseFolder: () => ipcRenderer.invoke("app:choose-folder"),
  health: () => ipcRenderer.invoke("app:health"),
  scanFolder: (request) => ipcRenderer.invoke("app:scan-folder", request),
  scanFolderStream: (request) => ipcRenderer.invoke("app:scan-folder-stream", request),
  listPhotos: (request) => ipcRenderer.invoke("app:list-photos", request),
  getThumbnail: (request) => ipcRenderer.invoke("app:get-thumbnail", request),
  deletePhoto: (request) => ipcRenderer.invoke("app:delete-photo", request),
  moveToRejected: (request) => ipcRenderer.invoke("app:move-to-rejected", request),
  getMetadata: (request) => ipcRenderer.invoke("app:get-metadata", request),
  abortThumbnails: () => ipcRenderer.invoke("app:abort-thumbnails"),
  onScanProgress: (handler) => {
    ipcRenderer.on("app:scan-folder-progress", (_event, payload) => handler(payload));
  },
});
