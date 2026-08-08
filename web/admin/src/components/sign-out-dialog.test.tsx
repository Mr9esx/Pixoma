import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import { userEvent } from 'vitest/browser'
import { SignOutDialog } from './sign-out-dialog'

describe('SignOutDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('closes without navigating to sign-in', async () => {
    const onOpenChange = vi.fn()
    const { getByRole } = await render(
      <SignOutDialog open onOpenChange={onOpenChange} />
    )

    await userEvent.click(getByRole('button', { name: /^Close$/i }))

    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it('does not call onOpenChange(false) when Cancel is clicked', async () => {
    const onOpenChange = vi.fn()
    const { getByRole } = await render(
      <SignOutDialog open onOpenChange={onOpenChange} />
    )

    await userEvent.click(getByRole('button', { name: /^Cancel$/i }))

    expect(onOpenChange).not.toHaveBeenCalledWith(false)
  })
})
