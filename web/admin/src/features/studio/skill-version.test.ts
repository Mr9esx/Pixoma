import { describe, expect, it } from 'vitest'
import {
  listSkillFileChanges,
  nextSkillVersion,
  skillVersionError,
} from './skill-version'

describe('Skill 版本', () => {
  it('新建时使用 1.0.0，后续保存默认递增补丁号', () => {
    expect(nextSkillVersion()).toBe('1.0.0')
    expect(nextSkillVersion('1.2.3')).toBe('1.2.4')
    expect(skillVersionError('1.3.0', '1.2.3')).toBe('')
  })

  it('只接受高于当前版本的 x.y.z 版本号', () => {
    for (const value of ['v1.2.4', '01.2.4', '1.2.4-beta.1', '1.2.4+build']) {
      expect(skillVersionError(value, '1.2.3')).not.toBe('')
    }
    expect(skillVersionError('1.2.3', '1.2.3')).not.toBe('')
    expect(skillVersionError('1.2.2', '1.2.3')).not.toBe('')
  })

  it('识别新增、修改、删除及二进制文件变更', () => {
    const changes = listSkillFileChanges(
      [
        { path: 'SKILL.md', content: '旧说明' },
        { path: 'old.md', content: '旧文件' },
        { path: 'assets/logo.png', content: 'AAAA', binary: true },
      ],
      [
        { path: 'SKILL.md', content: '新说明' },
        { path: 'new.md', content: '新文件' },
        { path: 'assets/logo.png', content: 'BBBB', binary: true },
      ]
    )

    expect(changes.map(({ path, kind }) => ({ path, kind }))).toEqual([
      { path: 'SKILL.md', kind: 'modified' },
      { path: 'assets/logo.png', kind: 'modified' },
      { path: 'new.md', kind: 'added' },
      { path: 'old.md', kind: 'deleted' },
    ])
    expect(changes[1].after?.binary).toBe(true)
  })
})
