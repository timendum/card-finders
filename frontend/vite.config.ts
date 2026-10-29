/// <reference types="vitest/config" />
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// The Go backend serves the built assets from `frontend/dist` (see
// backend/internal/server/server.go default StaticDir "frontend/dist"), so the
// build must emit there. In dev, Vite serves the app and proxies the API and
// WebSocket surfaces to the Go server on :8080 so the browser talks to one
// origin.
const BACKEND = "http://localhost:8080";

export default defineConfig({
  plugins: [react()],
  build: {
    // Emits to frontend/dist (relative to this config's root).
    outDir: "dist",
    emptyOutDir: true,
  },
  server: {
    proxy: {
      "/api": {
        target: BACKEND,
        changeOrigin: true,
      },
      "/ws": {
        target: BACKEND,
        changeOrigin: true,
        ws: true,
        rewriteWsOrigin: true,
      },
    },
  },
  test: {
    globals: true,
    environment: "node",
    include: ["src/**/*.{test,spec}.{ts,tsx}"],
  },
});
