import react, { reactCompilerPreset } from '@vitejs/plugin-react'
import babel from '@rolldown/plugin-babel'
import { defineConfig } from 'vite'

// The dev server forwards API paths to the Go backend, so the page and API share an origin and the auth cookies just work.
// Paths aren't rewritten: the refresh cookie is scoped to /auth. Add each new API prefix here.
const api = 'http://localhost:8080'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    babel({ presets: [reactCompilerPreset()] })
  ],
  server: {
    proxy: { '/auth': api, '/user': api },
  },
})
