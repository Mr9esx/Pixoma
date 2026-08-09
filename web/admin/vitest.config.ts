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
      'src/lib/api/instances.test.ts',
      'src/lib/api/cases.test.ts',
      'src/lib/api/tg-menu.test.ts',
      'src/lib/api/tasks.test.ts',
      'src/lib/api/task-errors.test.ts',
      'src/lib/api/users.test.ts',
      'src/lib/api/sessions.test.ts',
      'src/lib/api/query-keys.test.ts',
      'src/lib/dashboard/aggregate.test.ts',
      'src/config/menu.test.ts',
      'src/lib/i18n/locale.test.ts',
      'src/components/master-detail/master-detail.contract.test.ts',
      'src/components/layout/shell-layout.contract.test.ts',
      'src/styles/theme-neutral.contract.test.ts',
      'src/features/cases/list-panel.contract.test.ts',
      'src/features/cases/menu-placements.contract.test.ts',
      'src/components/filters/list-filter.contract.test.ts',
    ],
  },
})
