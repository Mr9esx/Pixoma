import { describe, expect, it } from 'vitest'
import type { AttributeDescriptor, RoutingConfig, TopicRecord } from '../types'
import { validateRouting, validateRule } from './validate'

const topics: TopicRecord[] = [
  { key: 'default', name: '默认 Topic', enabled: true, created_at: '', updated_at: '' },
  { key: 'fast-gpu', name: '高规格 GPU', enabled: true, created_at: '', updated_at: '' },
  { key: 'batch-night', name: '夜间批量', enabled: true, created_at: '', updated_at: '' },
  { key: 'cpu-low', name: '低成本 CPU', enabled: false, created_at: '', updated_at: '' },
]

const attributes: AttributeDescriptor[] = [
  { key: 'user.is_premium', context: 'user', label: '用户是否付费', schema: { type: 'boolean' } },
  { key: 'case.category', context: 'case', label: 'Case 分类', schema: { type: 'string', enum: ['image', 'video', 'audio'] } },
  { key: 'case.tags', context: 'case', label: 'Case 标签', schema: { type: 'array', items: { type: 'string' } } },
  { key: 'user.age', context: 'user', label: '用户年龄', schema: { type: 'number' } },
]

const okRule = (when: unknown = { field: 'user.is_premium', op: 'eq', value: true }, topic = 'fast-gpu') => ({
  when,
  topic,
})

describe('validateRule', () => {
  it('合法规则返回 null', () => {
    expect(validateRule(okRule() as never, 0, topics, attributes)).toBeNull()
  })

  it('未连线（topic 缺失）报 topic-missing', () => {
    const issue = validateRule({ when: { field: 'user.is_premium', op: 'eq', value: true } }, 2, topics, attributes)
    expect(issue?.kind).toBe('topic-missing')
    expect(issue?.index).toBe(2)
  })

  it('引用不存在的 Topic 报 topic-unknown', () => {
    const issue = validateRule(okRule(undefined, 'ghost-gpu') as never, 0, topics, attributes)
    expect(issue?.kind).toBe('topic-unknown')
  })

  it('引用已禁用 Topic 报 topic-disabled', () => {
    const issue = validateRule(okRule(undefined, 'cpu-low') as never, 0, topics, attributes)
    expect(issue?.kind).toBe('topic-disabled')
  })

  it('条件字段未注册报 condition-unknown-field', () => {
    const issue = validateRule(
      okRule({ field: 'user.not_registered', op: 'eq', value: true }) as never,
      0,
      topics,
      attributes,
    )
    expect(issue?.kind).toBe('condition-unknown-field')
  })

  it('操作符与字段类型不匹配报 condition-bad-op', () => {
    // boolean 字段不支持 in
    const issue = validateRule(
      okRule({ field: 'user.is_premium', op: 'in', value: [true] }) as never,
      0,
      topics,
      attributes,
    )
    expect(issue?.kind).toBe('condition-bad-op')
  })

  it('空 AND/OR 组合报 condition-empty-group', () => {
    const and = validateRule(okRule({ and: [] }) as never, 0, topics, attributes)
    expect(and?.kind).toBe('condition-empty-group')
    const or = validateRule(okRule({ or: [] }) as never, 0, topics, attributes)
    expect(or?.kind).toBe('condition-empty-group')
  })

  it('嵌套组合内的非法子条件被递归查出', () => {
    const issue = validateRule(
      okRule({ or: [{ field: 'case.category', op: 'eq', value: 'image' }, { field: 'ghost', op: 'eq', value: 1 }] }) as never,
      1,
      topics,
      attributes,
    )
    expect(issue?.kind).toBe('condition-unknown-field')
    expect(issue?.index).toBe(1)
  })

  it('值类型不正确报 condition-bad-value（boolean 字段给字符串）', () => {
    const issue = validateRule(
      okRule({ field: 'user.is_premium', op: 'eq', value: 'yes' }) as never,
      0,
      topics,
      attributes,
    )
    expect(issue?.kind).toBe('condition-bad-value')
  })

  it('enum 字段值不在枚举内报 condition-bad-value', () => {
    const issue = validateRule(
      okRule({ field: 'case.category', op: 'eq', value: 'pdf' }) as never,
      0,
      topics,
      attributes,
    )
    expect(issue?.kind).toBe('condition-bad-value')
  })

  it('exists 操作符不需要值', () => {
    expect(validateRule(okRule({ field: 'user.is_premium', op: 'exists' }) as never, 0, topics, attributes)).toBeNull()
  })

  it('传入绑定集合且 Topic 未绑定节点时报 topic-unbound', () => {
    const bound = new Set(['batch-night'])
    const issue = validateRule(okRule() as never, 0, topics, attributes, bound)
    expect(issue?.kind).toBe('topic-unbound')
    expect(issue?.message).toContain('未绑定计算节点')
  })

  it('传入绑定集合且 Topic 已绑定节点时通过', () => {
    const bound = new Set(['fast-gpu'])
    expect(validateRule(okRule() as never, 0, topics, attributes, bound)).toBeNull()
  })
})

