import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import ui from '@nuxt/ui/vite';

export default defineConfig({
  plugins: [
    vue(),
    ui({
      ui: {
        colors: {
          primary: 'sky',
          neutral: 'slate'
        },
      },
    }),
  ],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://172.17.4.21',
        changeOrigin: true,
      },
      // Static uploads (FullPic/SmallPic) are served by backend web root in dev
      '/img': {
        target: 'http://172.17.4.21',
        changeOrigin: true,
      },
    },
  },
});