/// <reference types="vitest/config" />
import { defineConfig } from "vite";
import { fileURLToPath, URL } from "node:url";
import { createRequire } from "node:module";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { viteSingleFile } from "vite-plugin-singlefile";

const require = createRequire(import.meta.url);
const pkg = require("./package.json") as { version: string };

// The management panel ships as a single self-contained HTML document.
// `vite-plugin-singlefile` inlines every JS/CSS asset into `dist/index.html`,
// which CI then copies to `dist/management.html` for the Go backend to embed.
export default defineConfig({
  plugins: [react(), tailwindcss(), viteSingleFile()],
  define: {
    __APP_VERSION__: JSON.stringify(pkg.version),
  },
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  build: {
    target: "es2022",
    cssCodeSplit: false,
    assetsInlineLimit: 100_000_000,
    chunkSizeWarningLimit: 100_000,
    reportCompressedSize: false,
  },
  test: {
    globals: true,
    environment: "jsdom",
    setupFiles: ["./src/test/setup.ts"],
    css: false,
  },
});
