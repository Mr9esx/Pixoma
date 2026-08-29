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
const NAV_USER = join(here, 'nav-user.tsx')

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

  it('AppSidebar mounts only the integrated NavUser in SidebarFooter', () => {
    const source = read(APP_SIDEBAR)
    expect(source).toContain('SidebarFooter')
    expect(source).toContain('NavUser')
    expect(source).not.toContain('import { ThemeSwitch }')
    expect(source).not.toContain('import { LanguageSwitcher }')
    expect(source).toMatch(/<SidebarFooter>\s*<NavUser/)
  })

  it('NavUser renders an expanded user card and collapsed dropdown', () => {
    const source = read(NAV_USER)
    expect(source).toContain('fetchCurrentUser')
    expect(source).toContain("queryKey: ['current-user']")
    expect(source).toContain('queryFn: fetchCurrentUser')
    expect(source).toContain('getDisplayNameInitials')
    expect(source).toContain('size-6')
    expect(source).toContain('h-12')
    expect(source).toContain('ArrowRightFromLine')
    expect(source).toContain('DropdownMenuTrigger')
    expect(source).toContain('DropdownMenuContent')
    expect(source).toContain('nav-user-sign-out')
    expect(source).toContain('common.signOut')
  })

  it('AppTitle shows Pixoma without template subtitle', () => {
    const source = read(APP_TITLE)
    expect(source).toContain('Pixoma')
    expect(source).not.toContain('Shadcn-Admin')
    expect(source).not.toContain('Vite + ShadcnUI')
  })

  it('Logo accessible name is Pixoma', () => {
    const source = read(LOGO)
    expect(source).toContain("alt = 'Pixoma'")
    expect(source).toContain("src='/images/logo.png'")
    expect(source).toContain("id='pixoma-admin-logo'")
    expect(source).toContain('<img')
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

  it('NavUser integrates language + theme switching in its dropdown', () => {
    const source = read(NAV_USER)
    expect(source).toContain('DropdownMenu')
    expect(source).toContain("data-testid='nav-user-lang-zh'")
    expect(source).toContain("data-testid='nav-user-lang-en'")
    expect(source).toContain("data-testid='nav-user-theme-light'")
    expect(source).toContain("data-testid='nav-user-theme-dark'")
  })

  it('content region switches to fixed layout when a child opts in', () => {
    const source = read(APP_LAYOUT)
    expect(source).toContain('contentRegionClassName')
    const region = read(join(here, 'content-region.ts'))
    expect(region).toContain(
      'mx-auto w-full max-w-[1440px] has-[>[data-layout=fixed]]:flex has-[>[data-layout=fixed]]:overflow-hidden'
    )
  })
})
