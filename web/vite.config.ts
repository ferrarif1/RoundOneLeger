import { defineConfig } from 'vite';
import type { ProxyOptions } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'node:path';
import type { ServerResponse, IncomingMessage } from 'node:http';

const API_TARGET = process.env.VITE_API_PROXY_TARGET?.trim() || 'http://127.0.0.1:8080';

const createProxy = (routePrefix: string): ProxyOptions => ({
  target: API_TARGET,
  changeOrigin: true,
  secure: false,
  configure: (proxy) => {
    proxy.on('error', (error, req, res) => {
      const message =
        (error as NodeJS.ErrnoException).code === 'ECONNREFUSED'
          ? '本地后端未启动或端口配置错误，请先运行 `go run ./cmd/server` 再刷新页面。'
          : error.message;
      const url = (req as IncomingMessage).url || routePrefix;
      console.error(`[proxy] ${url} failed: ${error.message}`);
      const response = res as ServerResponse;
      if (!response.headersSent) {
        response.writeHead(502, { 'Content-Type': 'application/json' });
      }
      if (!response.writableEnded) {
        response.end(JSON.stringify({ error: 'proxy_error', message }));
      }
    });
  },
});

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src')
    }
  },
  server: {
    port: 5173,
    host: '0.0.0.0',
    proxy: {
      '/auth': createProxy('/auth'),
      '/api': createProxy('/api')
    }
  },
  preview: {
    port: 4173,
    host: '0.0.0.0'
  }
});
