import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ mode }) => {
  // build/preview 时 API 用相对路径 /api（nginx 代理）
  // dev 时 API 用 localhost:8181
  const apiBase = mode === 'production' ? '/api' : 'http://localhost:8181/api'

  return {
    plugins: [react()],
    server: {
      port: 8100,
      host: '0.0.0.0'
    },
    preview: {
      port: 8100,
      host: '0.0.0.0'
    },
    define: {
      'import.meta.env.VITE_API_BASE': JSON.stringify(apiBase)
    }
  }
})