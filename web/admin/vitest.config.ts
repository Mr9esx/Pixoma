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
      'src/config/menu.test.ts',
      'src/lib/i18n/locale.test.ts',
      'src/components/master-detail/master-detail.contract.test.ts',
    ],
  },
})
