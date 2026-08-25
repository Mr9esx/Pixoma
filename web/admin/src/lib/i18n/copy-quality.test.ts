import { readFileSync } from 'node:fs'
import { describe, it, expect } from 'vitest'
import zh from './locales/zh.json'

// Voice profile: docs/voice-profile.md
// 这是 contract test,锁住"不要回退到旧 register"。
// 改 doc 的规则时可以同步改这里;改了 zh.json 让 test fail 就是在提醒违反规则。

type Nested = string | number | boolean | Nested[] | { [k: string]: Nested }

function flatten(obj: Nested, prefix = ''): Array<[string, string]> {
  const out: Array<[string, string]> = []
  if (obj && typeof obj === 'object' && !Array.isArray(obj)) {
    for (const [k, v] of Object.entries(obj as Record<string, Nested>)) {
      const key = prefix ? `${prefix}.${k}` : k
      if (typeof v === 'string') {
        out.push([key, v])
      } else if (v && typeof v === 'object') {
        out.push(...flatten(v, key))
      }
    }
  }
  return out
}

const strings = flatten(zh as Nested).filter(([, v]) => v && /[一-鿿]/.test(v))

function stripPlaceholders(s: string) {
  return s.replace(/\{\{[^}]+\}\}/g, '')
}

describe('zh.json copy quality (pixoma-voice)', () => {
  it('does not use "请" as a softener at sentence start', () => {
    const offenders: Array<[string, string]> = []
    for (const [k, v] of strings) {
      const s = stripPlaceholders(v)
      // "请" at start of string, or after 。！？， — but not inside 请求/请柬/etc
      if (/^请(?=[一-鿿])/.test(s) || /[。！？，]请(?=[一-鿿])/.test(s)) {
        if (!s.includes('请求')) {
          offenders.push([k, v])
        }
      }
    }
    expect(offenders).toEqual([])
  })

  it('does not use "您" (远距代词)', () => {
    const offenders = strings.filter(([, v]) => v.includes('您'))
    expect(offenders).toEqual([])
  })

  it('does not use exclamation marks', () => {
    const offenders = strings.filter(([, v]) => /[！!]/.test(v))
    expect(offenders).toEqual([])
  })

  it('does not use 卖萌 particles (温馨提示/哎呀/亲)', () => {
    const offenders = strings.filter(([, v]) =>
      /(温馨提示|哎呀|^亲[爱们])/.test(v)
    )
    expect(offenders).toEqual([])
  })

  it('does not use 首先/其次/最后 (应直接列)', () => {
    const offenders = strings.filter(([, v]) => /(首先|其次|最后)/.test(v))
    expect(offenders).toEqual([])
  })

  it('does not use 远距动词 "前往"', () => {
    const offenders = strings.filter(([, v]) => v.includes('前往'))
    expect(offenders).toEqual([])
  })

  it('does not use 名词化包装 "查找关键字"', () => {
    const offenders = strings.filter(([, v]) => v.includes('查找关键字'))
    expect(offenders).toEqual([])
  })
})
