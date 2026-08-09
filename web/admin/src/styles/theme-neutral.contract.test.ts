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

  it('source has no hardcoded slate color utilities', () => {
    expect(collectSlateHits(SRC_ROOT)).toEqual([])
  })
})
