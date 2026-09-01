import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { PenLine } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { patchCase } from '@/lib/api/cases'
import { queryKeys } from '@/lib/api/query-keys'
import type { CaseRecord, RoutingConfig } from '@/lib/api/types'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { TaskFlowTable } from '@/features/task-flow/task-flow-table'
import type { CaseContextData } from './use-case-references'

/** Case 详情「处理流程」：编辑 routing + 关联上下文面板。 */
export function CaseContextSection({
  record,
  data,
}: {
  record: CaseRecord
  data: Pick<CaseContextData, 'topics' | 'attributes' | 'edges' | 'presence'>
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [routing, setRouting] = useState<RoutingConfig | undefined>(
    record.routing
  )
  const [editOpen, setEditOpen] = useState(false)
  const { topics, attributes, edges, presence } = data

  const save = useMutation({
    mutationFn: () => patchCase(record.id, { routing }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: queryKeys.cases.detail(record.id),
      })
      toast.success(t('configContext.saved'))
    },
  })

  return (
    <>
      <TaskFlowTable
        preview
        hideActions
        routing={routing}
        topics={topics}
        attributes={attributes}
        edges={edges}
        presence={presence}
        onChange={setRouting}
        headerActions={
          <Button
            type='button'
            variant='outline'
            size='sm'
            className='h-7 gap-1.5 px-2.5 text-xs'
            onClick={() => setEditOpen(true)}
            data-edit-routing
          >
            <PenLine className='size-3.5' />
            {t('configContext.editFlow')}
          </Button>
        }
      />

      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent className='top-0 left-0 h-screen w-screen max-w-none translate-x-0 translate-y-0 gap-0 overflow-y-auto rounded-none p-0 sm:max-w-none'>
          <DialogHeader className='sr-only'>
            <DialogTitle>{t('configContext.editFlow')}</DialogTitle>
          </DialogHeader>
          <TaskFlowTable
            routing={routing}
            topics={topics}
            attributes={attributes}
            edges={edges}
            presence={presence}
            onChange={setRouting}
            title={record.name}
            className='h-full rounded-none border-0 shadow-none'
            headerActions={
              <Button
                type='button'
                size='sm'
                disabled={save.isPending}
                onClick={() => save.mutate()}
                data-case-routing-save
              >
                {t('configContext.saveRouting')}
              </Button>
            }
          />
        </DialogContent>
      </Dialog>
    </>
  )
}
