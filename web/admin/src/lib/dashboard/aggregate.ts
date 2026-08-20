export type AggregateInput = {
  instances: { id: string; enabled: boolean }[]
  cases: { id: number; enabled: boolean }[]
  tasks: { id: string; status: string }[]
}

export type DashboardStats = {
  instanceTotal: number
  instanceEnabled: number
  caseEnabled: number
  taskByStatus: Record<string, number>
}

export function aggregateDashboard(input: AggregateInput): DashboardStats {
  const taskByStatus: Record<string, number> = {}
  for (const t of input.tasks) {
    taskByStatus[t.status] = (taskByStatus[t.status] ?? 0) + 1
  }
  return {
    instanceTotal: input.instances.length,
    instanceEnabled: input.instances.filter((i) => i.enabled).length,
    caseEnabled: input.cases.filter((c) => c.enabled).length,
    taskByStatus,
  }
}
