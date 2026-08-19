import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'

// Wails dev picks the port up automatically (wails.json: frontend:dev:serverUrl=auto).
export default defineConfig({
  plugins: [svelte(), tailwindcss()],
  server: { port: 5173, strictPort: true },
  build: { outDir: 'dist', emptyOutDir: true }
})
