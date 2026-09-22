import { useMemo, useState, type ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  BotMessageSquare,
  ChevronDown,
  ChevronRight,
  Clock3,
  FileCode2,
  Hammer,
  ListTree,
  Search,
  Sparkles,
  Waypoints,
} from 'lucide-react'
import {
  listStudioRunEvents,
  listStudioSessionRuns,
  type StudioRun,
  type StudioRunEvent,
} from '@/lib/api/studio'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

type TraceKind = 'assistant' | 'reasoning' | 'tool' | 'model' | 'event'

type TraceRecord = {
  id: string
  kind: TraceKind
  sequence: number
  title: string
  content: string
  details?: Record<string, unknown>
  startedAt: string
  endedAt?: string
  status?: 'done' | 'running' | 'failed'
}

export function StudioTrace({ sessionId }: { sessionId: string }) {
  const runs = useQuery({
    queryKey: ['studio', 'session-runs', sessionId],
    queryFn: () => listStudioSessionRuns(sessionId),
    refetchInterval: 1500,
  })
  const [selectedRunID, setSelectedRunID] = useState<string>()
  const [query, setQuery] = useState('')
  const runID = selectedRunID ?? runs.data?.[0]?.id
  const events = useQuery({
    queryKey: ['studio', 'run-events', runID],
    queryFn: () => listStudioRunEvents(runID!),
    enabled: Boolean(runID),
    refetchInterval: 1500,
  })
  const records = useMemo(() => projectTrace(events.data ?? []), [events.data])
  const visibleRecords = useMemo(() => filterTrace(records, query), [records, query])
  const attempts = useMemo(() => projectModelAttempts(events.data ?? []), [events.data])

  return (
    <div className='flex min-h-0 flex-1 bg-background'>
      <RunList
        loading={runs.isLoading}
        runID={runID}
        runs={runs.data ?? []}
        onSelect={setSelectedRunID}
      />
      <ScrollArea className='min-w-0 flex-1'>
        <div className='mx-auto flex max-w-5xl flex-col gap-5 p-5'>
          <header className='flex flex-wrap items-end justify-between gap-3'>
            <div className='flex flex-col gap-1'>
              <div className='flex items-center gap-2 text-xs font-medium text-muted-foreground'>
                <Waypoints className='size-3.5' />
                运行轨迹
              </div>
              <h2 className='text-lg font-semibold tracking-tight'>执行 Trace</h2>
            </div>
            <TraceSummary records={records} attempts={attempts.length} />
          </header>

          <TraceTimeline records={records} />

          <div className='flex flex-wrap items-center justify-between gap-3 border-y py-2'>
            <div className='flex items-center gap-2 text-xs text-muted-foreground'>
              <ListTree className='size-3.5' />
              轨迹账本
              <span className='font-mono tabular-nums'>{visibleRecords.length}</span>
            </div>
            <label className='relative block'>
              <Search className='pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground' />
              <input
                aria-label='搜索轨迹'
                className='h-8 w-44 rounded-md border bg-background pr-3 pl-8 text-xs outline-none transition-colors placeholder:text-muted-foreground focus-visible:ring-[3px] focus-visible:ring-ring/50'
                onChange={(event) => setQuery(event.currentTarget.value)}
                placeholder='搜索输出与工具'
                value={query}
              />
            </label>
          </div>

          {events.isLoading ? <TraceLoading /> : null}
          {!events.isLoading && runID && records.length === 0 ? <TraceEmpty /> : null}
          <div className='flex flex-col gap-1'>
            {visibleRecords.map((record, index) => (
              <TraceLedgerRow key={record.id} record={record} index={index + 1} />
            ))}
          </div>
        </div>
      </ScrollArea>
    </div>
  )
}

function RunList({
  runs,
  runID,
  onSelect,
  loading,
}: {
  runs: StudioRun[]
  runID?: string
  onSelect: (id: string) => void
  loading: boolean
}) {
  return (
    <ScrollArea className='w-56 shrink-0 border-e bg-muted/20'>
      <div className='flex flex-col gap-2 p-3'>
        <p className='px-2 text-[11px] font-semibold tracking-wide text-muted-foreground uppercase'>
          执行记录
        </p>
        {loading ? <Skeleton className='h-14 w-full' /> : null}
        {runs.map((run) => (
          <Button
            key={run.id}
            variant='ghost'
            className={cn(
              'h-auto w-full justify-start rounded-md border px-3 py-2 text-start',
              runID === run.id
                ? 'border-border bg-background hover:bg-background'
                : 'border-transparent hover:bg-background'
            )}
            onClick={() => onSelect(run.id)}
          >
            <span className='flex min-w-0 flex-1 flex-col gap-1'>
              <span className='flex items-center gap-2 text-sm font-medium'>
                <RunIcon run={run} />
                <span className='truncate'>运行 {run.id.slice(0, 6)}</span>
              </span>
              <span className='font-mono text-[11px] text-muted-foreground tabular-nums'>
                {run.status}
              </span>
            </span>
          </Button>
        ))}
        {!loading && runs.length === 0 ? (
          <p className='px-2 text-xs leading-5 text-muted-foreground'>暂无执行记录</p>
        ) : null}
      </div>
    </ScrollArea>
  )
}

