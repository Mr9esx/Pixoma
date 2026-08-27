import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import {
  FolderOpen,
  PenLine,
  SearchX,
  Settings2,
  Tags,
  Trash2,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { deleteCase, getCase } from '@/lib/api/cases'
import { ApiError } from '@/lib/api/client'
import { caseDeleteErrorMessage } from '@/lib/api/localized-errors'
import { queryKeys } from '@/lib/api/query-keys'
import { listSessions } from '@/lib/api/sessions'
import { listTasks } from '@/lib/api/tasks'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { NotFoundState } from '@/components/feedback/not-found-state'
import { LongText } from '@/components/long-text'
import { MetaChip } from '@/components/meta-chip'
import { SectionHead } from '@/components/section-head'
import { CaseContextSection } from '@/features/config-context/case-context-section'
import { useCaseReferences } from '@/features/config-context/use-case-references'
import { kit } from '@/features/edges/kit-classes'
import { LinkHealthAlert } from '@/features/link-health/link-health-alert'
import { LinkHealthSection } from '@/features/link-health/link-health-section'
import { CaseForm } from './case-form'
import { WorkflowConfigView } from './sections/workflow-config-view'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

type Props = {
  id: number
}

export function CaseDetailPanel({ id }: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const [editDialog, setEditDialog] = useState<'info' | 'workflow' | null>(null)
  const [ackRefs, setAckRefs] = useState(false)

  const detailQuery = useQuery({
    queryKey: queryKeys.cases.detail(id),
    queryFn: () => getCase(id),
  })
  const record = detailQuery.data
  const { topics, attributes, edges, presence, placements, caseRefs } =
    useCaseReferences(record)
  const pendingTasksQuery = useQuery({
    queryKey: ['cases', id, 'pending-tasks'] as const,
    queryFn: () => listTasks({ case_id: id, status: 'pending' }),
  })
  const activeSessionsQuery = useQuery({
    queryKey: ['cases', id, 'active-sessions'] as const,
    queryFn: async () => {
      const [collecting, confirming] = await Promise.all([
        listSessions({ case_id: id, status: 'collecting' }),
        listSessions({ case_id: id, status: 'confirming' }),
      ])
      return collecting.length + confirming.length
    },
  })

  const deleteMutation = useMutation({
    mutationFn: async () => {
      if (!record) throw new Error('case missing')
      return deleteCase(record.id, ackRefs)
    },
    onSuccess: async (summary) => {
      const refs = summary.removed_placements?.length ?? 0
      if (
        refs > 0 ||
        (summary.failed_tasks ?? 0) > 0 ||
        (summary.terminated_sessions ?? 0) > 0
      ) {
        toast.success(
          t('cases.deleteSuccessSummary', {
            refs,
            tasks: summary.failed_tasks ?? 0,
            sessions: summary.terminated_sessions ?? 0,
          })
        )
      } else {
        toast.success(t('cases.deleteSuccess'))
      }
      await queryClient.invalidateQueries({ queryKey: queryKeys.cases.all })
      queryClient.removeQueries({ queryKey: queryKeys.cases.detail(id) })
      void navigate({ to: '/cases', state: { backToList: true } } as never)
    },
    onError: (err) => {
      const detail = caseDeleteErrorMessage(err, t) ?? errorMessage(err)
      toast.error(
        detail
          ? `${t('cases.deleteFailed')}：${detail}`
          : t('cases.deleteFailed')
      )
    },
  })

  if (detailQuery.isLoading) {
    return (
      <div className={kit.pageSection} data-testid='case-detail-panel'>
        <LoadingSkeleton rows={8} />
      </div>
    )
  }

  const notFound =
    (detailQuery.error instanceof ApiError &&
      detailQuery.error.status === 404) ||
    (!record && !detailQuery.error)

  if (notFound) {
    return (
      <NotFoundState
        icon={<SearchX />}
        title={t('cases.notFoundTitle')}
        description={t('cases.notFoundDesc')}
        actions={
          <>
            <Button asChild className={kit.btnPrimary}>
              <Link to='/cases'>{t('cases.backToList')}</Link>
            </Button>
            <Button
              asChild
              variant='outline'
              className='h-8 gap-1.5 rounded-md px-3 text-xs'
            >
              <Link to='/cases/$caseId' params={{ caseId: 'new' }}>
                {t('cases.createHeading')}
              </Link>
            </Button>
          </>
        }
      />
    )
  }

  if (detailQuery.isError) {
    return (
      <div className={kit.pageSection} data-testid='case-detail-panel'>
        <ErrorBanner
          message={errorMessage(detailQuery.error)}
          onRetry={() => void detailQuery.refetch()}
        />
      </div>
    )
  }

  if (!record) return null

  return (
    <section className={kit.pageSection} data-testid='case-detail-panel'>
      <div className='flex min-w-0 flex-col gap-[6px]'>
        <div className='flex flex-wrap items-center justify-between gap-3'>
          <div className='flex min-w-0 flex-wrap items-center gap-2'>
            <h2 className={kit.title}>{record.name || record.id}</h2>
          </div>
          <div className='flex shrink-0 flex-wrap gap-2'>
            <Button
              type='button'
              className={kit.btnPrimary}
              onClick={() => setEditDialog('info')}
            >
              <PenLine className='size-3.5' />
              {t('cases.edit')}
            </Button>
            <Button
              type='button'
              variant='outline'
              className={kit.btnGhost}
              onClick={() => setEditDialog('workflow')}
            >
              <Settings2 className='size-3.5' />
              {t('cases.editWorkflow')}
            </Button>
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button
                  type='button'
                  variant='destructive'
                  className='h-8 gap-1.5 rounded-md px-3 text-xs'
                  disabled={deleteMutation.isPending}
                >
                  <Trash2 className='size-3.5' />
                  {t('cases.deleteWorkflow')}
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>
                    {t('cases.deleteWorkflowTitle')}
                  </AlertDialogTitle>
                  <AlertDialogDescription>
                    {t('cases.deleteWorkflowBody', {
                      name: record.name || record.id,
                    })}
                    {placements.length > 0 ? (
                      <div className='mt-3 space-y-2'>
                        <p className='font-medium'>
                          {t('cases.deleteWillRemoveRefs', {
                            count: placements.length,
                          })}
                        </p>
                        <ul className='max-h-32 overflow-auto rounded-md border bg-muted/20 p-3 text-xs'>
                          {placements.map((p) => (
                            <li key={`${p.channel_id}:${p.item_id}`}>
                              {p.channel_name || p.channel_id} ·{' '}
                              {p.path.map((s) => s.label).join(' / ')}
                            </li>
                          ))}
                        </ul>
                      </div>
                    ) : null}
                    {(pendingTasksQuery.data?.length ?? 0) > 0 ||
                    (activeSessionsQuery.data ?? 0) > 0 ? (
                      <p className='mt-3 text-xs text-muted-foreground'>
                        {(pendingTasksQuery.data?.length ?? 0) > 0
                          ? t('cases.deleteWillFailTasks', {
                              count: pendingTasksQuery.data?.length ?? 0,
                            })
                          : null}
                        {(activeSessionsQuery.data ?? 0) > 0
                          ? t('cases.deleteWillEndSessions', {
                              count: activeSessionsQuery.data ?? 0,
                            })
                          : null}
                      </p>
                    ) : null}
                    <label
                      htmlFor='case-delete-ack'
                      className='mt-4 flex items-center gap-2 text-sm'
                    >
                      <Checkbox
                        id='case-delete-ack'
                        checked={ackRefs}
                        onCheckedChange={(v) => setAckRefs(v === true)}
                        data-testid='case-delete-ack'
                      />
                      <span>{t('cases.deleteAckRefs')}</span>
                    </label>
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel
                    type='button'
                    disabled={deleteMutation.isPending}
                  >
                    {t('common.cancel')}
                  </AlertDialogCancel>
                  <AlertDialogAction
                    type='button'
                    className='bg-destructive text-white hover:bg-destructive/90'
                    disabled={deleteMutation.isPending || !ackRefs}
                    onClick={() => deleteMutation.mutate()}
                  >
                    {t('common.delete')}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        </div>
        {record.description ? (
          <LongText className='max-w-full text-sm text-muted-foreground'>
            {record.description}
          </LongText>
        ) : null}
        <div className='mt-4 flex max-w-full flex-wrap items-center gap-2 text-xs'>
          <MetaChip
            icon={<Tags className='size-3.5' />}
            label={t('cases.fieldTags')}
            value={record.tags?.join(', ')}
            divider
          />
          <MetaChip
            icon={<FolderOpen className='size-3.5' />}
            label={t('cases.fieldCategories')}
            value={record.categories?.join(', ')}
          />
        </div>
      </div>

      {caseRefs ? (
        <LinkHealthAlert
          name={record.name}
          health={caseRefs.health}
          anchorTo='#link-health-section'
        />
      ) : null}

      <section className='flex flex-col gap-4'>
        <SectionHead
          title={t('cases.sectionConfig')}
          hint={t('cases.sectionConfigHint')}
        />
        <WorkflowConfigView
          record={record}
          onSaved={(next) =>
            queryClient.setQueryData(queryKeys.cases.detail(id), next)
          }
        />
      </section>

      <section id='case-routing-section' className='flex flex-col gap-4'>
        <SectionHead
          title={t('cases.sectionProcessing')}
          hint={t('cases.sectionProcessingHint')}
        />
        <CaseContextSection
          record={record}
          data={{ topics, attributes, edges, presence }}
        />
      </section>

      {caseRefs ? (
        <LinkHealthSection
          title={t('linkHealth.title')}
          health={caseRefs.health}
          upstream={{
            title: t('linkHealth.relatedEntries'),
            items: caseRefs.menuEntries,
          }}
          downstream={{
            title: t('linkHealth.routeTopics'),
            items: caseRefs.topics,
          }}
        />
      ) : null}

      <Dialog
        open={editDialog !== null}
        onOpenChange={(v) => (v ? undefined : setEditDialog(null))}
      >
        <DialogContent className='flex max-h-[85vh] flex-col gap-0 p-0 sm:max-w-3xl'>
          <DialogHeader className='border-b px-5 py-4'>
            <DialogTitle>
              {editDialog === 'info'
                ? t('cases.editInfo')
                : t('cases.editWorkflow')}
            </DialogTitle>
          </DialogHeader>
          <div className='min-h-0 flex-1 overflow-auto px-5 py-4'>
            {editDialog === 'info' ? (
              <CaseForm
                key={`info-${record.id}`}
                mode='edit'
                initial={record}
                showWorkflow={false}
                onSaved={(next) => {
                  queryClient.setQueryData(queryKeys.cases.detail(id), next)
                  setEditDialog(null)
                }}
              />
            ) : editDialog === 'workflow' ? (
              <CaseForm
                key={`wf-${record.id}`}
                mode='edit'
                initial={record}
                showBasics={false}
                onSaved={(next) => {
                  queryClient.setQueryData(queryKeys.cases.detail(id), next)
                  setEditDialog(null)
                }}
              />
            ) : null}
          </div>
          <div className='flex items-center justify-end gap-2 border-t px-5 py-3'>
            <Button
              type='button'
              variant='outline'
              className='h-8 gap-1.5 px-3 text-xs'
              onClick={() => setEditDialog(null)}
            >
              {t('common.cancel')}
            </Button>
            <Button
              type='submit'
              form='case-edit-form'
              className='h-8 gap-1.5 px-3 text-xs'
            >
              {t('common.save')}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </section>
  )
}
