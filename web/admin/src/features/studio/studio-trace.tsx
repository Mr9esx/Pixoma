import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type PointerEvent,
  type RefObject,
} from 'react'
import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { useVirtualizer } from '@tanstack/react-virtual'
import { ChevronDown, ChevronRight, Clock3, Search, X } from 'lucide-react'
import { createPortal } from 'react-dom'
import ReactMarkdown from 'react-markdown'
import {
  getStudioSessionTrajectory,
  getStudioTrajectoryRecord,
  type StudioTrajectoryDetail,
  type StudioTrajectoryRecord,
  type StudioRun,
} from '@/lib/api/studio'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { IconButtonTooltip } from '@/components/ui/icon-button-tooltip'
import { Input } from '@/components/ui/input'
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from '@/components/ui/resizable'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { formatDuration, formatRecordTiming, recordPositions, systemPromptFromRequest } from './studio-trace-data'

type RunGroup = {
  run: StudioRun
  records: StudioTrajectoryRecord[]
  turnNumber: number
}
type LedgerRow =
  | { key: string; kind: 'load' }
  | { key: string; kind: 'loading' }
  | { key: string; kind: 'error' }
  | { key: string; kind: 'empty'; label: string }
  | { key: string; kind: 'run'; group: RunGroup; collapsed: boolean }
  | {
      key: string
      kind: 'fragment'
      title: string
      firstRecord: StudioTrajectoryRecord
    }
  | {
      key: string
      kind: 'tools'
      count: number
      folded: boolean
      groupKey: string
    }
  | { key: string; kind: 'record'; record: StudioTrajectoryRecord }
type FractionRange = { start: number; end: number }
type Tab =
  | 'overview'
  | 'system'
  | 'preview'
  | 'raw'
  | 'input'
  | 'output'
  | 'schema'
  | 'usage'
  | 'timing'
const tabs: { id: Tab; title: string }[] = [
  { id: 'overview', title: '概述' },
  { id: 'system', title: '系统提示词' },
  { id: 'preview', title: '预览' },
  { id: 'raw', title: '原始内容' },
  { id: 'input', title: '输入' },
  { id: 'output', title: '输出' },
  { id: 'schema', title: 'Schema' },
  { id: 'usage', title: '用量' },
  { id: 'timing', title: '耗时' },
]
const kindName: Record<string, string> = {
  user: '用户',
  model: '模型',
  tool: '工具',
  assistant: '助手',
  reasoning: '思考',
  compaction: '压缩',
  event: '事件',
}
const kindBadgeVariant: Record<string, 'default' | 'secondary' | 'outline'> = {
  user: 'outline',
  model: 'default',
  tool: 'secondary',
  assistant: 'outline',
  reasoning: 'secondary',
  compaction: 'default',
  event: 'outline',
}
const statusName: Record<string, string> = {
  running: '进行中',
  done: '已完成',
  failed: '失败',
}

function KindBadge({ kind }: { kind: string }) {
  return (
    <Badge variant={kindBadgeVariant[kind] ?? 'outline'} className='text-sm'>
      {kindName[kind] ?? kind}
    </Badge>
  )
}

