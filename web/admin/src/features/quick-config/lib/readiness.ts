import { DEFAULT_TOPIC_KEY } from '../../task-flow/types'

export type ReadinessLevel = 'ready' | 'warn' | 'gap'

type WorkflowLike = {
  bindings?: { workflow?: unknown }
  inputs?: unknown[]
  input_schema?: unknown
}

/**
 * G1：工作流就绪 = bindings.workflow 非空、inputs 非空、input_schema 存在。
 */
export function workflowStatus(caseLike: WorkflowLike): ReadinessLevel {
  const workflow = caseLike.bindings?.workflow
  const hasWorkflow = Boolean(workflow && Object.keys(workflow).length > 0)
  const hasInputs = Boolean(caseLike.inputs && caseLike.inputs.length > 0)
  const hasSchema = caseLike.input_schema != null
  return hasWorkflow && hasInputs && hasSchema ? 'ready' : 'gap'
}

export type ProcessingInput = {
  workflow: ReadinessLevel
  rules: { topic?: string }[]
  enabledTopics: string[]
  boundTopics: string[]
  onlineTopics: string[]
  placements: unknown[]
  /** 是否已在向导选定运行节点。 */
  selectedNodeSelected: boolean
  /** 是否为默认路由分支（无规则，default 路由）。 */
  hasDefaultRoute: boolean
}

export type Readiness = {
  workflow: ReadinessLevel
  processing: ReadinessLevel
  placements: ReadinessLevel
  node: ReadinessLevel
}

/**
 * 就绪模型：
 * - G1 工作流：由 workflowStatus 决定。
 * - G2 处理流程：默认路由（default 分支）或至少一条规则；规则未连 Topic / 使用中 Topic 未绑定启用节点 → gap；
 *   全部绑定但无在线节点 → warn。
 * - G3 节点：运行节点已选即 ready（离线不阻塞发布，离线警告由处理流程在线态承载）。
 * - G4 投放：至少一个 menu placement。
 */
export function computeReadiness(input: ProcessingInput): Readiness {
  let processing: ReadinessLevel
  if (!input.hasDefaultRoute && input.rules.length === 0) {
    processing = 'gap'
  } else if (input.hasDefaultRoute && input.rules.length === 0) {
    const hasUnbound = !input.boundTopics.includes(DEFAULT_TOPIC_KEY)
    const hasDisabled = !input.enabledTopics.includes(DEFAULT_TOPIC_KEY)
    if (hasUnbound || hasDisabled) {
      processing = 'gap'
    } else if (input.onlineTopics.includes(DEFAULT_TOPIC_KEY)) {
      processing = 'ready'
    } else {
      processing = 'warn'
    }
  } else {
    const usedTopics = input.rules
      .map((rule) => rule.topic)
      .filter((topic): topic is string => Boolean(topic))
    const hasUnwiredRule = input.rules.some((rule) => !rule.topic)
    if (hasUnwiredRule) {
      processing = 'gap'
    } else {
      const hasUnbound = usedTopics.some(
        (topic) => !input.boundTopics.includes(topic)
      )
      const hasDisabled = usedTopics.some(
        (topic) => !input.enabledTopics.includes(topic)
      )
      if (hasUnbound || hasDisabled) {
        processing = 'gap'
      } else if (
        usedTopics.every((topic) => input.onlineTopics.includes(topic))
      ) {
        processing = 'ready'
      } else {
        processing = 'warn'
      }
    }
  }

  return {
    workflow: input.workflow === 'ready' ? 'ready' : 'gap',
    processing,
    placements: input.placements.length > 0 ? 'ready' : 'gap',
    node: input.selectedNodeSelected ? 'ready' : 'gap',
  }
}
