import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const apiBase = mode === 'production' ? '/api' : 'http://localhost:8181/api'
  const hmrHost = env.VITE_HMR_HOST || 'localhost'

  return {
    plugins: [react()],
    server: {
      port: 8100,
      host: '0.0.0.0',
      hmr: {
        host: hmrHost,
        port: 8100
      }
    },
    define: {
      'import.meta.env.VITE_API_BASE': JSON.stringify(apiBase)
    }
  }
})