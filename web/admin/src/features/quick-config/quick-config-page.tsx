import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  ArrowRight,
  ChevronLeft,
  ChevronRight,
  Search,
  Sparkles,
  Trash2,
} from 'lucide-react'
import { getCase, listCases } from '@/lib/api/cases'
import { queryKeys } from '@/lib/api/query-keys'
import type { CaseRecord } from '@/lib/api/types'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { clearQuickConfigSession, loadQuickConfigSession } from './lib/session'
import { QuickConfigFlow } from './quick-config-flow'
import type { PendingMenuEntry, WizardMode } from './types'

export function QuickConfigPage() {
  const [wizard, setWizard] = useState<{
    mode: WizardMode
    caseRecord?: CaseRecord
    routing?: CaseRecord['routing']
    pendingEntries?: PendingMenuEntry[]
  } | null>(null)
  const [selectedCase, setSelectedCase] = useState<CaseRecord | null>(null)
  const [query, setQuery] = useState('')
  const [page, setPage] = useState(0)
  const [sessionEpoch, setSessionEpoch] = useState(0)
  const PAGE_SIZE = 8

  const casesQuery = useQuery({
    queryKey: queryKeys.cases.all,
    queryFn: () => listCases(),
  })

  const session = useMemo(
    () =>
      typeof window === 'undefined'
        ? null
        : loadQuickConfigSession(window.localStorage),
    // 废弃草稿后递增 epoch，触发重新读取 localStorage（此时已无会话）。
    [sessionEpoch]
  )

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase()
    return (casesQuery.data ?? []).filter(
      (caseRecord) =>
        !needle ||
        caseRecord.name.toLowerCase().includes(needle) ||
        (caseRecord.description ?? '').toLowerCase().includes(needle)
    )
  }, [casesQuery.data, query])

  const pageCount = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE))
  const safePage = Math.min(page, pageCount - 1)
  const visibleCases = filtered.slice(
    safePage * PAGE_SIZE,
    safePage * PAGE_SIZE + PAGE_SIZE
  )

  async function resume() {
    if (!session) return
    const draft =
      session.caseDraft &&
      typeof session.caseDraft === 'object' &&
      session.caseDraft !== null &&
      'id' in session.caseDraft &&
      'name' in session.caseDraft
        ? (session.caseDraft as CaseRecord)
        : undefined
    const caseRecord =
      draft ?? (session.caseId != null ? await getCase(session.caseId) : undefined)
    if (!caseRecord) return
    setWizard({
      mode: session.mode,
      caseRecord,
      routing:
        (session.routing as CaseRecord['routing'] | undefined) ??
        caseRecord.routing,
      pendingEntries: session.pendingEntries,
    })
  }

  if (wizard) {
    return (
        <QuickConfigFlow
          mode={wizard.mode}
          initialCase={wizard.caseRecord}
          initialRouting={wizard.routing}
          initialPendingEntries={wizard.pendingEntries}
          onExit={() => setWizard(null)}
        />
    )
  }

  return (
    <div className='mx-auto flex h-full min-h-0 w-full max-w-3xl flex-col gap-4 px-6 py-6'>
      <div>
        <h1 className='text-lg font-semibold'>从一个工作流开始</h1>
        <p className='text-sm text-muted-foreground'>
          三步向导：工作流配置 → 处理流程 → 投放；完成页就绪清单全绿即可发布。
        </p>
      </div>

      {session ? (
        <AlertDialog>
          <div className='flex shrink-0 items-center justify-between gap-3 rounded-lg border border-border bg-background px-3 py-2'>
            <div className='flex min-w-0 items-center gap-2 text-sm'>
              <span className='text-muted-foreground'>继续上次配置</span>
              <span className='truncate font-medium'>
                {session.caseDraft &&
                typeof session.caseDraft === 'object' &&
                session.caseDraft !== null &&
                'name' in session.caseDraft
                  ? String((session.caseDraft as { name?: unknown }).name ?? '')
                  : session.caseId != null
                    ? `Case #${session.caseId}`
                    : '未命名工作流'}
              </span>
            </div>
            <div className='flex shrink-0 items-center gap-2'>
              <AlertDialogTrigger asChild>
                <Button
                  type='button'
                  size='sm'
                  variant='outline'
                  className='gap-1.5 text-destructive hover:text-destructive'
                >
                  <Trash2 className='size-3.5' />
                  废弃
                </Button>
              </AlertDialogTrigger>
              <Button type='button' size='sm' onClick={() => void resume()}>
                继续
                <ArrowRight className='size-4' />
              </Button>
            </div>
          </div>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>废弃未完成的配置？</AlertDialogTitle>
              <AlertDialogDescription>
                将清除本地保存的草稿（工作流草稿、处理流程与待投放菜单），
                此操作不可恢复。
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel type='button'>取消</AlertDialogCancel>
              <AlertDialogAction
                type='button'
                className='bg-destructive text-white hover:bg-destructive/90'
                onClick={() => {
                  clearQuickConfigSession(window.localStorage)
                  setSessionEpoch((e) => e + 1)
                }}
              >
                废弃
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      ) : null}

      <Card className='flex min-h-0 flex-1 flex-col gap-3'>
        <CardHeader className='pb-0'>
          <CardTitle className='text-sm'>选择已有工作流</CardTitle>
        </CardHeader>
        <CardContent className='flex min-h-0 flex-1 flex-col gap-2 pt-0'>
          <div className='relative shrink-0'>
            <Search className='absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground' />
            <Input
              className='pl-9'
              placeholder='搜索名称 / 描述…'
              value={query}
              onChange={(e) => {
                setQuery(e.target.value)
                setPage(0)
              }}
            />
          </div>
          {casesQuery.isLoading ? (
            <LoadingSkeleton rows={3} />
          ) : (
            <div
              className={`min-h-0 flex-1 overflow-auto rounded-lg border border-border ${
                visibleCases.length === 0
                  ? 'grid grid-rows-[minmax(0,1fr)]'
                  : ''
              }`}
            >
              <Table
                className={visibleCases.length === 0 ? 'h-full' : undefined}
              >
                <TableHeader>
                  <TableRow className='bg-muted/40'>
                    <TableHead className='w-10'>
                      <span className='sr-only'>选择</span>
                    </TableHead>
                    <TableHead className='w-2/5'>名称</TableHead>
                    <TableHead>描述</TableHead>
                    <TableHead className='w-20'>状态</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {visibleCases.map((caseRecord) => (
                    <TableRow
                      key={caseRecord.id}
                      className={`cursor-pointer ${
                        selectedCase?.id === caseRecord.id
                          ? 'bg-muted/60'
                          : ''
                      }`}
                      onClick={() => setSelectedCase(caseRecord)}
                    >
                      <TableCell>
                        <Checkbox
                          checked={selectedCase?.id === caseRecord.id}
                          onCheckedChange={() => setSelectedCase(caseRecord)}
                          aria-label={`选择 ${caseRecord.name}`}
                          onClick={(e) => e.stopPropagation()}
                        />
                      </TableCell>
                      <TableCell className='font-medium'>
                        {caseRecord.name}
                      </TableCell>
                      <TableCell className='text-muted-foreground'>
                        {caseRecord.description ?? `#${caseRecord.id}`}
                      </TableCell>
                      <TableCell className='text-muted-foreground'>
                        {caseRecord.enabled ? '已启用' : '已停用'}
                      </TableCell>
                    </TableRow>
                  ))}
                  {visibleCases.length === 0 ? (
                    <TableRow>
                      <TableCell
                        colSpan={4}
                        className='py-8 text-center text-muted-foreground'
                      >
                        没有匹配的工作流
                      </TableCell>
                    </TableRow>
                  ) : null}
                </TableBody>
              </Table>
            </div>
          )}
          <div className='flex shrink-0 items-center justify-between text-xs text-muted-foreground'>
            <span>
              共 {filtered.length} 个 · 第 {safePage + 1} / {pageCount} 页
            </span>
            <div className='flex items-center gap-1'>
              <Button
                type='button'
                variant='outline'
                size='icon'
                className='size-7'
                disabled={safePage === 0}
                onClick={() => setPage((p) => Math.max(0, p - 1))}
                aria-label='上一页'
              >
                <ChevronLeft className='size-4' />
              </Button>
              <Button
                type='button'
                variant='outline'
                size='icon'
                className='size-7'
                disabled={safePage >= pageCount - 1}
                onClick={() => setPage((p) => Math.min(pageCount - 1, p + 1))}
                aria-label='下一页'
              >
                <ChevronRight className='size-4' />
              </Button>
            </div>
          </div>
          <div className='flex shrink-0 items-center justify-between gap-3 pt-1'>
            <button
              type='button'
              className='qc-create-btn'
              onClick={() => setWizard({ mode: 'create' })}
            >
              <Sparkles className='size-4' />
              创建新的工作流
            </button>
            <Button
              type='button'
              disabled={selectedCase == null}
              onClick={() =>
                selectedCase &&
                setWizard({
                  mode: 'existing',
                  caseRecord: selectedCase,
                })
              }
            >
              下一步
              <ArrowRight className='size-4' />
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