function TraceSummary({ records, attempts }: { records: TraceRecord[]; attempts: number }) {
  const tools = records.filter((record) => record.kind === 'tool').length
  return (
    <div className='flex items-center gap-3 font-mono text-[11px] text-muted-foreground tabular-nums'>
      <span>{records.length} records</span>
      <span>{attempts} model</span>
      <span>{tools} tools</span>
    </div>
  )
}

function TraceTimeline({ records }: { records: TraceRecord[] }) {
  const first = records[0]
  const last = records[records.length - 1]
  const total = Math.max(
    1,
    first && last
      ? new Date(last.endedAt ?? last.startedAt).getTime() - new Date(first.startedAt).getTime()
      : 1
  )
  return (
    <section aria-label='轨迹总览' className='border bg-muted/20'>
      <div className='flex items-center justify-between border-b px-3 py-2'>
        <h3 className='text-xs font-medium'>轨迹总览</h3>
        <span className='font-mono text-[11px] text-muted-foreground tabular-nums'>
          {formatDuration(total)}
        </span>
      </div>
      <div className='grid grid-cols-[44px_minmax(0,1fr)]'>
        <div className='flex flex-col justify-around border-e py-2 pr-2 text-end text-[10px] text-muted-foreground'>
          <span>输出</span>
          <span>模型</span>
          <span>工具</span>
        </div>
        <div className='relative h-14 overflow-hidden'>
          <div className='absolute inset-x-0 top-1/3 border-t border-dashed' />
          <div className='absolute inset-x-0 top-2/3 border-t border-dashed' />
          {records.map((record) => <TraceSpan key={record.id} record={record} first={first} total={total} />)}
        </div>
      </div>
    </section>
  )
}

function TraceSpan({ record, first, total }: { record: TraceRecord; first?: TraceRecord; total: number }) {
  const started = new Date(record.startedAt).getTime()
  const ended = new Date(record.endedAt ?? record.startedAt).getTime()
  const offset = first ? ((started - new Date(first.startedAt).getTime()) / total) * 100 : 0
  const width = Math.max(1.5, ((ended - started) / total) * 100)
  return (
    <span
      aria-label={record.title}
      className={cn(
        'absolute h-2 rounded-sm',
        record.kind === 'tool' && 'bg-warning',
        record.kind === 'model' && 'bg-accent-brand',
        record.kind === 'reasoning' && 'bg-muted-foreground',
        record.kind === 'assistant' && 'bg-primary',
        record.kind === 'event' && 'bg-border'
      )}
      style={{
        left: `${Math.min(98.5, Math.max(0, offset))}%`,
        top: record.kind === 'tool' ? '39px' : record.kind === 'model' ? '26px' : '12px',
        width: `${Math.min(100 - offset, width)}%`,
      }}
    />
  )
}

function TraceLedgerRow({ record, index }: { record: TraceRecord; index: number }) {
  const [open, setOpen] = useState(record.kind === 'assistant' || record.kind === 'reasoning')
  const hasDetails = record.content !== '' || record.details !== undefined
  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <article className='border bg-background'>
        <CollapsibleTrigger asChild>
          <button
            type='button'
            className='grid w-full grid-cols-[2.25rem_5.5rem_minmax(0,1fr)_auto] items-center gap-2 px-3 py-2 text-start transition-colors hover:bg-muted/40 focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:outline-none'
          >
            <span className='font-mono text-[11px] text-muted-foreground tabular-nums'>#{index}</span>
            <TraceKindBadge kind={record.kind} />
            <span className='min-w-0 truncate text-sm'>{record.title}</span>
            <span className='flex items-center gap-2'>
              <time className='font-mono text-[11px] text-muted-foreground tabular-nums'>
                {formatTime(record.startedAt)}
              </time>
              {hasDetails ? (
                open ? <ChevronDown className='size-3.5 text-muted-foreground' /> : <ChevronRight className='size-3.5 text-muted-foreground' />
              ) : null}
            </span>
          </button>
        </CollapsibleTrigger>
        {hasDetails ? (
          <CollapsibleContent>
            <div className='border-t bg-muted/20 px-3 py-3 pl-32'>
              {record.content !== '' ? (
                <pre className='overflow-x-auto font-mono text-xs leading-5 whitespace-pre-wrap text-foreground'>{record.content}</pre>
              ) : null}
              {record.details ? (
                <pre className='mt-3 overflow-x-auto border-t pt-3 font-mono text-[11px] leading-5 whitespace-pre-wrap text-muted-foreground'>
                  {JSON.stringify(record.details, null, 2)}
                </pre>
              ) : null}
            </div>
          </CollapsibleContent>
        ) : null}
      </article>
    </Collapsible>
  )
}

