import { matter } from 'gray-matter-es'
import { isScalar, parseDocument } from 'yaml'
import type { StudioSkill, StudioSkillFile } from '@/lib/api/studio'

export type SkillDraft = {
  files: StudioSkillFile[]
  enabled: boolean
  legacy?: { name: string; description: string }
}

export function applySavedSkill(
  current: SkillDraft,
  submitted: SkillDraft,
  saved: StudioSkill
): {
  draft: SkillDraft
  baseline: SkillDraft
  skill: StudioSkill
  close: boolean
} {
  const close = JSON.stringify(current) === JSON.stringify(submitted)
  const baseline = initDraft(saved)
  return {
    draft: close ? baseline : current,
    baseline,
    skill: saved,
    close,
  }
}

type SkillMetadata = { name: string; description: string; error?: string }

const maxFiles = 128
const maxFileBytes = 256 << 10
const maxPackageBytes = 1 << 20
const namePattern = /^[a-z0-9]+(?:-[a-z0-9]+)*$/

export function initDraft(skill?: StudioSkill): SkillDraft {
  if (skill?.files?.length) {
    return {
      files: skill.files.map((file) => ({ ...file })),
      enabled: skill.enabled,
    }
  }
  if (skill) {
    return {
      files: [{ path: 'SKILL.md', content: skill.prompt }],
      enabled: skill.enabled,
      legacy: { name: skill.name, description: skill.description },
    }
  }
  return {
    files: [
      { path: 'SKILL.md', content: '---\nname: \ndescription: \n---\n\n' },
    ],
    enabled: true,
  }
}

function manifestContent(draft: SkillDraft): string {
  const file = draft.files.find((entry) => entry.path === 'SKILL.md')
  if (!file || file.binary || file.directory) {
    throw new Error('技能包必须包含文本文件 SKILL.md')
  }
  return file.content
}

function parseManifest(content: string) {
  const normalized = content.replace(/\r\n/g, '\n')
  if (!normalized.startsWith('---\n')) {
    throw new Error('SKILL.md 必须以 YAML 元数据开头')
  }
  const parsed = matter(normalized)
  if (!parsed.matter) {
    throw new Error('SKILL.md YAML 元数据缺少结束标记')
  }
  const document = parseDocument(parsed.matter)
  if (document.errors.length) {
    throw new Error(`SKILL.md YAML 元数据无效：${document.errors[0].message}`)
  }
  return { parsed, document }
}

export function readMetadata(draft: SkillDraft): SkillMetadata {
  if (draft.legacy) return draft.legacy
  try {
    const { parsed } = parseManifest(manifestContent(draft))
    if (
      (parsed.data.name != null && typeof parsed.data.name !== 'string') ||
      (parsed.data.description != null &&
        typeof parsed.data.description !== 'string')
    ) {
      return { name: '', description: '', error: 'SKILL.md 元数据类型无效' }
    }
    return {
      name: parsed.data.name ?? '',
      description: parsed.data.description ?? '',
    }
  } catch (error) {
    return {
      name: '',
      description: '',
      error: error instanceof Error ? error.message : 'SKILL.md 无法读取',
    }
  }
}

export function updateMetadata(
  draft: SkillDraft,
  patch: Partial<Pick<SkillMetadata, 'name' | 'description'>>
): SkillDraft {
  if (draft.legacy) {
    return { ...draft, legacy: { ...draft.legacy, ...patch } }
  }
  const { parsed, document } = parseManifest(manifestContent(draft))
  for (const key of ['name', 'description'] as const) {
    const value = patch[key]
    if (value === undefined) continue
    const node = document.get(key, true)
    if (isScalar(node)) node.value = value
    else document.set(key, value)
  }
  const content = `---\n${document.toString().trimEnd()}\n---\n${parsed.content}`
  return updateFileContent(draft, 'SKILL.md', content)
}

export function updateFileContent(
  draft: SkillDraft,
  path: string,
  content: string
): SkillDraft {
  const file = draft.files.find((entry) => entry.path === path)
  if (!file || file.binary || file.directory) {
    throw new Error(`无法编辑文件 ${path}`)
  }
  return {
    ...draft,
    files: draft.files.map((entry) =>
      entry.path === path ? { ...entry, content } : entry
    ),
  }
}

function validPath(path: string): boolean {
  return (
    path !== '' &&
    !path.startsWith('/') &&
    new TextEncoder().encode(path).length <= 512 &&
    !/[\\:]/.test(path) &&
    !path.includes('\x00') &&
    path.split('/').length <= 16 &&
    path
      .split('/')
      .every((segment) => segment !== '' && segment !== '.' && segment !== '..')
  )
}

function pathConflicts(files: StudioSkillFile[]): string | undefined {
  const paths = new Set<string>()
  for (const file of files) {
    if (!validPath(file.path)) return `文件路径无效：${file.path}`
    if (paths.has(file.path)) return `文件已存在：${file.path}`
    paths.add(file.path)
  }
  for (const file of files) {
    if (
      !file.directory &&
      files.some((other) => other.path.startsWith(`${file.path}/`))
    ) {
      return `文件与目录路径冲突：${file.path}`
    }
  }
}

function promoteLegacy(draft: SkillDraft): SkillDraft {
  if (!draft.legacy || draft.files.length === 1) return draft
  const content = matter({
    content: manifestContent(draft),
    data: {},
  }).stringify({
    name: draft.legacy.name.trim(),
    description: draft.legacy.description.trim(),
  })
  return {
    files: draft.files.map((file) =>
      file.path === 'SKILL.md' ? { ...file, content } : file
    ),
    enabled: draft.enabled,
  }
}

