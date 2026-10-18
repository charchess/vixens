import { defineConfig } from "vite";

export default defineConfig({
  build: {
    manifest: "asset-manifest.json",
    sourcemap: false,
  },
});
