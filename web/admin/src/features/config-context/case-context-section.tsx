import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { PenLine } from 'lucide-react'
import { listEdges, listPresence } from '@/lib/api/edges'
import { getCaseMenuPlacements } from '@/lib/api/channel-menu'
import { patchCase } from '@/lib/api/cases'
import { listRoutingAttributes } from '@/lib/api/routing'
import { listTopics } from '@/lib/api/topics'
import { queryKeys } from '@/lib/api/query-keys'
import type { CaseRecord, RoutingConfig } from '@/lib/api/types'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { TaskFlowEditor } from '@/features/task-flow/task-flow-editor'
import type { EdgePresence, EdgeRecord, TopicRecord } from '@/features/task-flow/types'
import { LinkHealthAlert } from '@/features/link-health/link-health-alert'
import { LinkHealthSection } from '@/features/link-health/link-health-section'
import { caseReferences } from '@/features/link-health/lib/references'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

/** Case 详情「处理流程」：编辑 routing + 关联上下文面板。 */
export function CaseContextSection({ record }: { record: CaseRecord }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [routing, setRouting] = useState<RoutingConfig | undefined>(record.routing)
  const [editOpen, setEditOpen] = useState(false)

  const topicsQuery = useQuery({ queryKey: queryKeys.topics.all, queryFn: () => listTopics() })
  const attributesQuery = useQuery({
    queryKey: ['routing', 'attributes'],
    queryFn: () => listRoutingAttributes(),
  })
  const edgesQuery = useQuery({ queryKey: queryKeys.edges.all, queryFn: listEdges })
  const presenceQuery = useQuery({ queryKey: queryKeys.edges.presence, queryFn: listPresence })
  const placementsQuery = useQuery({
    queryKey: queryKeys.cases.menuPlacements(record.id),
    queryFn: () => getCaseMenuPlacements(record.id),
  })

  const topics: TopicRecord[] = topicsQuery.data ?? []
  const attributes = attributesQuery.data?.attributes ?? []
  const edges: EdgeRecord[] = (edgesQuery.data ?? []).map((e) => ({
    id: e.id,
    name: e.name,
    enabled: e.enabled,
    subscribe_topics: e.subscribe_topics ?? [],
    effective_topics: e.effective_topics ?? [],
  }))
  const presence: EdgePresence[] = presenceQuery.data ?? []

  const save = useMutation({
    mutationFn: () => patchCase(record.id, { routing }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.cases.detail(record.id) })
      toast.success(t('configContext.saved'))
    },
  })

  const linkInput = {
    cases: [record],
    edges,
    presence,
    placements: placementsQuery.data ?? [],
  }
  const caseRefs = useMemo(
    () => caseReferences(record.id, linkInput),
    [record.id, linkInput],
  )

  return (
    <div className='space-y-4'>
      <LinkHealthAlert
        name={record.name}
        health={caseRefs.health}
        anchorTo='#link-health-section'
      />
      <div className='relative'>
        <TaskFlowEditor
          preview
          routing={routing}
          topics={topics}
          attributes={attributes}
          edges={edges}
          presence={presence}
          caseName={record.name}
          onChange={setRouting}
        />
        <Button
          type='button'
          size='sm'
          className='absolute right-3 top-3'
          onClick={() => setEditOpen(true)}
          data-edit-routing
        >
          <PenLine className='size-3.5' />
          {t('configContext.editFlow')}
        </Button>
      </div>

      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent className='left-0 top-0 h-screen w-screen max-w-none translate-x-0 translate-y-0 gap-0 overflow-y-auto rounded-none p-0 sm:max-w-none'>
          <DialogHeader className='sr-only'>
            <DialogTitle>{t('configContext.editFlow')}</DialogTitle>
          </DialogHeader>
          <TaskFlowEditor
            routing={routing}
            topics={topics}
            attributes={attributes}
            edges={edges}
            presence={presence}
            caseName={record.name}
            onChange={setRouting}
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
          {save.error ? (
            <p className='px-4 py-3 text-sm text-destructive' role='alert'>
              {errorMessage(save.error)}
            </p>
          ) : null}
        </DialogContent>
      </Dialog>
      <LinkHealthSection
        title={t('linkHealth.title')}
        health={caseRefs.health}
        upstream={{ title: t('linkHealth.relatedEntries'), items: caseRefs.menuEntries }}
        downstream={{ title: t('linkHealth.routeTopics'), items: caseRefs.topics }}
      />
    </div>
  )
}
