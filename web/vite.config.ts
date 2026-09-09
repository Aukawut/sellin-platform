import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')

  return {
    plugins: [react(), tailwindcss()],
    server: {
      port: 5173,
      proxy: {
        // ยิงผ่าน proxy ตอน dev เพื่อให้ browser มองว่าเป็น origin เดียวกัน
        // cookie ของ refresh token จึงเดินทางได้ตามปกติโดยไม่ต้องผ่อนกฎ CORS
        '/api': {
          target: env.API_URL || 'http://localhost:8080',
          changeOrigin: true,
        },
      },
    },
    build: {
      sourcemap: true,
      chunkSizeWarningLimit: 1000, // in KB
    },
  }
})
