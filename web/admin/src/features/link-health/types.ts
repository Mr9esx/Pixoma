export type HealthState = 'ok' | 'warn' | 'pending'

export interface ReferenceItem {
  id: string
  name: string
  state: HealthState
  to: string
}

export interface HealthBreakpoint {
  stage: 'entry' | 'workflow' | 'topic' | 'node'
  fix: 'config' | 'runtime'
  key: string
  params?: Record<string, string>
  action: { to: string; key: string }
  guide: string
}

export interface EntityHealth {
  state: HealthState
  breakpoints: HealthBreakpoint[]
}
