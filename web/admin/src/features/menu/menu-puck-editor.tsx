import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { Puck, type Data } from '@measured/puck'
import '@measured/puck/no-external.css'
import './menu-puck.css'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useBlocker, useNavigate } from '@tanstack/react-router'
import { ArrowLeft } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { listCases } from '@/lib/api/cases'
import { getChannel } from '@/lib/api/channels'
import { getMenu, putMenu } from '@/lib/api/channel-menu'
import { queryKeys } from '@/lib/api/query-keys'
import { Button } from '@/components/ui/button'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { PixomaLoading } from '@/components/feedback/pixoma-loading'
import { fromPuckData, toPuckData } from './puck-map'
import { menuPuckConfig } from './puck-config'
import { MenuPuckMetaProvider } from './puck-fields'
import { toWorkflowRef } from './node-view'
import { validateMenuTree } from './validate-tree'
import { FitPhonePane } from './fit-phone-pane'

function HiddenPuckHeader() {
  return <span className='hidden' />
}

function MenuPuckFrame({ children }: { children: ReactNode }) {
  return (
    <div className='flex h-full min-h-0 flex-1 flex-col overflow-hidden'>
      {children}
    </div>
  )
}

const MENU_PUCK_IFRAME = { enabled: false } as const
const MENU_PUCK_OVERRIDES = {
  header: HiddenPuckHeader,
  puck: MenuPuckFrame,
}