export function StudioTrace({
  sessionId,
  toolbarTarget,
}: {
  sessionId: string
  toolbarTarget: HTMLDivElement | null
}) {
  const [selectedID, setSelectedID] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [collapsedRuns, setCollapsedRuns] = useState<Set<string>>(new Set())
  const [callsCollapsed, setCallsCollapsed] = useState(false)
  const [expandedCallGroups, setExpandedCallGroups] = useState<Set<string>>(
    new Set()
  )
  const [actualDuration, setActualDuration] = useState(false)
  const [range, setRange] = useState<FractionRange | null>(null)
  const [viewport, setViewport] = useState<FractionRange>({ start: 0, end: 1 })
  const ledgerRef = useRef<HTMLDivElement>(null)
  const restoreScroll = useRef<{ height: number; top: number } | null>(null)
  const [tailInitialized, setTailInitialized] = useState(false)
  const trajectory = useInfiniteQuery({
    queryKey: ['studio', 'session-trajectory', sessionId],
    queryFn: ({ pageParam }) =>
      getStudioSessionTrajectory(sessionId, pageParam),
    initialPageParam: '',
    getNextPageParam: (page) => (page.has_more ? page.next_cursor : undefined),
    refetchInterval: (query) => {
      const status = query.state.data?.pages[0]?.runs[0]?.run.status
      return status === 'running' ||
        status === 'queued' ||
        status === 'waiting_approval' ||
        status === 'waiting_clarification'
        ? 5000
        : false
    },
  })
  const newestFirst = useMemo(
    () => trajectory.data?.pages.flatMap((page) => page.runs) ?? [],
    [trajectory.data]
  )
  const totalRuns = trajectory.data?.pages[0]?.total_runs ?? newestFirst.length
  const runs = useMemo(
    () =>
      newestFirst
        .map((group, index) => ({ ...group, turnNumber: totalRuns - index }))
        .reverse(),
    [newestFirst, totalRuns]
  )
  const records = useMemo(() => runs.flatMap((group) => group.records), [runs])
  const selected =
    selectedID === null
      ? undefined
      : records.find((record) => record.id === selectedID)
  const detail = useQuery({
    queryKey: [
      'studio',
      'trajectory-record',
      sessionId,
      selected?.run_id,
      selected?.id,
    ],
    queryFn: () =>
      getStudioTrajectoryRecord(sessionId, selected!.run_id, selected!.id),
    enabled: Boolean(selected),
    refetchInterval: selected?.status === 'running' ? 3000 : false,
  })
  const needle = search.trim().toLocaleLowerCase()
  const matched = useMemo(
    () =>
      new Set(
        records
          .filter(
            (record) =>
              needle === '' ||
              (record.title + ' ' + (record.preview ?? '') + ' ' + record.kind)
                .toLocaleLowerCase()
                .includes(needle)
          )
          .map((record) => record.id)
      ),
    [records, needle]
  )
  const visibleRuns = useMemo(
    () =>
      runs
        .map((group) => ({
          ...group,
          records: group.records.filter((record) => matched.has(record.id)),
        }))
        .filter((group) => group.records.length > 0),
    [runs, matched]
  )
  const ledgerRows = useMemo(() => {
    const rows: LedgerRow[] = []
    if (trajectory.hasNextPage) rows.push({ key: 'load', kind: 'load' })
    if (trajectory.isLoading) rows.push({ key: 'loading', kind: 'loading' })
    else if (trajectory.isError) rows.push({ key: 'error', kind: 'error' })
    else if (visibleRuns.length === 0)
      rows.push({
        key: 'empty',
        kind: 'empty',
        label: records.length === 0 ? '暂无轨迹记录' : '没有匹配的记录',
      })

    for (const group of visibleRuns) {
      const collapsed = collapsedRuns.has(group.run.id)
      rows.push({
        key: 'run:' + group.run.id,
        kind: 'run',
        group,
        collapsed,
      })
      if (collapsed) continue

      const grouped = new Map<string, StudioTrajectoryRecord[]>()
      for (const record of group.records) {
        const name = record.step
          ? '步骤 ' + record.step
          : record.kind === 'user'
            ? '输入'
            : '事件'
        const entries = grouped.get(name)
        if (entries) entries.push(record)
        else grouped.set(name, [record])
      }
      for (const [name, entries] of grouped) {
        const groupKey = group.run.id + ':' + name
        rows.push({
          key: 'fragment:' + groupKey,
          kind: 'fragment',
          title: name,
          firstRecord: entries[0],
        })
        const toolCount = entries.filter(
          (record) => record.kind === 'tool'
        ).length
        const folded =
          callsCollapsed && toolCount > 0 && !expandedCallGroups.has(groupKey)
        if (callsCollapsed && toolCount > 0)
          rows.push({
            key: 'tools:' + groupKey,
            kind: 'tools',
            count: toolCount,
            folded,
            groupKey,
          })
        for (const record of entries) {
          if (!folded || record.kind !== 'tool')
            rows.push({ key: 'record:' + record.id, kind: 'record', record })
        }
      }
    }
    return rows
  }, [
    trajectory.hasNextPage,
    trajectory.isLoading,
    trajectory.isError,
    visibleRuns,
    records.length,
    collapsedRuns,
    callsCollapsed,
    expandedCallGroups,
  ])
  const allCollapsed =
    runs.length > 0 && runs.every((group) => collapsedRuns.has(group.run.id))
  const rangeMatches = useMemo(
    () =>
      range === null
        ? null
        : new Set(
            recordPositions(records, actualDuration)
              .filter(
                (position) =>
                  position.end >= range.start && position.start <= range.end
              )
              .map((position) => position.record.id)
          ),
    [records, actualDuration, range]
  )

  useEffect(() => {
    const pane = ledgerRef.current
    if (!pane || trajectory.isLoading || tailInitialized) return
    pane.scrollTop = pane.scrollHeight
    setTailInitialized(true)
  }, [trajectory.isLoading, records.length, tailInitialized])
  useEffect(() => {
    const pane = ledgerRef.current
    const previous = restoreScroll.current
    if (!pane || !previous) return
    pane.scrollTop = previous.top + pane.scrollHeight - previous.height
    restoreScroll.current = null
  }, [trajectory.data?.pages.length])

  function focus(record: StudioTrajectoryRecord) {
    setSelectedID(record.id)
  }
  async function loadEarlier() {
    const pane = ledgerRef.current
    if (pane)
      restoreScroll.current = { height: pane.scrollHeight, top: pane.scrollTop }
    await trajectory.fetchNextPage()
  }

  return (
    <div
      className='flex min-h-0 flex-1 flex-col bg-card text-sm'
      data-slot='studio-trajectory'
    >
      {toolbarTarget
        ? createPortal(
            <>
              <Button
                size='sm'
                variant={actualDuration ? 'secondary' : 'ghost'}
                aria-pressed={actualDuration}
                aria-label={
                  actualDuration
                    ? '时间轴按记录等宽显示'
                    : '时间轴按实际时长显示'
                }
                title={
                  actualDuration
                    ? '时间轴按记录等宽显示'
                    : '时间轴按实际时长显示'
                }
                onClick={() => {
                  setActualDuration(!actualDuration)
                  setRange(null)
                  setViewport({ start: 0, end: 1 })
                }}
              >
                <Clock3 />
                {actualDuration ? '按记录等宽显示' : '按实际时长显示'}
              </Button>
              <Button
                size='sm'
                variant='ghost'
                aria-pressed={allCollapsed}
                aria-label={allCollapsed ? '展开所有轮次' : '收起所有轮次'}
                title={allCollapsed ? '展开所有轮次' : '收起所有轮次'}
                onClick={() =>
                  setCollapsedRuns(
                    allCollapsed
                      ? new Set()
                      : new Set(runs.map((group) => group.run.id))
                  )
                }
              >
                {allCollapsed ? <ChevronRight /> : <ChevronDown />}
                {allCollapsed ? '展开轮次' : '收起轮次'}
              </Button>
              <Button
                size='sm'
                variant='ghost'
                aria-pressed={callsCollapsed}
                aria-label={
                  callsCollapsed ? '展开所有工具调用' : '收起所有工具调用'
                }
                title={callsCollapsed ? '展开所有工具调用' : '收起所有工具调用'}
                onClick={() => {
                  setCallsCollapsed(!callsCollapsed)
                  setExpandedCallGroups(new Set())
                }}
              >
                {callsCollapsed ? <ChevronRight /> : <ChevronDown />}
                {callsCollapsed ? '展开工具调用' : '收起工具调用'}
              </Button>
              <div className='relative ml-1 w-32 shrink-0'>
                <Search
                  aria-hidden='true'
                  className='pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground'
                />
                <Input
                  type='search'
                  aria-label='搜索轨迹'
                  placeholder='搜索'
                  className='h-9 bg-background pl-9 text-sm'
                  value={search}
                  onChange={(event) => setSearch(event.currentTarget.value)}
                />
              </div>
            </>,
            toolbarTarget
          )
        : null}
      <TrajectoryOverview
        records={records}
        selectedID={selectedID}
        actualDuration={actualDuration}
        range={range}
        onRangeChange={setRange}
        viewport={viewport}
        onViewportChange={setViewport}
        onSelect={focus}
        onLoadEarlier={trajectory.hasNextPage ? loadEarlier : undefined}
      />
      <ResizablePanelGroup
        orientation='horizontal'
        className='relative min-h-0 flex-1 overflow-hidden'
      >
        <ResizablePanel
          id='trace-ledger'
          defaultSize={selected ? '62%' : '100%'}
          minSize='30%'
          className='min-w-0'
        >
          <TraceLedger
            rows={ledgerRows}
            scrollAreaRef={ledgerRef}
            selectedID={selectedID}
            rangeMatches={rangeMatches}
            fetchingEarlier={trajectory.isFetchingNextPage}
            onLoadEarlier={loadEarlier}
            onSelect={focus}
            onToggleRun={(id) =>
              setCollapsedRuns((current) => {
                const next = new Set(current)
                if (next.has(id)) next.delete(id)
                else next.add(id)
                return next
              })
            }
            onToggleCalls={(key) =>
              setExpandedCallGroups((current) => {
                const next = new Set(current)
                if (next.has(key)) next.delete(key)
                else next.add(key)
                return next
              })
            }
          />
        </ResizablePanel>
        {selected ? (
          <>
            <ResizableHandle className='max-md:hidden' />
            <ResizablePanel
              id='trace-details'
              defaultSize='38%'
              minSize='30%'
              className='min-w-0 max-md:absolute max-md:inset-0 max-md:z-20 max-md:w-full!'
            >
              <RecordDetails
                record={selected}
                detail={detail.data}
                loading={detail.isLoading}
                error={detail.isError}
                onClose={() => setSelectedID(null)}
              />
            </ResizablePanel>
          </>
        ) : null}
      </ResizablePanelGroup>
    </div>
  )
}

