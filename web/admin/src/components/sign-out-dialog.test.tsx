import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import { userEvent } from 'vitest/browser'
import { SignOutDialog } from './sign-out-dialog'

describe('SignOutDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('calls onSignOut then closes on confirm', async () => {
    const onOpenChange = vi.fn()
    const onSignOut = vi.fn()
    const { getByRole } = await render(
      <SignOutDialog
        open
        onOpenChange={onOpenChange}
        onSignOut={onSignOut}
      />
    )

    await userEvent.click(getByRole('button', { name: /^退出登录$/ }))

    expect(onSignOut).toHaveBeenCalledOnce()
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it('closes on confirm when no onSignOut handler is provided', async () => {
    const onOpenChange = vi.fn()
    const { getByRole } = await render(
      <SignOutDialog open onOpenChange={onOpenChange} />
    )

    await userEvent.click(getByRole('button', { name: /^退出登录$/ }))

    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it('does not close when Cancel is clicked', async () => {
    const onOpenChange = vi.fn()
    const onSignOut = vi.fn()
    const { getByRole } = await render(
      <SignOutDialog
        open
        onOpenChange={onOpenChange}
        onSignOut={onSignOut}
      />
    )

    await userEvent.click(getByRole('button', { name: /^取消$/ }))

    expect(onSignOut).not.toHaveBeenCalled()
    expect(onOpenChange).not.toHaveBeenCalledWith(false)
  })
})
