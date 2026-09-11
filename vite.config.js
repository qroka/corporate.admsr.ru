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
        container: {
          base: 'p-0 sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0',
        },
        main: {
          base: 'min-h-[calc(100vh-var(--ui-header-height))] w-full max-w-[1600px] mx-auto',
        },
        pageHeader: {
          slots: {
            root: 'relative border-b border-default py-4',
          },
        },
        // Год в сегментах date picker: дефолтный w-11 слишком узкий для text-base / кастомного шрифта
        inputDate: {
          slots: {
            segment: [
              'rounded-sm text-center outline-hidden whitespace-nowrap data-placeholder:text-dimmed data-[segment=literal]:text-muted data-invalid:text-error data-disabled:cursor-not-allowed data-disabled:opacity-75',
              'transition-colors',
            ],
          },
          variants: {
            size: {
              xs: {
                segment: 'data-[segment=day]:w-8 data-[segment=month]:w-8 data-[segment=year]:w-12',
              },
              sm: {
                segment: 'data-[segment=day]:w-8 data-[segment=month]:w-8 data-[segment=year]:w-12',
              },
              md: {
                segment: 'data-[segment=day]:w-9 data-[segment=month]:w-9 data-[segment=year]:w-14',
              },
              lg: {
                segment: 'data-[segment=day]:w-10 data-[segment=month]:w-10 data-[segment=year]:w-14',
              },
              xl: {
                segment: 'data-[segment=day]:w-10 data-[segment=month]:w-10 data-[segment=year]:w-16',
              },
            },
          },
        },
      },
    }),
  ],
  server: {
    port: 5173,
    proxy: {
      // Бэкенд теперь редиректит 80→443, поэтому ходим сразу по https.
      // secure:false — игнорируем несовпадение wildcard-сертификата с IP (только для dev).
      '/api': {
        target: 'https://172.17.4.21',
        changeOrigin: true,
        secure: false,
      },
      // Static uploads (FullPic/SmallPic) are served by backend web root in dev
      '/img': {
        target: 'https://172.17.4.21',
        changeOrigin: true,
        secure: false,
      },
    },
  },
});