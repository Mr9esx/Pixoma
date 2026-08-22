import { useCallback, useEffect, useState, type ReactNode } from 'react'
import { useFormity, type Flow, type s } from '@formity/react'
import type { CaseRecord, RoutingConfig } from '@/lib/api/types'
import type { MenuPlacement } from '@/lib/api/channel-menu'
import { saveQuickConfigSession } from './lib/session'
import { Step1Workflow } from './step1-workflow'
import { Step2Processing } from './step2-processing'
import { Step3Channels } from './step3-channels'
import { DoneScreen } from './done-screen'
import type {
  PendingMenuEntry,
  WizardMode,
  WizardShared,
  WizardSummary,
} from './types'

type WizardSchema = {
  render: React.ReactNode
  struct: [
    s.Form<Record<string, never>>,
    s.Form<Record<string, never>>,
    s.Form<Record<string, never>>,
    s.Form<Record<string, never>>,
    s.Return<WizardSummary>,
  ]
  inputs: Record<string, never>
  params: WizardShared
}

const flow: Flow<WizardSchema> = [
  {
    form: {
      fields: () => ({}),
      render: ({ params, next, back, jump }) => (
        <StepBridge n={1} onStepChange={params.onStepChange}>
          <Step1Workflow shared={params} next={next} back={back} jump={jump} />
        </StepBridge>
      ),
    },
  },
  {
    form: {
      fields: () => ({}),
      render: ({ params, next, back, jump }) => (
        <StepBridge n={2} onStepChange={params.onStepChange}>
          <Step2Processing shared={params} next={next} back={back} jump={jump} />
        </StepBridge>
      ),
    },
  },
  {
    form: {
      fields: () => ({}),
      render: ({ params, next, back, jump }) => (
        <StepBridge n={3} onStepChange={params.onStepChange}>
          <Step3Channels shared={params} next={next} back={back} jump={jump} />
        </StepBridge>
      ),
    },
  },
  {
    form: {
      fields: () => ({}),
      render: ({ params, next, back, jump }) => (
        <StepBridge n={4} onStepChange={params.onStepChange}>
          <DoneScreen shared={params} next={next} back={back} jump={jump} />
        </StepBridge>
      ),
    },
  },
  {
    return: () => ({ caseId: 0, ruleCount: 0, placementCount: 0 }),
  },
]

/** 通知外层当前步骤，供会话持久化与恢复使用。 */
function StepBridge({
  n,
  onStepChange,
  children,
}: {
  n: number
  onStepChange: (step: number) => void
  children: ReactNode
}) {
  useEffect(() => {
    onStepChange(n)
  }, [n, onStepChange])
  return <>{children}</>
}

export function QuickConfigFlow({
  mode,
  initialCase,
  initialRouting,
  initialPendingEntries,
  onExit,
}: {
  mode: WizardMode
  initialCase?: CaseRecord
  initialRouting?: RoutingConfig
  initialPendingEntries?: PendingMenuEntry[]
  onExit: () => void
}) {
  const [caseId, setCaseId] = useState<number | null>(
    initialCase?.id ?? null
  )
  const [caseRecord, setCaseRecord] = useState<CaseRecord | null>(
    initialCase ?? null
  )
  const [routing, setRouting] = useState<RoutingConfig | undefined>(
    initialRouting ?? initialCase?.routing
  )
  const [placements, setPlacements] = useState<MenuPlacement[]>([])
  const [pendingEntries, setPendingEntries] = useState<PendingMenuEntry[]>(
    initialPendingEntries ?? []
  )
  const [step, setStep] = useState(1)

  const updateCase = useCallback((next: CaseRecord) => {
    setCaseId(next.id > 0 ? next.id : null)
    setCaseRecord(next)
    setRouting((prev) => prev ?? next.routing)
  }, [])
  const updateRouting = useCallback(
    (next: RoutingConfig) => setRouting(next),
    []
  )
  const updatePlacements = useCallback(
    (next: MenuPlacement[]) => setPlacements(next),
    []
  )
  const updatePendingEntries = useCallback(
    (next: PendingMenuEntry[]) => setPendingEntries(next),
    []
  )
  const onStepChange = useCallback((n: number) => setStep(n), [])

  const shared: WizardShared = {
    mode,
    caseId,
    caseRecord,
    routing,
    placements,
    pendingEntries,
    onStepChange,
    updateCase,
    updateRouting,
    updatePlacements,
    updatePendingEntries,
    onExit,
  }

  useEffect(() => {
    // 只有收集到草稿或已有 Case 时才保存会话，避免空会话导致「Case #null」。
    if (caseRecord == null && caseId == null) return
    saveQuickConfigSession(window.localStorage, {
      caseId,
      mode,
      step,
      caseDraft: caseRecord,
      routing,
      pendingEntries,
      updatedAt: new Date().toISOString(),
    })
  }, [caseId, mode, caseRecord, routing, pendingEntries])

  const rendered = useFormity<WizardSchema>({ flow, params: shared })
  return <>{rendered}</>
}
