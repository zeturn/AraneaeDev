import {defineConfig} from 'vite';
import vue from '@vitejs/plugin-vue';
import path from 'path';

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) {
            return;
          }

          const [, subPath = ''] = id.split('node_modules/');
          const cleanSubPath = subPath.replace(/\\/g, '/');
          const parts = cleanSubPath.split('/');
          const packageName = parts[0].startsWith('@')
            ? `${parts[0]}/${parts[1] || ''}`
            : parts[0];

          if (id.includes('echarts') || id.includes('zrender')) {
            return 'vendor-echarts';
          }
          if (id.includes('element-plus')) {
            return 'vendor-element-plus';
          }
          if (id.includes('vue') || id.includes('pinia') || id.includes('vue-router')) {
            return 'vendor-vue';
          }
          if (id.includes('axios')) {
            return 'vendor-axios';
          }
          if (packageName === '@popperjs/core') {
            return 'vendor-popper';
          }
          if (packageName === 'lodash-es') {
            return 'vendor-lodash';
          }
          if (packageName === '@ctrl/tinycolor') {
            return 'vendor-tinycolor';
          }

          return 'vendor-misc';
        },
      },
    },
  },
});
