import { useEffect, useMemo, useRef, useState, type PointerEvent } from 'react'
import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { ChevronDown, ChevronRight, Clock3, Search, X } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import {
  getStudioSessionTrajectory,
  getStudioTrajectoryRecord,
  type StudioTrajectoryDetail,
  type StudioTrajectoryRecord,
  type StudioRun,
} from '@/lib/api/studio'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { recordPositions, systemPromptFromRequest } from './studio-trace-data'

type RunGroup = {
  run: StudioRun
  records: StudioTrajectoryRecord[]
  turnNumber: number
}
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
const statusName: Record<string, string> = {
  running: '进行中',
  done: '完成',
  failed: '失败',
}

export function StudioTrace({ sessionId }: { sessionId: string }) {
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
  const [detailsWidth, setDetailsWidth] = useState<number | null>(null)
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
        status === 'waiting_approval'
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
    document
      .getElementById('trajectory-record-' + record.id)
      ?.scrollIntoView({ block: 'center', behavior: 'smooth' })
  }
  async function loadEarlier() {
    const pane = ledgerRef.current
    if (pane)
      restoreScroll.current = { height: pane.scrollHeight, top: pane.scrollTop }
    await trajectory.fetchNextPage()
  }

  return (
    <div
      className='flex min-h-0 flex-1 flex-col bg-background text-xs'
      data-slot='studio-trajectory'
    >
      <div
        role='toolbar'
        aria-label='轨迹工具栏'
        className='flex h-8 shrink-0 items-center gap-1 border-b bg-background px-1.5'
      >
        <Button
          size='xs'
          variant={actualDuration ? 'secondary' : 'ghost'}
          aria-pressed={actualDuration}
          aria-label={actualDuration ? '使用等宽操作' : '使用实际时长'}
          onClick={() => {
            setActualDuration(!actualDuration)
            setRange(null)
            setViewport({ start: 0, end: 1 })
          }}
        >
          <Clock3 />
          时长
        </Button>
        <Button
          size='xs'
          variant='ghost'
          aria-pressed={allCollapsed}
          aria-label={allCollapsed ? '展开所有轮次' : '收起所有轮次'}
          onClick={() =>
            setCollapsedRuns(
              allCollapsed
                ? new Set()
                : new Set(runs.map((group) => group.run.id))
            )
          }
        >
          {allCollapsed ? '⊞' : '⊟'} 轮次
        </Button>
        <Button
          size='xs'
          variant='ghost'
          aria-pressed={callsCollapsed}
          aria-label={callsCollapsed ? '展开所有调用' : '收起所有调用'}
          onClick={() => {
            setCallsCollapsed(!callsCollapsed)
            setExpandedCallGroups(new Set())
          }}
        >
          {callsCollapsed ? '⊞' : '⊟'} 调用
        </Button>
        <label className='ml-auto flex h-6 w-40 items-center gap-1 rounded-sm border bg-muted/40 px-1.5 text-muted-foreground focus-within:border-ring'>
          <Search className='size-3' />
          <input
            type='search'
            aria-label='搜索轨迹'
            placeholder='搜索'
            className='min-w-0 flex-1 bg-transparent text-xs text-foreground outline-none placeholder:text-muted-foreground'
            value={search}
            onChange={(event) => setSearch(event.currentTarget.value)}
          />
        </label>
      </div>
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
      <div className='flex min-h-0 flex-1 overflow-hidden'>
        <div
          ref={ledgerRef}
          className='min-w-0 flex-1 overflow-auto'
          onClick={(event) => {
            if (event.target === event.currentTarget) setSelectedID(null)
          }}
        >
          <table
            className='w-full table-fixed border-collapse text-xs'
            aria-label='轨迹账本'
          >
            <colgroup>
              <col className='w-21' />
              <col />
            </colgroup>
            <tbody>
              {trajectory.hasNextPage ? (
                <tr>
                  <td colSpan={2} className='border-b p-0'>
                    <button
                      className='h-8 w-full text-muted-foreground hover:bg-accent'
                      disabled={trajectory.isFetchingNextPage}
                      onClick={loadEarlier}
                    >
                      {trajectory.isFetchingNextPage
                        ? '正在加载…'
                        : '加载更早的记录'}
                    </button>
                  </td>
                </tr>
              ) : null}
              {trajectory.isLoading ? (
                <tr>
                  <td colSpan={2} className='p-4'>
                    <Skeleton className='h-16 w-full' />
                  </td>
                </tr>
              ) : null}
              {trajectory.isError ? (
                <tr>
                  <td colSpan={2} className='p-4 text-destructive'>
                    轨迹读取失败
                  </td>
                </tr>
              ) : null}
              {!trajectory.isLoading && records.length === 0 ? (
                <tr>
                  <td colSpan={2} className='p-4 text-muted-foreground'>
                    暂无轨迹记录
                  </td>
                </tr>
              ) : null}
              {visibleRuns.map((group) => (
                <RunLedger
                  key={group.run.id}
                  group={group}
                  collapsed={collapsedRuns.has(group.run.id)}
                  onToggle={() =>
                    setCollapsedRuns((current) => {
                      const next = new Set(current)
                      if (next.has(group.run.id)) next.delete(group.run.id)
                      else next.add(group.run.id)
                      return next
                    })
                  }
                  selectedID={selectedID}
                  onSelect={focus}
                  rangeMatches={rangeMatches}
                  callsCollapsed={callsCollapsed}
                  expandedCallGroups={expandedCallGroups}
                  onToggleCalls={(key) =>
                    setExpandedCallGroups((current) => {
                      const next = new Set(current)
                      if (next.has(key)) next.delete(key)
                      else next.add(key)
                      return next
                    })
                  }
                />
              ))}
            </tbody>
          </table>
        </div>
        {selected ? (
          <RecordDetails
            record={selected}
            detail={detail.data}
            loading={detail.isLoading}
            error={detail.isError}
            width={detailsWidth}
            onResize={setDetailsWidth}
            onClose={() => setSelectedID(null)}
          />
        ) : null}
      </div>
    </div>
  )
}

