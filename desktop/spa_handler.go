package main

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

const wailsBridgeSnippet = `<script>
(function() {
  try {
    window.go = window.go || {};
    window.go.main = window.go.main || {};
    window.go.main.DeskWindowManager = {
      ExchangeToken: function(t) {
        if (window.wails && window.wails.Call && window.wails.Call.ByName) {
          return window.wails.Call.ByName("main.DeskWindowManager.ExchangeToken", t);
        }
        return Promise.reject(new Error("Wails runtime not yet initialized"));
      },
      GetMonitors: function() {
        if (window.wails && window.wails.Call && window.wails.Call.ByName) {
          return window.wails.Call.ByName("main.DeskWindowManager.GetMonitors");
        }
        return Promise.resolve([]);
      },
      GetScreens: function() {
        if (window.wails && window.wails.Call && window.wails.Call.ByName) {
          return window.wails.Call.ByName("main.DeskWindowManager.GetScreens");
        }
        return Promise.resolve([]);
      },
      GetOpenWindowIDs: function() {
        if (window.wails && window.wails.Call && window.wails.Call.ByName) {
          return window.wails.Call.ByName("main.DeskWindowManager.GetOpenWindowIDs");
        }
        return Promise.resolve([]);
      },
      SpawnWindow: function(o) {
        if (window.wails && window.wails.Call && window.wails.Call.ByName) {
          return window.wails.Call.ByName("main.DeskWindowManager.SpawnWindow", o);
        }
        return Promise.reject(new Error("Wails runtime not yet initialized"));
      },
      CloseWindow: function(id) {
        if (window.wails && window.wails.Call && window.wails.Call.ByName) {
          return window.wails.Call.ByName("main.DeskWindowManager.CloseWindow", id);
        }
        return Promise.reject(new Error("Wails runtime not yet initialized"));
      },
      RelayMessage: function(c, p) {
        if (window.wails && window.wails.Call && window.wails.Call.ByName) {
          return window.wails.Call.ByName("main.DeskWindowManager.RelayMessage", c, p);
        }
        return Promise.reject(new Error("Wails runtime not yet initialized"));
      },
      GetHeartbeat: function() {
        if (window.wails && window.wails.Call && window.wails.Call.ByName) {
          return window.wails.Call.ByName("main.DeskWindowManager.GetHeartbeat");
        }
        return Promise.resolve({ windowCount: 1, openWindowIds: ['win_main'], screenCount: 1, vaultTokenCount: 0, timestamp: Date.now() });
      }
    };
  } catch (e) {
    console.error('[WailsBridge] Init error:', e);
  }
})();
</script>
<script type="module">
  try {
    const runtime = await import('/wails/runtime.js');
    window.wails = window.wails || runtime;
  } catch(e){}
</script>
</head>`

// SPAHandler serves static files from an fs.FS, falling back to index.html
// for client-side routing paths (e.g. /view/page/:slug, /workspace).
type SPAHandler struct {
	fsys          fs.FS
	fileServer    http.Handler
	indexBytes    []byte
	customHandler func(w http.ResponseWriter, r *http.Request) bool
}

// NewSPAHandler creates a new SPAHandler backed by the provided filesystem.
func NewSPAHandler(fsys fs.FS) *SPAHandler {
	var indexData []byte
	if fsys != nil {
		if data, err := fs.ReadFile(fsys, "index.html"); err == nil {
			// Inject bridge script before </head> if present
			content := string(data)
			if strings.Contains(content, "</head>") {
				content = strings.Replace(content, "</head>", wailsBridgeSnippet, 1)
			}
			indexData = []byte(content)
		}
	}

	h := &SPAHandler{
		fsys:       fsys,
		fileServer: http.FileServer(http.FS(fsys)),
		indexBytes: indexData,
	}
	initVerifyHandler(h)
	return h
}

func (h *SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.customHandler != nil && h.customHandler(w, r) {
		return
	}

	if h.fsys == nil {
		http.NotFound(w, r)
		return
	}

	cleanPath := path.Clean(r.URL.Path)
	cleanPath = strings.TrimPrefix(cleanPath, "/")

	if cleanPath == "" || cleanPath == "." {
		h.serveIndex(w, r)
		return
	}

	// Check if the requested path corresponds to an existing file
	f, err := h.fsys.Open(cleanPath)
	if err == nil {
		stat, statErr := f.Stat()
		_ = f.Close()
		if statErr == nil && !stat.IsDir() {
			// Serve the real static asset (e.g. .js, .css, .svg, .png)
			h.fileServer.ServeHTTP(w, r)
			return
		}
	}

	// Route does not match a physical asset; serve index.html for SPA routing
	h.serveIndex(w, r)
}

func (h *SPAHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
	if len(h.indexBytes) == 0 {
		http.Error(w, "index.html not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(h.indexBytes)
}
