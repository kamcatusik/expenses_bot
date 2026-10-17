import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// В dev-режиме /api проксируется на локальный бэкенд.
export default defineConfig({
  plugins: [react()],
  server: { proxy: { '/api': 'http://localhost:8080' } },
})
