import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    environment: 'node',
    env: {
      VITE_ADMIN_API_BASE: 'http://127.0.0.1:8081',
    },
    include: [
      'src/scaffold.contract.test.ts',
      'src/auth-gates.contract.test.ts',
      'src/lib/api/client.test.ts',
    ],
  },
})