function TraceLedger({
  rows,
  scrollAreaRef,
  selectedID,
  rangeMatches,
  fetchingEarlier,
  onLoadEarlier,
  onSelect,
  onToggleRun,
  onToggleCalls,
}: {
  rows: LedgerRow[]
  scrollAreaRef: RefObject<HTMLDivElement | null>
  selectedID: string | null
  rangeMatches: Set<string> | null
  fetchingEarlier: boolean
  onLoadEarlier: () => void
  onSelect: (record: StudioTrajectoryRecord) => void
  onToggleRun: (id: string) => void
  onToggleCalls: (key: string) => void
}) {
  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => scrollAreaRef.current,
    getItemKey: (index) => rows[index].key,
    estimateSize: (index) => {
      const kind = rows[index].kind
      return kind === 'run'
        ? 48
        : kind === 'load' || kind === 'tools'
          ? 40
          : kind === 'loading'
            ? 96
            : kind === 'error' || kind === 'empty'
              ? 52
              : 44
    },
    overscan: 6,
    initialRect: { width: 0, height: 600 },
  })
  const lastFocusedID = useRef<string | null>(null)
  useEffect(() => {
    if (selectedID === lastFocusedID.current) return
    lastFocusedID.current = selectedID
    if (selectedID === null) return
    const index = rows.findIndex(
      (row) => row.kind === 'record' && row.record.id === selectedID
    )
    if (index >= 0) virtualizer.scrollToIndex(index, { align: 'center' })
  }, [selectedID, rows, virtualizer])

  const virtualRows = virtualizer.getVirtualItems()
  const paddingTop = virtualRows[0]?.start ?? 0
  const paddingBottom =
    virtualizer.getTotalSize() - (virtualRows.at(-1)?.end ?? 0)
  return (
    <div ref={scrollAreaRef} className='h-full min-w-0 overflow-y-auto'>
      <Table
        className='table-fixed border-collapse text-sm'
        wrapperClassName='overflow-visible'
        aria-label='轨迹账本'
        aria-rowcount={rows.length}
      >
        <colgroup>
          <col className='w-21' />
          <col />
        </colgroup>
        <TableBody>
          {paddingTop > 0 ? (
            <TableRow
              aria-hidden='true'
              className='border-0 hover:bg-transparent'
            >
              <TableCell
                colSpan={2}
                className='p-0'
                style={{ height: paddingTop }}
              />
            </TableRow>
          ) : null}
          {virtualRows.map((virtualRow) => (
            <LedgerRowView
              key={virtualRow.key}
              index={virtualRow.index}
              row={rows[virtualRow.index]}
              selectedID={selectedID}
              rangeMatches={rangeMatches}
              fetchingEarlier={fetchingEarlier}
              onLoadEarlier={onLoadEarlier}
              onSelect={onSelect}
              onToggleRun={onToggleRun}
              onToggleCalls={onToggleCalls}
            />
          ))}
          {paddingBottom > 0 ? (
            <TableRow
              aria-hidden='true'
              className='border-0 hover:bg-transparent'
            >
              <TableCell
                colSpan={2}
                className='p-0'
                style={{ height: paddingBottom }}
              />
            </TableRow>
          ) : null}
        </TableBody>
      </Table>
    </div>
  )
}

