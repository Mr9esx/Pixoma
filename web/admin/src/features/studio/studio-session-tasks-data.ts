import type { StudioFlowNode, StudioWorkflowExecution } from '@/lib/api/studio'

export function isActiveWorkflowExecution(execution: StudioWorkflowExecution) {
  return execution.status === 'submitted' && !['succeeded', 'failed', 'cancelled'].includes(execution.task_status ?? '')
}

export function workflowExecutionTitle(execution: StudioWorkflowExecution, nodes: StudioFlowNode[]) {
  return nodes.find((node) => node.id === execution.operation_node_id)?.title || `工作流 ${execution.workflow_id}`
}