function TraceKindBadge({ kind }: { kind: TraceKind }) {
  const labels: Record<TraceKind, string> = { assistant: '输出', reasoning: '思考', tool: '工具', model: '模型', event: '事件' }
  const icons: Record<TraceKind, ReactNode> = {
    assistant: <BotMessageSquare />, reasoning: <Sparkles />, tool: <Hammer />, model: <Clock3 />, event: <FileCode2 />,
  }
  return <Badge variant='outline'>{icons[kind]}{labels[kind]}</Badge>
}

function TraceLoading() {
  return <div className='flex flex-col gap-1'>{[0, 1, 2].map((index) => <Skeleton key={index} className='h-10 w-full' />)}</div>
}

function TraceEmpty() {
  return (
    <Empty>
      <EmptyHeader>
        <EmptyMedia variant='icon'><Waypoints /></EmptyMedia>
        <EmptyTitle>尚未产生轨迹</EmptyTitle>
        <EmptyDescription>发起一次对话后在这里查看模型与工具记录</EmptyDescription>
      </EmptyHeader>
    </Empty>
  )
}

function projectTrace(events: StudioRunEvent[]): TraceRecord[] {
  const records: TraceRecord[] = []
  const byID = new Map<string, TraceRecord>()
  const upsert = (id: string, record: TraceRecord) => {
    const current = byID.get(id)
    if (current) return current
    byID.set(id, record)
    records.push(record)
    return record
  }
  for (const event of events) {
    const payload = event.payload
    const messageID = stringValue(payload.message_id) || event.id
    const toolCallID = stringValue(payload.tool_call_id) || event.id
    switch (event.type) {
      case 'TEXT_MESSAGE_START':
        upsert(`text:${messageID}`, createRecord(`text:${messageID}`, 'assistant', event, '模型输出'))
        break
      case 'TEXT_MESSAGE_CONTENT': {
        const record = upsert(`text:${messageID}`, createRecord(`text:${messageID}`, 'assistant', event, '模型输出'))
        record.content += stringValue(payload.delta)
        break
      }
      case 'TEXT_MESSAGE_END': {
        const record = upsert(`text:${messageID}`, createRecord(`text:${messageID}`, 'assistant', event, '模型输出'))
        record.content = stringValue(payload.content) || record.content
        record.endedAt = event.created_at
        record.status = 'done'
        break
      }
      case 'REASONING_MESSAGE_START':
        upsert(`reasoning:${messageID}`, createRecord(`reasoning:${messageID}`, 'reasoning', event, '模型思考'))
        break
      case 'REASONING_MESSAGE_CONTENT': {
        const record = upsert(`reasoning:${messageID}`, createRecord(`reasoning:${messageID}`, 'reasoning', event, '模型思考'))
        record.content += stringValue(payload.content) || stringValue(payload.delta)
        break
      }
      case 'REASONING_MESSAGE_END': {
        const record = byID.get(`reasoning:${messageID}`)
        if (record) {
          record.endedAt = event.created_at
          record.status = 'done'
        }
        break
      }
      case 'TOOL_CALL_START':
        upsert(`tool:${toolCallID}`, createRecord(`tool:${toolCallID}`, 'tool', event, stringValue(payload.tool_name) || '调用工具'))
        break
      case 'TOOL_CALL_ARGS': {
        const record = upsert(`tool:${toolCallID}`, createRecord(`tool:${toolCallID}`, 'tool', event, '调用工具'))
        record.details = { ...(record.details ?? {}), 参数: `${stringValue(record.details?.参数)}${stringValue(payload.delta)}` }
        break
      }
      case 'TOOL_CALL_RESULT': {
        const record = upsert(`tool:${toolCallID}`, createRecord(`tool:${toolCallID}`, 'tool', event, '调用工具'))
        record.content = stringValue(payload.content)
        record.status = boolValue(payload.is_error) ? 'failed' : 'done'
        break
      }
      case 'TOOL_CALL_END': {
        const record = byID.get(`tool:${toolCallID}`)
        if (record) {
          record.title = stringValue(payload.tool_name) || record.title
          record.endedAt = event.created_at
        }
        break
      }
      default:
        if (!isModelTraceEvent(event.type) && event.type !== 'RUN_STARTED' && event.type !== 'RUN_FINISHED') {
          records.push({
            ...createRecord(`event:${event.id}`, 'event', event, eventLabel(event.type)),
            content: stringValue(payload.name) || stringValue(payload.content),
            details: payload,
            endedAt: event.created_at,
            status: 'done',
          })
        }
    }
  }
  for (const attempt of projectModelAttempts(events)) {
    const source = attempt.started ?? attempt.firstToken ?? attempt.finished ?? attempt.failed
    if (!source) continue
    const record = createRecord(`model:${attempt.id}`, 'model', source, stringValue(source.payload.model) || '模型请求')
    record.endedAt = attempt.finished?.created_at ?? attempt.failed?.created_at
    record.status = attempt.failed ? 'failed' : attempt.finished ? 'done' : 'running'
    record.details = {
      首Token: formatMetric(attempt.firstToken?.payload.elapsed_ms),
      耗时: formatMetric(attempt.finished?.payload.elapsed_ms ?? attempt.failed?.payload.elapsed_ms),
      输入: formatMetric(attempt.finished?.payload.input_tokens, false),
      输出: formatMetric(attempt.finished?.payload.output_tokens, false),
      ...(attempt.failed ? { 错误: stringValue(attempt.failed.payload.error) } : {}),
    }
    records.push(record)
  }
  return records.sort((left, right) => left.sequence - right.sequence)
}