function LedgerRowView({
  row,
  index,
  selectedID,
  rangeMatches,
  fetchingEarlier,
  onLoadEarlier,
  onSelect,
  onToggleRun,
  onToggleCalls,
}: {
  row: LedgerRow
  index: number
  selectedID: string | null
  rangeMatches: Set<string> | null
  fetchingEarlier: boolean
  onLoadEarlier: () => void
  onSelect: (record: StudioTrajectoryRecord) => void
  onToggleRun: (id: string) => void
  onToggleCalls: (key: string) => void
}) {
  const ariaRowIndex = index + 1
  if (row.kind === 'load')
    return (
      <TableRow aria-rowindex={ariaRowIndex}>
        <TableCell colSpan={2} className='p-0'>
          <Button
            variant='ghost'
            className='h-10 w-full rounded-none text-muted-foreground'
            disabled={fetchingEarlier}
            onClick={onLoadEarlier}
          >
            {fetchingEarlier ? '正在加载…' : '加载更早的记录'}
          </Button>
        </TableCell>
      </TableRow>
    )
  if (row.kind === 'loading')
    return (
      <TableRow aria-rowindex={ariaRowIndex}>
        <TableCell colSpan={2} className='p-4'>
          <Skeleton className='h-16 w-full' />
        </TableCell>
      </TableRow>
    )
  if (row.kind === 'error' || row.kind === 'empty')
    return (
      <TableRow aria-rowindex={ariaRowIndex}>
        <TableCell
          colSpan={2}
          className={cn(
            'p-4',
            row.kind === 'error' ? 'text-destructive' : 'text-muted-foreground'
          )}
        >
          {row.kind === 'error' ? '轨迹读取失败' : row.label}
        </TableCell>
      </TableRow>
    )
  if (row.kind === 'run') {
    const { group, collapsed } = row
    const duration =
      group.run.completed_at && group.run.started_at
        ? Math.max(
            0,
            Date.parse(group.run.completed_at) -
              Date.parse(group.run.started_at)
          )
        : null
    return (
      <TableRow
        aria-rowindex={ariaRowIndex}
        className='h-12 border-y bg-muted hover:bg-muted'
        data-run-id={group.run.id}
      >
        <TableCell colSpan={2} className='border-y px-4 py-0'>
          <Button
            variant='ghost'
            className='h-11 w-full justify-start px-1 text-sm'
            onClick={() => onToggleRun(group.run.id)}
            aria-expanded={!collapsed}
          >
            {collapsed ? (
              <ChevronRight className='size-3.5' />
            ) : (
              <ChevronDown className='size-3.5' />
            )}
            <span>第 {group.turnNumber} 轮</span>
            <span className='ml-auto flex items-center gap-4 font-mono text-sm font-normal text-muted-foreground'>
              <span>
                {
                  group.records.filter((record) => record.kind === 'model')
                    .length
                }{' '}
                模型
              </span>
              <span>
                {
                  group.records.filter((record) => record.kind === 'tool')
                    .length
                }{' '}
                工具
              </span>
              <span>
                {duration === null ? '进行中' : formatDuration(duration)}
              </span>
            </span>
          </Button>
        </TableCell>
      </TableRow>
    )
  }
  if (row.kind === 'fragment')
    return (
      <TableRow aria-rowindex={ariaRowIndex} className='h-11'>
        <TableCell
          colSpan={2}
          className='px-5 py-0 font-medium text-foreground'
        >
          {row.title}
          <span className='ml-3 font-normal text-muted-foreground'>
            {row.firstRecord.kind === 'model' ? row.firstRecord.title : ''}
          </span>
        </TableCell>
      </TableRow>
    )
  if (row.kind === 'tools')
    return (
      <TableRow aria-rowindex={ariaRowIndex} className='h-10 border-border/50'>
        <TableCell className='px-2 py-0 text-right font-mono text-sm text-muted-foreground'>
          工具
        </TableCell>
        <TableCell className='px-2 py-0'>
          <Button
            variant='ghost'
            size='sm'
            aria-label={
              (row.folded ? '展开 ' : '收起 ') + row.count + ' 次工具调用'
            }
            aria-expanded={!row.folded}
            onClick={() => onToggleCalls(row.groupKey)}
            className='text-muted-foreground hover:text-foreground'
          >
            {row.folded ? (
              <ChevronRight className='size-3.5' />
            ) : (
              <ChevronDown className='size-3.5' />
            )}
            {row.count} 次工具调用
          </Button>
        </TableCell>
      </TableRow>
    )

  const { record } = row
  const outside = rangeMatches !== null && !rangeMatches.has(record.id)
  return (
    <TableRow
      id={'trajectory-record-' + record.id}
      aria-rowindex={ariaRowIndex}
      tabIndex={0}
      aria-selected={selectedID === record.id}
      onClick={() => onSelect(record)}
      onKeyDown={(event) => {
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault()
          onSelect(record)
        }
      }}
      className={cn(
        'h-11 cursor-pointer border-border/50 hover:bg-accent/50 focus-visible:outline-ring',
        selectedID === record.id && 'bg-accent',
        outside && 'opacity-30'
      )}
      data-kind={record.kind}
    >
      <TableCell className='px-2 py-0 text-right'>
        <KindBadge kind={record.kind} />
      </TableCell>
      <TableCell className='px-2 py-0'>
        <div className='flex min-w-0 items-center gap-3'>
          <span className='min-w-0 flex-1 truncate'>
            {record.preview || record.title}
          </span>
          {record.kind === 'model' && record.attempt && record.attempt > 1 ? (
            <span className='text-sm text-muted-foreground'>
              重试 {record.attempt}
            </span>
          ) : null}
          <span
            className={cn(
              'shrink-0 font-mono text-sm text-muted-foreground',
              record.status === 'failed' && 'text-destructive'
            )}
          >
            {formatRecordTiming(record)}
          </span>
        </div>
      </TableCell>
    </TableRow>
  )
}

