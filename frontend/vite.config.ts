/**
 * @file vite.config.ts
 * @description Vite build and development configuration for the Cubemail frontend.
 * Defines the development server, API proxy, plugins, and the output directory of the build.
 */

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  define: {
    API_BASE: JSON.stringify('/api/v1'),
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      }
    }
  },
  build: {
    outDir: '../web/dist',
    emptyOutDir: true,
  }
})
