import { createFileRoute } from '@tanstack/react-router'
import { TaskFlowPrototype } from '@/features/task-flow/task-flow-prototype'

export const Route = createFileRoute('/_app/task-flow-prototype')({
  component: TaskFlowPrototype,
})
