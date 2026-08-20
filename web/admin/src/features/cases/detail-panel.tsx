import { useState, type ReactNode } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Coins, FolderOpen, Hash, KeyRound, PenLine, Tags } from 'lucide-react'
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
import { kit } from '@/features/edges/kit-classes'
import { StatusTag } from '@/features/edges/presence-tags'
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
  id: string
}

export function CaseDetailPanel({ id }: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [editOpen, setEditOpen] = useState(false)

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
        <div className='flex flex-wrap items-start justify-between gap-3'>
          <div className='flex min-w-0 flex-col gap-[6px]'>
            <div className='flex flex-wrap items-center gap-2'>
              <h2 className={kit.title}>{record.name || record.id}</h2>
              <StatusTag on={record.enabled}>
                {record.enabled ? t('cases.enabled') : t('cases.disabled')}
              </StatusTag>
            </div>
            {record.description ? (
              <p className={kit.desc}>{record.description}</p>
            ) : null}
            <div className='mt-4 flex max-w-full flex-wrap items-center gap-2 text-xs'>
              <MetaChip
                icon={<Hash className='size-3.5' />}
                label={t('cases.fieldId')}
                value={record.id}
                divider
              />
              <MetaChip
                icon={<KeyRound className='size-3.5' />}
                label={t('cases.fieldMenuKey')}
                value={record.menu_key}
                divider
              />
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
          <div className='flex shrink-0 gap-2'>
            <Button
              type='button'
              className={kit.btnPrimary}
              onClick={() => setEditOpen(true)}
            >
              <PenLine className='size-3.5' />
              {t('cases.editHeading')}
            </Button>
          </div>
        </div>

        <SectionHead
          title={t('cases.sectionConfig')}
          hint={t('cases.sectionConfigHint')}
        />
        <WorkflowConfigView record={record} />

        <SectionHead
          title={t('cases.sectionEntries')}
          hint={t('cases.sectionEntriesHint')}
        />
        <MenuPlacementsSection caseId={record.id} showHeading={false} />
      </div>

      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent className='flex max-h-[85vh] flex-col sm:max-w-3xl'>
          <DialogHeader>
            <DialogTitle>{t('cases.editHeading')}</DialogTitle>
          </DialogHeader>
          <div className='min-h-0 flex-1 overflow-auto'>
            <CaseForm
              key={`edit-${record.id}`}
              mode='edit'
              initial={record}
              onSaved={(next) => {
                queryClient.setQueryData(queryKeys.cases.detail(id), next)
                setEditOpen(false)
              }}
            />
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
