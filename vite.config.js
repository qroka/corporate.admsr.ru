import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import ui from '@nuxt/ui/vite';

export default defineConfig({
  plugins: [
    vue(),
    ui({
      ui: {
        colors: {
          primary: 'emerald',
          neutral: 'zinc'
        },
      },
    }),
  ],
  server: {
    port: 5173,
    proxy: {
      // Проксируем /api/* → XAMPP (http://localhost/...)
      // Если проект лежит в подпапке htdocs, поменяй rewrite:
      //   '/corporate/api' → убери rewrite и измени путь в fetch-вызовах
      '/api': {
        target: 'http://localhost',
        changeOrigin: true,
      },
    },
  },
});

