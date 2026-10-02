import { createRef } from 'react'
import { userEvent } from 'vitest/browser'
import { render } from 'vitest-browser-react'
import { expect, it } from 'vitest'
import '@/styles/index.css'
import { PromptInput, PromptInputBody, PromptInputFooter } from '@/components/ai-elements/prompt-input'
import { Button } from '@/components/ui/button'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { StudioComposer, type StudioComposerHandle } from './studio-composer'
import { retainStudioComposerFocus } from './studio-composer-focus'

it('资产引用菜单在尚无候选项时仍显示空状态', async () => {
  const screen = await render(<StudioComposer placeholder='输入消息' />)
  await screen.getByRole('textbox', { name: '输入消息' }).fill('@')
  await expect.element(screen.getByText('没有匹配的资产')).toBeVisible()
})

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

for (const reference of [
  { kind: 'skill' as const, id: 'skill-1', label: '分镜草稿' },
  { kind: 'asset' as const, id: 'asset-1', versionId: 'version-1', label: '产品照片' },
  { kind: 'workflow' as const, id: 'workflow-1', label: '角色三视图' },
]) {
  it(`keeps the caret height equal on both sides of a ${reference.kind} badge`, async () => {
    const composer = createRef<StudioComposerHandle>()
    const screen = await render(<StudioComposer ref={composer} placeholder='输入消息' />)
    const textbox = screen.getByRole('textbox', { name: '输入消息' })
    await textbox.click()
    composer.current?.insertReference(reference)
    await expect.element(screen.getByText(reference.label)).toBeVisible()

    const afterHeight = window.getSelection()!.getRangeAt(0).getBoundingClientRect().height
    await userEvent.keyboard('{ArrowLeft}')
    const beforeHeight = window.getSelection()!.getRangeAt(0).getBoundingClientRect().height
    expect(beforeHeight).toBeGreaterThan(0)
    expect(Math.abs(beforeHeight - afterHeight)).toBeLessThanOrEqual(1)
  })
}

