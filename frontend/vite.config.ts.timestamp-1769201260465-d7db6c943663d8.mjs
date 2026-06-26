// vite.config.ts
import { defineConfig, loadEnv } from "file:///Users/eganpj/GitHub/semlayer/node_modules/vite/dist/node/index.js";
import react from "file:///Users/eganpj/GitHub/semlayer/node_modules/@vitejs/plugin-react/dist/index.js";
import path from "path";
import { fileURLToPath } from "url";
var __vite_injected_original_import_meta_url = "file:///Users/eganpj/GitHub/semlayer/frontend/vite.config.ts";
var __filename = fileURLToPath(__vite_injected_original_import_meta_url);
var __dirname = path.dirname(__filename);
var vite_config_default = defineConfig(async ({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  console.log("[vite.config] backendTarget:", env.VITE_BACKEND_TARGET || process.env.VITE_BACKEND_TARGET || "http://localhost:8080");
  console.log("[vite.config] apiBase:", env.VITE_API_BASE_URL || process.env.VITE_API_BASE_URL || env.VITE_BACKEND_TARGET || process.env.VITE_BACKEND_TARGET || "http://localhost:8080");
  const plugins = [react()];
  try {
    let visualizer;
    try {
      ({ visualizer } = await import("vite-plugin-visualizer"));
    } catch (e) {
      ({ visualizer } = await import("file:///Users/eganpj/GitHub/semlayer/node_modules/rollup-plugin-visualizer/dist/plugin/index.js"));
    }
    plugins.push(
      visualizer({ filename: "dist/stats.html", open: false, gzipSize: true, brotliSize: true, json: true, template: "sunburst" })
    );
  } catch (e) {
  }
  try {
    const monacoPluginModule = await import("file:///Users/eganpj/GitHub/semlayer/node_modules/vite-plugin-monaco-editor/dist/index.js");
    const monacoFactory = monacoPluginModule && (monacoPluginModule.default || monacoPluginModule);
    if (typeof monacoFactory === "function") {
      plugins.push(monacoFactory({}));
    }
  } catch (e) {
  }
  return {
    plugins,
    resolve: {
      dedupe: ["react", "react-dom"],
      alias: {
        "monaco-yaml": path.resolve(__dirname, "src/shims/monaco-yaml-shim.ts"),
        "@": path.resolve(__dirname, "src"),
        "@internal": path.resolve(__dirname, "../internal"),
        react: path.resolve(__dirname, "../node_modules/react"),
        "react-dom": path.resolve(__dirname, "../node_modules/react-dom")
      }
    },
    server: {
      host: "0.0.0.0",
      port: Number(process.env.PORT) || 5173,
      strictPort: true,
      headers: {
        "Cache-Control": "no-cache, no-store, must-revalidate",
        "Pragma": "no-cache",
        "Expires": "0"
      },
      // Hardened proxy settings:
      // - Dedicated `/api/profiler` route with long timeouts for long-running profiler jobs.
      // - General `/api` forwarding with reasonable timeouts.
      // - xfwd/changeOrigin to preserve upstream expectations.
      // Toggle proxying in development by setting VITE_USE_PROXY=true in your `.env.local`.
      proxy: (() => {
        const useProxy = String(env.VITE_USE_PROXY || process.env.VITE_USE_PROXY || "false").toLowerCase() === "true";
        console.log("[vite.config] VITE_USE_PROXY:", env.VITE_USE_PROXY, "useProxy:", useProxy);
        if (!useProxy) return {};
        const backendTarget = env.VITE_BACKEND_TARGET || process.env.VITE_BACKEND_TARGET || "http://localhost:8080";
        const apiBase = env.VITE_API_BASE_URL || process.env.VITE_API_BASE_URL || backendTarget;
        return {
          // Special-case profiler endpoints which may run for a long time.
          "/api/profiler": {
            target: backendTarget,
            changeOrigin: true,
            secure: false,
            xfwd: true,
            ws: false,
            // profiler endpoints are not websockets; disable ws for stability
            // Keep long timeouts for profiling runs (10 minutes)
            timeout: 10 * 60 * 1e3,
            proxyTimeout: 10 * 60 * 1e3,
            // Provide helpful logging on proxy errors to ease debugging
            onError(err, _req, res) {
              console.error("[vite proxy] /api/profiler error", err && err.message ? err.message : err);
              try {
                if (!res.headersSent) {
                  res.writeHead && res.writeHead(502);
                  res.end && res.end("Proxy error");
                }
              } catch (e) {
              }
            }
          },
          // Generic API proxy. Use this when you want the frontend origin to mask backend origins (avoid configuring CORS).
          "/api/": {
            target: apiBase,
            changeOrigin: true,
            secure: false,
            xfwd: true,
            ws: false,
            // Shorter timeout for regular API calls (2 minutes)
            timeout: 2 * 60 * 1e3,
            proxyTimeout: 2 * 60 * 1e3,
            // Helpful dev-time logging: print proxied request details so we can
            // confirm the dev server is forwarding /api calls to the intended target.
            configure(proxy, options) {
              try {
                proxy.on && proxy.on("proxyReq", (_proxyReq, req) => {
                  console.log("[vite proxy] proxying request", { method: req.method, url: req.url, target: options.target });
                });
                proxy.on && proxy.on("error", (err, req) => {
                  console.error("[vite proxy] proxy error", err && err.message ? err.message : err, { url: req && req.url });
                });
              } catch (e) {
              }
            },
            onError(err, _req, res) {
              console.error("[vite proxy] /api error", err && err.message ? err.message : err);
              try {
                if (!res.headersSent) {
                  res.writeHead && res.writeHead(502);
                  res.end && res.end("Proxy error");
                }
              } catch (e) {
              }
            }
          },
          // Proxy GraphQL endpoint used by the app's Apollo client when running locally.
          // When VITE_GRAPHQL_ENDPOINT uses a relative path '/v1/graphql' the dev server
          // should forward it to the configured backend target so the browser does not
          // inadvertently return index.html (SPA fallback) for GraphQL requests.
          "/v1/graphql": {
            target: apiBase,
            changeOrigin: true,
            secure: false,
            xfwd: true,
            // GraphQL may use websockets for subscriptions; allow ws forwarding.
            ws: true,
            timeout: 60 * 1e3,
            proxyTimeout: 60 * 1e3,
            configure(proxy, options) {
              try {
                proxy.on && proxy.on("proxyReq", (_proxyReq, req) => {
                  console.log("[vite proxy] proxying GraphQL", { method: req.method, url: req.url, target: options.target });
                });
                proxy.on && proxy.on("error", (err, req) => {
                  console.error("[vite proxy] graphql proxy error", err && err.message ? err.message : err, { url: req && req.url });
                });
              } catch (e) {
              }
            },
            onError(err, _req, res) {
              console.error("[vite proxy] /v1/graphql error", err && err.message ? err.message : err);
              try {
                if (!res.headersSent) {
                  res.writeHead && res.writeHead(502);
                  res.end && res.end("Proxy error");
                }
              } catch (e) {
              }
            }
          }
        };
      })()
    },
    optimizeDeps: {
      include: [
        "@mui/material",
        "@mui/x-date-pickers",
        "@mui/x-date-pickers/LocalizationProvider",
        "@mui/x-date-pickers/AdapterDateFns",
        "@mui/x-date-pickers/DatePicker",
        "@mui/icons-material",
        "chart.js",
        "chartjs-adapter-date-fns",
        "react-chartjs-2",
        "react-diff-view",
        "diff"
      ],
      exclude: [
        "echarts",
        "zrender",
        "echarts-for-react",
        // Common problematic dependencies that may cause issues with Vite's dep optimizer
        "monaco-editor",
        "monaco-yaml",
        "@monaco-editor/react",
        // Ant Design v5 patch for React 19 may be incompatible with current React 18 dev installs
        // and can confuse Vite's optimizer; keep it excluded so the optimizer won't pre-bundle it.
        "@ant-design/v5-patch-for-react-19"
        // Add any other dependencies that cause optimization issues
      ]
    },
    // Note: monaco-yaml is dynamically imported at runtime. Avoid pre-bundling it
    // to prevent dependency resolution issues with certain language server packages.
    build: {
      chunkSizeWarningLimit: 2300,
      rollupOptions: {
        external: ["echarts", "zrender", "echarts-for-react"],
        output: {
          globals: { echarts: "echarts", zrender: "zrender", "echarts-for-react": "EChartsReact" },
          manualChunks(id) {
            if (!id.includes("node_modules")) return void 0;
            if (id.includes("node_modules/react")) return "vendor-react";
            if (id.includes("node_modules/react-dom")) return "vendor-react";
            if (id.includes("node_modules/reactflow")) return "vendor-reactflow";
            if (id.includes("node_modules") && /(?:^|\/)(@(apache\-)?echarts|echarts|zrender|echarts-for-react|react-echarts)/.test(id)) {
              return "vendor-echarts";
            }
            if (id.includes("node_modules/react-chartjs-2") || id.includes("node_modules/chart.js")) return "vendor-chartjs";
            if (id.includes("node_modules/recharts")) return "vendor-recharts";
            if (id.includes("node_modules/react-diff-view") || id.includes("node_modules/react-diff-viewer")) return "vendor-diff";
            if (id.includes("node_modules/monaco-editor/esm/vs/language/")) return "vendor-monaco-languages";
            if (id.includes("node_modules/monaco-editor/esm/vs/basic-languages/")) return "vendor-monaco-basic-languages";
            if (id.includes("node_modules/monaco-editor")) return "vendor-monaco-core";
            if (id.includes("node_modules/highlight.js")) return "vendor-highlight";
            if (id.includes("node_modules/refractor")) return "vendor-refractor";
            if (id.includes("node_modules/@mui/material")) return "vendor-mui-material";
            if (id.includes("node_modules/@mui/icons-material")) return "vendor-mui-icons";
            if (id.includes("node_modules/@mui/lab")) return "vendor-mui-lab";
            if (id.includes("node_modules/@emotion")) return "vendor-emotion";
            if (id.includes("node_modules/@mui/")) return "vendor-mui-shared";
            const parts = id.split("node_modules/")[1].split("/");
            let pkg = parts[0];
            if (pkg && pkg.startsWith("@") && parts.length > 1) {
              pkg = `${pkg}/${parts[1]}`;
            }
            return `vendor-${pkg.replace("/", "_").replace("@", "")}`;
          }
        }
      },
      commonjsOptions: {
        transformMixedEsModules: true
      }
    }
  };
});
export {
  vite_config_default as default
};
//# sourceMappingURL=data:application/json;base64,ewogICJ2ZXJzaW9uIjogMywKICAic291cmNlcyI6IFsidml0ZS5jb25maWcudHMiXSwKICAic291cmNlc0NvbnRlbnQiOiBbImNvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9kaXJuYW1lID0gXCIvVXNlcnMvZWdhbnBqL0dpdEh1Yi9zZW1sYXllci9mcm9udGVuZFwiO2NvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9maWxlbmFtZSA9IFwiL1VzZXJzL2VnYW5wai9HaXRIdWIvc2VtbGF5ZXIvZnJvbnRlbmQvdml0ZS5jb25maWcudHNcIjtjb25zdCBfX3ZpdGVfaW5qZWN0ZWRfb3JpZ2luYWxfaW1wb3J0X21ldGFfdXJsID0gXCJmaWxlOi8vL1VzZXJzL2VnYW5wai9HaXRIdWIvc2VtbGF5ZXIvZnJvbnRlbmQvdml0ZS5jb25maWcudHNcIjsvKiBlc2xpbnQtZGlzYWJsZSBuby1yZXN0cmljdGVkLXN5bnRheCAqL1xuaW1wb3J0IHsgZGVmaW5lQ29uZmlnLCBsb2FkRW52IH0gZnJvbSAndml0ZSc7XG5pbXBvcnQgcmVhY3QgZnJvbSAnQHZpdGVqcy9wbHVnaW4tcmVhY3QnO1xuaW1wb3J0IHBhdGggZnJvbSAncGF0aCc7XG5pbXBvcnQgeyBmaWxlVVJMVG9QYXRoIH0gZnJvbSAndXJsJztcblxuLy8gX19kaXJuYW1lIHdvcmthcm91bmQgZm9yIEVTTVxuY29uc3QgX19maWxlbmFtZSA9IGZpbGVVUkxUb1BhdGgoaW1wb3J0Lm1ldGEudXJsKTtcbmNvbnN0IF9fZGlybmFtZSA9IHBhdGguZGlybmFtZShfX2ZpbGVuYW1lKTtcblxuZXhwb3J0IGRlZmF1bHQgZGVmaW5lQ29uZmlnKGFzeW5jICh7IG1vZGUgfSkgPT4ge1xuICAvLyBMb2FkIGVudmlyb25tZW50IHZhcmlhYmxlcyBzbyBzZXJ2ZXItc2lkZSBjb25maWcgKGxpa2UgcHJveHkpIHJlc3BlY3RzIC5lbnYubG9jYWxcbiAgY29uc3QgZW52ID0gbG9hZEVudihtb2RlLCBwcm9jZXNzLmN3ZCgpLCAnJyk7XG4gIC8vIExvZyByZXNvbHZlZCBiYWNrZW5kIHRhcmdldHMgdG8gYWlkIGRlYnVnZ2luZyBwcm94eSBpc3N1ZXNcbiAgY29uc29sZS5sb2coJ1t2aXRlLmNvbmZpZ10gYmFja2VuZFRhcmdldDonLCBlbnYuVklURV9CQUNLRU5EX1RBUkdFVCB8fCBwcm9jZXNzLmVudi5WSVRFX0JBQ0tFTkRfVEFSR0VUIHx8ICdodHRwOi8vbG9jYWxob3N0OjgwODAnKTtcbiAgY29uc29sZS5sb2coJ1t2aXRlLmNvbmZpZ10gYXBpQmFzZTonLCAoZW52LlZJVEVfQVBJX0JBU0VfVVJMIHx8IHByb2Nlc3MuZW52LlZJVEVfQVBJX0JBU0VfVVJMIHx8IGVudi5WSVRFX0JBQ0tFTkRfVEFSR0VUIHx8IHByb2Nlc3MuZW52LlZJVEVfQkFDS0VORF9UQVJHRVQgfHwgJ2h0dHA6Ly9sb2NhbGhvc3Q6ODA4MCcpKTtcbiAgY29uc3QgcGx1Z2luczogYW55W10gPSBbcmVhY3QoKV07XG5cbiAgLy8gVHJ5IHRvIGxvYWQgb3B0aW9uYWwgcGx1Z2lucyBpZiBpbnN0YWxsZWRcbiAgdHJ5IHtcbiAgICBsZXQgdmlzdWFsaXplcjogYW55O1xuICAgIHRyeSB7XG4gICAgICAoeyB2aXN1YWxpemVyIH0gPSBhd2FpdCBpbXBvcnQoJ3ZpdGUtcGx1Z2luLXZpc3VhbGl6ZXInKSk7XG4gICAgfSBjYXRjaCAoZSkge1xuICAgICAgKHsgdmlzdWFsaXplciB9ID0gYXdhaXQgaW1wb3J0KCdyb2xsdXAtcGx1Z2luLXZpc3VhbGl6ZXInKSk7XG4gICAgfVxuICAgIHBsdWdpbnMucHVzaChcbiAgICAgIHZpc3VhbGl6ZXIoeyBmaWxlbmFtZTogJ2Rpc3Qvc3RhdHMuaHRtbCcsIG9wZW46IGZhbHNlLCBnemlwU2l6ZTogdHJ1ZSwgYnJvdGxpU2l6ZTogdHJ1ZSwganNvbjogdHJ1ZSwgdGVtcGxhdGU6ICdzdW5idXJzdCcgfSlcbiAgICApO1xuICB9IGNhdGNoIChlKSB7IH1cblxuICB0cnkge1xuICAgIGNvbnN0IG1vbmFjb1BsdWdpbk1vZHVsZTogYW55ID0gYXdhaXQgaW1wb3J0KCd2aXRlLXBsdWdpbi1tb25hY28tZWRpdG9yJyk7XG4gICAgY29uc3QgbW9uYWNvRmFjdG9yeTogYW55ID0gbW9uYWNvUGx1Z2luTW9kdWxlICYmIChtb25hY29QbHVnaW5Nb2R1bGUuZGVmYXVsdCB8fCBtb25hY29QbHVnaW5Nb2R1bGUpO1xuICAgIGlmICh0eXBlb2YgbW9uYWNvRmFjdG9yeSA9PT0gJ2Z1bmN0aW9uJykge1xuICAgICAgcGx1Z2lucy5wdXNoKG1vbmFjb0ZhY3Rvcnkoe30pKTtcbiAgICB9XG4gIH0gY2F0Y2ggKGUpIHsgfVxuXG4gIHJldHVybiB7XG4gICAgcGx1Z2lucyxcbiAgICByZXNvbHZlOiB7XG4gICAgICBkZWR1cGU6IFsncmVhY3QnLCAncmVhY3QtZG9tJ10sXG4gICAgICBhbGlhczoge1xuICAgICAgICAnbW9uYWNvLXlhbWwnOiBwYXRoLnJlc29sdmUoX19kaXJuYW1lLCAnc3JjL3NoaW1zL21vbmFjby15YW1sLXNoaW0udHMnKSxcbiAgICAgICAgJ0AnOiBwYXRoLnJlc29sdmUoX19kaXJuYW1lLCAnc3JjJyksXG4gICAgICAgICdAaW50ZXJuYWwnOiBwYXRoLnJlc29sdmUoX19kaXJuYW1lLCAnLi4vaW50ZXJuYWwnKSxcbiAgICAgICAgcmVhY3Q6IHBhdGgucmVzb2x2ZShfX2Rpcm5hbWUsICcuLi9ub2RlX21vZHVsZXMvcmVhY3QnKSxcbiAgICAgICAgJ3JlYWN0LWRvbSc6IHBhdGgucmVzb2x2ZShfX2Rpcm5hbWUsICcuLi9ub2RlX21vZHVsZXMvcmVhY3QtZG9tJyksXG4gICAgICB9LFxuICAgIH0sXG4gICAgc2VydmVyOiB7XG4gICAgICBob3N0OiAnMC4wLjAuMCcsXG4gICAgICBwb3J0OiBOdW1iZXIocHJvY2Vzcy5lbnYuUE9SVCkgfHwgNTE3MyxcbiAgICAgIHN0cmljdFBvcnQ6IHRydWUsXG4gICAgICBoZWFkZXJzOiB7XG4gICAgICAgICdDYWNoZS1Db250cm9sJzogJ25vLWNhY2hlLCBuby1zdG9yZSwgbXVzdC1yZXZhbGlkYXRlJyxcbiAgICAgICAgJ1ByYWdtYSc6ICduby1jYWNoZScsXG4gICAgICAgICdFeHBpcmVzJzogJzAnLFxuICAgICAgfSxcbiAgICAgIC8vIEhhcmRlbmVkIHByb3h5IHNldHRpbmdzOlxuICAgICAgLy8gLSBEZWRpY2F0ZWQgYC9hcGkvcHJvZmlsZXJgIHJvdXRlIHdpdGggbG9uZyB0aW1lb3V0cyBmb3IgbG9uZy1ydW5uaW5nIHByb2ZpbGVyIGpvYnMuXG4gICAgICAvLyAtIEdlbmVyYWwgYC9hcGlgIGZvcndhcmRpbmcgd2l0aCByZWFzb25hYmxlIHRpbWVvdXRzLlxuICAgICAgLy8gLSB4ZndkL2NoYW5nZU9yaWdpbiB0byBwcmVzZXJ2ZSB1cHN0cmVhbSBleHBlY3RhdGlvbnMuXG4gICAgICAvLyBUb2dnbGUgcHJveHlpbmcgaW4gZGV2ZWxvcG1lbnQgYnkgc2V0dGluZyBWSVRFX1VTRV9QUk9YWT10cnVlIGluIHlvdXIgYC5lbnYubG9jYWxgLlxuICAgICAgcHJveHk6ICgoKTogUmVjb3JkPHN0cmluZywgYW55PiA9PiB7XG4gICAgICAgIGNvbnN0IHVzZVByb3h5ID0gU3RyaW5nKGVudi5WSVRFX1VTRV9QUk9YWSB8fCBwcm9jZXNzLmVudi5WSVRFX1VTRV9QUk9YWSB8fCAnZmFsc2UnKS50b0xvd2VyQ2FzZSgpID09PSAndHJ1ZSc7XG4gICAgICAgIGNvbnNvbGUubG9nKCdbdml0ZS5jb25maWddIFZJVEVfVVNFX1BST1hZOicsIGVudi5WSVRFX1VTRV9QUk9YWSwgJ3VzZVByb3h5OicsIHVzZVByb3h5KTtcbiAgICAgICAgaWYgKCF1c2VQcm94eSkgcmV0dXJuIHt9O1xuICAgICAgICBjb25zdCBiYWNrZW5kVGFyZ2V0ID0gKGVudi5WSVRFX0JBQ0tFTkRfVEFSR0VUIHx8IHByb2Nlc3MuZW52LlZJVEVfQkFDS0VORF9UQVJHRVQpIHx8ICdodHRwOi8vbG9jYWxob3N0OjgwODAnO1xuICAgICAgICBjb25zdCBhcGlCYXNlID0gKGVudi5WSVRFX0FQSV9CQVNFX1VSTCB8fCBwcm9jZXNzLmVudi5WSVRFX0FQSV9CQVNFX1VSTCkgfHwgYmFja2VuZFRhcmdldDtcblxuICAgICAgICByZXR1cm4ge1xuICAgICAgICAgIC8vIFNwZWNpYWwtY2FzZSBwcm9maWxlciBlbmRwb2ludHMgd2hpY2ggbWF5IHJ1biBmb3IgYSBsb25nIHRpbWUuXG4gICAgICAgICAgJy9hcGkvcHJvZmlsZXInOiB7XG4gICAgICAgICAgICB0YXJnZXQ6IGJhY2tlbmRUYXJnZXQsXG4gICAgICAgICAgICBjaGFuZ2VPcmlnaW46IHRydWUsXG4gICAgICAgICAgICBzZWN1cmU6IGZhbHNlLFxuICAgICAgICAgICAgeGZ3ZDogdHJ1ZSxcbiAgICAgICAgICAgIHdzOiBmYWxzZSwgLy8gcHJvZmlsZXIgZW5kcG9pbnRzIGFyZSBub3Qgd2Vic29ja2V0czsgZGlzYWJsZSB3cyBmb3Igc3RhYmlsaXR5XG4gICAgICAgICAgICAvLyBLZWVwIGxvbmcgdGltZW91dHMgZm9yIHByb2ZpbGluZyBydW5zICgxMCBtaW51dGVzKVxuICAgICAgICAgICAgdGltZW91dDogMTAgKiA2MCAqIDEwMDAsXG4gICAgICAgICAgICBwcm94eVRpbWVvdXQ6IDEwICogNjAgKiAxMDAwLFxuICAgICAgICAgICAgLy8gUHJvdmlkZSBoZWxwZnVsIGxvZ2dpbmcgb24gcHJveHkgZXJyb3JzIHRvIGVhc2UgZGVidWdnaW5nXG4gICAgICAgICAgICBvbkVycm9yKGVycjogYW55LCBfcmVxOiBhbnksIHJlczogYW55KSB7XG4gICAgICAgICAgICAgIC8vIGVzbGludC1kaXNhYmxlLW5leHQtbGluZSBuby1jb25zb2xlXG4gICAgICAgICAgICAgIGNvbnNvbGUuZXJyb3IoJ1t2aXRlIHByb3h5XSAvYXBpL3Byb2ZpbGVyIGVycm9yJywgZXJyICYmIGVyci5tZXNzYWdlID8gZXJyLm1lc3NhZ2UgOiBlcnIpO1xuICAgICAgICAgICAgICB0cnkge1xuICAgICAgICAgICAgICAgIGlmICghcmVzLmhlYWRlcnNTZW50KSB7XG4gICAgICAgICAgICAgICAgICByZXMud3JpdGVIZWFkICYmIHJlcy53cml0ZUhlYWQoNTAyKTtcbiAgICAgICAgICAgICAgICAgIHJlcy5lbmQgJiYgcmVzLmVuZCgnUHJveHkgZXJyb3InKTtcbiAgICAgICAgICAgICAgICB9XG4gICAgICAgICAgICAgIH0gY2F0Y2ggKGUpIHsgfVxuICAgICAgICAgICAgfSxcbiAgICAgICAgICB9LFxuXG4gICAgICAgICAgLy8gR2VuZXJpYyBBUEkgcHJveHkuIFVzZSB0aGlzIHdoZW4geW91IHdhbnQgdGhlIGZyb250ZW5kIG9yaWdpbiB0byBtYXNrIGJhY2tlbmQgb3JpZ2lucyAoYXZvaWQgY29uZmlndXJpbmcgQ09SUykuXG4gICAgICAgICAgJy9hcGkvJzoge1xuICAgICAgICAgICAgdGFyZ2V0OiBhcGlCYXNlLFxuICAgICAgICAgICAgY2hhbmdlT3JpZ2luOiB0cnVlLFxuICAgICAgICAgICAgc2VjdXJlOiBmYWxzZSxcbiAgICAgICAgICAgIHhmd2Q6IHRydWUsXG4gICAgICAgICAgICB3czogZmFsc2UsXG4gICAgICAgICAgICAvLyBTaG9ydGVyIHRpbWVvdXQgZm9yIHJlZ3VsYXIgQVBJIGNhbGxzICgyIG1pbnV0ZXMpXG4gICAgICAgICAgICB0aW1lb3V0OiAyICogNjAgKiAxMDAwLFxuICAgICAgICAgICAgcHJveHlUaW1lb3V0OiAyICogNjAgKiAxMDAwLFxuICAgICAgICAgICAgLy8gSGVscGZ1bCBkZXYtdGltZSBsb2dnaW5nOiBwcmludCBwcm94aWVkIHJlcXVlc3QgZGV0YWlscyBzbyB3ZSBjYW5cbiAgICAgICAgICAgIC8vIGNvbmZpcm0gdGhlIGRldiBzZXJ2ZXIgaXMgZm9yd2FyZGluZyAvYXBpIGNhbGxzIHRvIHRoZSBpbnRlbmRlZCB0YXJnZXQuXG4gICAgICAgICAgICBjb25maWd1cmUocHJveHk6IGFueSwgb3B0aW9uczogYW55KSB7XG4gICAgICAgICAgICAgIHRyeSB7XG4gICAgICAgICAgICAgICAgcHJveHkub24gJiYgcHJveHkub24oJ3Byb3h5UmVxJywgKF9wcm94eVJlcTogYW55LCByZXE6IGFueSkgPT4ge1xuICAgICAgICAgICAgICAgICAgLy8gZXNsaW50LWRpc2FibGUtbmV4dC1saW5lIG5vLWNvbnNvbGVcbiAgICAgICAgICAgICAgICAgIGNvbnNvbGUubG9nKCdbdml0ZSBwcm94eV0gcHJveHlpbmcgcmVxdWVzdCcsIHsgbWV0aG9kOiByZXEubWV0aG9kLCB1cmw6IHJlcS51cmwsIHRhcmdldDogb3B0aW9ucy50YXJnZXQgfSk7XG4gICAgICAgICAgICAgICAgfSk7XG4gICAgICAgICAgICAgICAgcHJveHkub24gJiYgcHJveHkub24oJ2Vycm9yJywgKGVycjogYW55LCByZXE6IGFueSkgPT4ge1xuICAgICAgICAgICAgICAgICAgLy8gZXNsaW50LWRpc2FibGUtbmV4dC1saW5lIG5vLWNvbnNvbGVcbiAgICAgICAgICAgICAgICAgIGNvbnNvbGUuZXJyb3IoJ1t2aXRlIHByb3h5XSBwcm94eSBlcnJvcicsIGVyciAmJiBlcnIubWVzc2FnZSA/IGVyci5tZXNzYWdlIDogZXJyLCB7IHVybDogcmVxICYmIHJlcS51cmwgfSk7XG4gICAgICAgICAgICAgICAgfSk7XG4gICAgICAgICAgICAgIH0gY2F0Y2ggKGUpIHsgfVxuICAgICAgICAgICAgfSxcbiAgICAgICAgICAgIG9uRXJyb3IoZXJyOiBhbnksIF9yZXE6IGFueSwgcmVzOiBhbnkpIHtcbiAgICAgICAgICAgICAgLy8gZXNsaW50LWRpc2FibGUtbmV4dC1saW5lIG5vLWNvbnNvbGVcbiAgICAgICAgICAgICAgY29uc29sZS5lcnJvcignW3ZpdGUgcHJveHldIC9hcGkgZXJyb3InLCBlcnIgJiYgZXJyLm1lc3NhZ2UgPyBlcnIubWVzc2FnZSA6IGVycik7XG4gICAgICAgICAgICAgIHRyeSB7XG4gICAgICAgICAgICAgICAgaWYgKCFyZXMuaGVhZGVyc1NlbnQpIHtcbiAgICAgICAgICAgICAgICAgIHJlcy53cml0ZUhlYWQgJiYgcmVzLndyaXRlSGVhZCg1MDIpO1xuICAgICAgICAgICAgICAgICAgcmVzLmVuZCAmJiByZXMuZW5kKCdQcm94eSBlcnJvcicpO1xuICAgICAgICAgICAgICAgIH1cbiAgICAgICAgICAgICAgfSBjYXRjaCAoZSkgeyB9XG4gICAgICAgICAgICB9LFxuICAgICAgICAgIH0sXG4gICAgICAgICAgLy8gUHJveHkgR3JhcGhRTCBlbmRwb2ludCB1c2VkIGJ5IHRoZSBhcHAncyBBcG9sbG8gY2xpZW50IHdoZW4gcnVubmluZyBsb2NhbGx5LlxuICAgICAgICAgIC8vIFdoZW4gVklURV9HUkFQSFFMX0VORFBPSU5UIHVzZXMgYSByZWxhdGl2ZSBwYXRoICcvdjEvZ3JhcGhxbCcgdGhlIGRldiBzZXJ2ZXJcbiAgICAgICAgICAvLyBzaG91bGQgZm9yd2FyZCBpdCB0byB0aGUgY29uZmlndXJlZCBiYWNrZW5kIHRhcmdldCBzbyB0aGUgYnJvd3NlciBkb2VzIG5vdFxuICAgICAgICAgIC8vIGluYWR2ZXJ0ZW50bHkgcmV0dXJuIGluZGV4Lmh0bWwgKFNQQSBmYWxsYmFjaykgZm9yIEdyYXBoUUwgcmVxdWVzdHMuXG4gICAgICAgICAgJy92MS9ncmFwaHFsJzoge1xuICAgICAgICAgICAgdGFyZ2V0OiBhcGlCYXNlLFxuICAgICAgICAgICAgY2hhbmdlT3JpZ2luOiB0cnVlLFxuICAgICAgICAgICAgc2VjdXJlOiBmYWxzZSxcbiAgICAgICAgICAgIHhmd2Q6IHRydWUsXG4gICAgICAgICAgICAvLyBHcmFwaFFMIG1heSB1c2Ugd2Vic29ja2V0cyBmb3Igc3Vic2NyaXB0aW9uczsgYWxsb3cgd3MgZm9yd2FyZGluZy5cbiAgICAgICAgICAgIHdzOiB0cnVlLFxuICAgICAgICAgICAgdGltZW91dDogNjAgKiAxMDAwLFxuICAgICAgICAgICAgcHJveHlUaW1lb3V0OiA2MCAqIDEwMDAsXG4gICAgICAgICAgICBjb25maWd1cmUocHJveHk6IGFueSwgb3B0aW9uczogYW55KSB7XG4gICAgICAgICAgICAgIHRyeSB7XG4gICAgICAgICAgICAgICAgcHJveHkub24gJiYgcHJveHkub24oJ3Byb3h5UmVxJywgKF9wcm94eVJlcTogYW55LCByZXE6IGFueSkgPT4ge1xuICAgICAgICAgICAgICAgICAgLy8gZXNsaW50LWRpc2FibGUtbmV4dC1saW5lIG5vLWNvbnNvbGVcbiAgICAgICAgICAgICAgICAgIGNvbnNvbGUubG9nKCdbdml0ZSBwcm94eV0gcHJveHlpbmcgR3JhcGhRTCcsIHsgbWV0aG9kOiByZXEubWV0aG9kLCB1cmw6IHJlcS51cmwsIHRhcmdldDogb3B0aW9ucy50YXJnZXQgfSk7XG4gICAgICAgICAgICAgICAgfSk7XG4gICAgICAgICAgICAgICAgcHJveHkub24gJiYgcHJveHkub24oJ2Vycm9yJywgKGVycjogYW55LCByZXE6IGFueSkgPT4ge1xuICAgICAgICAgICAgICAgICAgLy8gZXNsaW50LWRpc2FibGUtbmV4dC1saW5lIG5vLWNvbnNvbGVcbiAgICAgICAgICAgICAgICAgIGNvbnNvbGUuZXJyb3IoJ1t2aXRlIHByb3h5XSBncmFwaHFsIHByb3h5IGVycm9yJywgZXJyICYmIGVyci5tZXNzYWdlID8gZXJyLm1lc3NhZ2UgOiBlcnIsIHsgdXJsOiByZXEgJiYgcmVxLnVybCB9KTtcbiAgICAgICAgICAgICAgICB9KTtcbiAgICAgICAgICAgICAgfSBjYXRjaCAoZSkgeyB9XG4gICAgICAgICAgICB9LFxuICAgICAgICAgICAgb25FcnJvcihlcnI6IGFueSwgX3JlcTogYW55LCByZXM6IGFueSkge1xuICAgICAgICAgICAgICAvLyBlc2xpbnQtZGlzYWJsZS1uZXh0LWxpbmUgbm8tY29uc29sZVxuICAgICAgICAgICAgICBjb25zb2xlLmVycm9yKCdbdml0ZSBwcm94eV0gL3YxL2dyYXBocWwgZXJyb3InLCBlcnIgJiYgZXJyLm1lc3NhZ2UgPyBlcnIubWVzc2FnZSA6IGVycik7XG4gICAgICAgICAgICAgIHRyeSB7XG4gICAgICAgICAgICAgICAgaWYgKCFyZXMuaGVhZGVyc1NlbnQpIHtcbiAgICAgICAgICAgICAgICAgIHJlcy53cml0ZUhlYWQgJiYgcmVzLndyaXRlSGVhZCg1MDIpO1xuICAgICAgICAgICAgICAgICAgcmVzLmVuZCAmJiByZXMuZW5kKCdQcm94eSBlcnJvcicpO1xuICAgICAgICAgICAgICAgIH1cbiAgICAgICAgICAgICAgfSBjYXRjaCAoZSkgeyB9XG4gICAgICAgICAgICB9LFxuICAgICAgICAgIH0sXG4gICAgICAgIH07XG4gICAgICB9KSgpLFxuICAgIH0sXG4gICAgb3B0aW1pemVEZXBzOiB7XG4gICAgICBpbmNsdWRlOiBbXG4gICAgICAgICdAbXVpL21hdGVyaWFsJyxcbiAgICAgICAgJ0BtdWkveC1kYXRlLXBpY2tlcnMnLFxuICAgICAgICAnQG11aS94LWRhdGUtcGlja2Vycy9Mb2NhbGl6YXRpb25Qcm92aWRlcicsXG4gICAgICAgICdAbXVpL3gtZGF0ZS1waWNrZXJzL0FkYXB0ZXJEYXRlRm5zJyxcbiAgICAgICAgJ0BtdWkveC1kYXRlLXBpY2tlcnMvRGF0ZVBpY2tlcicsXG4gICAgICAgICdAbXVpL2ljb25zLW1hdGVyaWFsJyxcbiAgICAgICAgJ2NoYXJ0LmpzJyxcbiAgICAgICAgJ2NoYXJ0anMtYWRhcHRlci1kYXRlLWZucycsXG4gICAgICAgICdyZWFjdC1jaGFydGpzLTInLFxuICAgICAgICAncmVhY3QtZGlmZi12aWV3JyxcbiAgICAgICAgJ2RpZmYnLFxuICAgICAgXSxcbiAgICAgIGV4Y2x1ZGU6IFtcbiAgICAgICAgJ2VjaGFydHMnLFxuICAgICAgICAnenJlbmRlcicsXG4gICAgICAgICdlY2hhcnRzLWZvci1yZWFjdCcsXG4gICAgICAgIC8vIENvbW1vbiBwcm9ibGVtYXRpYyBkZXBlbmRlbmNpZXMgdGhhdCBtYXkgY2F1c2UgaXNzdWVzIHdpdGggVml0ZSdzIGRlcCBvcHRpbWl6ZXJcbiAgICAgICAgJ21vbmFjby1lZGl0b3InLFxuICAgICAgICAnbW9uYWNvLXlhbWwnLFxuICAgICAgICAnQG1vbmFjby1lZGl0b3IvcmVhY3QnLFxuICAgICAgICAvLyBBbnQgRGVzaWduIHY1IHBhdGNoIGZvciBSZWFjdCAxOSBtYXkgYmUgaW5jb21wYXRpYmxlIHdpdGggY3VycmVudCBSZWFjdCAxOCBkZXYgaW5zdGFsbHNcbiAgICAgICAgLy8gYW5kIGNhbiBjb25mdXNlIFZpdGUncyBvcHRpbWl6ZXI7IGtlZXAgaXQgZXhjbHVkZWQgc28gdGhlIG9wdGltaXplciB3b24ndCBwcmUtYnVuZGxlIGl0LlxuICAgICAgICAnQGFudC1kZXNpZ24vdjUtcGF0Y2gtZm9yLXJlYWN0LTE5JyxcbiAgICAgICAgLy8gQWRkIGFueSBvdGhlciBkZXBlbmRlbmNpZXMgdGhhdCBjYXVzZSBvcHRpbWl6YXRpb24gaXNzdWVzXG4gICAgICBdLFxuICAgIH0sXG4gICAgLy8gTm90ZTogbW9uYWNvLXlhbWwgaXMgZHluYW1pY2FsbHkgaW1wb3J0ZWQgYXQgcnVudGltZS4gQXZvaWQgcHJlLWJ1bmRsaW5nIGl0XG4gICAgLy8gdG8gcHJldmVudCBkZXBlbmRlbmN5IHJlc29sdXRpb24gaXNzdWVzIHdpdGggY2VydGFpbiBsYW5ndWFnZSBzZXJ2ZXIgcGFja2FnZXMuXG4gICAgYnVpbGQ6IHtcbiAgICAgIGNodW5rU2l6ZVdhcm5pbmdMaW1pdDogMjMwMCxcbiAgICAgIHJvbGx1cE9wdGlvbnM6IHtcbiAgICAgICAgZXh0ZXJuYWw6IFsnZWNoYXJ0cycsICd6cmVuZGVyJywgJ2VjaGFydHMtZm9yLXJlYWN0J10sXG4gICAgICAgIG91dHB1dDoge1xuICAgICAgICAgIGdsb2JhbHM6IHsgZWNoYXJ0czogJ2VjaGFydHMnLCB6cmVuZGVyOiAnenJlbmRlcicsICdlY2hhcnRzLWZvci1yZWFjdCc6ICdFQ2hhcnRzUmVhY3QnIH0sXG4gICAgICAgICAgbWFudWFsQ2h1bmtzKGlkOiBzdHJpbmcpIHtcbiAgICAgICAgICAgIGlmICghaWQuaW5jbHVkZXMoJ25vZGVfbW9kdWxlcycpKSByZXR1cm4gdW5kZWZpbmVkO1xuICAgICAgICAgICAgaWYgKGlkLmluY2x1ZGVzKCdub2RlX21vZHVsZXMvcmVhY3QnKSkgcmV0dXJuICd2ZW5kb3ItcmVhY3QnO1xuICAgICAgICAgICAgaWYgKGlkLmluY2x1ZGVzKCdub2RlX21vZHVsZXMvcmVhY3QtZG9tJykpIHJldHVybiAndmVuZG9yLXJlYWN0JztcbiAgICAgICAgICAgIGlmIChpZC5pbmNsdWRlcygnbm9kZV9tb2R1bGVzL3JlYWN0ZmxvdycpKSByZXR1cm4gJ3ZlbmRvci1yZWFjdGZsb3cnO1xuICAgICAgICAgICAgaWYgKGlkLmluY2x1ZGVzKCdub2RlX21vZHVsZXMnKSAmJiAvKD86XnxcXC8pKEAoYXBhY2hlXFwtKT9lY2hhcnRzfGVjaGFydHN8enJlbmRlcnxlY2hhcnRzLWZvci1yZWFjdHxyZWFjdC1lY2hhcnRzKS8udGVzdChpZCkpIHtcbiAgICAgICAgICAgICAgcmV0dXJuICd2ZW5kb3ItZWNoYXJ0cyc7XG4gICAgICAgICAgICB9XG4gICAgICAgICAgICBpZiAoaWQuaW5jbHVkZXMoJ25vZGVfbW9kdWxlcy9yZWFjdC1jaGFydGpzLTInKSB8fCBpZC5pbmNsdWRlcygnbm9kZV9tb2R1bGVzL2NoYXJ0LmpzJykpIHJldHVybiAndmVuZG9yLWNoYXJ0anMnO1xuICAgICAgICAgICAgaWYgKGlkLmluY2x1ZGVzKCdub2RlX21vZHVsZXMvcmVjaGFydHMnKSkgcmV0dXJuICd2ZW5kb3ItcmVjaGFydHMnO1xuICAgICAgICAgICAgaWYgKGlkLmluY2x1ZGVzKCdub2RlX21vZHVsZXMvcmVhY3QtZGlmZi12aWV3JykgfHwgaWQuaW5jbHVkZXMoJ25vZGVfbW9kdWxlcy9yZWFjdC1kaWZmLXZpZXdlcicpKSByZXR1cm4gJ3ZlbmRvci1kaWZmJztcbiAgICAgICAgICAgIGlmIChpZC5pbmNsdWRlcygnbm9kZV9tb2R1bGVzL21vbmFjby1lZGl0b3IvZXNtL3ZzL2xhbmd1YWdlLycpKSByZXR1cm4gJ3ZlbmRvci1tb25hY28tbGFuZ3VhZ2VzJztcbiAgICAgICAgICAgIGlmIChpZC5pbmNsdWRlcygnbm9kZV9tb2R1bGVzL21vbmFjby1lZGl0b3IvZXNtL3ZzL2Jhc2ljLWxhbmd1YWdlcy8nKSkgcmV0dXJuICd2ZW5kb3ItbW9uYWNvLWJhc2ljLWxhbmd1YWdlcyc7XG4gICAgICAgICAgICBpZiAoaWQuaW5jbHVkZXMoJ25vZGVfbW9kdWxlcy9tb25hY28tZWRpdG9yJykpIHJldHVybiAndmVuZG9yLW1vbmFjby1jb3JlJztcbiAgICAgICAgICAgIGlmIChpZC5pbmNsdWRlcygnbm9kZV9tb2R1bGVzL2hpZ2hsaWdodC5qcycpKSByZXR1cm4gJ3ZlbmRvci1oaWdobGlnaHQnO1xuICAgICAgICAgICAgaWYgKGlkLmluY2x1ZGVzKCdub2RlX21vZHVsZXMvcmVmcmFjdG9yJykpIHJldHVybiAndmVuZG9yLXJlZnJhY3Rvcic7XG4gICAgICAgICAgICBpZiAoaWQuaW5jbHVkZXMoJ25vZGVfbW9kdWxlcy9AbXVpL21hdGVyaWFsJykpIHJldHVybiAndmVuZG9yLW11aS1tYXRlcmlhbCc7XG4gICAgICAgICAgICBpZiAoaWQuaW5jbHVkZXMoJ25vZGVfbW9kdWxlcy9AbXVpL2ljb25zLW1hdGVyaWFsJykpIHJldHVybiAndmVuZG9yLW11aS1pY29ucyc7XG4gICAgICAgICAgICBpZiAoaWQuaW5jbHVkZXMoJ25vZGVfbW9kdWxlcy9AbXVpL2xhYicpKSByZXR1cm4gJ3ZlbmRvci1tdWktbGFiJztcbiAgICAgICAgICAgIGlmIChpZC5pbmNsdWRlcygnbm9kZV9tb2R1bGVzL0BlbW90aW9uJykpIHJldHVybiAndmVuZG9yLWVtb3Rpb24nO1xuICAgICAgICAgICAgaWYgKGlkLmluY2x1ZGVzKCdub2RlX21vZHVsZXMvQG11aS8nKSkgcmV0dXJuICd2ZW5kb3ItbXVpLXNoYXJlZCc7XG4gICAgICAgICAgICBjb25zdCBwYXJ0cyA9IGlkLnNwbGl0KCdub2RlX21vZHVsZXMvJylbMV0uc3BsaXQoJy8nKTtcbiAgICAgICAgICAgIGxldCBwa2cgPSBwYXJ0c1swXTtcbiAgICAgICAgICAgIGlmIChwa2cgJiYgcGtnLnN0YXJ0c1dpdGgoJ0AnKSAmJiBwYXJ0cy5sZW5ndGggPiAxKSB7XG4gICAgICAgICAgICAgIHBrZyA9IGAke3BrZ30vJHtwYXJ0c1sxXX1gO1xuICAgICAgICAgICAgfVxuICAgICAgICAgICAgcmV0dXJuIGB2ZW5kb3ItJHtwa2cucmVwbGFjZSgnLycsICdfJykucmVwbGFjZSgnQCcsICcnKX1gO1xuICAgICAgICAgIH0sXG4gICAgICAgIH0sXG4gICAgICB9LFxuICAgICAgY29tbW9uanNPcHRpb25zOiB7XG4gICAgICAgIHRyYW5zZm9ybU1peGVkRXNNb2R1bGVzOiB0cnVlLFxuICAgICAgfSxcbiAgICB9LFxuICB9O1xufSk7XG4iXSwKICAibWFwcGluZ3MiOiAiO0FBQ0EsU0FBUyxjQUFjLGVBQWU7QUFDdEMsT0FBTyxXQUFXO0FBQ2xCLE9BQU8sVUFBVTtBQUNqQixTQUFTLHFCQUFxQjtBQUpzSixJQUFNLDJDQUEyQztBQU9yTyxJQUFNLGFBQWEsY0FBYyx3Q0FBZTtBQUNoRCxJQUFNLFlBQVksS0FBSyxRQUFRLFVBQVU7QUFFekMsSUFBTyxzQkFBUSxhQUFhLE9BQU8sRUFBRSxLQUFLLE1BQU07QUFFOUMsUUFBTSxNQUFNLFFBQVEsTUFBTSxRQUFRLElBQUksR0FBRyxFQUFFO0FBRTNDLFVBQVEsSUFBSSxnQ0FBZ0MsSUFBSSx1QkFBdUIsUUFBUSxJQUFJLHVCQUF1Qix1QkFBdUI7QUFDakksVUFBUSxJQUFJLDBCQUEyQixJQUFJLHFCQUFxQixRQUFRLElBQUkscUJBQXFCLElBQUksdUJBQXVCLFFBQVEsSUFBSSx1QkFBdUIsdUJBQXdCO0FBQ3ZMLFFBQU0sVUFBaUIsQ0FBQyxNQUFNLENBQUM7QUFHL0IsTUFBSTtBQUNGLFFBQUk7QUFDSixRQUFJO0FBQ0YsT0FBQyxFQUFFLFdBQVcsSUFBSSxNQUFNLE9BQU8sd0JBQXdCO0FBQUEsSUFDekQsU0FBUyxHQUFHO0FBQ1YsT0FBQyxFQUFFLFdBQVcsSUFBSSxNQUFNLE9BQU8saUdBQTBCO0FBQUEsSUFDM0Q7QUFDQSxZQUFRO0FBQUEsTUFDTixXQUFXLEVBQUUsVUFBVSxtQkFBbUIsTUFBTSxPQUFPLFVBQVUsTUFBTSxZQUFZLE1BQU0sTUFB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
