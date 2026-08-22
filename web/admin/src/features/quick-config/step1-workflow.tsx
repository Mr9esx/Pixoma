import { CaseForm } from '@/features/cases/case-form'
import { useTranslation } from 'react-i18next'
import type { CaseRecord } from '@/lib/api/types'
import { WizardChrome } from './wizard-chrome'
import type { StepActions, WizardShared } from './types'

type Props = StepActions & { shared: WizardShared }

/** Step 1 工作流配置：新建（强制导入，复用 CaseForm）或调整已有（已加载可编辑）。 */
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
        <CaseForm
          mode='create'
          initialName={t('quickConfig.unnamedWorkflow')}
          collectOnly
          splitPane
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
        <CaseForm
          mode='edit'
          initial={caseRecord}
          collectOnly
          splitPane
          onCollect={handleCollect}
          formId='quick-config-step1-form'
        />
      ) : null}
    </WizardChrome>
  )
}