describe('validateRouting', () => {
  it('空配置（无 rules）合法（全部回退默认 Topic）', () => {
    const result = validateRouting(undefined, topics, attributes)
    expect(result.valid).toBe(true)
    expect(result.issues).toHaveLength(0)
  })

  it('空规则且默认 Topic 已绑定时合法', () => {
    const result = validateRouting(undefined, topics, attributes, new Set(['default']))
    expect(result.valid).toBe(true)
  })

  it('空规则且默认 Topic 未绑定时报 topic-unbound', () => {
    const result = validateRouting(undefined, topics, attributes, new Set(['fast-gpu']))
    expect(result.valid).toBe(false)
    expect(result.issues).toHaveLength(1)
    expect(result.issues[0].kind).toBe('topic-unbound')
    expect(result.issues[0].message).toContain('默认 Topic')
  })

  it('混合规则：只标记出问题的下标', () => {
    const routing: RoutingConfig = {
      rules: [
        { when: { field: 'user.is_premium', op: 'eq', value: true }, topic: 'fast-gpu' },
        { when: { field: 'case.category', op: 'eq', value: 'image' } }, // 未连线
        { when: { field: 'case.tags', op: 'in', value: ['night'] }, topic: 'cpu-low' }, // 已禁用
      ],
    }
    const result = validateRouting(routing, topics, attributes)
    expect(result.valid).toBe(false)
    expect(result.issues.map((i) => i.index)).toEqual([1, 2])
    expect(result.invalidIndexes.has(1)).toBe(true)
    expect(result.invalidIndexes.has(2)).toBe(true)
    expect(result.invalidIndexes.has(0)).toBe(false)
  })

  it('每规则只报一条（topic 问题优先于条件问题）', () => {
    const routing: RoutingConfig = {
      rules: [{ when: { field: 'ghost', op: 'eq', value: 1 } }], // 缺 topic 且字段非法
    }
    const result = validateRouting(routing, topics, attributes)
    expect(result.issues).toHaveLength(1)
    expect(result.issues[0].kind).toBe('topic-missing')
  })

  it('未传入绑定集合时不检查绑定（保持兼容）', () => {
    const result = validateRouting(
      { rules: [{ when: { field: 'user.is_premium', op: 'eq', value: true }, topic: 'fast-gpu' }] },
      topics,
      attributes,
    )
    expect(result.valid).toBe(true)
  })

  it('传入绑定集合且存在未绑定 Topic 时报 topic-unbound', () => {
    const routing: RoutingConfig = {
      rules: [
        { when: { field: 'user.is_premium', op: 'eq', value: true }, topic: 'fast-gpu' },
        { when: { field: 'case.category', op: 'eq', value: 'image' }, topic: 'batch-night' },
      ],
    }
    const result = validateRouting(routing, topics, attributes, new Set(['fast-gpu']))
    expect(result.valid).toBe(false)
    expect(result.issues.map((i) => i.index)).toEqual([1])
    expect(result.issues[0].kind).toBe('topic-unbound')
  })
})
