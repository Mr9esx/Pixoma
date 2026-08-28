import { beforeAll, describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { UseCasesSection } from '@/components/sections/UseCasesSection'
import { initI18n } from '@/i18n'

describe('UseCasesSection', () => {
  beforeAll(async () => {
    await initI18n()
  })

  it('渲染使用场景卡片', () => {
    render(<UseCasesSection />)
    expect(screen.getByRole('heading', { level: 2, name: /场景|Cases|Uses/i })).toBeInTheDocument()
  })
})
