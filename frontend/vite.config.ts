import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'
import { fileURLToPath, URL } from 'node:url'

// The backend does not send CORS headers, so API traffic is proxied through
// the frontend origins below. The app uses a relative base URL (see .env) to
// keep those requests same-origin.
const apiProxy = {
  '/api': {
    target: 'http://localhost:1337',
    changeOrigin: true,
  },
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    proxy: apiProxy,
  },
  preview: {
    proxy: apiProxy,
  },
})
