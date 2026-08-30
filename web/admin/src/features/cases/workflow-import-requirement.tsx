import { useTranslation } from 'react-i18next'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'

export function WorkflowImportRequirement() {
  const { t } = useTranslation()

  return (
    <Alert variant='info' data-testid='workflow-import-requirement'>
      <AlertTitle>{t('quickConfig.workflowImportTitle')}</AlertTitle>
      <AlertDescription>
        {t('quickConfig.workflowImportDescription')}
      </AlertDescription>
    </Alert>
  )
}
