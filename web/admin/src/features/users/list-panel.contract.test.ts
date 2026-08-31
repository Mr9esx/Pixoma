import { readFileSync } from 'node:fs'
import { it, expect } from 'vitest'

const panel = readFileSync(new URL('./list-panel.tsx', import.meta.url), 'utf8')

it('offers task access actions with all supported modes', () => {
  expect(panel).toMatch(/DropdownMenu/)
  expect(panel).toMatch(/accessAlwaysAllowed/)
  expect(panel).toMatch(/accessPaid/)
  expect(panel).toMatch(/accessDenied/)
  expect(panel).toMatch(/onSetAccess/)
})