function RunLedger({
  group,
  collapsed,
  onToggle,
  selectedID,
  onSelect,
  rangeMatches,
  callsCollapsed,
  expandedCallGroups,
  onToggleCalls,
}: {
  group: RunGroup
  collapsed: boolean
  onToggle: () => void
  selectedID: string | null
  onSelect: (record: StudioTrajectoryRecord) => void
  rangeMatches: Set<string> | null
  callsCollapsed: boolean
  expandedCallGroups: Set<string>
  onToggleCalls: (key: string) => void
}) {
  const grouped = new Map<string, StudioTrajectoryRecord[]>()
  for (const record of group.records) {
    const key = record.step
      ? '步骤 ' + record.step
      : record.kind === 'user'
        ? '输入'
        : '事件'
    grouped.set(key, [...(grouped.get(key) ?? []), record])
  }
  const duration =
    group.run.completed_at && group.run.started_at
      ? Math.max(
          0,
          Date.parse(group.run.completed_at) - Date.parse(group.run.started_at)
        )
      : null
  return (
    <>
      <tr
        className='sticky top-0 z-10 h-11 bg-muted/80 backdrop-blur-sm'
        data-run-id={group.run.id}
      >
        <td colSpan={2} className='border-y px-4'>
          <button
            className='flex w-full items-center gap-2 text-left font-medium'
            onClick={onToggle}
            aria-expanded={!collapsed}
          >
            {collapsed ? (
              <ChevronRight className='size-3.5' />
            ) : (
              <ChevronDown className='size-3.5' />
            )}
            <span>第 {group.turnNumber} 轮</span>
            <span className='ml-auto flex items-center gap-4 font-mono text-[11px] font-normal text-muted-foreground'>
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
          </button>
        </td>
      </tr>
      {!collapsed
        ? [...grouped].map(([name, groupRecords]) => (
            <FragmentRows
              key={name}
              title={name}
              records={groupRecords}
              selectedID={selectedID}
              onSelect={onSelect}
              rangeMatches={rangeMatches}
              callsCollapsed={callsCollapsed}
              callsExpanded={expandedCallGroups.has(group.run.id + ':' + name)}
              onToggleCalls={() => onToggleCalls(group.run.id + ':' + name)}
            />
          ))
        : null}
    </>
  )
}

