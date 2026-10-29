import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// During development the browser talks to Vite (port 5173) and Vite forwards
// every /api request to the Go backend, so no CORS problems appear in dev.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': { target: process.env.VITE_BACKEND_URL || 'http://localhost:8080', changeOrigin: true },
    },
  },
});
