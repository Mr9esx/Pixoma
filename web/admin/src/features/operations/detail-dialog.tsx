import { Dialog, DialogContent } from '@/components/ui/dialog'
import { resourceDetailDialogClassName } from '@/features/resource-modal'
import { SessionDetailPanel } from '@/features/sessions/detail-panel'
import { TaskDetailPanel } from '@/features/tasks/detail-panel'
import { UserDetailPanel } from '@/features/users/detail-panel'
import type { OperationsDetailTarget } from './types'

type Props = {
  target: OperationsDetailTarget | null
  onClose: () => void
  onOpenRelated: (target: OperationsDetailTarget) => void
}

export function OperationsDetailDialog({
  target,
  onClose,
  onOpenRelated,
}: Props) {
  return (
    <Dialog
      open={target != null}
      onOpenChange={(open) => {
        if (!open) onClose()
      }}
    >
      <DialogContent className={resourceDetailDialogClassName}>
        {target?.kind === 'task' ? (
          <TaskDetailPanel
            key={target.id}
            id={target.id}
            onOpenRelated={onOpenRelated}
          />
        ) : null}
        {target?.kind === 'session' ? (
          <SessionDetailPanel
            key={target.id}
            id={target.id}
            onOpenRelated={onOpenRelated}
          />
        ) : null}
        {target?.kind === 'user' ? (
          <UserDetailPanel key={target.id} id={target.id} />
        ) : null}
      </DialogContent>
    </Dialog>
  )
}