function filterTrace(records: TraceRecord[], query: string) {
  const keyword = query.trim().toLocaleLowerCase()
  if (keyword === '') return records
  return records.filter((record) => `${record.title}\n${record.content}`.toLocaleLowerCase().includes(keyword))
}

function createRecord(id: string, kind: TraceKind, event: StudioRunEvent, title: string): TraceRecord {
  return { id, kind, sequence: event.sequence, title, content: '', startedAt: event.created_at, status: 'running' }
}

type ModelAttempt = { id: string; started?: StudioRunEvent; firstToken?: StudioRunEvent; finished?: StudioRunEvent; failed?: StudioRunEvent }

function projectModelAttempts(events: StudioRunEvent[]): ModelAttempt[] {
  const attempts = new Map<string, ModelAttempt>()
  for (const event of events) {
    if (!isModelTraceEvent(event.type)) continue
    const id = stringValue(event.payload.attempt_id) || event.id
    const attempt = attempts.get(id) ?? { id }
    if (event.type === 'MODEL_REQUEST_STARTED') attempt.started = event
    if (event.type === 'MODEL_FIRST_TOKEN') attempt.firstToken = event
    if (event.type === 'MODEL_REQUEST_FINISHED') attempt.finished = event
    if (event.type === 'MODEL_REQUEST_FAILED') attempt.failed = event
    attempts.set(id, attempt)
  }
  return [...attempts.values()]
}

function isModelTraceEvent(type: string) { return ['MODEL_REQUEST_STARTED', 'MODEL_FIRST_TOKEN', 'MODEL_REQUEST_FINISHED', 'MODEL_REQUEST_FAILED'].includes(type) }
function eventLabel(type: string) {
  return ({
    ASSET_CREATED: '创建资产',
    FLOW_UPDATED: '更新资产路线',
    APPROVAL_REQUIRED: '请求批准',
    APPROVAL_RESOLVED: '批准已处理',
  } as Record<string, string>)[type] ?? type
}
function stringValue(value: unknown) { return typeof value === 'string' ? value : '' }
function boolValue(value: unknown) { return value === true }
function formatMetric(value: unknown, milliseconds = true) { return typeof value === 'number' ? `${value.toLocaleString()}${milliseconds ? ' ms' : ''}` : '—' }
function formatTime(value: string) { return new Date(value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' }) }
function formatDuration(value: number) { return value < 1000 ? `${value} ms` : `${(value / 1000).toFixed(1)} s` }

function RunIcon({ run }: { run: StudioRun }) {
  return <span className={cn('size-2 rounded-full', run.status === 'succeeded' ? 'bg-success' : run.status === 'failed' || run.status === 'cancelled' ? 'bg-warning' : 'bg-muted-foreground')} />
}
