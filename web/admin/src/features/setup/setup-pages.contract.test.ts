import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const root = join(dirname(fileURLToPath(import.meta.url)), '../../..')

function read(rel: string) {
  return readFileSync(join(root, rel), 'utf8')
}

describe('login and setup pages', () => {
  it('share the split sign-in shell', () => {
    expect(read('src/features/setup/login-page.tsx')).toMatch(/AuthShell/)
    expect(read('src/features/setup/setup-wizard.tsx')).toMatch(/AuthShell/)
    expect(read('src/features/setup/auth-shell.tsx')).toMatch(/lg:grid-cols-2/)
  })

  it('keeps login as a single form without step back', () => {
    const login = read('src/features/setup/login-page.tsx')
    expect(login).not.toMatch(/上一步/)
    expect(login).not.toMatch(/setupStepsFor/)
    expect(login).not.toMatch(/previousSetupStep/)
  })

  it('uses multi-step only on the setup wizard and allows going back', () => {
    const wizard = read('src/features/setup/setup-wizard.tsx')
    expect(wizard).toMatch(/上一步/)
    expect(wizard).toMatch(/previousSetupStep/)
    expect(wizard).toMatch(/steps\.length/)
  })

  it('does not expose mock or a ComfyUI node step', () => {
    const wizard = read('src/features/setup/setup-wizard.tsx')
    const steps = read('src/features/setup/setup-steps.ts')
    expect(wizard).not.toMatch(/Comfy Mock/)
    expect(wizard).not.toMatch(/htmlFor='comfy-url'/)
    expect(wizard).not.toMatch(/ComfyUI 节点/)
    expect(steps).not.toMatch(/ComfyUI 节点/)
    expect(wizard).toMatch(/comfyui_base_url: ''/)
    expect(wizard).toMatch(/暂时跳过/)
  })

  it('asks for the new password twice and lets Telegram be skipped', () => {
    const wizard = read('src/features/setup/setup-wizard.tsx')
    expect(wizard).not.toMatch(/htmlFor='old-password'/)
    expect(wizard).toMatch(/htmlFor='confirm-password'/)
    expect(wizard).toMatch(/telegram_bot_token: ''/)
  })
})
