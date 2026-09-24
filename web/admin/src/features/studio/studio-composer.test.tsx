import { createRef } from 'react'
import { userEvent } from 'vitest/browser'
import { render } from 'vitest-browser-react'
import { expect, it } from 'vitest'
import '@/styles/index.css'
import { PromptInput, PromptInputBody, PromptInputFooter } from '@/components/ai-elements/prompt-input'
import { StudioComposer, type StudioComposerHandle } from './studio-composer'

it('fills the chat input width when the toolbar is below it', async () => {
  const screen = await render(
    <div style={{ width: 640 }}>
      <PromptInput onSubmit={() => {}} inputGroupClassName='h-auto overflow-visible bg-background'>
        <PromptInputBody>
          <StudioComposer placeholder='输入消息' />
        </PromptInputBody>
        <PromptInputFooter />
      </PromptInput>
    </div>
  )
  const editor = screen.getByRole('textbox', { name: '输入消息' }).element()
  const inputGroup = editor.closest('[data-slot="input-group"]')
  const composer = editor.parentElement?.parentElement
  expect(inputGroup).not.toBeNull()
  expect(composer).not.toBeNull()
  expect(Math.abs(composer!.getBoundingClientRect().width - inputGroup!.getBoundingClientRect().width)).toBeLessThanOrEqual(2)
})

it('aligns placeholder text with the editor first line', async () => {
  const screen = await render(
    <div style={{ width: 640 }}>
      <PromptInput onSubmit={() => {}} inputGroupClassName='h-auto overflow-visible bg-background'>
        <PromptInputBody>
          <StudioComposer placeholder='输入消息' />
        </PromptInputBody>
        <PromptInputFooter />
      </PromptInput>
    </div>
  )
  const editor = screen.getByRole('textbox', { name: '输入消息' }).element()
  const placeholder = screen.getByText('输入消息').element()
  expect(Math.abs(editor.getBoundingClientRect().top - placeholder.getBoundingClientRect().top)).toBeLessThanOrEqual(1)
  expect(getComputedStyle(editor).lineHeight).toBe(getComputedStyle(placeholder).lineHeight)
})

it('keeps the caret height after an inline reference equal to plain text', async () => {
  const composer = createRef<StudioComposerHandle>()
  const screen = await render(<StudioComposer ref={composer} placeholder='输入消息' />)
  const textbox = screen.getByRole('textbox', { name: '输入消息' })
  await textbox.fill('前')

  const plainCaretHeight = window.getSelection()!.getRangeAt(0).getBoundingClientRect().height
  composer.current?.insertReference({ kind: 'skill', id: 'skill-1', label: '分镜草稿' })
  await expect.element(screen.getByText('分镜草稿')).toBeVisible()
  const referenceCaretHeight = window.getSelection()!.getRangeAt(0).getBoundingClientRect().height
  expect(referenceCaretHeight).toBeGreaterThan(0)
  expect(Math.abs(referenceCaretHeight - plainCaretHeight)).toBeLessThanOrEqual(1)
})

it('uses distinct theme colors for Skill and asset references', async () => {
  const composer = createRef<StudioComposerHandle>()
  const screen = await render(
    <div data-testid='theme-root'>
      <StudioComposer ref={composer} placeholder='输入消息' />
    </div>
  )
  composer.current?.insertReference({ kind: 'skill', id: 'skill-1', label: '分镜草稿' })
  composer.current?.insertReference({ kind: 'asset', id: 'asset-1', versionId: 'version-1', label: '产品照片' })

  await expect.element(screen.getByText('分镜草稿')).toBeVisible()
  await expect.element(screen.getByText('产品照片')).toBeVisible()
  const skillBadge = screen.getByText('分镜草稿').element().closest('[data-slot="badge"]')!
  const assetBadge = screen.getByText('产品照片').element().closest('[data-slot="badge"]')!
  expect(getComputedStyle(skillBadge).backgroundColor).not.toBe(getComputedStyle(assetBadge).backgroundColor)
  expect(getComputedStyle(skillBadge).borderTopWidth).toBe('0px')
  expect(getComputedStyle(assetBadge).borderTopWidth).toBe('0px')
  expect(getComputedStyle(skillBadge).color).toBe(getComputedStyle(assetBadge).color)

  const lightSkillColor = getComputedStyle(skillBadge).backgroundColor
  screen.getByTestId('theme-root').element().classList.add('dark')
  expect(getComputedStyle(skillBadge).backgroundColor).not.toBe(lightSkillColor)
  expect(getComputedStyle(skillBadge).backgroundColor).not.toBe(getComputedStyle(assetBadge).backgroundColor)
})

it('inserts and deletes an inline Skill as one editable unit', async () => {
  const composer = createRef<StudioComposerHandle>()
  const screen = await render(<StudioComposer ref={composer} placeholder='输入消息' />)
  const textbox = screen.getByRole('textbox', { name: '输入消息' })

  await textbox.fill('用 /')
  expect(composer.current?.serialize().text).toBe('用 /')
  expect(composer.current?.serialize().selectedSkillIds).toEqual([])

  composer.current?.insertReference({ kind: 'skill', id: 'skill-1', label: '分镜草稿' })
  await expect.element(screen.getByText('分镜草稿')).toBeVisible()
  expect(composer.current?.serialize().selectedSkillIds).toEqual(['skill-1'])

  await userEvent.keyboard('{Backspace}')
  expect(composer.current?.serialize().selectedSkillIds).toEqual([])
})

