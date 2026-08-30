import { useMemo, useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
  clearQuickConfigSession,
  loadQuickConfigSession,
} from './lib/session'
import { QuickConfigFlow } from './quick-config-flow'
import type { CaseRecord } from '@/lib/api/types'

type WizardStart = {
  caseRecord?: CaseRecord
  topicKey?: string | null
  selectedEdgeId?: string | null
}

function startWithoutSession(): WizardStart | null {
  if (typeof window === 'undefined') return null
  return loadQuickConfigSession(window.localStorage) ? null : {}
}

export function QuickConfigPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [wizard, setWizard] = useState<WizardStart | null>(startWithoutSession)
  const [sessionEpoch, setSessionEpoch] = useState(0)

  const session = useMemo(
    () =>
      typeof window === 'undefined'
        ? null
        : loadQuickConfigSession(window.localStorage),
    [sessionEpoch]
  )

  if (wizard) {
    return (
      <QuickConfigFlow
        initialCase={wizard.caseRecord}
        initialTopicKey={wizard.topicKey}
        initialSelectedEdgeId={wizard.selectedEdgeId}
        onExit={() => void navigate({ to: '/' })}
      />
    )
  }

  return (
    <div className='mx-auto flex h-full min-h-0 w-full max-w-3xl flex-col gap-4 px-6 py-6'>
      <h1 className='text-lg font-semibold'>{t('quickConfig.pageTitle')}</h1>
      {session ? (
        <div className='flex shrink-0 items-center justify-between gap-3 rounded-lg border border-border bg-background px-3 py-2'>
          <span className='truncate text-sm'>
            {session.caseDraft &&
            typeof session.caseDraft === 'object' &&
            session.caseDraft !== null &&
            'name' in session.caseDraft
              ? String((session.caseDraft as { name?: unknown }).name ?? '')
              : t('quickConfig.unnamedWorkflow')}
          </span>
          <div className='flex gap-2'>
            <Button
              type='button'
              size='sm'
              variant='outline'
              onClick={() => {
                clearQuickConfigSession(window.localStorage)
                setSessionEpoch((e) => e + 1)
                setWizard({})
              }}
            >
              {t('quickConfig.discardDraft')}
            </Button>
            <Button
              type='button'
              size='sm'
              onClick={() =>
                setWizard({
                  caseRecord: session.caseDraft as CaseRecord,
                  topicKey: session.topicKey,
                  selectedEdgeId: session.selectedEdgeId,
                })
              }
            >
              {t('quickConfig.continueDraft')}
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  )
}
