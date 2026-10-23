import {defineConfig} from 'vite'
import {svelte} from '@sveltejs/vite-plugin-svelte'

// https://vitejs.dev/config/
// `vite build --mode installer` builds the setup UI (installer/) into
// ../installer/dist; it shares styles and components with the launcher.
export default defineConfig(({mode}) => mode === 'installer'
  ? {
      root: 'installer',
      base: './',
      plugins: [svelte()],
      build: {outDir: '../../installer/dist', emptyOutDir: true},
    }
  : {plugins: [svelte()]})
