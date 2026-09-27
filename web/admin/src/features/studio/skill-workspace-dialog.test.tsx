import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@/styles/index.css'
import { describe, expect, it } from 'vitest'
import { render } from 'vitest-browser-react'
import { page, userEvent } from 'vitest/browser'
import type { StudioSkill } from '@/lib/api/studio'
import { SkillWorkspaceDialog } from './skill-workspace-dialog'

const imported = {
  name: 'sample-skill',
  description: '整理资料',
  files: [
    {
      path: 'SKILL.md',
      content:
        '---\nname: sample-skill\ndescription: 整理资料\n---\n\n# 操作说明\n\n按照资料清单工作。',
    },
    { path: 'references/guide.md', content: '# 参考资料\n\n这里有相关资料。' },
  ],
}

function renderWorkspace(
  skill: Pick<StudioSkill, 'name' | 'description' | 'files'> | null = imported
) {
  const queryClient = new QueryClient()
  return render(
    <QueryClientProvider client={queryClient}>
      <SkillWorkspaceDialog
        imported={skill ?? undefined}
        open
        onOpenChange={() => {}}
      />
    </QueryClientProvider>
  )
}

describe('SkillWorkspaceDialog', () => {
  it('打开编辑器时不显示按钮提示，键盘聚焦后显示提示', async () => {
    const screen = await renderWorkspace()
    const dialog = await screen
      .getByRole('dialog', { name: '导入技能' })
      .element()

    expect(document.activeElement).toBe(dialog)
    expect(document.querySelector('[data-slot="tooltip-content"]')).toBeNull()

    await userEvent.keyboard('{Tab}')
    expect(document.activeElement).toBe(
      await screen.getByRole('button', { name: '关闭', exact: true }).element()
    )
    await expect
      .element(screen.getByRole('tooltip', { name: '关闭' }))
      .toBeVisible()
  })

  it('资源管理器和基础信息标题为 14px，工具按钮图标为 16px', async () => {
    const screen = await renderWorkspace()
    for (const title of ['资源管理器', '基础信息']) {
      const element = await screen.getByText(title, { exact: true }).element()
      expect(getComputedStyle(element).fontSize).toBe('14px')
    }

    for (const label of [
      '新建文件',
      '新建文件夹',
      '上传文件',
      '隐藏基础信息',
    ]) {
      const button = await screen
        .getByRole('button', { name: label, exact: true })
        .element()
      const icon = button.querySelector('svg')!
      expect(button.getBoundingClientRect().width).toBe(32)
      expect(button.getBoundingClientRect().height).toBe(32)
      expect(icon.getBoundingClientRect().width).toBe(16)
      expect(icon.getBoundingClientRect().height).toBe(16)
    }
  })

  it('标题展示未保存状态，不展示技能名称', async () => {
    const screen = await renderWorkspace()
    const header = (await screen.getByRole('dialog').element()).querySelector(
      '[data-slot="dialog-header"]'
    )!

    expect(header.textContent).not.toContain('sample-skill')
    expect(header.querySelector('[data-slot="badge"]')?.textContent).toBe(
      '未保存'
    )
  })

  it('新建 Skill 有更改时才在标题显示未保存状态', async () => {
    const screen = await renderWorkspace(null)
    const header = (await screen.getByRole('dialog').element()).querySelector(
      '[data-slot="dialog-header"]'
    )!

    expect(header.querySelector('[data-slot="badge"]')).toBeNull()
    await screen.getByRole('textbox', { name: '名称' }).fill('sample-skill')
    await expect
      .element(
        screen.getByRole('dialog').getByText('未保存', { exact: true }).first()
      )
      .toBeVisible()
    expect(header.querySelector('[data-slot="badge"]')?.textContent).toBe(
      '未保存'
    )
  })

  it('编辑工作区占据宽屏的主要区域', async () => {
    await page.viewport(1280, 800)
    const screen = await renderWorkspace()
    const dialog = screen.getByRole('dialog', { name: '导入技能' })
    const width = (await dialog.element()).getBoundingClientRect().width
    expect(window.innerWidth).toBeGreaterThan(768)
    expect(width).toBeGreaterThan(window.innerWidth * 0.9)
  })

  it('基础信息按钮在展开与折叠后保持位置，并位于对应工具栏', async () => {
    await page.viewport(1280, 800)
    const screen = await renderWorkspace()
    const inspector = screen.getByRole('button', { name: '隐藏基础信息' })
    const close = screen.getByRole('button', { name: '关闭', exact: true })
    const inspectorRect = (await inspector.element()).getBoundingClientRect()
    expect(
      (await inspector.element()).closest('#skill-inspector')
    ).not.toBeNull()
    expect(
      (await close.element()).closest('[data-slot="dialog-header"]')
    ).not.toBeNull()
    expect(
      (await inspector.element()).closest('[data-slot="dialog-header"]')
    ).toBeNull()

    await inspector.click()
    const show = screen.getByRole('button', { name: '显示基础信息' })
    await expect.element(show).toBeVisible()
    const showElement = await show.element()
    const showRect = showElement.getBoundingClientRect()
    const modeGroup = await screen
      .getByRole('group', { name: '编辑视图' })
      .element()

    expect(modeGroup.getAttribute('data-slot')).toBe('button-group')
    expect(showElement.previousElementSibling).toBe(modeGroup)
    expect(showElement.getAttribute('aria-expanded')).toBe('false')
    expect(showRect.width).toBe(inspectorRect.width)
    expect(showRect.height).toBe(inspectorRect.height)
    expect(
      Math.abs(
        showRect.left +
          showRect.width / 2 -
          (inspectorRect.left + inspectorRect.width / 2)
      )
    ).toBeLessThanOrEqual(1)
    expect(
      Math.abs(
        showRect.top +
          showRect.height / 2 -
          (inspectorRect.top + inspectorRect.height / 2)
      )
    ).toBeLessThanOrEqual(1)

    await show.click()
    await expect.element(inspector).toBeVisible()
    expect((await inspector.element()).getAttribute('aria-expanded')).toBe(
      'true'
    )
  })

  it('编辑、预览、分栏展示当前选中状态', async () => {
    const screen = await renderWorkspace()
    const tabs = ['编辑', '预览', '分栏'] as const
    const modeGroup = await screen
      .getByRole('group', { name: '编辑视图' })
      .element()

    expect(modeGroup.getAttribute('data-slot')).toBe('button-group')

    for (const name of tabs) {
      await screen.getByRole('button', { name }).click()
      const active = await screen.getByRole('button', { name }).element()
      const inactive = await screen
        .getByRole('button', { name: name === '编辑' ? '预览' : '编辑' })
        .element()

      expect(active.getAttribute('data-variant')).toBe('ghost')
      expect(active.getAttribute('aria-pressed')).toBe('true')
      expect(inactive.getAttribute('aria-pressed')).toBe('false')
      expect(getComputedStyle(active).backgroundColor).not.toBe(
        getComputedStyle(inactive).backgroundColor
      )
    }
  })

  it('保存确认框允许修改版本号并查看每个文件的差异', async () => {
    const screen = await renderWorkspace()
    await screen.getByRole('button', { name: '保存', exact: true }).click()

    const review = screen.getByRole('dialog', { name: '确认保存技能' })
    await expect.element(review).toBeVisible()
    const version = screen.getByRole('textbox', { name: '版本号' })
    await expect.element(version).toHaveValue('1.0.0')
    await version.fill('1.2.3')
    await expect.element(version).toHaveValue('1.2.3')

    const skillFile = review.getByRole('button', { name: /SKILL\.md.*新增/ })
    await expect.element(skillFile).toHaveAttribute('aria-current', 'true')
    expect(
      (await review.element()).querySelector('.cm-content')?.textContent
    ).toContain('name: sample-skill')

    const referenceFile = review.getByRole('button', {
      name: /references\/guide\.md.*新增/,
    })
    await referenceFile.click()
    await expect.element(referenceFile).toHaveAttribute('aria-current', 'true')
    expect(
      (await review.element()).querySelector('.cm-content')?.textContent
    ).toContain('# 参考资料')
  })

  it('Ctrl+S 与 Cmd+S 均打开保存确认框', async () => {
    const screen = await renderWorkspace()
    const dialog = screen.getByRole('dialog', { name: '导入技能' })

    for (const modifier of ['ctrlKey', 'metaKey'] as const) {
      const event = new KeyboardEvent('keydown', {
        key: 's',
        [modifier]: true,
        bubbles: true,
        cancelable: true,
      })
      ;(await dialog.element()).dispatchEvent(event)
      expect(event.defaultPrevented).toBe(true)
      await expect
        .element(screen.getByRole('dialog', { name: '确认保存技能' }))
        .toBeVisible()
      await screen.getByRole('button', { name: '取消' }).click()
    }
  })

  it('同步元数据并渲染 Markdown 预览', async () => {
    const screen = await renderWorkspace()
    await screen.getByRole('textbox', { name: '名称' }).fill('updated-skill')
    await expect
      .element(screen.getByRole('textbox', { name: '名称' }))
      .toHaveValue('updated-skill')
    expect(document.querySelector('.cm-content')?.textContent).toContain(
      'name: updated-skill'
    )

    await screen.getByRole('button', { name: '预览' }).click()
    await expect
      .element(screen.getByRole('heading', { name: '操作说明' }))
      .toBeVisible()
  })

  it('支持文件树右键重命名及未保存退出确认', async () => {
    const screen = await renderWorkspace()
    await screen
      .getByRole('treeitem', { name: 'guide.md' })
      .click({ button: 'right' })
    await screen.getByRole('menuitem', { name: '重命名' }).click()
    await screen
      .getByRole('textbox', { name: '文件路径' })
      .fill('references/new-guide.md')
    await screen.getByRole('button', { name: '确认' }).click()
    await expect
      .element(screen.getByRole('treeitem', { name: 'new-guide.md' }))
      .toBeVisible()

    await screen.getByRole('button', { name: '关闭', exact: true }).click()
    await expect.element(screen.getByRole('alertdialog')).toBeVisible()
    await screen.getByRole('button', { name: '继续编辑' }).click()
    await expect
      .element(screen.getByRole('dialog', { name: '导入技能' }))
      .toBeVisible()
  })

  it('键盘菜单作用于当前焦点文件', async () => {
    const screen = await renderWorkspace()
    const target = document.querySelector<HTMLElement>(
      '[data-skill-path="references/guide.md"]'
    )!
    target.focus()
    expect(document.activeElement).toBe(target)
    target.dispatchEvent(
      new KeyboardEvent('keydown', {
        key: 'F10',
        shiftKey: true,
        bubbles: true,
      })
    )
    await expect
      .element(screen.getByRole('menuitem', { name: '新建文件', exact: true }))
      .toBeVisible()
    await screen.getByRole('menuitem', { name: '重命名' }).click()

    await expect
      .element(screen.getByRole('textbox', { name: '文件路径' }))
      .toHaveValue('references/guide.md')
  })

  it('右键文件夹上传到该文件夹', async () => {
    const screen = await renderWorkspace()
    await screen
      .getByRole('treeitem', { name: 'references' })
      .click({ button: 'right' })
    await screen.getByRole('menuitem', { name: '上传文件' }).click()

    const input = document.querySelector<HTMLInputElement>('input[type=file]')!
    const transfer = new DataTransfer()
    transfer.items.add(
      new File(['# 新资料'], 'upload.md', { type: 'text/markdown' })
    )
    input.files = transfer.files
    input.dispatchEvent(new Event('change', { bubbles: true }))

    await expect
      .element(screen.getByRole('treeitem', { name: 'upload.md' }))
      .toBeVisible()
    expect(
      document.querySelector('[data-skill-path="references/upload.md"]')
    ).not.toBeNull()
  })

  it('在文件树文件夹内新建文件', async () => {
    const screen = await renderWorkspace()
    await screen
      .getByRole('treeitem', { name: 'references' })
      .click({ button: 'right' })
    await screen
      .getByRole('menuitem', { name: '新建文件', exact: true })
      .click()
    await expect
      .element(screen.getByRole('textbox', { name: '文件路径' }))
      .toHaveValue('references/')
    await screen
      .getByRole('textbox', { name: '文件路径' })
      .fill('references/check.md')
    await screen.getByRole('button', { name: '确认' }).click()
    await expect
      .element(screen.getByRole('treeitem', { name: 'check.md' }))
      .toBeVisible()
  })

  it('预览包内图片并打开相对路径文档', async () => {
    const screen = await renderWorkspace({
      name: 'sample-skill',
      description: '整理资料',
      files: [
        {
          path: 'SKILL.md',
          content:
            '---\nname: sample-skill\ndescription: 整理资料\n---\n\n![预览图](assets/logo.png)\n\n[指南](references/guide.md)\n\n[外部资料](https://example.org)',
        },
        {
          path: 'assets/logo.png',
          content:
            'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jXXYAAAAASUVORK5CYII=',
          binary: true,
        },
        { path: 'references/guide.md', content: '# 参考资料' },
      ],
    })

    await screen.getByRole('button', { name: '预览' }).click()
    const image = document.querySelector<HTMLImageElement>('img[alt="预览图"]')
    expect(image?.getAttribute('src')).toMatch(/^data:image\/png;base64,/)

    await expect
      .element(screen.getByRole('link', { name: '外部资料' }))
      .toHaveAttribute('target', '_blank')
    await expect
      .element(screen.getByRole('link', { name: '外部资料' }))
      .toHaveAttribute('rel', 'noopener noreferrer')

    await screen.getByRole('link', { name: '指南' }).click()
    await expect
      .element(screen.getByRole('treeitem', { name: 'guide.md' }))
      .toHaveAttribute('aria-selected', 'true')
  })

  it('长 Markdown 文档可以在编辑器内部滚动', async () => {
    await renderWorkspace({
      name: 'sample-skill',
      description: '整理资料',
      files: [
        {
          path: 'SKILL.md',
          content: `---\nname: sample-skill\ndescription: 整理资料\n---\n\n${Array.from(
            { length: 300 },
            (_, index) => `第 ${index + 1} 行操作说明。`
          ).join('\n')}`,
        },
      ],
    })

    const scroller = document.querySelector<HTMLElement>('.cm-scroller')!
    expect(scroller.scrollHeight).toBeGreaterThan(scroller.clientHeight)
    scroller.scrollTop = 400
    await new Promise((resolve) => requestAnimationFrame(resolve))
    expect(scroller.scrollTop).toBeGreaterThan(0)
  })
})
