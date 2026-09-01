import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'
// https://vite.dev/config/
export default defineConfig({
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/auth': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/query': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/webhooks': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    assetsDir: 'static',
    rollupOptions: {
      maxParallelFileOps: 50,
    },
  },
  plugins: [
    react({
      babel: {
        plugins: [['babel-plugin-react-compiler']],
      },
    }),
  ],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
})