function TrajectoryOverview({
  records,
  selectedID,
  actualDuration,
  range,
  onRangeChange,
  viewport,
  onViewportChange,
  onSelect,
  onLoadEarlier,
}: {
  records: StudioTrajectoryRecord[]
  selectedID: string | null
  actualDuration: boolean
  range: FractionRange | null
  onRangeChange: (range: FractionRange | null) => void
  viewport: FractionRange
  onViewportChange: (range: FractionRange) => void
  onSelect: (record: StudioTrajectoryRecord) => void
  onLoadEarlier?: () => void
}) {
  const trackRef = useRef<HTMLDivElement>(null)
  const drag = useRef<{ start: number; x: number; pointerId: number } | null>(
    null
  )
  const pan = useRef<{
    x: number
    pointerId: number
    viewport: FractionRange
    moved: boolean
  } | null>(null)
  const [draft, setDraft] = useState<FractionRange | null>(null)
  const positions = useMemo(
    () => recordPositions(records, actualDuration),
    [records, actualDuration]
  )
  function pointerFraction(event: PointerEvent<HTMLDivElement>) {
    const bounds = event.currentTarget.getBoundingClientRect()
    return Math.max(
      0,
      Math.min(1, (event.clientX - bounds.left) / Math.max(1, bounds.width))
    )
  }
  return (
    <section
      aria-label='轨迹时间线'
      className='grid h-24 shrink-0 grid-cols-[64px_minmax(0,1fr)] border-b bg-muted/30'
    >
      <div
        aria-hidden='true'
        className='grid grid-rows-3 border-r text-center text-sm text-muted-foreground'
      >
        <span className='flex items-center justify-center'>输入</span>
        <span className='flex items-center justify-center'>模型</span>
        <span className='flex items-center justify-center'>工具</span>
      </div>
      <div
        ref={trackRef}
        className='relative cursor-crosshair touch-none overflow-hidden'
        aria-label='时间线概览；水平拖动可聚焦事件'
        tabIndex={0}
        onPointerDown={(event) => {
          if (event.button === 2) {
            pan.current = {
              x: event.clientX,
              pointerId: event.pointerId,
              viewport,
              moved: false,
            }
            event.currentTarget.setPointerCapture(event.pointerId)
            return
          }
          if (event.button !== 0) return
          const start = pointerFraction(event)
          drag.current = { start, x: event.clientX, pointerId: event.pointerId }
          event.currentTarget.setPointerCapture(event.pointerId)
        }}
        onPointerMove={(event) => {
          if (pan.current?.pointerId === event.pointerId) {
            const current = pan.current
            if (Math.abs(event.clientX - current.x) >= 4) current.moved = true
            if (current.moved) {
              const duration = current.viewport.end - current.viewport.start
              const shift =
                ((current.x - event.clientX) /
                  Math.max(1, event.currentTarget.clientWidth)) *
                duration
              const start = Math.max(
                0,
                Math.min(1 - duration, current.viewport.start + shift)
              )
              onViewportChange({ start, end: start + duration })
            }
            return
          }
          if (
            drag.current?.pointerId !== event.pointerId ||
            Math.abs(event.clientX - drag.current.x) < 4
          )
            return
          setDraft(ordered(drag.current.start, pointerFraction(event)))
        }}
        onPointerUp={(event) => {
          if (pan.current?.pointerId === event.pointerId) {
            if (!pan.current.moved) onRangeChange(null)
            pan.current = null
            event.currentTarget.releasePointerCapture(event.pointerId)
            return
          }
          if (drag.current?.pointerId !== event.pointerId) return
          if (draft)
            onRangeChange({
              start:
                viewport.start + draft.start * (viewport.end - viewport.start),
              end: viewport.start + draft.end * (viewport.end - viewport.start),
            })
          else {
            const point =
              viewport.start +
              pointerFraction(event) * (viewport.end - viewport.start)
            const lane = Math.floor(
              (event.clientY -
                event.currentTarget.getBoundingClientRect().top -
                10) /
                32
            )
            const hit = positions.find(
              (position) =>
                position.lane === lane &&
                point >= position.start &&
                point <= Math.max(position.end, position.start + 0.01)
            )
            if (hit) onSelect(hit.record)
          }
          drag.current = null
          setDraft(null)
          event.currentTarget.releasePointerCapture(event.pointerId)
        }}
        onPointerCancel={() => {
          drag.current = null
          pan.current = null
          setDraft(null)
        }}
        onDoubleClick={() => {
          onRangeChange(null)
          onViewportChange({ start: 0, end: 1 })
        }}
        onContextMenu={(event) => {
          event.preventDefault()
        }}
        onWheel={(event) => {
          event.preventDefault()
          const bounds = event.currentTarget.getBoundingClientRect()
          const anchor = Math.max(
            0,
            Math.min(
              1,
              (event.clientX - bounds.left) / Math.max(1, bounds.width)
            )
          )
          const current = viewport.end - viewport.start
          const next = Math.max(
            0.05,
            Math.min(1, current * Math.exp(event.deltaY * 0.0015))
          )
          const center = viewport.start + anchor * current
          const start = Math.max(0, Math.min(1 - next, center - anchor * next))
          onViewportChange({ start, end: start + next })
        }}
        onKeyDown={(event) => {
          if (event.key === 'Escape') {
            onRangeChange(null)
            onViewportChange({ start: 0, end: 1 })
          }
        }}
      >
        {onLoadEarlier ? (
          <IconButtonTooltip label='加载更早的记录'>
            <Button
              variant='ghost'
              size='icon-sm'
              className='absolute inset-y-0 left-0 z-20 h-full w-8 rounded-none bg-card/80 text-muted-foreground'
              aria-label='加载更早的记录'
              onClick={(event) => {
                event.stopPropagation()
                onLoadEarlier()
              }}
            >
              ‹
            </Button>
          </IconButtonTooltip>
        ) : null}
        {records.length === 0 ? (
          <span className='absolute inset-0 flex items-center justify-center text-muted-foreground'>
            无计时数据
          </span>
        ) : null}
        {positions.map(({ record, start, end, lane }) => {
          if (end < viewport.start || start > viewport.end) return null
          const left =
            (start - viewport.start) / (viewport.end - viewport.start)
          const width = Math.max(
            0.002,
            (end - start) / (viewport.end - viewport.start)
          )
          return (
            <Tooltip key={record.id}>
              <TooltipTrigger asChild>
                <Button
                  type='button'
                  variant='ghost'
                  size='icon-sm'
                  aria-label={'选择' + record.title}
                  onClick={(event) => {
                    event.stopPropagation()
                    onSelect(record)
                  }}
                  className={cn(
                    'absolute h-3 min-w-0.5 rounded-none p-0 opacity-80 transition-opacity hover:opacity-100',
                    selectedID === record.id && 'ring-1 ring-foreground'
                  )}
                  style={{
                    backgroundColor: record.status === 'failed' ? 'var(--destructive)' : lane === 0 ? 'var(--primary)' : lane === 1 ? 'var(--chart-3)' : 'var(--chart-4)',
                    top: 10 + lane * 32,
                    left: left * 100 + '%',
                    width: width * 100 + '%',
                  }}
                />
              </TooltipTrigger>
              <TooltipContent showArrow={false} sideOffset={6} className='border bg-popover text-sm text-popover-foreground shadow-md'>
                {(kindName[record.kind] ?? record.kind) +
                  ' · ' +
                  record.title +
                  ' · ' +
                  formatRecordTiming(record)}
              </TooltipContent>
            </Tooltip>
          )
        })}
        {(draft ??
        (range && {
          start:
            (range.start - viewport.start) / (viewport.end - viewport.start),
          end: (range.end - viewport.start) / (viewport.end - viewport.start),
        })) ? (
          <span
            className='pointer-events-none absolute inset-y-0 border-x border-primary bg-primary/10'
            style={{
              left:
                (((draft ?? range)!.start - (draft ? 0 : viewport.start)) /
                  (draft ? 1 : viewport.end - viewport.start)) *
                  100 +
                '%',
              width:
                (((draft ?? range)!.end - (draft ?? range)!.start) /
                  (draft ? 1 : viewport.end - viewport.start)) *
                  100 +
                '%',
            }}
          />
        ) : null}
      </div>
    </section>
  )
}

