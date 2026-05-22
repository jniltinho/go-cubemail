import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: '../web/dist',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (
            id.includes('primeicons') ||
            id.includes('primevue') ||
            id.includes('@primevue') ||
            id.includes('@primeuix')
          ) {
            return 'primevue'
          }
          if (
            id.includes('node_modules/vue') ||
            id.includes('pinia') ||
            id.includes('vue-router')
          ) {
            return 'vendor'
          }
        },
      },
    },
  },
})
