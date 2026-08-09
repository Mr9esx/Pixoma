import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const srcRoot = join(here, '../..')

const APP_LAYOUT = join(srcRoot, 'routes/_app.tsx')
const APP_SIDEBAR = join(here, 'app-sidebar.tsx')
const APP_TITLE = join(here, 'app-title.tsx')
const LOGO = join(srcRoot, 'assets/logo.tsx')
const LANG_SWITCHER = join(here, 'language-switcher.tsx')

function read(path: string) {
  return readFileSync(path, 'utf8')
}

describe('admin shell layout (sidebar footer + no content header)', () => {
  it('_app.tsx has no shell-level header with language/theme', () => {
    const source = read(APP_LAYOUT)
    expect(source).not.toMatch(/<header[^>]*>[\s\S]*LanguageSwitcher/)
    expect(source).not.toMatch(/<header[^>]*>[\s\S]*ThemeSwitch/)
    expect(source).not.toContain('import { LanguageSwitcher }')
    expect(source).not.toContain('import { ThemeSwitch }')
  })

  it('AppSidebar mounts tools in SidebarFooter', () => {
    const source = read(APP_SIDEBAR)
    expect(source).toContain('SidebarFooter')
    expect(source).toContain('LanguageSwitcher')
    expect(source).toContain('ThemeSwitch')
  })

  it('AppTitle shows Pixoma without template subtitle', () => {
    const source = read(APP_TITLE)
    expect(source).toContain('Pixoma')
    expect(source).not.toContain('Shadcn-Admin')
    expect(source).not.toContain('Vite + ShadcnUI')
  })

  it('Logo accessible name is Pixoma', () => {
    const source = read(LOGO)
    expect(source).toContain('<title>Pixoma</title>')
    expect(source).toContain("id='pixoma-admin-logo'")
    expect(source).not.toContain('Shadcn-Admin')
    expect(source).not.toContain('shadcn-admin')
  })

  it('document meta brands as Pixoma (no template copy)', () => {
    const html = read(join(srcRoot, '../index.html'))
    const pkg = read(join(srcRoot, '../package.json'))
    expect(html).toContain('<title>Pixoma</title>')
    expect(html).toContain('Pixoma 管理控制台')
    expect(html).not.toMatch(/[Ss]hadcn/)
    expect(html).not.toContain('shadcn-admin.netlify.app')
    expect(pkg).toContain('"name": "pixoma-admin"')
    expect(pkg).not.toContain('shadcn-admin')
  })

  it('LanguageSwitcher uses dropdown trigger (not dual text buttons)', () => {
    const source = read(LANG_SWITCHER)
    expect(source).toContain('DropdownMenu')
    expect(source).toContain("data-testid='language-switcher'")
    expect(source).not.toMatch(
      /size='sm'[\s\S]*lang\.zh[\s\S]*size='sm'[\s\S]*lang\.en/
    )
  })
})