function withFiles(draft: SkillDraft, files: StudioSkillFile[]): SkillDraft {
  const error = pathConflicts(files)
  if (error) throw new Error(error)
  return promoteLegacy({ ...draft, files })
}

export function addFile(draft: SkillDraft, file: StudioSkillFile): SkillDraft {
  if (file.directory) throw new Error('请使用 addFolder 新建目录')
  return withFiles(draft, [...draft.files, { ...file }])
}

export function addFolder(draft: SkillDraft, path: string): SkillDraft {
  return withFiles(draft, [
    ...draft.files,
    { path, content: '', directory: true },
  ])
}

function affectedFiles(draft: SkillDraft, path: string): StudioSkillFile[] {
  const files = draft.files.filter(
    (file) => file.path === path || file.path.startsWith(`${path}/`)
  )
  if (!files.length) throw new Error(`文件路径不存在：${path}`)
  return files
}

export function renamePath(
  draft: SkillDraft,
  source: string,
  target: string
): SkillDraft {
  if (source === 'SKILL.md') throw new Error('SKILL.md 不能重命名')
  if (!validPath(target)) throw new Error(`文件路径无效：${target}`)
  const affected = affectedFiles(draft, source)
  if (source === target) return draft
  if (
    (affected.length > 1 ||
      affected[0].directory ||
      affected[0].path !== source) &&
    target.startsWith(`${source}/`)
  ) {
    throw new Error('目录不能移动到自身内部')
  }
  const moved = affected.map((file) => ({
    ...file,
    path: `${target}${file.path.slice(source.length)}`,
  }))
  const remaining = draft.files.filter((file) => !affected.includes(file))
  return withFiles(draft, [...remaining, ...moved])
}

export function copyPath(
  draft: SkillDraft,
  source: string,
  target: string
): SkillDraft {
  if (!validPath(target)) throw new Error(`文件路径无效：${target}`)
  const copied = affectedFiles(draft, source).map((file) => ({
    ...file,
    path: `${target}${file.path.slice(source.length)}`,
  }))
  return withFiles(draft, [...draft.files, ...copied])
}

export function deletePath(draft: SkillDraft, path: string): SkillDraft {
  if (path === 'SKILL.md') throw new Error('SKILL.md 不能删除')
  const affected = affectedFiles(draft, path)
  return {
    ...draft,
    files: draft.files.filter((file) => !affected.includes(file)),
  }
}

function decodedSize(file: StudioSkillFile): number {
  if (file.directory) {
    if (file.binary || file.content)
      throw new Error(`目录内容无效：${file.path}`)
    return 0
  }
  if (!file.binary) {
    if (file.content.includes('\x00'))
      throw new Error(`文本文件编码无效：${file.path}`)
    return new TextEncoder().encode(file.content).length
  }
  if (
    !/^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/.test(
      file.content
    )
  ) {
    throw new Error(`二进制文件编码无效：${file.path}`)
  }
  return atob(file.content).length
}

export function validateDraft(draft: SkillDraft): string[] {
  const errors: string[] = []
  if (draft.files.length < 1 || draft.files.length > maxFiles) {
    errors.push(`技能包文件数量必须在 1 到 ${maxFiles} 个之间`)
  }
  const pathError = pathConflicts(draft.files)
  if (pathError) errors.push(pathError)
  let totalBytes = 0
  for (const file of draft.files) {
    try {
      const size = decodedSize(file)
      if (size > maxFileBytes) errors.push(`文件超过大小限制：${file.path}`)
      totalBytes += size
    } catch (error) {
      errors.push(
        error instanceof Error ? error.message : `文件无效：${file.path}`
      )
    }
  }
  if (totalBytes > maxPackageBytes) errors.push('技能包超过大小限制')
  const metadata = readMetadata(draft)
  if (metadata.error) errors.push(metadata.error)
  if (draft.legacy) {
    if (!metadata.name.trim()) errors.push('技能名称不能为空')
    if (!metadata.description.trim()) errors.push('技能说明不能为空')
    if (!draft.files[0]?.content.trim()) errors.push('技能缺少操作说明')
  } else {
    if (
      metadata.name.trim().length > 64 ||
      !namePattern.test(metadata.name.trim())
    ) {
      errors.push('SKILL.md name 必须为 1 到 64 位小写字母、数字和连字符')
    }
    if (
      !metadata.description.trim() ||
      Array.from(metadata.description.trim()).length > 1024
    ) {
      errors.push('SKILL.md description 必须在 1 到 1024 个字符之间')
    }
    try {
      const { parsed } = parseManifest(manifestContent(draft))
      if (!parsed.content.trim()) errors.push('SKILL.md 缺少操作说明')
    } catch {
      // 元数据错误已由 readMetadata 报告。
    }
  }
  return errors
}

export function serializeDraft(draft: SkillDraft): {
  name: string
  description: string
  prompt: string
  files?: StudioSkillFile[]
  enabled: boolean
} {
  const packaged = promoteLegacy(draft)
  const metadata = readMetadata(packaged)
  const normalized = metadata.error
    ? packaged
    : updateMetadata(packaged, {
        name: metadata.name.trim(),
        description: metadata.description.trim(),
      })
  const errors = validateDraft(normalized)
  if (errors.length) throw new Error(errors.join('；'))
  const { name, description } = readMetadata(normalized)
  return {
    name,
    description,
    prompt: normalized.legacy ? manifestContent(normalized) : '',
    ...(normalized.legacy
      ? {}
      : { files: normalized.files.map((file) => ({ ...file })) }),
    enabled: normalized.enabled,
  }
}
