const fs = require('fs');
const path = require('path');

const targetPath = path.resolve(__dirname, '../apps/desktop/index.html');
const content = fs.readFileSync(targetPath, 'utf8');

const scriptIndex = content.indexOf('<script>');
if (scriptIndex === -1) {
  console.error("No <script> tag found.");
  process.exit(1);
}

const jsPart = content.substring(scriptIndex);

const newUI = `<!doctype html>
<html lang="ko">
  <head>
    <meta charset="UTF-8" />
    <meta http-equiv="X-UA-Compatible" content="IE=edge" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Photo Viewer - Adobe Bridge Style</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">
    <style>
      :root {
        color-scheme: dark;
        --bg-base: #0f1012;
        --bg-panel: rgba(22, 24, 28, 0.65);
        --bg-card: rgba(30, 33, 39, 0.5);
        --text-main: #f0f3f6;
        --text-muted: #8b95a5;
        --accent: #3b82f6;
        --accent-hover: #4f8cf6;
        --border-color: rgba(255, 255, 255, 0.08);
      }
      
      * { box-sizing: border-box; }
      
      body {
        margin: 0;
        font-family: 'Inter', system-ui, sans-serif;
        background-color: var(--bg-base);
        background-image: 
          radial-gradient(ellipse at top left, rgba(59, 130, 246, 0.08), transparent 40%),
          radial-gradient(ellipse at bottom right, rgba(139, 92, 246, 0.08), transparent 40%);
        background-attachment: fixed;
        color: var(--text-main);
        overflow: hidden; /* Prevent body scroll */
      }
      
      /* Glassmorphism utility */
      .glass {
        background: var(--bg-panel);
        backdrop-filter: blur(16px);
        -webkit-backdrop-filter: blur(16px);
        border: 1px solid var(--border-color);
      }

      .shell {
        display: grid;
        grid-template-rows: 56px 1fr 36px;
        height: 100vh;
      }
      
      /* Header */
      header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 0 24px;
        border-bottom: 1px solid var(--border-color);
        background: var(--bg-panel);
        backdrop-filter: blur(12px);
        z-index: 10;
      }
      
      .brand {
        font-size: 16px;
        font-weight: 600;
        letter-spacing: -0.01em;
        display: flex;
        align-items: center;
        gap: 12px;
      }
      .brand-icon {
        width: 24px;
        height: 24px;
        background: linear-gradient(135deg, #3b82f6, #8b5cf6);
        border-radius: 6px;
        display: flex;
        align-items: center;
        justify-content: center;
        color: white;
        font-size: 14px;
        font-weight: bold;
      }

      .header-right {
        display: flex;
        align-items: center;
        gap: 16px;
        font-size: 13px;
      }

      /* Main Layout */
      main {
        display: grid;
        grid-template-columns: 280px 1fr 320px;
        overflow: hidden;
      }
      
      .sidebar {
        border-right: 1px solid var(--border-color);
        display: flex;
        flex-direction: column;
        overflow-y: auto;
        padding: 20px;
        gap: 24px;
      }
      
      .right-panel {
        border-left: 1px solid var(--border-color);
        display: flex;
        flex-direction: column;
        overflow-y: auto;
        padding: 20px;
        gap: 20px;
        background: rgba(15, 16, 18, 0.4);
      }
      
      .center-area {
        display: flex;
        flex-direction: column;
        background: rgba(10, 11, 13, 0.7);
        position: relative;
      }

      /* Sections & Titles */
      .section-title {
        font-size: 12px;
        text-transform: uppercase;
        letter-spacing: 0.08em;
        color: var(--text-muted);
        font-weight: 600;
        margin-bottom: 12px;
      }

      /* Forms & Controls */
      input {
        width: 100%;
        background: rgba(255, 255, 255, 0.04);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 8px;
        color: var(--text-main);
        padding: 10px 12px;
        font-family: inherit;
        font-size: 13px;
        transition: all 0.2s;
        outline: none;
      }
      input:focus {
        border-color: var(--accent);
        background: rgba(255, 255, 255, 0.08);
      }
      
      .btn {
        background: rgba(255, 255, 255, 0.08);
        color: var(--text-main);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 8px;
        padding: 8px 14px;
        font-family: inherit;
        font-size: 13px;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
      }
      .btn:hover:not(:disabled) {
        background: rgba(255, 255, 255, 0.12);
        border-color: rgba(255, 255, 255, 0.2);
      }
      .btn-primary {
        background: var(--accent);
        color: white;
        border-color: transparent;
      }
      .btn-primary:hover:not(:disabled) {
        background: var(--accent-hover);
        box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
      }
      .btn:disabled {
        opacity: 0.5;
        cursor: not-allowed;
      }

      /* Library tools */
      .library-tools {
        display: flex;
        flex-direction: column;
        gap: 12px;
      }
      .toolbar-hstack {
        display: flex;
        gap: 8px;
      }

      .dropzone {
        border: 2px dashed rgba(255, 255, 255, 0.12);
        border-radius: 12px;
        padding: 32px 20px;
        text-align: center;
        font-size: 13px;
        color: var(--text-muted);
        transition: all 0.2s;
        background: rgba(255, 255, 255, 0.02);
      }
      .dropzone.active {
        border-color: var(--accent);
        background: rgba(59, 130, 246, 0.08);
        color: var(--accent);
      }

      /* Main Grid */
      .grid-toolbar {
        padding: 16px 24px;
        border-bottom: 1px solid var(--border-color);
        display: flex;
        justify-content: space-between;
        align-items: center;
        background: rgba(15, 16, 18, 0.8);
        backdrop-filter: blur(8px);
        z-index: 2;
      }
      .grid-title {
        font-weight: 500;
        font-size: 14px;
      }
      
      .grid-container {
        flex: 1;
        overflow-y: auto;
        padding: 24px;
      }

      .grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
        gap: 20px;
      }

      /* Photo Cards */
      .photo {
        border-radius: 12px;
        padding: 10px;
        background: var(--bg-card);
        border: 1px solid transparent;
        transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
        cursor: pointer;
        position: relative;
      }
      .photo:hover {
        transform: translateY(-4px);
        background: rgba(255, 255, 255, 0.04);
        border-color: rgba(255, 255, 255, 0.1);
        box-shadow: 0 8px 24px rgba(0, 0, 0, 0.2);
      }
      .photo.entering {
        animation: scaleIn 0.35s cubic-bezier(0.175, 0.885, 0.32, 1.275);
      }
      .photo.selected {
        background: rgba(59, 130, 246, 0.1);
        border-color: var(--accent);
        box-shadow: 0 4px 16px rgba(59, 130, 246, 0.2);
      }
      
      .photo img {
        width: 100%;
        aspect-ratio: 3 / 2;
        object-fit: cover;
        border-radius: 8px;
        background: #08090a;
        box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.05);
      }
      
      .photo-name {
        font-size: 13px;
        font-weight: 500;
        margin-top: 12px;
        margin-bottom: 4px;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        color: var(--text-main);
      }
      .photo-meta {
        font-size: 11px;
        color: var(--text-muted);
        display: flex;
        justify-content: space-between;
      }

      /* Selection Preview */
      .preview-box {
        aspect-ratio: 3 / 2;
        border-radius: 12px;
        overflow: hidden;
        background: #050506;
        border: 1px solid var(--border-color);
        display: flex;
        align-items: center;
        justify-content: center;
      }
      .preview-box img {
        max-width: 100%;
        max-height: 100%;
        object-fit: contain;
      }

      /* Metadata */
      .meta-dl {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 8px 16px;
        font-size: 12px;
      }
      .meta-dl div {
        display: contents;
      }
      .meta-dl strong {
        color: var(--text-muted);
        font-weight: 500;
      }
      .meta-dl span {
        text-align: right;
        color: var(--text-main);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
      }

      /* Footer & Logs */
      footer {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 0 24px;
        font-size: 12px;
        border-top: 1px solid var(--border-color);
        background: var(--bg-panel);
        color: var(--text-muted);
      }

      .log-panel {
        position: fixed;
        left: 20px;
        bottom: 56px;
        width: 400px;
        height: 250px;
        background: rgba(15, 16, 18, 0.95);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 12px;
        box-shadow: 0 12px 40px rgba(0, 0, 0, 0.5);
        display: flex;
        flex-direction: column;
        z-index: 40;
        backdrop-filter: blur(20px);
        overflow: hidden;
        opacity: 0;
        pointer-events: none;
        transform: translateY(10px);
        transition: all 0.2s;
      }
      .log-panel.show {
        opacity: 1;
        pointer-events: auto;
        transform: translateY(0);
      }
      .log-header {
        padding: 12px 16px;
        border-bottom: 1px solid var(--border-color);
        display: flex;
        justify-content: space-between;
        align-items: center;
        font-size: 12px;
        font-weight: 600;
      }
      .log-list {
        flex: 1;
        overflow-y: auto;
        padding: 12px 16px;
        font-family: 'Consolas', 'Cascadia Code', monospace;
        font-size: 11px;
        line-height: 1.6;
        color: #a1a1aa;
      }
      .log-item { margin-bottom: 6px; word-break: break-all; }
      .log-time { color: var(--accent); margin-right: 8px; }

      /* Fullscreen Viewer */
      .viewer {
        position: fixed;
        inset: 0;
        background: rgba(0, 0, 0, 0.95);
        z-index: 100;
        display: flex;
        flex-direction: column;
        opacity: 0;
        pointer-events: none;
        transition: opacity 0.25s;
        backdrop-filter: blur(8px);
      }
      .viewer.open {
        opacity: 1;
        pointer-events: auto;
      }
      
      .viewer-bar {
        height: 64px;
        padding: 0 24px;
        display: flex;
        justify-content: space-between;
        align-items: center;
        background: linear-gradient(to bottom, rgba(0,0,0,0.8), transparent);
        color: white;
        position: absolute;
        top: 0; left: 0; right: 0;
        z-index: 101;
      }
      
      .viewer-stage {
        flex: 1;
        display: flex;
        align-items: center;
        justify-content: center;
        overflow: hidden;
        cursor: grab;
      }
      .viewer-stage:active { cursor: grabbing; }
      
      .viewer-image {
        max-width: 100vw;
        max-height: 100vh;
        object-fit: contain;
        transform-origin: center center;
        transition: transform 0.1s;
        box-shadow: 0 0 100px rgba(0,0,0,0.5);
      }

      /* Helpers */
      .text-accent { color: var(--accent); }
      .muted-text { color: var(--text-muted); }
      
      @keyframes scaleIn {
        from { opacity: 0; transform: scale(0.9) translateY(10px); }
        to { opacity: 1; transform: scale(1) translateY(0); }
      }
      
      /* Scrollbars */
      ::-webkit-scrollbar { width: 8px; height: 8px; }
      ::-webkit-scrollbar-track { background: transparent; }
      ::-webkit-scrollbar-thumb { background: rgba(255, 255, 255, 0.1); border-radius: 4px; }
      ::-webkit-scrollbar-thumb:hover { background: rgba(255, 255, 255, 0.2); }
    </style>
  </head>
  <body>
    <div class="shell">
      <header>
        <div class="brand">
          <div class="brand-icon">P</div>
          PhotoBridge
        </div>
        <div class="header-right">
          <div id="health" class="muted-text">Checking system...</div>
          <button class="btn" id="toggleLogBtn">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="4 17 10 11 4 5"></polyline><line x1="12" y1="19" x2="20" y2="19"></line></svg>
            Logs
          </button>
        </div>
      </header>
      
      <main>
        <aside class="sidebar glass">
          <div>
            <div class="section-title">Source Library</div>
            <div class="library-tools">
              <input id="libraryRoot" placeholder="e.g. C:\\workspace\\photos" />
              <div class="toolbar-hstack">
                <button id="pickBtn" class="btn" style="flex:1">Browse...</button>
                <button id="scanBtn" class="btn btn-primary" style="flex:1">Scan</button>
              </div>
            </div>
          </div>
          
          <div>
            <div class="section-title">Import</div>
            <div id="dropzone" class="dropzone">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="margin-bottom: 8px; opacity: 0.5;">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
                <polyline points="17 8 12 3 7 8"></polyline>
                <line x1="12" y1="3" x2="12" y2="15"></line>
              </svg>
              <br/>
              Drag & Drop folders here
            </div>
            <div id="scanStatus" class="muted-text" style="font-size: 12px; margin-top: 12px; text-align: center;">Waiting for input</div>
          </div>
        </aside>
        
        <section class="center-area">
          <div class="grid-toolbar">
            <div class="grid-title">All Photos</div>
            <div class="toolbar-hstack">
              <!-- Placeholder for future view options -->
              <span class="muted-text" style="font-size:12px;">Sorted by Capture Date</span>
            </div>
          </div>
          <div class="grid-container" id="gridContainer">
            <div class="grid" id="grid">
              <div class="muted-text" style="grid-column: 1/-1; text-align: center; padding: 40px;">
                No folder selected. Scan a directory to view photos.
              </div>
            </div>
          </div>
        </section>
        
        <aside class="right-panel glass">
          <div>
            <div class="section-title">Preview</div>
            <div class="preview-box">
              <img id="selectionPreview" alt="No image selected" style="opacity: 0.2" />
            </div>
          </div>
          
          <div>
            <div class="section-title">Metadata</div>
            <div id="exifStatus" class="muted-text" style="font-size: 12px; margin-bottom: 12px;">Select a photo</div>
            <div id="exifList" class="meta-dl">
              <!-- Exif rows injected here -->
            </div>
          </div>
        </aside>
      </main>
      
      <footer>
        <div id="footer">Waiting for gRPC backend...</div>
        <div>PhotoBridge MVP</div>
      </footer>
      
      <!-- Overlaid Panels -->
      <div class="log-panel" id="logPanel">
        <div class="log-header">
          Diagnostic Logs
          <button id="clearLogBtn" class="btn" style="padding: 4px 8px; font-size: 11px;">Clear</button>
        </div>
        <div id="logList" class="log-list"></div>
      </div>
      
      <div id="viewer" class="viewer" aria-hidden="true">
        <div class="viewer-bar">
          <div id="viewerTitle" style="font-weight: 500; font-size: 14px; text-shadow: 0 2px 4px rgba(0,0,0,0.5);">Image Viewer</div>
          <button id="closeViewerBtn" class="btn" style="background: rgba(255,255,255,0.1); border:none;">Close (Esc)</button>
        </div>
        <div class="viewer-stage" id="viewerStage">
          <img id="viewerImage" class="viewer-image" alt="" />
        </div>
      </div>
    </div>
`;

const updatedContent = newUI + jsPart;
fs.writeFileSync(targetPath, updatedContent, 'utf8');
console.log("Successfully updated index.html UI!");
