import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// 优先使用 VITE_BACKEND_URL；否则读取 backend/.env 的 SERVER_PORT，
// 避免后端改了端口、前端代理还指向 8080。
function backendPort(): string {
  try {
    const envPath = fileURLToPath(new URL('../backend/.env', import.meta.url))
    const content = readFileSync(envPath, 'utf8')
    const match = content.match(/^\s*SERVER_PORT\s*=\s*["']?(\d+)["']?\s*(?:#.*)?$/m)
    return match?.[1] ?? '8080'
  } catch {
    return '8080'
  }
}

const backend = process.env.VITE_BACKEND_URL || `http://localhost:${backendPort()}`
const toBackend = () => ({ target: backend, changeOrigin: true })

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/api': toBackend(),
      // OAuth 授权页与 token 端点需要与前端同源，Cookie 才能正常携带
      '/oauth': toBackend(),
      '/.well-known': toBackend(),
      // Agent 通过网页地址下载技能文档
      '/skills': toBackend(),
    },
  },
})
