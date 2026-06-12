import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// During `npm run dev`, /api requests are proxied to the Go `easy-infra serve`
// backend (default :8080). The production build is emitted to dist/, which the
// Go binary embeds and serves itself.
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/api": "http://localhost:8080",
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
});