export function MenuPuckEditor({ channelId }: { channelId: string }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [data, setData] = useState<Data | null>(null)
  const [savedSnap, setSavedSnap] = useState('')
  const [leaveOpen, setLeaveOpen] = useState(false)
  const [issues, setIssues] = useState<string[]>([])
  const allowLeave = useRef(false)

  const channelQuery = useQuery({
    queryKey: queryKeys.channels.detail(channelId),
    queryFn: () => getChannel(channelId),
  })
  const menuQuery = useQuery({
    queryKey: queryKeys.channels.menu(channelId),
    queryFn: () => getMenu(channelId),
  })
  const casesQuery = useQuery({
    queryKey: [...queryKeys.cases.all, { limit: 200 }] as const,
    queryFn: () => listCases({ limit: 200 }),
  })
  const workflows = useMemo(
    () => (casesQuery.data ?? []).map((c) => toWorkflowRef(c)),
    [casesQuery.data]
  )

  useEffect(() => {
    if (!menuQuery.data) return
    const next = toPuckData(menuQuery.data)
    setData(next)
    setSavedSnap(JSON.stringify(next))
  }, [menuQuery.data])

  const dirty = Boolean(data && JSON.stringify(data) !== savedSnap)

  const blocker = useBlocker({
    shouldBlockFn: () => dirty && !allowLeave.current,
    withResolver: true,
    enableBeforeUnload: false,
  })

  useEffect(() => {
    const onBeforeUnload = (e: BeforeUnloadEvent) => {
      if (!dirty) return
      e.preventDefault()
      e.returnValue = ''
    }
    window.addEventListener('beforeunload', onBeforeUnload)
    return () => window.removeEventListener('beforeunload', onBeforeUnload)
  }, [dirty])

  const saveMutation = useMutation({
    mutationFn: async (next: Data) => {
      const tree = fromPuckData(next, channelId)
      const found = validateMenuTree(tree)
      if (found.length > 0) {
        setIssues(found.map((i) => t(i.key)))
        throw new Error('invalid')
      }
      setIssues([])
      return putMenu(channelId, tree)
    },
    onSuccess: async () => {
      if (data) setSavedSnap(JSON.stringify(data))
      await queryClient.invalidateQueries({
        queryKey: queryKeys.channels.menu(channelId),
      })
      toast.success(t('common.successSaved'))
    },
  })

  const goBack = () => {
    void navigate({ to: '/channels/$id', params: { id: channelId } })
  }

  const requestLeave = () => {
    if (dirty) {
      setLeaveOpen(true)
      return
    }
    goBack()
  }

  const confirmLeave = () => {
    allowLeave.current = true
    setLeaveOpen(false)
    if (blocker.status === 'blocked') blocker.proceed()
    else goBack()
  }

  const keepEditing = () => {
    setLeaveOpen(false)
    if (blocker.status === 'blocked') blocker.reset()
  }

  const leaveDialogOpen = leaveOpen || blocker.status === 'blocked'

  const headerTitle = channelQuery.data?.name || t('menu.editMenuTitle')
  const puckData = data
  const metaValue = useMemo(
    () => ({ workflows, channelId }),
    [workflows, channelId]
  )

  if (menuQuery.isLoading || !puckData) {
    return <LoadingSkeleton rows={8} />
  }
  if (menuQuery.isError) {
    return <ErrorBanner message={t('menu.mapLoadFailed')} />
  }

  return (
    <div
      data-testid='menu-puck-editor'
      className='menu-puck-shell flex h-full min-h-0 flex-1 flex-col gap-3'
    >
      <div className='flex h-11 shrink-0 items-center gap-2'>
        <Button
          type='button'
          variant='ghost'
          size='icon'
          data-testid='menu-editor-back'
          title={t('common.backToList')}
          aria-label={t('common.backToList')}
          onClick={requestLeave}
        >
          <ArrowLeft />
        </Button>
        <h1 className='truncate text-lg font-semibold'>{headerTitle}</h1>
      </div>
      {issues.length > 0 ? (
        <ErrorBanner message={issues[0]} />
      ) : saveMutation.isError && saveMutation.error.message !== 'invalid' ? (
        <ErrorBanner message={t('menu.saveFailed')} />
      ) : null}
      <div className='flex min-h-0 flex-1 flex-col overflow-hidden rounded-md border'>
        <MenuPuckMetaProvider value={metaValue}>
        <div className='flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden'>
        <Puck
          config={menuPuckConfig}
          data={puckData}
          onChange={setData}
          iframe={MENU_PUCK_IFRAME}
          overrides={MENU_PUCK_OVERRIDES}
        >
          <div className='grid h-full min-h-0 min-w-0 flex-1 grid-cols-1 grid-rows-[auto_minmax(12rem,1fr)_auto] min-[920px]:grid-cols-[220px_minmax(0,1fr)_380px] min-[920px]:grid-rows-[minmax(0,1fr)]'>
            <div
              data-testid='menu-palette'
              className='min-h-0 overflow-auto border-b p-3 min-[920px]:border-r min-[920px]:border-b-0'
            >
              <Puck.Components />
            </div>
            <FitPhonePane>
              <Puck.Preview />
            </FitPhonePane>
            <div
              data-testid='menu-fields'
              className='min-h-0 overflow-y-auto border-t p-2 min-[920px]:border-t-0 min-[920px]:border-l'
            >
              <Puck.Fields wrapFields={false} />
            </div>
          </div>
        </Puck>
        </div>
        </MenuPuckMetaProvider>
        <footer className='flex shrink-0 flex-wrap items-center gap-2 border-t bg-card px-5 py-3'>
          <Button
            type='button'
            variant='outline'
            data-testid='discard-menu'
            disabled={saveMutation.isPending || !dirty}
            onClick={() => {
              if (!dirty) return
              setLeaveOpen(true)
            }}
          >
            {t('menu.discard')}
          </Button>
          <Button
            type='button'
            data-testid='save-menu'
            disabled={saveMutation.isPending}
            onClick={() => saveMutation.mutate(puckData)}
          >
            {saveMutation.isPending ? <PixomaLoading /> : null}
            {t('common.save')}
          </Button>
        </footer>
      </div>
      <AlertDialog
        open={leaveDialogOpen}
        onOpenChange={(open) => {
          if (!open) keepEditing()
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('menu.discardEdits')}</AlertDialogTitle>
            <AlertDialogDescription className='sr-only'>
              {t('menu.discardEdits')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel type='button' onClick={keepEditing}>
              {t('common.cancel')}
            </AlertDialogCancel>
            <AlertDialogAction type='button' onClick={confirmLeave}>
              {t('menu.discard')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
