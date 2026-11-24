/**
 * @file vite.config.ts
 * @description Vite+ configuration for the Cubemail frontend: the single file for the dev server and API proxy
 * (`vp dev`), the production build (`vp build`), and the formatter and linter (`vp check`).
 */

import { defineConfig, lazyPlugins } from 'vite-plus'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  // lazyPlugins: the Vite plugins load for dev/build only, not for check/lint/fmt
  plugins: lazyPlugins(() => [vue(), tailwindcss()]),
  define: {
    API_BASE: JSON.stringify('/api/v1'),
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
  },
  // unit tests (vp test): they read source files only, no browser and no backend
  test: { environment: 'node', include: ['tests/unit/**/*.test.mjs'] },
  // the style the code base already had: single quotes, no semicolons
  fmt: { singleQuote: true, semi: false, printWidth: 120 },
  lint: {
    jsPlugins: [{ name: 'vite-plus', specifier: 'vite-plus/oxlint-plugin' }],
    rules: {
      'vite-plus/prefer-vite-plus-imports': 'error',
      // needs strictNullChecks, which tsconfig.json leaves off
      'typescript/no-useless-default-assignment': 'off',
    },
    // .ts files are type-checked here; .vue files by vue-tsc (npm run type-check)
    options: { typeAware: true, typeCheck: true },
  },
})
