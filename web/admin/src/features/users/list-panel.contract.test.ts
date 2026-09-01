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

it('shows platform and generic user identity without telegram fields', () => {
  expect(panel).toMatch(/fieldPlatform/)
  expect(panel).toMatch(/fieldUserInfo/)
  expect(panel).toMatch(/accessor\('channel_name'/)
  expect(panel).toMatch(/formatUserLabel/)
  expect(panel).toMatch(/external_user_id/)
  expect(panel).not.toMatch(/tg_user_id|fieldTgUserId/)
  expect(panel).toMatch(/q/)
})

it('pins actions and exposes task access as a standalone trigger', () => {
  expect(panel).toMatch(/columnPinning: \{ right: \['actions'\] \}/)
  expect(panel).toMatch(/t\('common\.actions'\)/)
  expect(panel).toMatch(/t\('users\.fieldAccess'\)/)
  expect(panel).toMatch(/formatDateTime/)
})