function RecordDetails({
  record,
  detail,
  loading,
  error,
  onClose,
}: {
  record: StudioTrajectoryRecord
  detail?: StudioTrajectoryDetail
  loading: boolean
  error: boolean
  onClose: () => void
}) {
  const [activeTab, setActiveTab] = useState<Tab>('overview')
  const available = tabs.filter(
    (tab) =>
      tab.id === 'overview' ||
      tab.id === 'raw' ||
      (tab.id === 'system' && record.kind === 'model') ||
      (tab.id === 'preview' &&
        (record.kind === 'assistant' ||
          record.kind === 'reasoning' ||
          record.kind === 'tool' ||
          record.kind === 'user' ||
          record.kind === 'compaction')) ||
      (tab.id === 'input' &&
        (record.kind === 'model' || record.kind === 'tool')) ||
      (tab.id === 'output' &&
        (record.kind === 'model' || record.kind === 'tool')) ||
      (tab.id === 'schema' && record.kind === 'tool') ||
      (tab.id === 'usage' &&
        (record.kind === 'model' || record.kind === 'compaction')) ||
      (tab.id === 'timing' && record.kind !== 'user')
  )
  const current = available.some((tab) => tab.id === activeTab)
    ? activeTab
    : 'overview'
  return (
    <aside
      aria-label='事件详情'
      className='flex h-full min-h-0 flex-col bg-card'
    >
      <div className='flex h-12 shrink-0 items-center justify-between gap-2 px-4'>
        <div className='flex min-w-0 items-center gap-2'>
          <KindBadge kind={record.kind} />
          <span className='truncate text-sm text-muted-foreground'>
            {record.step ? '步骤 ' + record.step + ' · ' : ''}
            {record.title}
          </span>
        </div>
        <IconButtonTooltip label='关闭详情'>
          <Button
            variant='ghost'
            size='icon-sm'
            aria-label='关闭详情'
            onClick={onClose}
          >
            <X />
          </Button>
        </IconButtonTooltip>
      </div>
      <Tabs
        value={current}
        onValueChange={(value) => setActiveTab(value as Tab)}
        className='min-h-0 flex-1 gap-0'
      >
        <div className='flex shrink-0 items-center border-y px-4 py-2'>
          <TabsList
            aria-label='事件详情'
            className='max-w-full justify-start overflow-x-auto'
          >
            {available.map((tab) => (
              <TabsTrigger key={tab.id} value={tab.id} className='flex-none'>
                {tab.title}
              </TabsTrigger>
            ))}
          </TabsList>
        </div>
        <TabsContent
          value={current}
          className='min-h-0 flex-1 overflow-hidden'
          key={record.id + current}
        >
          <ScrollArea className='h-full'>
            <div className='p-4'>
              {loading ? (
                <Skeleton className='h-32 w-full' />
              ) : error ? (
                <p className='text-destructive'>详情读取失败</p>
              ) : detail ? (
                <InspectorContent tab={current} detail={detail} />
              ) : null}
            </div>
          </ScrollArea>
        </TabsContent>
      </Tabs>
    </aside>
  )
}

