import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { AdminDemo } from '@/components/demos/AdminDemo'

describe('AdminDemo', () => {
  it('渲染节点与任务信息', () => {
    render(<AdminDemo />)
    expect(screen.getByText(/节点 1/)).toBeInTheDocument()
    expect(screen.getByText(/导入/)).toBeInTheDocument()
  })
})
