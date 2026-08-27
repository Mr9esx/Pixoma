import type { CaseRecord } from '@/lib/api/types'
import { WorkflowGraphPreview } from './workflow-graph-preview'

type Props = {
  record: CaseRecord
  onSaved?: (next: CaseRecord) => void
}

export function WorkflowConfigView({ record, onSaved }: Props) {
  return (
    <div className='min-w-0 space-y-3' data-testid='case-workflow-config-view'>
      <WorkflowGraphPreview record={record} onSaved={onSaved} />
    </div>
  )
}
