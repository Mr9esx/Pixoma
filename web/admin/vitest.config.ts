import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    environment: 'node',
    include: [
      'src/scaffold.contract.test.ts',
      'src/auth-gates.contract.test.ts',
    ],
  },
})
