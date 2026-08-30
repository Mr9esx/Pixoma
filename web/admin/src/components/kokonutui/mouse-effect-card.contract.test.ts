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
    expect(source).toContain('bg-muted-foreground/60')
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

  it('renders children as an independent block and vertically centers content', () => {
    const source = read('mouse-effect-card.tsx')
    expect(source).toMatch(/\{children\s*\?/)
    expect(source).toMatch(/justify-center/)
    expect(source).toMatch(/items-start justify-center text-left/)
  })

  it('allows the card height to drive a full-height centered content area', () => {
    const source = read('mouse-effect-card.tsx')
    expect(source).toContain(
      'className="relative h-full min-h-[140px] w-full overflow-hidden p-0"'
    )
    expect(source).toContain('h-full min-h-[140px] flex-col')
    expect(source).toContain('items-center justify-center')
  })

  it('renders the welcome card at a fixed 288px height', () => {
    const card = read('../../features/dashboard/workbench-welcome-card.tsx')
    expect(card).toContain('h-[288px]')
    expect(card).toContain('justify-center')
    expect(card).toContain('text-center')
  })
})
