import type { Next, Back, Jump } from '@formity/react'
import type { CaseRecord, RoutingConfig } from '@/lib/api/types'
import type { MenuPlacement } from '@/lib/api/channel-menu'

export type WizardMode = 'create' | 'existing'

export type PendingMenuEntry = {
  channelId: string
  label: string
  mode: 'direct' | 'list'
}

export type WizardShared = {
  mode: WizardMode
  /** 已有 Case 的 id（新建时为 null，完成页落库后回填）。 */
  caseId: number | null
  /** Case 草稿（Step 1 收集，未落库）。 */
  caseRecord: CaseRecord | null
  routing: RoutingConfig | undefined
  /** 已有投放（Step 3 读取展示）。 */
  placements: MenuPlacement[]
  /** 待提交的菜单入口（完成页统一写回渠道菜单）。 */
  pendingEntries: PendingMenuEntry[]
  onStepChange: (step: number) => void
  updateCase: (next: CaseRecord) => void
  updateRouting: (next: RoutingConfig) => void
  updatePlacements: (next: MenuPlacement[]) => void
  updatePendingEntries: (next: PendingMenuEntry[]) => void
  onExit: () => void
}

export type WizardSummary = {
  caseId: number
  ruleCount: number
  placementCount: number
}

export type StepActions = {
  next: Next<Record<string, never>>
  back: Back<Record<string, never>>
  jump: Jump<Record<string, never>>
}
