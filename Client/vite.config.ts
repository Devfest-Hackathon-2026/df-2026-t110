import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

const apiProxy = { '/api': 'http://127.0.0.1:8080' }

export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: apiProxy,
  },
  preview: {
    proxy: apiProxy,
  },
  build: {
    rolldownOptions: {
      output: {
        codeSplitting: {
          groups: [
            { name: 'three', test: /node_modules[\\/]three[\\/]/, maxSize: 350_000 },
            { name: 'blockly', test: /node_modules[\\/]blockly[\\/]/, maxSize: 350_000 },
          ],
        },
      },
    },
  },
})
