import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import {
  AtSign,
  Bot,
  CalendarCheck,
  Clock,
  PenLine,
  Power,
  SearchX,
  Trash2,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  checkChannelReachability,
  deleteChannel,
  getChannel,
  setChannelEnabled,
  updateChannel,
  type ChannelReachability,
} from '@/lib/api/channels'
import { ApiError } from '@/lib/api/client'
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
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Reveal } from '@/components/ui/reveal'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { NotFoundState } from '@/components/feedback/not-found-state'
import { Pill } from '@/components/kibo-ui/pill'
import { MetaChip } from '@/components/meta-chip'
import { SectionHead } from '@/components/section-head'
import { kit } from '@/features/edges/kit-classes'
import { channelReferences } from '@/features/link-health/lib/references'
import { LinkHealthAlert } from '@/features/link-health/link-health-alert'
import { LinkHealthSection } from '@/features/link-health/link-health-section'
import { MenuCardEditor } from '@/features/menu/menu-card-editor'
import { TextTemplatesEditor } from '@/features/text-templates/text-templates-editor'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

export function ChannelDetailPanel({ id }: { id: string }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [editOpen, setEditOpen] = useState(false)
  const [name, setName] = useState('')
  const [token, setToken] = useState('')

  const channelQuery = useQuery({
    queryKey: queryKeys.channels.detail(id),
    queryFn: () => getChannel(id),
  })
  const ch = channelQuery.data
  const extra = ch?.extra_info
  const botUsername =
    typeof extra?.username === 'string' && extra.username ? extra.username : ''

  const reachabilityQuery = useQuery({
    queryKey: ['channels', id, 'reachability'],
    queryFn: () => checkChannelReachability(id),
    enabled: Boolean(ch),
  })
  const channelHealth = useMemo(
    () => channelReferences(id, reachabilityQuery.data),
    [id, reachabilityQuery.data]
  )

  const updateMutation = useMutation({
    mutationFn: () =>
      updateChannel(id, {
        name: name.trim() || undefined,
        token: token.trim() || undefined,
      }),
    onSuccess: () => {
      setToken('')
      setEditOpen(false)
      void queryClient.invalidateQueries({
        queryKey: queryKeys.channels.detail(id),
      })
      void queryClient.invalidateQueries({ queryKey: queryKeys.channels.all })
      toast.success(t('common.successSaved'))
    },
  })

  const enableMutation = useMutation({
    mutationFn: (enabled: boolean) => setChannelEnabled(id, enabled),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: queryKeys.channels.detail(id),
      })
      void queryClient.invalidateQueries({ queryKey: queryKeys.channels.all })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: () => deleteChannel(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.channels.all })
      toast.success(t('channels.deleted'))
    },
  })

  const [ackImpact, setAckImpact] = useState(false)
  const sessionsQuery = useQuery({
    queryKey: ['channels', id, 'active-sessions'] as const,
    queryFn: async () => {
      const [collecting, confirming] = await Promise.all([
        listSessions({ channel_id: id, status: 'collecting' }),
        listSessions({ channel_id: id, status: 'confirming' }),
      ])
      return collecting.length + confirming.length
    },
  })
  const inFlightTasksQuery = useQuery({
    queryKey: ['channels', id, 'in-flight-tasks'] as const,
    queryFn: async () => {
      const [pending, queued, running] = await Promise.all([
        listTasks({ channel_id: id, status: 'pending' }),
        listTasks({ channel_id: id, status: 'queued' }),
        listTasks({ channel_id: id, status: 'running' }),
      ])
      return pending.length + queued.length + running.length
    },
  })
  const hasImpact =
    (sessionsQuery.data ?? 0) > 0 || (inFlightTasksQuery.data ?? 0) > 0

  if (channelQuery.isLoading) {
    return (
      <div className={kit.pageSection}>
        <LoadingSkeleton rows={8} />
      </div>
    )
  }
  if (channelQuery.isError || !ch) {
    const notFound =
      (channelQuery.error instanceof ApiError &&
        channelQuery.error.status === 404) ||
      (!ch && !channelQuery.error)
    if (notFound) {
      return (
        <NotFoundState
          icon={<SearchX />}
          title={t('channels.notFoundTitle')}
          description={t('channels.notFoundDesc')}
          actions={
            <>
              <Button asChild className={kit.btnPrimary}>
                <Link to='/channels'>{t('channels.backToList')}</Link>
              </Button>
              <Button
                asChild
                variant='outline'
                className='h-8 gap-1.5 rounded-md px-3 text-xs'
              >
                <Link to='/channels/$id' params={{ id: 'new' }}>
                  {t('channels.new')}
                </Link>
              </Button>
            </>
          }
        />
      )
    }
    return (
      <div className={kit.pageSection}>
        <ErrorBanner
          message={errorMessage(channelQuery.error) ?? t('common.errorGeneric')}
          onRetry={() => void channelQuery.refetch()}
        />
      </div>
    )
  }

  return (
    <Reveal
      as='section'
      className={kit.pageSection}
      data-testid='channel-detail-panel'
    >
      <div className='flex min-w-0 flex-col gap-[6px]'>
        <div className='flex flex-wrap items-center justify-between gap-3'>
          <div className='flex min-w-0 flex-wrap items-center gap-2'>
            <h2 className={kit.title}>{ch.name}</h2>
            <Pill
              dot={ch.enabled ? 'success' : 'neutral'}
              className={
                ch.enabled
                  ? 'border-success/25 bg-success/10 text-success'
                  : 'border-border bg-muted text-muted-foreground'
              }
            >
              {ch.enabled ? t('channels.enabled') : t('channels.disabled')}
            </Pill>
            <ChannelReachabilityTag query={reachabilityQuery} />
          </div>
          <div className='flex shrink-0 flex-wrap gap-2'>
            <Button
              type='button'
              className={kit.btnPrimary}
              onClick={() => setEditOpen(true)}
            >
              <PenLine className='size-3.5' strokeWidth={2} />
              {t('channels.edit')}
            </Button>
            <Button
              type='button'
              variant='outline'
              className={kit.btnGhost}
              onClick={() => enableMutation.mutate(!ch.enabled)}
              disabled={enableMutation.isPending}
            >
              <Power className='size-3.5' strokeWidth={2} />
              {ch.enabled ? t('channels.disable') : t('channels.enable')}
            </Button>
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button
                  type='button'
                  variant='destructive'
                  className='h-8 gap-1.5 rounded-md px-3 text-xs'
                  disabled={deleteMutation.isPending}
                >
                  <Trash2 className='size-3.5' strokeWidth={2} />
                  {t('channels.delete')}
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>
                    {t('channels.deleteConfirmTitle')}
                  </AlertDialogTitle>
                  <AlertDialogDescription>
                    {t('channels.deleteConfirmBody', { name: ch.name })}
                    {(sessionsQuery.data ?? 0) > 0 ? (
                      <p className='mt-3'>
                        {t('channels.deleteWillEndSessions', {
                          count: sessionsQuery.data ?? 0,
                        })}
                      </p>
                    ) : null}
                    {(inFlightTasksQuery.data ?? 0) > 0 ? (
                      <p className='mt-1'>
                        {t('channels.deleteInFlightTasks', {
                          count: inFlightTasksQuery.data ?? 0,
                        })}
                      </p>
                    ) : null}
                    {hasImpact ? (
                      <label className='mt-4 flex items-start gap-2'>
                        <input
                          type='checkbox'
                          checked={ackImpact}
                          onChange={(e) => setAckImpact(e.target.checked)}
                          data-testid='channel-delete-ack'
                        />
                        <span>{t('channels.deleteAckImpact')}</span>
                      </label>
                    ) : null}
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel type='button'>
                    {t('common.cancel')}
                  </AlertDialogCancel>
                  <AlertDialogAction
                    type='button'
                    onClick={() => deleteMutation.mutate()}
                    className='bg-destructive text-white hover:bg-destructive/90'
                    disabled={hasImpact ? !ackImpact : false}
                  >
                    {t('channels.delete')}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        </div>
        <div className='mt-1 flex max-w-full flex-wrap items-center gap-2 text-xs'>
          <MetaChip
            icon={<Bot className='size-3.5' strokeWidth={2} />}
            label={t('channels.platform')}
            value={ch.platform}
            divider
          />
          {botUsername ? (
            <MetaChip
              icon={<AtSign className='size-3.5' strokeWidth={2} />}
              label={t('channels.botUsername')}
              value={`@${botUsername}`}
              divider
            />
          ) : null}
          <MetaChip
            icon={<CalendarCheck className='size-3.5' strokeWidth={2} />}
            label={t('channels.fieldCreatedAt')}
            value={formatTime(ch.created_at)}
            divider
          />
          <MetaChip
            icon={<Clock className='size-3.5' strokeWidth={2} />}
            label={t('channels.fieldUpdatedAt')}
            value={formatTime(ch.updated_at)}
          />
        </div>
      </div>

      {reachabilityQuery.data ? (
        <LinkHealthAlert
          name={ch.name}
          health={channelHealth}
          anchorTo='#link-health-section'
        />
      ) : null}

      {updateMutation.isError ? (
        <ErrorBanner message={errorMessage(updateMutation.error)} />
      ) : null}
      {deleteMutation.isError ? (
        <ErrorBanner message={errorMessage(deleteMutation.error)} />
      ) : null}

      <section id='channel-menu-section' className='flex flex-col gap-4'>
        <SectionHead
          title={t('channels.tabMenu')}
          hint={t('channels.tabMenuHint')}
        />
        <MenuCardEditor channelId={id} />
      </section>

      <section id='channel-text-section' className='flex flex-col gap-4'>
        <SectionHead
          title={t('channels.tabText')}
          hint={t('channels.tabTextHint')}
        />
        <TextTemplatesEditor channelId={id} />
      </section>

      {reachabilityQuery.data ? (
        <LinkHealthSection
          title={t('linkHealth.title')}
          health={channelHealth}
        />
      ) : null}

      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent className='sm:max-w-lg'>
          <DialogHeader>
            <DialogTitle>{t('channels.editInfo')}</DialogTitle>
          </DialogHeader>
          <div className='space-y-4'>
            <div className='space-y-1.5'>
              <Label>{t('channels.name')}</Label>
              <Input
                value={name || ch.name}
                onChange={(e) => setName(e.target.value)}
                autoComplete='off'
              />
            </div>
            <div className='space-y-1.5'>
              <Label>{t('channels.token')}</Label>
              <Input
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder={ch.token_masked}
                autoComplete='off'
              />
              <p className='text-xs text-muted-foreground'>
                {t('channels.tokenHint')}
              </p>
            </div>
          </div>
          <div className='flex justify-end gap-2'>
            <Button
              type='button'
              variant='outline'
              onClick={() => setEditOpen(false)}
            >
              {t('common.cancel')}
            </Button>
            <Button
              disabled={
                updateMutation.isPending || (!name.trim() && !token.trim())
              }
              onClick={() => updateMutation.mutate()}
            >
              {t('common.save')}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </Reveal>
  )
}