it('returns focus to the editor after choosing a workflow from a menu', async () => {
  const composer = createRef<StudioComposerHandle>()
  const screen = await render(
    <div>
      <StudioComposer ref={composer} placeholder='输入消息' />
      <DropdownMenu>
        <DropdownMenuTrigger asChild><Button type='button'>选择工作流</Button></DropdownMenuTrigger>
        <DropdownMenuContent onCloseAutoFocus={retainStudioComposerFocus}>
          <DropdownMenuItem onSelect={() => composer.current?.insertReference({ kind: 'workflow', id: '12', label: '角色三视图' })}>
            角色三视图
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
  const textbox = screen.getByRole('textbox', { name: '输入消息' })
  const trigger = screen.getByRole('button', { name: '选择工作流' })
  await trigger.click()
  await screen.getByRole('menuitem', { name: '角色三视图' }).click()
  await expect.element(screen.getByText('角色三视图')).toBeVisible()
  await expect.poll(() => document.activeElement === textbox.element()).toBe(true)
  await userEvent.keyboard('后')
  expect(composer.current?.serialize().text).toBe('「角色三视图」工作流后')

  await trigger.click()
  await userEvent.keyboard('{Escape}')
  await expect.poll(() => document.activeElement === trigger.element()).toBe(true)
})

it('inserts a workflow as an editable message reference', async () => {
  const composer = createRef<StudioComposerHandle>()
  const screen = await render(<StudioComposer ref={composer} placeholder='输入消息' />)
  composer.current?.insertReference({ kind: 'workflow', id: '12', label: '角色三视图' })

  await expect.element(screen.getByText('角色三视图')).toBeVisible()
  await expect.poll(() => document.activeElement === screen.getByRole('textbox', { name: '输入消息' }).element()).toBe(true)
  expect(composer.current?.serialize().parts).toEqual([
    { type: 'workflow_ref', workflow_id: '12', name: '角色三视图' },
  ])
  await userEvent.keyboard('{Backspace}')
  expect(composer.current?.serialize().parts).toEqual([])
})

it('keeps the caret outside the workflow badge after ArrowLeft', async () => {
  const composer = createRef<StudioComposerHandle>()
  const screen = await render(<StudioComposer ref={composer} placeholder='输入消息' />)
  await screen.getByRole('textbox', { name: '输入消息' }).click()
  composer.current?.insertReference({ kind: 'workflow', id: '12', label: '角色三视图' })
  const badgeLocator = screen.getByText('角色三视图')
  await expect.element(badgeLocator).toBeVisible()
  await userEvent.keyboard('{ArrowLeft}')
  const caret = window.getSelection()!.getRangeAt(0).getBoundingClientRect()
  expect(caret.left).toBeLessThan(badgeLocator.element().getBoundingClientRect().left)
  await userEvent.keyboard('前{ArrowRight}后')
  expect(composer.current?.serialize().text).toBe('前「角色三视图」工作流后')
})

for (const prefix of ['前文', '前👩‍💻']) {
  it(`moves through ${prefix} with each ArrowLeft after a badge`, async () => {
    const composer = createRef<StudioComposerHandle>()
    const screen = await render(<StudioComposer ref={composer} placeholder='输入消息' />)
    const textbox = screen.getByRole('textbox', { name: '输入消息' })
    await textbox.fill(prefix)
    composer.current?.insertReference({ kind: 'workflow', id: '12', label: '角色三视图' })
    await expect.element(screen.getByText('角色三视图')).toBeVisible()

    await userEvent.keyboard('{ArrowLeft}')
    const beforeBadge = window.getSelection()!.getRangeAt(0).getBoundingClientRect().left
    await userEvent.keyboard('{ArrowLeft}')
    const beforeLastCharacter = window.getSelection()!.getRangeAt(0).getBoundingClientRect().left
    expect(beforeLastCharacter).toBeLessThan(beforeBadge)
    await userEvent.keyboard('{ArrowLeft}')
    const beforeFirstCharacter = window.getSelection()!.getRangeAt(0).getBoundingClientRect().left
    expect(beforeFirstCharacter).toBeLessThan(beforeLastCharacter)
  })
}

it('moves to the previous line after crossing a badge at the line start', async () => {
  const composer = createRef<StudioComposerHandle>()
  const screen = await render(<StudioComposer ref={composer} placeholder='输入消息' />)
  const textbox = screen.getByRole('textbox', { name: '输入消息' })
  await textbox.fill('上一行')
  await userEvent.keyboard('{Shift>}{Enter}{/Shift}')
  composer.current?.insertReference({ kind: 'workflow', id: '12', label: '角色三视图' })
  await expect.element(screen.getByText('角色三视图')).toBeVisible()

  await userEvent.keyboard('{ArrowLeft}')
  const lineTop = window.getSelection()!.getRangeAt(0).getBoundingClientRect().top
  await userEvent.keyboard('{ArrowLeft}')
  const previousLineTop = window.getSelection()!.getRangeAt(0).getBoundingClientRect().top
  expect(previousLineTop).toBeLessThan(lineTop)
})

it('continues typing after replacing a clicked workflow badge', async () => {
  const composer = createRef<StudioComposerHandle>()
  const screen = await render(<StudioComposer ref={composer} placeholder='输入消息' />)
  const textbox = screen.getByRole('textbox', { name: '输入消息' })
  await textbox.fill('前')
  composer.current?.insertReference({ kind: 'workflow', id: '12', label: '角色三视图' })
  const badge = screen.getByText('角色三视图')
  await expect.element(badge).toBeVisible()

  await badge.click()
  await userEvent.keyboard('中后')
  expect(composer.current?.serialize().text).toBe('前中后')
})

for (const reference of [
  { kind: 'skill' as const, id: 'skill-1', label: '分镜草稿' },
  { kind: 'asset' as const, id: 'asset-1', versionId: 'version-1', label: '产品照片' },
  { kind: 'workflow' as const, id: 'workflow-1', label: '角色三视图' },
]) {
  it(`keeps the cursor out of a ${reference.kind} reference`, async () => {
    const composer = createRef<StudioComposerHandle>()
    const screen = await render(<StudioComposer ref={composer} placeholder='输入消息' />)
    const textbox = screen.getByRole('textbox', { name: '输入消息' })
    await textbox.fill('前')
    composer.current?.insertReference(reference)
    const badge = screen.getByText(reference.label)
    await expect.element(badge).toBeVisible()

    const nodeView = badge.element().closest('.node-studioReference')
    expect(nodeView?.getAttribute('contenteditable')).toBe('false')
    expect(getComputedStyle(nodeView!).userSelect).toBe('none')
    await badge.click()
    expect(window.getSelection()?.anchorNode?.parentElement?.closest('[data-slot="badge"]')).toBeNull()
    window.getSelection()?.collapse(badge.element().lastChild, 1)
    await expect.poll(() => window.getSelection()?.anchorNode?.parentElement?.closest('[data-slot="badge"]') ?? null).toBeNull()
    await userEvent.keyboard('中')
    expect(composer.current?.serialize().parts.some((part) => part.type === 'text' && part.text.includes('中'))).toBe(true)
    expect(composer.current?.serialize().parts.some((part) => part.type === `${reference.kind}_ref`)).toBe(true)
    await textbox.click()
    await userEvent.keyboard('后')
    expect(composer.current?.serialize().text).toContain('后')
  })
}

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

it('removes the reference and its caret spacer when deleting from before the reference', async () => {
  const composer = createRef<StudioComposerHandle>()
  const screen = await render(<StudioComposer ref={composer} placeholder='输入消息' />)
  const textbox = screen.getByRole('textbox', { name: '输入消息' })
  await textbox.fill('前')
  composer.current?.insertReference({ kind: 'skill', id: 'skill-1', label: '分镜草稿' })
  await expect.element(screen.getByText('分镜草稿')).toBeVisible()

  await userEvent.keyboard('{ArrowLeft}{Delete}')
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

it('inserts text before a reference after ArrowLeft', async () => {
  const composer = createRef<StudioComposerHandle>()
  const screen = await render(<StudioComposer ref={composer} placeholder='输入消息' />)
  const textbox = screen.getByRole('textbox', { name: '输入消息' })
  await textbox.fill('前')
  composer.current?.insertReference({ kind: 'skill', id: 'skill-1', label: '分镜草稿' })
  await expect.element(screen.getByText('分镜草稿')).toBeVisible()

  await userEvent.keyboard('{ArrowLeft}后')
  expect(composer.current?.serialize().text).toBe('前后「分镜草稿」技能')
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
        referenceItems={[{ kind: 'skill', id: 'skill-1', label: '分镜草稿' }]}
      />
    </div>
  )
  const textbox = screen.getByRole('textbox', { name: '输入消息' })

  await textbox.fill('用 /')
  expect(composer.current?.serialize().text).toBe('用 /')
  await expect.element(screen.getByRole('button', { name: '插入技能：分镜草稿' })).toBeVisible()

  await userEvent.keyboard('{Escape}')
  expect(composer.current?.serialize().text).toBe('用 /')
  expect(composer.current?.serialize().selectedSkillIds).toEqual([])

  await textbox.fill('用 /分')
  await screen.getByRole('button', { name: '插入技能：分镜草稿' }).click()
  expect(composer.current?.serialize().text).toBe('用 「分镜草稿」技能')
  expect(composer.current?.serialize().selectedSkillIds).toEqual(['skill-1'])
})

it('使用 @ 和键盘选择资产引用', async () => {
  const composer = createRef<StudioComposerHandle>()
  const screen = await render(
    <StudioComposer
      ref={composer}
      placeholder='输入消息'
      referenceItems={[
        { kind: 'skill', id: 'skill-1', label: '分镜草稿' },
        { kind: 'asset', id: 'asset-1', versionId: 'version-1', label: '产品照片' },
      ]}
    />
  )

  await screen.getByRole('textbox', { name: '输入消息' }).fill('用 @')
  await expect.element(screen.getByRole('button', { name: '插入资产：产品照片' })).toBeVisible()
  await userEvent.keyboard('{ArrowDown}{Enter}')
  expect(composer.current?.serialize().text).toBe('用 「产品照片」资产')
  expect(composer.current?.serialize().selectedAssets).toEqual([
    { assetId: 'asset-1', assetVersionId: 'version-1' },
  ])
})
