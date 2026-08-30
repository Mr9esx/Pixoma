import { useCallback, useEffect, useState, type ReactNode } from 'react'
import { useFormity, type Flow, type s } from '@formity/react'
import type { CaseRecord } from '@/lib/api/types'
import { saveQuickConfigSession } from './lib/session'
import type { TopicDraft } from './lib/session'
import { Step1Workflow } from './step1-workflow'
import { Step2Queue } from './step2-queue'
import { Step2Node } from './step2-node'
import { Step4Next } from './step4-next'
import type { WizardShared, WizardSummary } from './types'

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
          <Step2Queue shared={params} next={next} back={back} jump={jump} />
        </StepBridge>
      ),
    },
  },
  {
    form: {
      fields: () => ({}),
      render: ({ params, next, back, jump }) => (
        <StepBridge n={3} onStepChange={params.onStepChange}>
          <Step2Node shared={params} next={next} back={back} jump={jump} />
        </StepBridge>
      ),
    },
  },
  {
    form: {
      fields: () => ({}),
      render: ({ params, next, back, jump }) => (
        <StepBridge n={4} onStepChange={params.onStepChange}>
          <Step4Next shared={params} next={next} back={back} jump={jump} />
        </StepBridge>
      ),
    },
  },
  {
    return: () => ({ caseId: 0 }),
  },
]

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
  initialCase,
  initialTopicKey,
  initialSelectedEdgeId,
  onExit,
}: {
  initialCase?: CaseRecord
  initialTopicKey?: string | null
  initialSelectedEdgeId?: string | null
  onExit: () => void
}) {
  const [caseId, setCaseId] = useState<number | null>(
    initialCase && initialCase.id > 0 ? initialCase.id : null
  )
  const [caseRecord, setCaseRecord] = useState<CaseRecord | null>(
    initialCase ?? null
  )
  const [topicKey, setTopicKey] = useState<string | null>(
    initialTopicKey ?? null
  )
  const [topicDraft, setTopicDraft] = useState<TopicDraft | null>(null)
  const [selectedEdgeId, setSelectedEdgeId] = useState<string | null>(
    initialSelectedEdgeId ?? null
  )
  const [committed, setCommitted] = useState(false)
  const [step, setStep] = useState(1)

  const updateCase = useCallback((next: CaseRecord) => {
    setCaseId(next.id > 0 ? next.id : null)
    setCaseRecord(next)
  }, [])
  const updateTopic = useCallback(
    (key: string | null, draft: TopicDraft | null = null) => {
      setTopicKey(key)
      setTopicDraft(draft)
    },
    []
  )
  const updateSelectedEdge = useCallback(
    (id: string | null) => setSelectedEdgeId(id),
    []
  )
  const markCommitted = useCallback(() => setCommitted(true), [])
  const onStepChange = useCallback((n: number) => setStep(n), [])

  const shared: WizardShared = {
    caseId,
    caseRecord,
    topicKey,
    topicDraft,
    selectedEdgeId,
    committed,
    onStepChange,
    updateCase,
    updateTopic,
    updateSelectedEdge,
    markCommitted,
    onExit,
  }

  useEffect(() => {
    if (caseRecord == null && caseId == null) return
    if (committed) return
    saveQuickConfigSession(window.localStorage, {
      caseId,
      step,
      caseDraft: caseRecord,
      topicKey,
      topicDraft,
      selectedEdgeId,
      updatedAt: new Date().toISOString(),
    })
  }, [caseId, caseRecord, topicKey, topicDraft, selectedEdgeId, step, committed])

  const rendered = useFormity<WizardSchema>({ flow, params: shared })
  return <>{rendered}</>
}
