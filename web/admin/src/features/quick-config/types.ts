import type { Next, Back, Jump } from '@formity/react'
import type { CaseRecord } from '@/lib/api/types'
import type { TopicDraft } from './lib/session'

export type WizardShared = {
  caseId: number | null
  caseRecord: CaseRecord | null
  topicKey: string | null
  topicDraft: TopicDraft | null
  selectedEdgeId: string | null
  committed: boolean
  onStepChange: (step: number) => void
  updateCase: (next: CaseRecord) => void
  updateTopic: (key: string | null, draft?: TopicDraft | null) => void
  updateSelectedEdge: (id: string | null) => void
  markCommitted: () => void
  onExit: () => void
}

export type WizardSummary = {
  caseId: number
}

export type StepActions = {
  next: Next<Record<string, never>>
  back: Back<Record<string, never>>
  jump: Jump<Record<string, never>>
}
