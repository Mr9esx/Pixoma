import { useMutation } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { createCase, patchCase } from '@/lib/api/cases'
import type { StepActions, WizardShared } from './types'
import { WizardChrome } from './wizard-chrome'

type Props = StepActions & { shared: WizardShared }

/** Step 3 特殊规则：不需要走默认路由；需要先把工作流落库并跳转既有独立规则编辑页。 */
export function Step3Rules({ shared, next, back }: Props) {
  const { t } = useTranslation()

  const handover = useMutation({
    mutationFn: async () => {
      const draft = shared.caseRecord
      if (!draft) throw new Error('quickConfig.notReady')
      return shared.caseId != null
        ? patchCase(shared.caseId, draft)
        : createCase(draft)
    },
    onSuccess: (saved) => {
      shared.updateCase(saved)
      shared.updateRulesMode('editor')
      shared.updateRuleHandover(true)
      shared.onExit()
    },
  })

  return (
    <WizardChrome step={3} onBack={() => back({})}>
      <div className='space-y-3'>
        <h3 className='text-sm font-semibold'>
          {t('quickConfig.rulesQuestion')}
        </h3>
        <button
          type='button'
          onClick={() => {
            shared.updateRulesMode('default')
            next({})
          }}
          className='block w-full rounded-md border border-border px-3 py-2.5 text-left'
        >
          <span className='font-medium'>{t('quickConfig.noRules')}</span>
          <span className='mt-0.5 block text-xs text-muted-foreground'>
            {t('quickConfig.noRulesHint')}
          </span>
        </button>
        <button
          type='button'
          disabled={handover.isPending}
          onClick={() => handover.mutate()}
          className='block w-full rounded-md border border-border px-3 py-2.5 text-left'
        >
          <span className='font-medium'>{t('quickConfig.useRulesEditor')}</span>
          <span className='mt-0.5 block text-xs text-muted-foreground'>
            {t('quickConfig.useRulesEditorHint')}
          </span>
        </button>
      </div>
    </WizardChrome>
  )
}
