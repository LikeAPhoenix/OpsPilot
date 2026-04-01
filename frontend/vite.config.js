import { defineConfig } from 'vite';
import { resolve } from 'path';

export default defineConfig({
  build: {
    rollupOptions: {
      input: {
        main: resolve(__dirname, 'index.html'),
        login: resolve(__dirname, 'login.html')
      }
    }
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:6872',
        changeOrigin: true
      }
    }
  }
});
