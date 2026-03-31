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
          if (id.includes('echarts')) {
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
          return 'vendor-misc';
        },
      },
    },
  },
});
