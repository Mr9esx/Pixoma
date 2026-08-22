import { useState, type ReactNode } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Coins,
  FolderOpen,
  PenLine,
  Settings2,
  Tags,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { getCase } from '@/lib/api/cases'
import { queryKeys } from '@/lib/api/query-keys'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { LongText } from '@/components/long-text'
import { kit } from '@/features/edges/kit-classes'
import { CaseForm } from './case-form'
import { MenuPlacementsSection } from './sections/menu-placements'
import { WorkflowConfigView } from './sections/workflow-config-view'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function MetaChip({
  icon,
  label,
  value,
  divider = false,
}: {
  icon: ReactNode
  label: string
  value?: string
  divider?: boolean
}) {
  return (
    <div className={kit.metaChip}>
      <div className='flex min-w-0 items-center gap-2 text-muted-foreground'>
        {icon}
        <span className='shrink-0'>{label}</span>
        <span className='min-w-0 truncate font-medium text-foreground'>
          {value || '—'}
        </span>
      </div>
      {divider ? (
        <div
          data-orientation='vertical'
          role='none'
          className={kit.metaChipDivider}
        />
      ) : null}
    </div>
  )
}

function SectionHead({ title, hint }: { title: string; hint: string }) {
  return (
    <div className='flex flex-col gap-1'>
      <div className='flex items-center gap-3'>
        <h2 className={kit.sectionTitle}>{title}</h2>
        <span className={kit.sectionDash} />
      </div>
      <p className='text-xs text-muted-foreground'>{hint}</p>
    </div>
  )
}

type Props = {
  id: number
}

export function CaseDetailPanel({ id }: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [editDialog, setEditDialog] = useState<'info' | 'workflow' | null>(null)

  const detailQuery = useQuery({
    queryKey: queryKeys.cases.detail(id),
    queryFn: () => getCase(id),
  })

  if (detailQuery.isLoading) {
    return (
      <div data-testid='case-detail-panel'>
        <LoadingSkeleton rows={8} />
      </div>
    )
  }

  if (detailQuery.isError) {
    return (
      <div data-testid='case-detail-panel' className='space-y-3'>
        <ErrorBanner
          message={errorMessage(detailQuery.error)}
          onRetry={() => void detailQuery.refetch()}
        />
      </div>
    )
  }

  const record = detailQuery.data
  if (!record) return null

  return (
    <div
      className='flex min-h-0 flex-1 flex-col'
      data-testid='case-detail-panel'
    >
      <div className={`${kit.pageSection} min-h-0 flex-1 overflow-auto`}>
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
                {t('cases.editInfo')}
              </Button>
              <Button
                type='button'
                variant='outline'
                className='h-8 gap-1.5 px-3 text-xs'
                onClick={() => setEditDialog('workflow')}
              >
                <Settings2 className='size-3.5' />
                {t('cases.editWorkflow')}
              </Button>
            </div>
          </div>
          {record.description ? (
            <LongText className='max-w-full text-sm text-muted-foreground'>
              {record.description}
            </LongText>
          ) : null}
          <div className='mt-4 flex max-w-full flex-wrap items-center gap-2 text-xs'>
            <MetaChip
              icon={<Coins className='size-3.5' />}
              label={t('cases.fieldPrice')}
              value={String(record.price)}
              divider
            />
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

        <SectionHead
          title={t('cases.sectionConfig')}
          hint={t('cases.sectionConfigHint')}
        />
        <WorkflowConfigView
          record={record}
          onSaved={(next) => queryClient.setQueryData(queryKeys.cases.detail(id), next)}
        />

        <SectionHead
          title={t('cases.sectionEntries')}
          hint={t('cases.sectionEntriesHint')}
        />
        <MenuPlacementsSection caseId={record.id} showHeading={false} />
      </div>

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
    </div>
  )
}
