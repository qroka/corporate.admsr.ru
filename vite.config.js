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
  },
  optimizeDeps: {
    include: [
      '@nuxt/ui > prosemirror-state',
      '@nuxt/ui > prosemirror-transform',
      '@nuxt/ui > prosemirror-model',
      '@nuxt/ui > prosemirror-view',
      '@nuxt/ui > prosemirror-gapcursor',
    ],
  },
});

