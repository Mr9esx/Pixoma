import { describe, expect, it } from 'vitest'
import type { StudioSkill } from '@/lib/api/studio'
import {
  addFile,
  addFolder,
  applySavedSkill,
  copyPath,
  deletePath,
  initDraft,
  readMetadata,
  renamePath,
  serializeDraft,
  updateFileContent,
  updateMetadata,
  validateDraft,
} from './skill-document'

const savedSkill: StudioSkill = {
  id: 'skill-1',
  name: 'sample-skill',
  description: '整理资料',
  prompt: '[SKILL.md]\n---\nname: sample-skill\n---\n整理资料',
  files: [
    {
      path: 'SKILL.md',
      content:
        '---\n# 保留说明\nname: sample-skill # 名称注释\ndescription: 整理资料\nlicense: MIT\ntags:\n  - research\n---\n\n# 操作说明\n整理资料。\n',
    },
    { path: 'references', content: '', directory: true },
    { path: 'references/guide.md', content: '# 指南\n' },
  ],
  version: '1.0.0',
  enabled: true,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-02T00:00:00Z',
}

describe('Skill 草稿', () => {
  it('让右侧元数据编辑保留其他 YAML 属性、注释和正文', () => {
    const draft = updateMetadata(initDraft(savedSkill), {
      name: 'updated-skill',
      description: '更新说明',
    })
    const content = draft.files.find(
      (file) => file.path === 'SKILL.md'
    )?.content

    expect(readMetadata(draft)).toMatchObject({
      name: 'updated-skill',
      description: '更新说明',
    })
    expect(content).toContain('# 保留说明')
    expect(content).toContain('# 名称注释')
    expect(content).toContain('license: MIT')
    expect(content).toContain('- research')
    expect(content).toContain('# 操作说明\n整理资料。')
  })

  it('读取源码中编辑的名称和说明', () => {
    const draft = updateFileContent(
      initDraft(),
      'SKILL.md',
      '---\nname: source-name\ndescription: 源码说明\n---\n\n# 操作说明\n执行。\n'
    )

    expect(readMetadata(draft)).toMatchObject({
      name: 'source-name',
      description: '源码说明',
    })
    expect(serializeDraft(draft)).toMatchObject({
      name: 'source-name',
      description: '源码说明',
      enabled: true,
    })
  })

  it('编辑说明时保留正在输入的空格，保存时清理首尾空白', () => {
    const first = updateMetadata(initDraft(savedSkill), {
      description: '多词 ',
    })
    expect(readMetadata(first).description).toBe('多词 ')
    expect(first.files[0].content).toContain('description: "多词 "')

    const second = updateMetadata(first, { description: '多词 说明 ' })
    expect(readMetadata(second).description).toBe('多词 说明 ')
    expect(serializeDraft(second).description).toBe('多词 说明')
  })

  it('保存期间继续编辑时保留新输入，并以服务端版本作为比较基准', () => {
    const submitted = updateMetadata(initDraft(savedSkill), {
      description: '已提交说明',
    })
    const saved: StudioSkill = {
      ...savedSkill,
      description: '已提交说明',
      files: serializeDraft(submitted).files,
      updated_at: '2026-01-03T00:00:00Z',
    }
    const current = updateFileContent(
      submitted,
      'references/guide.md',
      '# 保存期间新增的内容'
    )

    const changed = applySavedSkill(current, submitted, saved)
    expect(changed.close).toBe(false)
    expect(
      changed.draft.files.find((file) => file.path === 'references/guide.md')
        ?.content
    ).toBe('# 保存期间新增的内容')
    expect(changed.baseline).toEqual(initDraft(saved))
    expect(changed.skill.id).toBe('skill-1')

    const unchanged = applySavedSkill(submitted, submitted, saved)
    expect(unchanged.close).toBe(true)
    expect(unchanged.draft).toEqual(initDraft(saved))
  })

  it('保留旧版 Skill 的 prompt 保存方式', () => {
    const legacy = initDraft({
      ...savedSkill,
      files: undefined,
      prompt: '旧版说明',
    })
    const changed = updateMetadata(
      updateFileContent(legacy, 'SKILL.md', '更新后的说明'),
      { description: '新版说明' }
    )

    expect(serializeDraft(changed)).toEqual({
      name: 'sample-skill',
      description: '新版说明',
      prompt: '更新后的说明',
      enabled: true,
    })
  })

  it('旧版 Skill 新增资源后保存完整文件包', () => {
    const legacy = initDraft({
      ...savedSkill,
      files: undefined,
      prompt: '# 原有操作说明\n执行任务。',
    })
    const draft = addFile(legacy, {
      path: 'references/guide.md',
      content: '# 附加资料',
    })

    const result = serializeDraft(draft)
    expect(result.files?.map((file) => file.path)).toEqual([
      'SKILL.md',
      'references/guide.md',
    ])
    expect(result.files?.[0].content).toContain('# 原有操作说明\n执行任务。')
    expect(readMetadata(draft)).toMatchObject({ name: 'sample-skill' })
  })

  it('保存目录、文本与二进制资源', () => {
    const draft = initDraft(savedSkill)
    const result = serializeDraft(draft)

    expect(result.files).toEqual(savedSkill.files)
    expect(result.prompt).toBe('')
  })

  it('新增空目录后可以重命名，并递归更新其子项', () => {
    const draft = addFolder(initDraft(savedSkill), 'assets')
    const renamed = renamePath(
      addFile(draft, { path: 'assets/readme.md', content: '内容' }),
      'assets',
      'examples'
    )

    expect(renamed.files.map((file) => file.path)).toContain('examples')
    expect(renamed.files.map((file) => file.path)).toContain(
      'examples/readme.md'
    )
    expect(renamed.files.map((file) => file.path)).not.toContain('assets')
  })

  it('保留原路径时草稿内容与文件顺序保持一致', () => {
    const draft = addFile(
      addFile(initDraft(savedSkill), { path: 'a.md', content: '文件 A' }),
      { path: 'b.md', content: '文件 B' }
    )
    const unchanged = renamePath(draft, 'a.md', 'a.md')

    expect(unchanged).toEqual(draft)
    expect(unchanged.files.map((file) => file.path)).toEqual(
      draft.files.map((file) => file.path)
    )
  })

  it('复制目录时复制所有子项，删除目录时移除所有子项', () => {
    const draft = copyPath(initDraft(savedSkill), 'references', 'examples')
    expect(draft.files.map((file) => file.path)).toContain('examples/guide.md')

    const deleted = deletePath(draft, 'references')
    expect(deleted.files.map((file) => file.path)).toEqual([
      'SKILL.md',
      'examples',
      'examples/guide.md',
    ])
  })

  it('阻止文件与目录的路径冲突，以及删除根 SKILL.md', () => {
    const draft = initDraft(savedSkill)

    expect(() => addFolder(draft, 'references/guide.md')).toThrow('已存在')
    expect(() => addFile(draft, { path: 'references', content: '' })).toThrow(
      '已存在'
    )
    expect(() => renamePath(draft, 'references', 'SKILL.md')).toThrow('已存在')
    expect(() => addFile(draft, { path: '../escape.md', content: '' })).toThrow(
      '路径无效'
    )
    expect(() => deletePath(draft, 'SKILL.md')).toThrow('SKILL.md')
  })

  it('限制文件路径的 UTF-8 字节数和路径段数', () => {
    const draft = initDraft(savedSkill)

    expect(() =>
      addFile(draft, { path: `${'中'.repeat(171)}.md`, content: '' })
    ).toThrow('路径无效')
    expect(() => addFolder(draft, Array(17).fill('folder').join('/'))).toThrow(
      '路径无效'
    )
    expect(() =>
      addFile(draft, {
        path: `${Array(16).fill('folder').join('/')}/guide.md`,
        content: '',
      })
    ).toThrow('路径无效')
  })

  it('指出无效元数据、缺失正文和无效二进制资源', () => {
    const draft = initDraft()
    expect(validateDraft(draft)).toEqual(
      expect.arrayContaining([
        expect.stringContaining('name'),
        expect.stringContaining('description'),
        expect.stringContaining('操作说明'),
      ])
    )

    const withResource = addFile(draft, {
      path: 'assets/logo.png',
      content: '%%%invalid',
      binary: true,
    })
    expect(validateDraft(withResource)).toEqual(
      expect.arrayContaining([expect.stringContaining('编码无效')])
    )
    expect(() => serializeDraft(withResource)).toThrow()
  })
})
