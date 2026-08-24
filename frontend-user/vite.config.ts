import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/n1': { target: 'http://127.0.0.1:28372', changeOrigin: true, rewrite: (p) => p.replace(/^\/n1/, '') },
      '/n2': { target: 'http://127.0.0.1:28373', changeOrigin: true, rewrite: (p) => p.replace(/^\/n2/, '') },
      '/n3': { target: 'http://127.0.0.1:28374', changeOrigin: true, rewrite: (p) => p.replace(/^\/n3/, '') },
      '/n4': { target: 'http://127.0.0.1:28375', changeOrigin: true, rewrite: (p) => p.replace(/^\/n4/, '') },
      '/n5': { target: 'http://127.0.0.1:28376', changeOrigin: true, rewrite: (p) => p.replace(/^\/n5/, '') },
    },
  },
})
