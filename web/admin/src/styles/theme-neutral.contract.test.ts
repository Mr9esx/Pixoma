import { readdirSync, readFileSync, statSync } from 'node:fs'
import { dirname, join, relative } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const adminRoot = join(here, '../..')
const COMPONENTS_JSON = join(adminRoot, 'components.json')
const THEME_CSS = join(here, 'theme.css')
const SRC_ROOT = join(adminRoot, 'src')

function read(path: string) {
  return readFileSync(path, 'utf8')
}

function darkBackgroundToken(css: string): string {
  const darkBlock = css.match(/\.dark\s*\{([\s\S]*?)\n\}/)
  expect(darkBlock, 'expected .dark { ... } block in theme.css').toBeTruthy()
  const match = darkBlock![1].match(/--background:\s*([^;]+);/)
  expect(match, 'expected --background in .dark').toBeTruthy()
  return match![1].trim()
}

function collectSlateHits(dir: string): string[] {
  const hits: string[] = []
  for (const name of readdirSync(dir)) {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) {
      hits.push(...collectSlateHits(path))
      continue
    }
    if (!/\.(tsx?|css)$/.test(name)) continue
    if (/\bslate-[0-9]{2,3}\b/.test(read(path))) {
      hits.push(relative(SRC_ROOT, path))
    }
  }
  return hits
}

describe('admin theme neutral base color', () => {
  it('components.json baseColor is neutral', () => {
    const config = JSON.parse(read(COMPONENTS_JSON)) as {
      tailwind?: { baseColor?: string }
    }
    expect(config.tailwind?.baseColor).toBe('neutral')
  })

  it('dark --background is true gray (chroma 0)', () => {
    const value = darkBackgroundToken(read(THEME_CSS))
    expect(value).toBe('oklch(0.145 0 0)')
  })

  it('theme defines font-heading for base dialog titles', () => {
    expect(read(THEME_CSS)).toMatch(/--font-heading:\s*var\(--font-sans\)/)
  })

  it('source has no hardcoded slate color utilities', () => {
    expect(collectSlateHits(SRC_ROOT)).toEqual([])
  })

  it('light --background is the studio canvas gray', () => {
    const root = read(THEME_CSS).match(/:root\s*\{([\s\S]*?)\n\}/)
    expect(root, 'expected :root { ... } block in theme.css').toBeTruthy()
    const match = root![1].match(/--background:\s*([^;]+);/)
    expect(match, 'expected --background in :root').toBeTruthy()
    expect(match![1].trim()).toBe('oklch(0.988 0 0)')
  })

  it('the shell leaves the page background in charge', () => {
    const layout = read(join(adminRoot, 'src/routes/_app.tsx'))
    expect(layout).not.toContain('bg-canvas')
    const base = read(join(adminRoot, 'src/styles/index.css'))
    expect(base).toMatch(/has-\[div\[data-variant='inset'\]\]:bg-sidebar/)
  })

  it('studio shell inherits the page background instead of re-tinting it', () => {
    // 半透明叠层会让实际颜色随页面底色漂移，外壳直接吃 --background。
    for (const rel of [
      'src/features/studio/studio-workspace.tsx',
      'src/features/studio/studio-library.tsx',
      'src/features/studio/studio-settings.tsx',
      'src/features/studio/studio-sidebar.tsx',
    ]) {
      const source = read(join(adminRoot, rel))
      for (const forbidden of [
        'overflow-hidden bg-muted/30',
        'flex-1 bg-muted/30',
        "'bg-muted/30 p-2'",
      ]) {
        expect(source, `${rel} 仍在外壳上叠 ${forbidden}`).not.toContain(forbidden)
      }
    }
  })
})

function extractFunction(source: string, name: string): string {
  const start = source.indexOf(`function ${name}(`)
  expect(start, `expected function ${name}(`).toBeGreaterThanOrEqual(0)
  const next = source.indexOf('\nfunction ', start + 1)
  return next === -1 ? source.slice(start) : source.slice(start, next)
}

describe('admin theme surface has no drop shadow', () => {
  it('Card surface has no drop shadow', () => {
    const src = read(join(SRC_ROOT, 'components/ui/card.tsx'))
    const cardFn = extractFunction(src, 'Card')
    expect(cardFn).not.toMatch(/\bshadow-sm\b/)
    expect(cardFn).toMatch(/\bborder\b/)
  })

  it('SidebarInset inset variant has no drop shadow', () => {
    const src = read(join(SRC_ROOT, 'components/ui/sidebar.tsx'))
    const insetFn = extractFunction(src, 'SidebarInset')
    expect(insetFn).not.toMatch(/\bshadow-sm\b/)
    expect(insetFn).toMatch(/rounded-2xl/)
  })

  it('button default variant has no shadow-xs', () => {
    const src = read(join(SRC_ROOT, 'components/ui/button.tsx'))
    const match = src.match(/default:\s*\n\s*'([^']+)'/)
    expect(match, 'expected default variant string').toBeTruthy()
    expect(match![1]).not.toMatch(/\bshadow-xs\b/)
  })

  it('form controls have no shadow-xs', () => {
    const files: [string, string][] = [
      ['components/ui/input.tsx', 'Input'],
      ['components/ui/textarea.tsx', 'Textarea'],
      ['components/ui/select.tsx', 'SelectTrigger'],
      ['components/ui/checkbox.tsx', 'Checkbox'],
      ['components/ui/switch.tsx', 'Switch'],
      ['components/ui/radio-group.tsx', 'RadioGroupItem'],
    ]
    for (const [rel, fn] of files) {
      const body = extractFunction(read(join(SRC_ROOT, rel)), fn)
      expect(body, rel).not.toMatch(/\bshadow-xs\b/)
    }
    expect(read(join(SRC_ROOT, 'components/ui/input-otp.tsx'))).not.toMatch(
      /\bshadow-xs\b/
    )
    expect(read(join(SRC_ROOT, 'components/ui/calendar.tsx'))).not.toMatch(
      /\bshadow-xs\b/
    )
    expect(read(join(SRC_ROOT, 'components/password-input.tsx'))).not.toMatch(
      /\bshadow-xs\b/
    )
  })

  it('filter-segment arrows have no shadow-sm', () => {
    const src = read(join(SRC_ROOT, 'components/filters/filter-segment.tsx'))
    expect(src).not.toMatch(/\bshadow-sm\b/)
  })

  it('select popover still has elevation', () => {
    const content = extractFunction(
      read(join(SRC_ROOT, 'components/ui/select.tsx')),
      'SelectContent'
    )
    expect(content).toMatch(/\bshadow-md\b/)
  })
})
