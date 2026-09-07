export type OperationsDetailKind = 'task' | 'session' | 'user'

export type OperationsDetailTarget = {
  kind: OperationsDetailKind
  id: string
}
