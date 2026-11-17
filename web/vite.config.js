import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  base: '/app/remote-desktop/',
  plugins: [vue()],
  server: {
    host: '127.0.0.1',
    port: 4173,
    proxy: { '/app/remote-desktop/api': { target: 'http://127.0.0.1:18080', ws: true } },
  },
  build: {
    chunkSizeWarningLimit: 1000,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules/element-plus')) return 'element-plus'
          if (id.includes('node_modules/lucide-vue-next')) return 'icons'
          if (id.includes('node_modules/vue') || id.includes('node_modules/@vue')) return 'vue-vendor'
        },
      },
    },
  },
})