function InspectorContent({
  tab,
  detail,
}: {
  tab: Tab
  detail: StudioTrajectoryDetail
}) {
  if (tab === 'overview') {
    const fields = {
      ...detail.overview,
      status: statusName[detail.record.status] ?? detail.record.status,
      step: detail.record.step,
      attempt: detail.record.attempt,
      ...(detail.usage ?? {}),
      ...(detail.timing ?? {}),
    }
    return (
      <dl className='flex flex-col gap-3'>
        {Object.entries(fields)
          .filter(
            ([, value]) => value !== undefined && value !== null && value !== ''
          )
          .map(([name, value]) => (
            <div
              key={name}
              className='grid grid-cols-[110px_minmax(0,1fr)] gap-3 py-1 text-sm'
            >
              <dt className='text-muted-foreground'>{name}</dt>
              <dd className='min-w-0 font-mono break-all'>
                {typeof value === 'object'
                  ? JSON.stringify(value)
                  : String(value)}
              </dd>
            </div>
          ))}
      </dl>
    )
  }
  const value =
    tab === 'system'
      ? systemPromptFromRequest(detail.input)
      : tab === 'preview'
        ? (detail.output ?? detail.input ?? detail.record.preview)
        : tab === 'raw'
          ? (detail.raw ?? detail.output)
          : tab === 'input'
            ? detail.input
            : tab === 'output'
              ? detail.output
              : tab === 'schema'
                ? detail.schema
                : tab === 'usage'
                  ? detail.usage
                  : detail.timing
  if (value === undefined || value === null || value === '')
    return <p className='py-4 text-sm text-muted-foreground'>未记录</p>
  if (tab === 'preview' && typeof value === 'string')
    return (
      <div className='prose prose-base dark:prose-invert max-w-none break-words'>
        <ReactMarkdown>{value}</ReactMarkdown>
      </div>
    )
  return (
    <pre className='overflow-auto font-mono text-sm leading-6 break-all whitespace-pre-wrap'>
      {typeof value === 'string' ? value : JSON.stringify(value, null, 2)}
    </pre>
  )
}

function ordered(a: number, b: number): FractionRange {
  return { start: Math.min(a, b), end: Math.max(a, b) }
}
