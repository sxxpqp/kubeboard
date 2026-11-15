import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  define: {
    // 期望的后端 API 契约版本，与 internal/api/version.go 的 APIVersion 同步递增。
    // 运行时与 /api/version 的 apiVersion 比对，不匹配则展示告警横幅（P0-⑤）。
    __EXPECTED_API_VERSION__: JSON.stringify('v2'),
  },
  server: {
    host: true,
    port: 8080,
    proxy: {
      '/api': 'http://localhost:9090',
    },
  },
  build: {
    outDir: 'dist',
  },
})
