import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  plugins: [svelte(), tailwindcss()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  server: {
    // `npm run dev` talks to a portal running locally on 3010.
    proxy: {
      '/api': 'http://localhost:3010',
      '/healthz': 'http://localhost:3010',
    },
  },
});
