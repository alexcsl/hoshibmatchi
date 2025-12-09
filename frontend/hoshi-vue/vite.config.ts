import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import path from "path";

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    host: true, // Listen on all addresses (0.0.0.0) for Docker
    allowedHosts: [
      'hoshi.local',
      'localhost',
      '.local', // Allow all .local domains
    ],
    watch: {
      usePolling: true, // Enable polling for file changes in Docker
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src")
    }
  }
});
