import { createElement } from 'react'
import '@/styles/index.css'
import { describe, expect, it } from 'vitest'
import { render } from 'vitest-browser-react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { StudioFlow } from './studio-flow'
import { StudioWorkbenchBackground } from './studio-workbench-background'

describe('Studio 工作台点阵', () => {
  it('切换标签页时保持同一个 React Flow 点阵', async () => {
    const screen = await render(
      createElement(
        Tabs,
        { defaultValue: 'flow', className: 'relative h-[400px] w-[600px]' },
        createElement(StudioWorkbenchBackground),
        createElement(
          TabsList,
          null,
          createElement(TabsTrigger, { value: 'flow' }, '制作流程'),
          createElement(TabsTrigger, { value: 'assets' }, '会话资产')
        ),
        createElement(
          TabsContent,
          { value: 'flow', className: 'min-h-0' },
          createElement(StudioFlow, {
            nodes: [],
            edges: [],
            background: false,
          })
        ),
        createElement(TabsContent, { value: 'assets' }, '资产内容')
      )
    )
    const background = screen.container.querySelector<SVGSVGElement>(
      'svg.react-flow__background'
    )!
    expect(
      screen.container.querySelectorAll('svg.react-flow__background')
    ).toHaveLength(1)
    const flowCanvas =
      screen.container.querySelector<HTMLElement>('.react-flow')!
    expect(getComputedStyle(flowCanvas).backgroundColor).toBe(
      'rgba(0, 0, 0, 0)'
    )
    const pattern = background.querySelector('pattern')!
    const dot = pattern.querySelector('circle')!
    const before = background.getBoundingClientRect()
    const origin = [
      pattern.getAttribute('x'),
      pattern.getAttribute('y'),
      pattern.getAttribute('patternTransform'),
    ]
    const dotColor = getComputedStyle(dot).fill
    const backgroundColor = getComputedStyle(background).backgroundColor
    expect(pattern.getAttribute('width')).toBe('20')
    expect(pattern.getAttribute('height')).toBe('20')
    expect(dot.getAttribute('r')).toBe('0.5')

    await screen.getByRole('tab', { name: '会话资产' }).click()

    expect(screen.container.querySelector('svg.react-flow__background')).toBe(
      background
    )
    expect(background.getBoundingClientRect()).toEqual(before)
    expect([
      pattern.getAttribute('x'),
      pattern.getAttribute('y'),
      pattern.getAttribute('patternTransform'),
    ]).toEqual(origin)
    expect(pattern.getAttribute('width')).toBe('20')
    expect(pattern.getAttribute('height')).toBe('20')
    expect(dot.getAttribute('r')).toBe('0.5')
    expect(getComputedStyle(dot).fill).toBe(dotColor)
    expect(getComputedStyle(background).backgroundColor).toBe(backgroundColor)
  })
})
