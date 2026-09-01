import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { CaseRecord } from '@/lib/api/types'
import { WorkflowEditor } from '@/features/cases/workflow-editor'
import type { StepActions, WizardShared } from './types'
import { WizardChrome } from './wizard-chrome'

type Props = StepActions & { shared: WizardShared }

export function Step1Workflow({ shared, next }: Props) {
  const { t } = useTranslation()
  const [workflowImported, setWorkflowImported] = useState(false)

  function handleCollect(payload: CaseRecord) {
    shared.updateCase(payload)
    next({})
  }

  return (
    <WizardChrome
      step={1}
      nextForm='quick-config-step1-form'
      nextLabel={t('quickConfig.next')}
      nextDisabled={!workflowImported}
    >
      <WorkflowEditor
        mode='create'
        initialName={t('quickConfig.unnamedWorkflow')}
        collectOnly
        splitPane
        stepRail
        hideProcessing
        redirectAfterSave={false}
        hideActions
        onWorkflowImportedChange={setWorkflowImported}
        onCollect={handleCollect}
        formId='quick-config-step1-form'
      />
    </WizardChrome>
  )
}
