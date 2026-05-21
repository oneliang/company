import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(() => {
  // 通过环境变量 PROD_MODE 控制，而不是 Vite mode
  const isProd = process.env.PROD_MODE === 'true'
  const apiBase = isProd ? '/api' : 'http://localhost:8181/api'

  return {
    plugins: [react()],
    server: {
      port: 8100,
      host: '0.0.0.0',
      // 生产模式禁用 HMR
      hmr: isProd ? false : undefined
    },
    define: {
      'import.meta.env.VITE_API_BASE': JSON.stringify(apiBase)
    }
  }
})