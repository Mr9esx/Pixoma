import { copyFileSync, mkdirSync } from "node:fs";
import path from "node:path";
import { defineConfig } from "vite";
import type { Plugin } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

function copyPixomaInstaller(): Plugin {
  return {
    name: "copy-pixoma-installer",
    apply: "build",
    closeBundle() {
      const distDirectory = path.resolve(import.meta.dirname, "./dist");
      mkdirSync(distDirectory, { recursive: true });
      copyFileSync(
        path.resolve(import.meta.dirname, "../../scripts/install.sh"),
        path.join(distDirectory, "install.sh"),
      );
    },
  };
}

export default defineConfig({
  plugins: [react(), tailwindcss(), copyPixomaInstaller()],
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "./src"),
      "next/image": path.resolve(
        import.meta.dirname,
        "./src/shims/next-image.tsx",
      ),
      "next/link": path.resolve(
        import.meta.dirname,
        "./src/shims/next-link.tsx",
      ),
    },
  },
  server: { host: "127.0.0.1", port: 5173 },
});
