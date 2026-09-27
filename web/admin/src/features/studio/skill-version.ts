import { gt, inc, prerelease, valid } from 'semver'
import type { StudioSkillFile } from '@/lib/api/studio'

export type SkillFileChange = {
  path: string
  kind: 'added' | 'modified' | 'deleted'
  before?: StudioSkillFile
  after?: StudioSkillFile
}

export function nextSkillVersion(current?: string): string {
  if (!current) return '1.0.0'
  const next = inc(current, 'patch')
  if (!next) throw new Error('当前技能版本号无效')
  return next
}

export function skillVersionError(version: string, current?: string): string {
  if (valid(version) !== version || prerelease(version) !== null) {
    return '版本号需为 x.y.z'
  }
  if (current && !gt(version, current)) return '版本号需高于当前版本'
  return ''
}

export function listSkillFileChanges(
  before: StudioSkillFile[],
  after: StudioSkillFile[]
): SkillFileChange[] {
  const beforeByPath = new Map(before.map((file) => [file.path, file]))
  const afterByPath = new Map(after.map((file) => [file.path, file]))
  const paths = [
    ...new Set([...beforeByPath.keys(), ...afterByPath.keys()]),
  ].sort()
  const changes: SkillFileChange[] = []

  for (const path of paths) {
    const previous = beforeByPath.get(path)
    const next = afterByPath.get(path)
    if (
      previous &&
      next &&
      previous.content === next.content &&
      Boolean(previous.binary) === Boolean(next.binary) &&
      Boolean(previous.directory) === Boolean(next.directory)
    ) {
      continue
    }
    changes.push({
      path,
      kind: !previous ? 'added' : !next ? 'deleted' : 'modified',
      before: previous,
      after: next,
    })
  }

  return changes
}
