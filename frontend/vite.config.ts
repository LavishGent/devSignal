import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// In development the SPA proxies API calls to the Go backend; in production
// the Go server serves the built SPA itself, so paths stay same-origin.
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})