it('removes the reference and its caret spacer when deleting from before the spacer', async () => {
  const composer = createRef<StudioComposerHandle>()
  const screen = await render(<StudioComposer ref={composer} placeholder='输入消息' />)
  const textbox = screen.getByRole('textbox', { name: '输入消息' })
  await textbox.fill('前')
  composer.current?.insertReference({ kind: 'skill', id: 'skill-1', label: '分镜草稿' })
  await expect.element(screen.getByText('分镜草稿')).toBeVisible()

  await userEvent.keyboard('{ArrowLeft}{Backspace}')
  expect(composer.current?.serialize().selectedSkillIds).toEqual([])
  expect(textbox.element().querySelector('p')?.textContent).toBe('前')
})

it('removes the caret spacer when deleting a selected reference', async () => {
  const composer = createRef<StudioComposerHandle>()
  const screen = await render(<StudioComposer ref={composer} placeholder='输入消息' />)
  const textbox = screen.getByRole('textbox', { name: '输入消息' })
  await textbox.fill('前')
  composer.current?.insertReference({ kind: 'skill', id: 'skill-1', label: '分镜草稿' })
  await expect.element(screen.getByText('分镜草稿')).toBeVisible()

  await screen.getByText('分镜草稿').click()
  await userEvent.keyboard('{Delete}')
  expect(composer.current?.serialize().selectedSkillIds).toEqual([])
  expect(textbox.element().querySelector('p')?.textContent).toBe('前')
})

it('keeps typed text after a reference and deletes the reference in one keystroke', async () => {
  const composer = createRef<StudioComposerHandle>()
  const screen = await render(<StudioComposer ref={composer} placeholder='输入消息' />)
  const textbox = screen.getByRole('textbox', { name: '输入消息' })
  await textbox.fill('前')
  composer.current?.insertReference({ kind: 'asset', id: 'asset-1', versionId: 'version-1', label: '照片' })
  await expect.element(screen.getByText('照片')).toBeVisible()

  await userEvent.keyboard('后')
  expect(composer.current?.serialize().text).toBe('前「照片」资产后')
  await userEvent.keyboard('{Backspace}{Backspace}')
  expect(composer.current?.serialize().selectedAssets).toEqual([])
  expect(textbox.element().querySelector('p')?.textContent).toBe('前')
})

it('omits the caret spacer when text is inserted before it', async () => {
  const composer = createRef<StudioComposerHandle>()
  const screen = await render(<StudioComposer ref={composer} placeholder='输入消息' />)
  const textbox = screen.getByRole('textbox', { name: '输入消息' })
  await textbox.fill('前')
  composer.current?.insertReference({ kind: 'skill', id: 'skill-1', label: '分镜草稿' })
  await expect.element(screen.getByText('分镜草稿')).toBeVisible()

  await userEvent.keyboard('{ArrowLeft}后')
  expect(composer.current?.serialize().text).toBe('前「分镜草稿」Skill后')
})

it('does not copy the caret spacer with a reference', async () => {
  const composer = createRef<StudioComposerHandle>()
  const screen = await render(<StudioComposer ref={composer} placeholder='输入消息' />)
  const textbox = screen.getByRole('textbox', { name: '输入消息' })
  await textbox.fill('前')
  composer.current?.insertReference({ kind: 'skill', id: 'skill-1', label: '分镜草稿' })
  await expect.element(screen.getByText('分镜草稿')).toBeVisible()

  await userEvent.keyboard('{Meta>}a{/Meta}')
  const clipboard = new DataTransfer()
  textbox.element().dispatchEvent(new ClipboardEvent('copy', {
    bubbles: true,
    cancelable: true,
    clipboardData: clipboard,
  }))
  expect(clipboard.getData('text/plain')).toBe('前分镜草稿')
})

it('keeps slash text until a menu item is chosen', async () => {
  const composer = createRef<StudioComposerHandle>()
  const screen = await render(
    <div style={{ paddingTop: 300 }}>
      <StudioComposer
        ref={composer}
        placeholder='输入消息'
        slashItems={[{ kind: 'skill', id: 'skill-1', label: '分镜草稿' }]}
      />
    </div>
  )
  const textbox = screen.getByRole('textbox', { name: '输入消息' })

  await textbox.fill('用 /')
  expect(composer.current?.serialize().text).toBe('用 /')
  await expect.element(screen.getByRole('button', { name: '插入 Skill：分镜草稿' })).toBeVisible()

  await userEvent.keyboard('{Escape}')
  expect(composer.current?.serialize().text).toBe('用 /')
  expect(composer.current?.serialize().selectedSkillIds).toEqual([])

  await textbox.fill('用 /分')
  await screen.getByRole('button', { name: '插入 Skill：分镜草稿' }).click()
  expect(composer.current?.serialize().text).toBe('用 「分镜草稿」Skill')
  expect(composer.current?.serialize().selectedSkillIds).toEqual(['skill-1'])
})

it('chooses a slash reference with the keyboard', async () => {
  const composer = createRef<StudioComposerHandle>()
  const screen = await render(
    <StudioComposer
      ref={composer}
      placeholder='输入消息'
      slashItems={[
        { kind: 'skill', id: 'skill-1', label: '分镜草稿' },
        { kind: 'asset', id: 'asset-1', versionId: 'version-1', label: '产品照片' },
      ]}
    />
  )

  await screen.getByRole('textbox', { name: '输入消息' }).fill('用 /')
  await expect.element(screen.getByRole('button', { name: '插入 Skill：分镜草稿' })).toBeVisible()
  await userEvent.keyboard('{ArrowDown}{Enter}')
  expect(composer.current?.serialize().text).toBe('用 「产品照片」资产')
  expect(composer.current?.serialize().selectedAssets).toEqual([
    { assetId: 'asset-1', assetVersionId: 'version-1' },
  ])
})
