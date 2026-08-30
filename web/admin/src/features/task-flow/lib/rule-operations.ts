import type { Condition, RoutingConfig } from '../types'

export function describeCondition(condition: Condition): string {
  if ('always' in condition) return '无条件'
  if ('and' in condition) {
    return `同时满足 ${condition.and.length} 个条件`
  }
  if ('or' in condition) {
    return `满足任一 ${condition.or.length} 个条件`
  }
  const opText: Record<string, string> = {
    eq: '=',
    ne: '≠',
    in: '∈',
    gt: '>',
    gte: '≥',
    lt: '<',
    lte: '≤',
    exists: '存在',
  }
  const value =
    condition.op === 'exists'
      ? ''
      : Array.isArray(condition.value)
        ? `[${condition.value.join(', ')}]`
        : String(condition.value ?? '')
  return `${condition.field} ${opText[condition.op] ?? condition.op}${value ? ` ${value}` : ''}`
}

export function moveRule(
  routing: RoutingConfig,
  index: number,
  direction: -1 | 1
): RoutingConfig {
  const target = index + direction
  if (index < 0 || target < 0 || target >= routing.rules.length) return routing
  const rules = [...routing.rules]
  const [item] = rules.splice(index, 1)
  rules.splice(target, 0, item)
  return { rules }
}

export function removeRule(
  routing: RoutingConfig,
  index: number
): RoutingConfig {
  return { rules: routing.rules.filter((_, i) => i !== index) }
}
