import { useState } from 'react'
import '@/styles/index.css'
import { expect, it } from 'vitest'
import { render } from 'vitest-browser-react'
import type { StudioPermissionMode } from '@/lib/api/studio'
import { PermissionPicker } from './studio-chat-controls'

it('requires confirmation before enabling full access', async () => {
  function Example() {
    const [mode, setMode] = useState<StudioPermissionMode>('request_approval')
    return <PermissionPicker value={mode} onChange={setMode} />
  }

  const screen = await render(<Example />)
  const requestApproval = screen.getByRole('button', {
    name: 'Agent 操作权限：请求批准',
  })

  await requestApproval.click()
  await screen.getByRole('menuitem', { name: /完全访问/ }).click()
  await expect.element(screen.getByRole('alertdialog')).toBeVisible()
  await screen.getByRole('button', { name: '取消' }).click()
  await expect.element(requestApproval).toBeVisible()

  await requestApproval.click()
  await screen.getByRole('menuitem', { name: /完全访问/ }).click()
  await screen.getByRole('button', { name: '确认开启' }).click()
  await expect
    .element(screen.getByRole('button', { name: 'Agent 操作权限：完全访问' }))
    .toBeVisible()
})
