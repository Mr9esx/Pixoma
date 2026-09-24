import '@/styles/index.css'
import { describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import { userEvent } from 'vitest/browser'
import type { StudioFlowNode } from '@/lib/api/studio'
import { StudioFlow } from './studio-flow'

const nodes: StudioFlowNode[] = [
  {
    id: 'stage-a',
    type: 'stage',
    title: '立住角色',
    position: { x: 100, y: 100 },
    sort_order: 10,
    updated_at: '',
  },
  {
    id: 'stage-b',
    type: 'stage',
    title: '排好分镜',
    position: { x: 450, y: 100 },
    sort_order: 20,
    updated_at: '',
  },
]

describe('StudioFlow', () => {
  it('reports a failed connection without keeping a phantom edge', async () => {
    const onEdgeCreate = vi.fn(async () => {
      throw new Error('连接失败')
    })
    const screen = await render(
      <div className='h-[600px] w-[1000px]'>
        <StudioFlow nodes={nodes} edges={[]} onEdgeCreate={onEdgeCreate} />
      </div>
    )
    await expect.element(screen.getByText('立住角色')).toBeVisible()
    const source = screen.container.querySelector(
      '[data-nodeid="stage-a"].react-flow__handle-right'
    ) as HTMLElement
    const target = screen.container.querySelector(
      '[data-nodeid="stage-b"].react-flow__handle-left'
    ) as HTMLElement
    source.click()
    target.click()
    expect(onEdgeCreate).toHaveBeenCalledWith({
      source: 'stage-a',
      target: 'stage-b',
    })
    await expect
      .element(screen.getByRole('alert'))
      .toHaveTextContent('连线创建失败')
    expect(screen.container.querySelectorAll('.react-flow__edge')).toHaveLength(
      0
    )
  })

  it('restores the saved node position after a rejected move', async () => {
    const onPositionsChange = vi.fn(async () => {
      throw new Error('保存失败')
    })
    const screen = await render(
      <div className='h-[600px] w-[1000px]'>
        <StudioFlow
          nodes={nodes}
          edges={[]}
          onPositionsChange={onPositionsChange}
        />
      </div>
    )
    await expect.element(screen.getByText('立住角色')).toBeVisible()
    const node = screen.container.querySelector(
      '.react-flow__node[data-id="stage-a"]'
    ) as HTMLElement
    const original = node.style.transform
    const box = node.getBoundingClientRect()
    const x = box.x + box.width / 2
    const y = box.y + box.height / 2
    node.dispatchEvent(
      new MouseEvent('mousedown', {
        bubbles: true,
        button: 0,
        buttons: 1,
        clientX: x,
        clientY: y,
        view: window,
      })
    )
    for (let step = 1; step <= 5; step++) {
      document.dispatchEvent(
        new MouseEvent('mousemove', {
          bubbles: true,
          button: 0,
          buttons: 1,
          clientX: x + step * 20,
          clientY: y + step * 10,
          view: window,
        })
      )
    }
    document.dispatchEvent(
      new MouseEvent('mouseup', {
        bubbles: true,
        button: 0,
        clientX: x + 100,
        clientY: y + 50,
        view: window,
      })
    )
    expect(onPositionsChange).toHaveBeenCalled()
    await expect
      .element(screen.getByRole('alert'))
      .toHaveTextContent('节点位置保存失败')
    expect(node.style.transform).toBe(original)
  })

  it('keeps a node when deletion is rejected', async () => {
    const onNodeDelete = vi.fn(async () => {
      throw new Error('删除失败')
    })
    const screen = await render(
      <div className='h-[600px] w-[1000px]'>
        <StudioFlow nodes={nodes} edges={[]} onNodeDelete={onNodeDelete} />
      </div>
    )
    await screen.getByText('立住角色').click()
    await userEvent.keyboard('{Delete}')
    expect(onNodeDelete).toHaveBeenCalledWith('stage-a')
    await expect
      .element(screen.getByRole('alert'))
      .toHaveTextContent('节点删除失败')
    await expect.element(screen.getByText('立住角色')).toBeVisible()
  })

  it('removes a node only after the server accepts deletion', async () => {
    const onNodeDelete = vi.fn(async () => undefined)
    const screen = await render(
      <div className='h-[600px] w-[1000px]'>
        <StudioFlow nodes={nodes} edges={[]} onNodeDelete={onNodeDelete} />
      </div>
    )
    await screen.getByText('立住角色').click()
    await userEvent.keyboard('{Delete}')
    expect(onNodeDelete).toHaveBeenCalledWith('stage-a')
    await expect.element(screen.getByText('立住角色')).not.toBeInTheDocument()
  })
})
