import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))

function read(name: string) {
  return readFileSync(join(here, name), 'utf8')
}

describe('kokonutui mouse-effect card workbench adaptation', () => {
  it('removes default brand copy and hardcoded promotional defaults', () => {
    const source = read('mouse-effect-card.tsx')
    expect(source).not.toContain('title = "Acme"')
    expect(source).not.toContain('Build interfaces with interactive patterns')
    expect(source).not.toContain('topText = "Case Study"')
    expect(source).not.toContain('primaryCtaText = "Get Started"')
    expect(source).not.toContain('footerText = "We do it all"')
  })

  it('uses semantic tokens instead of hardcoded zinc/white glow', () => {
    const source = read('mouse-effect-card.tsx')
    expect(source).not.toContain('bg-zinc-400')
    expect(source).not.toContain('bg-zinc-600')
    expect(source).not.toContain('bg-white/60')
    expect(source).not.toContain('bg-white/80')
    expect(source).toContain('bg-muted-foreground/30')
    expect(source).toContain('bg-foreground/5')
  })

  it('becomes full-width and content-driven, dropping fixed max-w and height', () => {
    const source = read('mouse-effect-card.tsx')
    expect(source).not.toContain('max-w-md')
    expect(source).not.toContain('h-[400px]')
    expect(source).toContain('w-full')
    expect(source).toContain('min-h-[140px]')
  })

  it('keeps the mouse dot motion effect and keyboard interaction', () => {
    const source = read('mouse-effect-card.tsx')
    expect(source).toContain('motion.div')
    expect(source).toContain('ResizeObserver')
    expect(source).toContain('onKeyDown')
    expect(source).toContain('handleMouseMove')
  })
})
