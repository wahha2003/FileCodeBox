import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    port: 3000,
    // 开发环境代理：将所有后端 API 路径转发到本地后端
    // 生产部署（EdgeOne Pages）时此配置不生效，
    // 由 src/utils/request.ts 中的 VITE_API_BASE_URL 环境变量控制后端地址
    proxy: {
      '/share': {
        target: 'http://localhost:12345',
        changeOrigin: true,
      },
      '/user': {
        target: 'http://localhost:12345',
        changeOrigin: true,
      },
      '/admin': {
        target: 'http://localhost:12345',
        changeOrigin: true,
      },
      '/chunk': {
        target: 'http://localhost:12345',
        changeOrigin: true,
      },
      // 公开配置接口
      '/api': {
        target: 'http://localhost:12345',
        changeOrigin: true,
      },
      // 初始化检测接口
      '/setup': {
        target: 'http://localhost:12345',
        changeOrigin: true,
      },
    },
  },
  build: {
    // 构建输出目录（EdgeOne Pages 部署时使用此目录）
    outDir: 'dist',
  },
})

