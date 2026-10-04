import { defineConfig } from 'vitest/config'
import solid from 'vite-plugin-solid'
import tailwindcss from '@tailwindcss/vite'

// `solid-js` ships separate development and browser entry points. Vitest runs
// with `mode === 'test'`, so only the test run resolves those conditions; a
// production build keeps Vite's default condition set.
const testResolve = { conditions: ['development', 'browser'] }

export default defineConfig(({ mode }) => ({
  plugins: [tailwindcss(), solid()],
  ...(mode === 'test' ? { resolve: testResolve } : {}),
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.{ts,tsx}'],
    setupFiles: ['./src/test/setup.ts'],
    clearMocks: true,
    restoreMocks: true,
    unstubEnvs: true,
    unstubGlobals: true,
  },
}))