function formatTime(iso: string): string {
  const ms = Date.parse(iso)
  if (!Number.isFinite(ms)) return iso
  return new Date(ms).toLocaleString()
}

function ChannelReachabilityTag({
  query,
}: {
  query: ReturnType<typeof useQuery<ChannelReachability, Error>>
}) {
  const { t } = useTranslation()
  if (query.isPending) {
    return (
      <Pill
        dot='neutral'
        className='border-border bg-muted text-muted-foreground'
      >
        {t('channels.checkingReachability')}
      </Pill>
    )
  }
  if (query.isError || !query.data) {
    return (
      <Pill
        dot='error'
        className='border-destructive/25 bg-destructive/10 text-destructive'
      >
        {t('channels.reachabilityFailed')}
      </Pill>
    )
  }
  const result = query.data
  if (result.kind === 'ok') {
    return (
      <Pill
        dot='success'
        className='border-success/25 bg-success/10 text-success'
      >
        {t('channels.reachabilityOK')}
      </Pill>
    )
  }
  if (result.kind === 'network') {
    return (
      <Pill
        dot='warning'
        className='border-warning/30 bg-warning/10 text-warning'
      >
        {t('channels.reachabilityNetwork')}
      </Pill>
    )
  }
  if (result.kind === 'auth') {
    return (
      <Pill
        dot='error'
        className='border-destructive/25 bg-destructive/10 text-destructive'
      >
        {t('channels.reachabilityAuth')}
      </Pill>
    )
  }
  return (
    <Pill
      dot='neutral'
      className='border-border bg-muted text-muted-foreground'
      title={result.message || undefined}
    >
      {t('channels.reachabilityFailed')}
    </Pill>
  )
}
