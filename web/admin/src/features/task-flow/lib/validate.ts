import {
  type AttributeDescriptor,
  type Condition,
  type RoutingConfig,
  type RoutingRule,
  type TopicRecord,
} from '../types'

/**
 * 保存前校验：与后端 internal/catalog/infrastructure/validation/ValidateRouting
 * 语义对齐（topic 必填 / topic 存在且启用 / 条件按属性目录与 op/值类型校验）。
 * 每条规则最多返回一条错误：先查 topic，再查条件——修复顺序即错误顺序。
 */

export type RuleIssueKind =
  | 'routing-empty'
  | 'topic-missing'
  | 'topic-unknown'
  | 'topic-disabled'
  | 'topic-unbound'
  | 'condition-unknown-field'
  | 'condition-bad-op'
  | 'condition-empty-group'
  | 'condition-bad-value'

export interface RuleIssue {
  /** 对应 rules 数组下标（错误定位到分支卡片）。 */
  index: number
  kind: RuleIssueKind
  message: string
}

interface ValidationResult {
  issues: RuleIssue[]
  /** 是否允许保存（无任何问题）。 */
  valid: boolean
  /** 有问题的规则下标集合（画布高亮用）。 */
  invalidIndexes: Set<number>
}

/** 与 condition-form 的 OPS_BY_TYPE 一致：类型允许的操作符集合。 */
const OPS_BY_TYPE: Record<string, string[]> = {
  boolean: ['eq', 'ne', 'exists'],
  string: ['eq', 'ne', 'in', 'exists'],
  number: ['eq', 'ne', 'gt', 'gte', 'lt', 'lte', 'exists'],
  array: ['in', 'exists'],
}

function topicName(topics: TopicRecord[], key: string): string {
  return topics.find((t) => t.key === key)?.name ?? key
}

/** 校验单条规则；返回 null 表示该规则合法。 */
export function validateRule(
  rule: RoutingRule,
  index: number,
  topics: TopicRecord[],
  attributes: AttributeDescriptor[],
  boundTopicKeys?: ReadonlySet<string>
): RuleIssue | null {
  const topic = rule.topic?.trim()
  if (!topic) {
    return {
      index,
      kind: 'topic-missing',
      message: `规则 #${index + 1}：未连接目标 Topic（从右侧圆点拖线到 Topic）`,
    }
  }
  const record = topics.find((t) => t.key === topic)
  if (!record) {
    return {
      index,
      kind: 'topic-unknown',
      message: `规则 #${index + 1}：目标 Topic「${topic}」不存在（引用已删除的 Topic）`,
    }
  }
  if (!record.enabled) {
    return {
      index,
      kind: 'topic-disabled',
      message: `规则 #${index + 1}：目标 Topic「${topicName(topics, topic)}」已禁用，请先启用或改连其他 Topic`,
    }
  }
  if (boundTopicKeys && !boundTopicKeys.has(topic)) {
    return {
      index,
      kind: 'topic-unbound',
      message: `规则 #${index + 1}：目标 Topic「${topicName(topics, topic)}」未绑定计算节点，请先为该 Topic 绑定启用节点`,
    }
  }
  if ('always' in rule.when) return null
  const condIssue = validateCondition(rule.when, attributes)
  if (condIssue) {
    return {
      index,
      kind: condIssue.kind,
      message: `规则 #${index + 1}：${condIssue.message}`,
    }
  }
  return null
}

function validateCondition(
  c: Condition,
  attributes: AttributeDescriptor[]
): Omit<RuleIssue, 'index'> | null {
  if ('always' in c) return null
  if ('and' in c) {
    if (c.and.length === 0) {
      return {
        kind: 'condition-empty-group',
        message: 'AND 组合为空，请添加至少一个子条件',
      }
    }
    for (const sub of c.and) {
      const issue = validateCondition(sub, attributes)
      if (issue) return issue
    }
    return null
  }
  if ('or' in c) {
    if (c.or.length === 0) {
      return {
        kind: 'condition-empty-group',
        message: 'OR 组合为空，请添加至少一个子条件',
      }
    }
    for (const sub of c.or) {
      const issue = validateCondition(sub, attributes)
      if (issue) return issue
    }
    return null
  }

  const leaf = c as Extract<Condition, { field: string }>
  const attr = attributes.find((a) => a.key === leaf.field)
  if (!attr) {
    return {
      kind: 'condition-unknown-field',
      message: `条件字段「${leaf.field}」未在属性目录注册，请重新选择字段`,
    }
  }
  const type = attr.schema.type ?? 'string'
  const allowed = OPS_BY_TYPE[type] ?? OPS_BY_TYPE.string
  if (!allowed.includes(leaf.op)) {
    return {
      kind: 'condition-bad-op',
      message: `字段「${attr.label}」不支持操作符「${leaf.op}」（可用：${allowed.join('、')}）`,
    }
  }
  if (leaf.op === 'exists') return null
  if (!validateValue(leaf.value, attr, leaf.op)) {
    return {
      kind: 'condition-bad-value',
      message: `字段「${attr.label}」的值类型不正确`,
    }
  }
  return null
}

function validateValue(
  value: unknown,
  attr: AttributeDescriptor,
  op: string
): boolean {
  const type = attr.schema.type ?? 'string'
  const schema = attr.schema
  // in 操作符：value 必须是数组，元素按 itemSchema 校验（对齐后端 validateValueType）。
  if (op === 'in') {
    if (!Array.isArray(value)) return false
    const itemType = schema.items?.type ?? type
    return value.every((v) => validateValueByType(v, itemType))
  }
  if (type === 'boolean') {
    return typeof value === 'boolean'
  }
  if (type === 'number') {
    return typeof value === 'number'
  }
  if (type === 'array') {
    if (!Array.isArray(value)) return false
    const itemType = schema.items?.type
    return value.every((v) =>
      itemType ? validateValueByType(v, itemType) : true
    )
  }
  // string
  if (typeof value !== 'string') return false
  if (schema.enum && schema.enum.length > 0) {
    return schema.enum.includes(value)
  }
  return true
}

function validateValueByType(value: unknown, type: string): boolean {
  if (type === 'boolean') return typeof value === 'boolean'
  if (type === 'number') return typeof value === 'number'
  return typeof value === 'string'
}

export function validateRouting(
  routing: RoutingConfig | undefined,
  topics: TopicRecord[],
  attributes: AttributeDescriptor[],
  boundTopicKeys?: ReadonlySet<string>
): ValidationResult {
  const rules = routing?.rules ?? []
  const issues: RuleIssue[] = []
  const invalidIndexes = new Set<number>()
  if (rules.length === 0) {
    return {
      issues: [
        { index: -1, kind: 'routing-empty', message: '至少需要一条路由规则' },
      ],
      valid: false,
      invalidIndexes,
    }
  }
  for (let i = 0; i < rules.length; i++) {
    const issue = validateRule(rules[i], i, topics, attributes, boundTopicKeys)
    if (issue) {
      issues.push(issue)
      invalidIndexes.add(i)
    }
  }
  return { issues, valid: issues.length === 0, invalidIndexes }
}
