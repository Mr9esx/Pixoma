export type ReadinessLevel = 'ready' | 'warn' | 'gap'

type WorkflowLike = {
  bindings?: { workflow?: unknown }
  inputs?: unknown[]
  input_schema?: unknown
}

/**
 * G1：工作流就绪 = bindings.workflow 非空、inputs 非空、input_schema 存在。
 * 与后端 Case 创建校验等价，避免向导产出空壳 Case。
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
}

export type Readiness = {
  workflow: ReadinessLevel
  processing: ReadinessLevel
  placements: ReadinessLevel
}

/**
 * 就绪模型：
 * - G1 工作流：由 workflowStatus 决定。
 * - G2 处理流程：至少一条规则、每条规则已连 Topic、每个使用中的 Topic 已绑定启用节点；
 *   全部绑定但无在线节点 → warn（不阻塞发布）。
 * - G3 投放：至少一个 menu placement。
 */
export function computeReadiness(input: ProcessingInput): Readiness {
  const usedTopics = input.rules
    .map((rule) => rule.topic)
    .filter((topic): topic is string => Boolean(topic))

  let processing: ReadinessLevel
  const hasUnwiredRule = input.rules.some((rule) => !rule.topic)
  if (input.rules.length === 0 || hasUnwiredRule) {
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
    } else if (usedTopics.every((topic) => input.onlineTopics.includes(topic))) {
      processing = 'ready'
    } else {
      processing = 'warn'
    }
  }

  return {
    workflow: input.workflow === 'ready' ? 'ready' : 'gap',
    processing,
    placements: input.placements.length > 0 ? 'ready' : 'gap',
  }
}
