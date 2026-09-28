import { useState } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@/styles/index.css'
import { expect, it } from 'vitest'
import { render } from 'vitest-browser-react'
import { StudioProjectDialog } from './studio-project-dialog'

it('keeps a project name editable and closes without saving on cancel', async () => {
  const client = new QueryClient()
  function Example() {
    const [open, setOpen] = useState(true)
    return (
      <QueryClientProvider client={client}>
        {open ? (
          <StudioProjectDialog
            open
            project={{
              id: 'project-1',
              name: '旧名称',
              created_at: '',
              updated_at: '',
            }}
            onOpenChange={setOpen}
          />
        ) : null}
      </QueryClientProvider>
    )
  }
  const screen = await render(<Example />)
  const name = screen.getByRole('textbox', { name: '项目名称' })
  await expect.element(name).toHaveValue('旧名称')
  await name.fill('新名称')
  await screen.getByRole('button', { name: '取消' }).click()
  await expect.element(name).not.toBeInTheDocument()
})
