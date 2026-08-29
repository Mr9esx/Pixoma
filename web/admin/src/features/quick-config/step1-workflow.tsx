import { useTranslation } from 'react-i18next'
import type { CaseRecord } from '@/lib/api/types'
import { WorkflowEditor } from '@/features/cases/workflow-editor'
import type { StepActions, WizardShared } from './types'
import { WizardChrome } from './wizard-chrome'

type Props = StepActions & { shared: WizardShared }

/** Step 1 工作流配置：与独立编辑页共用统一 WorkflowEditor。 */
export function Step1Workflow({ shared, next, back }: Props) {
  const { mode, caseRecord } = shared
  const { t } = useTranslation()

  function handleCollect(payload: CaseRecord) {
    shared.updateCase(payload)
    next({})
  }

  return (
    <WizardChrome
      step={1}
      onBack={mode === 'create' ? shared.onExit : () => back({})}
      nextLabel={t('quickConfig.next')}
      onNext={() =>
        (
          document.getElementById(
            'quick-config-step1-form'
          ) as HTMLFormElement | null
        )?.requestSubmit()
      }
    >
      {mode === 'create' ? (
        <WorkflowEditor
          mode='create'
          initialName={t('quickConfig.unnamedWorkflow')}
          collectOnly
          splitPane
          stepRail
          leftIntro={
            <p className='text-sm text-destructive' role='note'>
              {t('quickConfig.requireValidJson')}
            </p>
          }
          redirectAfterSave={false}
          hideActions
          onCollect={handleCollect}
          formId='quick-config-step1-form'
        />
      ) : caseRecord ? (
        <WorkflowEditor
          mode='edit'
          initial={caseRecord}
          collectOnly
          splitPane
          stepRail
          onCollect={handleCollect}
          formId='quick-config-step1-form'
        />
      ) : null}
    </WizardChrome>
  )
}