function FragmentRows({
  title,
  records,
  selectedID,
  onSelect,
  rangeMatches,
  callsCollapsed,
  callsExpanded,
  onToggleCalls,
}: {
  title: string
  records: StudioTrajectoryRecord[]
  selectedID: string | null
  onSelect: (record: StudioTrajectoryRecord) => void
  rangeMatches: Set<string> | null
  callsCollapsed: boolean
  callsExpanded: boolean
  onToggleCalls: () => void
}) {
  const toolCount = records.filter((record) => record.kind === 'tool').length
  const folded = callsCollapsed && toolCount > 0 && !callsExpanded
  const shown = folded
    ? records.filter((record) => record.kind !== 'tool')
    : records
  return (
    <>
      <tr className='h-9'>
        <td colSpan={2} className='border-b px-5 font-medium text-foreground'>
          {title}
          <span className='ml-3 font-normal text-muted-foreground'>
            {records[0]?.kind === 'model' ? records[0].title : ''}
          </span>
        </td>
      </tr>
      {callsCollapsed && toolCount > 0 ? (
        <tr className='h-8 border-b border-border/50'>
          <td className='px-2 text-right font-mono text-[11px] text-muted-foreground'>
            工具
          </td>
          <td className='px-2'>
            <button
              aria-label={
                (folded ? '展开 ' : '收起 ') + toolCount + ' 次工具调用'
              }
              aria-expanded={!folded}
              onClick={onToggleCalls}
              className='flex items-center gap-1 text-muted-foreground hover:text-foreground'
            >
              {folded ? (
                <ChevronRight className='size-3.5' />
              ) : (
                <ChevronDown className='size-3.5' />
              )}
              {toolCount} 次工具调用
            </button>
          </td>
        </tr>
      ) : null}
      {shown.map((record) => {
        const outside = rangeMatches !== null && !rangeMatches.has(record.id)
        return (
          <tr
            id={'trajectory-record-' + record.id}
            key={record.id}
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
              'h-8 cursor-pointer border-b border-border/50 hover:bg-accent/50 focus-visible:outline-ring',
              selectedID === record.id && 'bg-accent',
              outside && 'opacity-30'
            )}
            data-kind={record.kind}
          >
            <td className='px-2 text-right font-mono text-[11px] text-muted-foreground'>
              {record.kind === 'model' ? '● ' : ''}
              {kindName[record.kind] ?? record.kind}
            </td>
            <td className='px-2'>
              <div className='flex min-w-0 items-center gap-3'>
                <span className='min-w-0 flex-1 truncate'>
                  {record.preview || record.title}
                </span>
                {record.kind === 'model' &&
                record.attempt &&
                record.attempt > 1 ? (
                  <span className='text-[11px] text-muted-foreground'>
                    重试 {record.attempt}
                  </span>
                ) : null}
                <span
                  className={cn(
                    'shrink-0 font-mono text-[11px] text-muted-foreground',
                    record.status === 'failed' && 'text-destructive'
                  )}
                >
                  {record.status === 'running'
                    ? statusName.running
                    : record.ended_at
                      ? formatDuration(
                          Math.max(
                            0,
                            Date.parse(record.ended_at) -
                              Date.parse(record.started_at)
                          )
                        )
                      : (statusName[record.status] ?? record.status)}
                </span>
              </div>
            </td>
          </tr>
        )
      })}
    </>
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
      className='grid h-12.5 shrink-0 grid-cols-[44px_minmax(0,1fr)] border-b bg-muted/30'
    >
      <div
        aria-hidden='true'
        className='flex flex-col justify-around border-r pr-1 text-right text-[10px] text-muted-foreground'
      >
        <span>输入</span>
        <span>模型</span>
        <span>工具</span>
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
                7) /
                14
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
          <button
            className='absolute inset-y-0 left-0 z-20 w-6 bg-gradient-to-r from-background to-transparent text-muted-foreground'
            aria-label='加载更早的记录'
            onClick={(event) => {
              event.stopPropagation()
              onLoadEarlier()
            }}
          >
            ‹
          </button>
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
            <button
              key={record.id}
              type='button'
              title={
                (kindName[record.kind] ?? record.kind) +
                ' · ' +
                record.title +
                ' · ' +
                (record.ended_at
                  ? formatDuration(
                      Date.parse(record.ended_at) -
                        Date.parse(record.started_at)
                    )
                  : '进行中')
              }
              aria-label={'选择' + record.title}
              onClick={(event) => {
                event.stopPropagation()
                onSelect(record)
              }}
              className={cn(
                'absolute h-2 min-w-0.5 rounded-[1px] opacity-80 hover:opacity-100',
                lane === 0
                  ? 'bg-primary'
                  : lane === 1
                    ? 'bg-chart-3'
                    : 'bg-chart-4',
                record.status === 'failed' && 'bg-destructive',
                selectedID === record.id && 'ring-1 ring-foreground'
              )}
              style={{
                top: 7 + lane * 14,
                left: left * 100 + '%',
                width: width * 100 + '%',
              }}
            />
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
  width,
  onResize,
  onClose,
}: {
  record: StudioTrajectoryRecord
  detail?: StudioTrajectoryDetail
  loading: boolean
  error: boolean
  width: number | null
  onResize: (width: number | null) => void
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
  const drag = useRef<{ x: number; width: number } | null>(null)
  return (
    <aside
      aria-label='事件详情'
      className='relative flex min-h-0 w-[38%] max-w-[calc(100%-280px)] min-w-80 shrink-0 flex-col border-l bg-background max-md:absolute max-md:inset-0 max-md:z-20 max-md:w-full max-md:max-w-none'
      style={width === null ? undefined : { width }}
    >
      <div
        role='separator'
        aria-label='调整事件详情宽度'
        aria-orientation='vertical'
        tabIndex={0}
        className='absolute inset-y-0 -left-1 z-10 w-2 cursor-col-resize max-md:hidden'
        onDoubleClick={() => onResize(null)}
        onPointerDown={(event) => {
          if (event.button !== 0) return
          drag.current = {
            x: event.clientX,
            width:
              event.currentTarget.parentElement?.getBoundingClientRect()
                .width ?? 400,
          }
          event.currentTarget.setPointerCapture(event.pointerId)
        }}
        onPointerMove={(event) => {
          if (!drag.current) return
          const parent = event.currentTarget.parentElement?.parentElement
          onResize(
            Math.max(
              320,
              Math.min(
                (parent?.clientWidth ?? 900) - 280,
                drag.current.width + drag.current.x - event.clientX
              )
            )
          )
        }}
        onPointerUp={(event) => {
          drag.current = null
          event.currentTarget.releasePointerCapture(event.pointerId)
        }}
        onKeyDown={(event) => {
          if (event.key === 'ArrowLeft' || event.key === 'ArrowRight') {
            onResize(
              Math.max(
                320,
                (width ?? 400) + (event.key === 'ArrowLeft' ? 16 : -16)
              )
            )
            event.preventDefault()
          }
        }}
      />
      <div className='flex h-10.5 shrink-0 items-center justify-between gap-2 border-b px-3'>
        <div className='flex min-w-0 items-center gap-2'>
          <span className='rounded-sm bg-muted px-1.5 py-0.5 font-mono text-[11px] font-medium'>
            {kindName[record.kind] ?? record.kind}
          </span>
          <span className='truncate text-[11px] text-muted-foreground'>
            {record.step ? '步骤 ' + record.step + ' · ' : ''}
            {record.title}
          </span>
        </div>
        <Button
          variant='ghost'
          size='icon-xs'
          aria-label='关闭详情'
          onClick={onClose}
        >
          <X />
        </Button>
      </div>
      <div
        role='tablist'
        aria-label='事件详情'
        className='flex h-8.5 shrink-0 items-stretch overflow-x-auto border-b px-2'
      >
        {available.map((tab) => (
          <button
            key={tab.id}
            role='tab'
            aria-selected={current === tab.id}
            className={cn(
              'shrink-0 border-b-2 border-transparent px-2 text-xs text-muted-foreground hover:text-foreground',
              current === tab.id && 'border-primary text-primary'
            )}
            onClick={() => setActiveTab(tab.id)}
          >
            {tab.title}
          </button>
        ))}
      </div>
      <div
        role='tabpanel'
        className='min-h-0 flex-1 overflow-auto p-3'
        key={record.id + current}
      >
        {loading ? (
          <Skeleton className='h-32 w-full' />
        ) : error ? (
          <p className='text-destructive'>详情读取失败</p>
        ) : detail ? (
          <InspectorContent tab={current} detail={detail} />
        ) : null}
      </div>
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
      <dl className='space-y-0'>
        {Object.entries(fields)
          .filter(
            ([, value]) => value !== undefined && value !== null && value !== ''
          )
          .map(([name, value]) => (
            <div
              key={name}
              className='grid grid-cols-[110px_minmax(0,1fr)] gap-3 border-b py-2 text-xs'
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
    return <p className='py-4 text-xs text-muted-foreground'>未记录</p>
  if (tab === 'preview' && typeof value === 'string')
    return (
      <div className='prose prose-sm dark:prose-invert max-w-none break-words'>
        <ReactMarkdown>{value}</ReactMarkdown>
      </div>
    )
  return (
    <pre className='overflow-auto font-mono text-xs leading-5 break-all whitespace-pre-wrap'>
      {typeof value === 'string' ? value : JSON.stringify(value, null, 2)}
    </pre>
  )
}

function ordered(a: number, b: number): FractionRange {
  return { start: Math.min(a, b), end: Math.max(a, b) }
}
function formatDuration(ms: number) {
  return ms < 1000 ? Math.round(ms) + 'ms' : (ms / 1000).toFixed(1) + 's'
}
