import { fileURLToPath, URL } from "node:url";

import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// During `npm run dev`, /api requests are proxied to the Go `easy-infra serve`
// backend (default :8080). The production build is emitted to dist/, which the
// Go binary embeds and serves itself.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    // `@` maps to ./src so modules import by layer ("@/services/api") rather
    // than brittle relative paths. Mirrored in tsconfig.json `paths`.
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
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
