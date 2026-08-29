import type { Next, Back, Jump } from '@formity/react'
import type { CaseRecord, RoutingConfig } from '@/lib/api/types'
import type { MenuPlacement } from '@/lib/api/channel-menu'

export type WizardMode = 'create' | 'existing'
export type RulesMode = 'default' | 'editor'

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
  /** 待提交的菜单入口（完成页统一写回消息平台菜单）。 */
  pendingEntries: PendingMenuEntry[]
  /** 运行节点草稿（Step2，完成页统一下发 default 订阅）。 */
  selectedEdgeId: string | null
  /** 特殊规则分支模式：default 路由 / 跳转独立编辑页。 */
  rulesMode: RulesMode
  /** 规则分支是否已跳转独立编辑页（清空会话避免干扰）。 */
  ruleHandover: boolean
  onStepChange: (step: number) => void
  updateCase: (next: CaseRecord) => void
  updateRouting: (next: RoutingConfig) => void
  updatePlacements: (next: MenuPlacement[]) => void
  updatePendingEntries: (next: PendingMenuEntry[]) => void
  updateSelectedEdge: (id: string | null) => void
  updateRulesMode: (mode: RulesMode) => void
  updateRuleHandover: (value: boolean) => void
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
