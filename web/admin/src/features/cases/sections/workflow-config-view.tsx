import { useTranslation } from 'react-i18next'
import type { CaseRecord } from '@/lib/api/types'
import { WorkflowGraphPreview } from './workflow-graph-preview'

type Props = {
  record: CaseRecord
  onSaved?: (next: CaseRecord) => void
}

export function WorkflowConfigView({ record, onSaved }: Props) {
  const { t } = useTranslation()

  return (
    <div className='min-w-0 space-y-3' data-testid='case-workflow-config-view'>
      {record.workflow_filename ? (
        <p className='text-xs text-muted-foreground'>
          {t('cases.importFile')}: {record.workflow_filename}
        </p>
      ) : null}

      <WorkflowGraphPreview record={record} onSaved={onSaved} />
    </div>
  )
}
