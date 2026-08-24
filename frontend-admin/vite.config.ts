import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: "http://localhost:42818",
        changeOrigin: true,
      },
    },
  },
  build: {
    sourcemap: false,
  },
});
