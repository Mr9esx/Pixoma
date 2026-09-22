/// <reference types="vitest/config" />
import path from 'path'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { tanstackRouter } from '@tanstack/router-plugin/vite'
import { playwright } from '@vitest/browser-playwright'

function proxyTarget(): string {
  const addr =
    process.env.HTTP_ADDR || `127.0.0.1:${process.env.PORT || '8082'}`
  const port = addr.slice(addr.lastIndexOf(':') + 1)
  return `http://127.0.0.1:${port}`
}

// https://vite.dev/config/
export default defineConfig(({ command }) => ({
  plugins: [
    tanstackRouter({
      target: 'react',
      autoCodeSplitting: true,
    }),
    react(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  define:
    command === 'serve'
      ? { 'import.meta.env.VITE_ADMIN_API_BASE': JSON.stringify('') }
      : undefined,
  server: {
    host: '127.0.0.1',
    proxy: {
      '/api': {
        target: proxyTarget(),
        changeOrigin: true,
        ws: true,
      },
    },
  },
  test: {
    include: ['src/**/*.test.ts', 'src/features/studio/studio-chat.test.tsx'],
    silent: 'passed-only',
    unstubEnvs: true,
    browser: {
      enabled: true,
      provider: playwright(),
      instances: [{ browser: 'chromium' }],
    },
    coverage: {
      // include: ['src/**/*.{js,jsx,ts,tsx}'], // Uncomment to expand the report to all src/**/* so untested modules appear as 0% coverage.
      exclude: [
        'src/components/ui/**',
        'src/assets/**',
        'src/tanstack-table.d.ts',
        'src/routeTree.gen.ts',
        'src/test-utils/**',
        'src/routes/**',
      ],
    },
  },
}))
