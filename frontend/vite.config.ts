import react, { reactCompilerPreset } from '@vitejs/plugin-react'
import babel from '@rolldown/plugin-babel'
import { fileURLToPath } from 'node:url'
import { defineConfig } from 'vite'

// The dev server forwards API paths to the Go backend, so the page and API share an origin and the auth cookies just work.
// Paths aren't rewritten: the refresh cookie is scoped to /auth. Add each new API prefix here.
// Page loads ask for HTML and stay with Vite, so app URLs like /movies/:id never reach the API.
const api = {
  target: 'http://localhost:8080',
  bypass: (req: { headers: { accept?: string }; url?: string }) =>
    req.headers.accept?.includes('text/html') ? req.url : undefined,
}

// https://vite.dev/config/
export default defineConfig({
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  plugins: [
    react(),
    babel({ presets: [reactCompilerPreset()] })
  ],
  server: {
    proxy: { '/auth': api, '/user': api, '/movies': api, '/events': api },
  },
})
