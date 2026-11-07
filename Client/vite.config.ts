import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
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
