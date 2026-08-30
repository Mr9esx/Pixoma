import { useTranslation } from 'react-i18next'
import type { CaseRecord } from '@/lib/api/types'
import { WorkflowEditor } from '@/features/cases/workflow-editor'
import { WorkflowImportRequirement } from '@/features/cases/workflow-import-requirement'
import { Button } from '@/components/ui/button'
import { ArrowRight } from 'lucide-react'
import type { StepActions, WizardShared } from './types'
import { WizardChrome } from './wizard-chrome'

type Props = StepActions & { shared: WizardShared }

export function Step1Workflow({ shared, next }: Props) {
  const { t } = useTranslation()

  function handleCollect(payload: CaseRecord) {
    shared.updateCase(payload)
    next({})
  }

  return (
    <WizardChrome step={1} hideFooter>
      <WorkflowEditor
        mode='create'
        initialName={t('quickConfig.unnamedWorkflow')}
        collectOnly
        splitPane
        stepRail
        hideProcessing
        leftIntro={<WorkflowImportRequirement />}
        redirectAfterSave={false}
        hideActions
        onCollect={handleCollect}
        formId='quick-config-step1-form'
        footer={
          <div className='flex justify-end'>
            <Button type='submit'>
              {t('quickConfig.next')}
              <ArrowRight className='size-4' />
            </Button>
          </div>
        }
      />
    </WizardChrome>
  )
}
