import { useMemo, useState } from 'react'
import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { Bar, BarChart, CartesianGrid, Cell, ReferenceLine, XAxis, YAxis } from 'recharts'
import {
  getStudioContextRequest,
  getStudioCurrentContext,
  getStudioSessionContext,
  listStudioContextEvents,
  listStudioContextRequests,
  type StudioContextEvent,
  type StudioContextPart,
  type StudioContextRequest,
} from '@/lib/api/studio'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from '@/components/ui/chart'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { cn } from '@/lib/utils'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

const categories = [
  { id: 'system_prompt', name: '系统提示词', color: 'bg-chart-1', chartColor: 'var(--chart-1)' },
  { id: 'tool_definition', name: '工具定义', color: 'bg-chart-2', chartColor: 'var(--chart-2)' },
  { id: 'user_message', name: '用户消息', color: 'bg-chart-3', chartColor: 'var(--chart-3)' },
  { id: 'injection', name: '注入内容', color: 'bg-chart-4', chartColor: 'var(--chart-4)' },
  { id: 'skill_injection', name: 'Skill 注入', color: 'bg-chart-4', chartColor: 'var(--chart-4)' },
  { id: 'assistant_message', name: '助手消息', color: 'bg-chart-5', chartColor: 'var(--chart-5)' },
  { id: 'tool_result', name: '工具结果', color: 'bg-primary', chartColor: 'var(--primary)' },
] as const

const trendConfig: ChartConfig = {
  ...Object.fromEntries(categories.map((category) => [category.id, { label: category.name, color: category.chartColor }])),
  delta: { label: '增量', color: 'var(--primary)' },
}

type Category = (typeof categories)[number]['id']

function number(value: number | null | undefined) {
  return value === null || value === undefined
    ? '未记录'
    : Intl.NumberFormat('zh-CN', { notation: 'compact', maximumFractionDigits: 1 }).format(value)
}

function duration(value: number) {
  if (value < 1000) return `${Math.round(value)} 毫秒`
  if (value < 60000) return `${(value / 1000).toFixed(1)} 秒`
  return `${Math.floor(value / 60000)} 分 ${Math.round((value % 60000) / 1000)} 秒`
}

function totals(parts: StudioContextPart[]) {
  return Object.fromEntries(
    categories.map(({ id }) => [
      id,
      parts.filter((part) => part.category === id).reduce((sum, part) => sum + part.estimated_tokens, 0),
    ])
  ) as Record<Category, number>
}

function total(parts: StudioContextPart[]) {
  return parts.reduce((sum, part) => sum + part.estimated_tokens, 0)
}

type Segment = { label: string; amount: number; color: string }

function SegmentBar({ segments, totalAmount, unit, label }: {
  segments: Segment[]
  totalAmount: number
  unit: string
  label: string
}) {
  return (
    <div className='flex h-4 overflow-hidden rounded-sm bg-muted' aria-label={label}>
      {segments.filter((segment) => segment.amount > 0).map((segment) => (
        <Tooltip key={segment.label}>
          <TooltipTrigger
            className={cn('h-full min-w-0 hover:opacity-80 focus-visible:relative focus-visible:z-10 focus-visible:outline-2 focus-visible:outline-ring', segment.color)}
            style={{ width: `${segment.amount / Math.max(1, totalAmount) * 100}%` }}
            aria-label={`${segment.label} ${segment.amount.toLocaleString('zh-CN')} ${unit}`}
          />
          <TooltipContent className='border bg-popover text-sm text-popover-foreground shadow-md [&>svg]:bg-popover [&>svg]:fill-popover'>
            {segment.label} · {segment.amount.toLocaleString('zh-CN')} {unit} · {(segment.amount / Math.max(1, totalAmount) * 100).toFixed(1)}%
          </TooltipContent>
        </Tooltip>
      ))}
    </div>
  )
}

function Segments({ parts }: { parts: StudioContextPart[] }) {
  const values = totals(parts)
  const sum = total(parts)
  return (
    <div className='space-y-2'>
      <SegmentBar label='上下文组成' totalAmount={sum} unit='Token' segments={categories.map((category) => ({ label: category.name, amount: values[category.id], color: category.color }))} />
      <div className='flex flex-wrap gap-x-4 gap-y-1 text-sm text-muted-foreground'>
        {categories.filter((category) => values[category.id] > 0).map((category) => (
          <span key={category.id} className='flex items-center gap-1'>
            <span className={cn('size-2 rounded-[1px]', category.color)} />
            {category.name} {parts.filter((part) => part.category === category.id).length} 项 · {number(values[category.id])}
          </span>
        ))}
      </div>
    </div>
  )
}

export function StudioContext({ sessionId }: { sessionId: string }) {
  const [eventFilter, setEventFilter] = useState('')
  const overview = useQuery({
    queryKey: ['studio', 'context', sessionId],
    queryFn: () => getStudioSessionContext(sessionId),
    refetchInterval: 5000,
  })
  const requestsQuery = useInfiniteQuery({
    queryKey: ['studio', 'context-requests', sessionId],
    queryFn: ({ pageParam }) => listStudioContextRequests(sessionId, pageParam),
    initialPageParam: 0,
    getNextPageParam: (page) => page.has_more ? page.next_offset : undefined,
    refetchInterval: 5000,
  })
  const eventsQuery = useInfiniteQuery({
    queryKey: ['studio', 'context-events', sessionId, eventFilter],
    queryFn: ({ pageParam }) => listStudioContextEvents(sessionId, pageParam, eventFilter),
    initialPageParam: 0,
    getNextPageParam: (page) => page.has_more ? page.next_offset : undefined,
    refetchInterval: 5000,
  })
  const requests = useMemo(
    () => requestsQuery.data?.pages.flatMap((page) => page.requests) ?? [],
    [requestsQuery.data]
  )
  const events = useMemo(
    () => eventsQuery.data?.pages.flatMap((page) => page.events) ?? [],
    [eventsQuery.data]
  )
  const [selectedId, setSelectedId] = useState('current')
  const [search, setSearch] = useState('')
  const [category, setCategory] = useState('all')
  const [sortBy, setSortBy] = useState('order')
  const [trendGroup, setTrendGroup] = useState<'step' | 'turn'>('step')
  const [trendValue, setTrendValue] = useState<'total' | 'delta'>('total')
  const currentId = overview.data?.current?.attempt_id
  const attemptId = selectedId === 'current' ? currentId : selectedId
  const browser = useQuery({
    queryKey: ['studio', 'context-request', sessionId, selectedId, attemptId],
    queryFn: () => selectedId === 'current' ? getStudioCurrentContext(sessionId) : getStudioContextRequest(sessionId, attemptId!),
    enabled: Boolean(attemptId),
    refetchInterval: selectedId === 'current' ? 5000 : false,
  })
  const index = requests.findIndex((request) => request.attempt_id === attemptId)
  const previousId = index >= 0 ? requests[index + 1]?.attempt_id : undefined
  const previous = useQuery({
    queryKey: ['studio', 'context-request', sessionId, previousId],
    queryFn: () => getStudioContextRequest(sessionId, previousId!),
    enabled: Boolean(previousId),
  })
  const previousParts = new Map(previous.data?.parts.map((part) => [part.id, part.content]) ?? [])
  const currentPartIDs = new Set(browser.data?.parts.map((part) => part.id) ?? [])
  const removedParts = previous.data?.parts.filter((part) => !currentPartIDs.has(part.id)) ?? []
  const visibleParts = useMemo(() => {
    const items = browser.data?.parts.filter((part) =>
      (category === 'all' || part.category === category) &&
      `${part.label} ${part.source} ${part.content ?? ''}`.toLocaleLowerCase().includes(search.toLocaleLowerCase())
    ) ?? []
    if (sortBy === 'tokens') return [...items].sort((a, b) => b.estimated_tokens - a.estimated_tokens)
    if (sortBy === 'name') return [...items].sort((a, b) => a.label.localeCompare(b.label))
    return items
  }, [browser.data?.parts, category, search, sortBy])

  if (overview.isLoading) return <div className='p-5 text-sm text-muted-foreground'>上下文读取中…</div>
  if (overview.isError || !overview.data) return <div className='p-5 text-sm text-destructive'>上下文读取失败</div>

  const view = overview.data
  const current = view.current
  const focusedRequest = requests.find((request) => request.attempt_id === attemptId)
  const occupancy = current?.projected_tokens !== undefined && current.context_window_tokens > 0
    ? Math.round(current.projected_tokens / current.context_window_tokens * 100)
    : null
  const tokenTotal = view.tokens.input + view.tokens.output
  const timing = view.timing

  return (
    <ScrollArea className='min-h-0 flex-1 bg-card text-sm' data-slot='studio-context'>
      <div className='mx-auto flex max-w-360 flex-col gap-3 p-3 sm:p-4'>
        <div className='grid gap-3 xl:grid-cols-3'>
          <Card className='gap-3 py-4'>
            <CardHeader className='px-4'><CardTitle className='text-base'>上下文统计</CardTitle></CardHeader>
            <CardContent className='grid grid-cols-3 gap-2 px-4'>
              {[
                ['轮次', view.turns], ['步骤', view.steps], ['工具调用', view.tool_calls],
                ['注入', view.injections], ['压缩', view.compactions], ['剪枝', view.prunes],
              ].map(([label, value]) => (
                <div key={label} className='rounded-md border bg-muted/30 p-2'>
                  <div className='text-muted-foreground'>{label}</div>
                  <div className='mt-1 text-base font-semibold tabular-nums'>{number(Number(value))}</div>
                </div>
              ))}
            </CardContent>
          </Card>
          <Card className='gap-3 py-4'>
            <CardHeader className='px-4'><CardTitle className='text-base'>Token 统计</CardTitle></CardHeader>
            <CardContent className='space-y-2 px-4'>
              <div className='text-xl font-semibold tabular-nums'>{number(tokenTotal)}</div>
              <SegmentBar label='Token 用量组成' totalAmount={tokenTotal} unit='Token' segments={[
                { amount: view.tokens.cache_read, color: 'bg-chart-1', label: '缓存读取' },
                { amount: view.tokens.uncached, color: 'bg-chart-2', label: '未缓存输入' },
                { amount: view.tokens.cache_write, color: 'bg-chart-3', label: '缓存写入' },
                { amount: view.tokens.unclassified_input, color: 'bg-chart-5', label: '缓存状态未知' },
                { amount: view.tokens.output, color: 'bg-chart-4', label: '输出' },
              ]} />
              <div className='grid grid-cols-2 gap-1 text-muted-foreground'>
                <span>缓存读取 {number(view.tokens.cache_read)}</span>
                <span>未缓存输入 {number(view.tokens.uncached)}</span>
                <span>缓存写入 {number(view.tokens.cache_write)}</span>
                {view.tokens.unclassified_input > 0 ? <span>缓存状态未知 {number(view.tokens.unclassified_input)}</span> : null}
                <span>输出 {number(view.tokens.output)}</span>
                <span>推理 {number(view.tokens.reasoning)}</span>
                <span>缓存命中 {view.tokens.cache_known_input > 0 ? `${(view.tokens.cache_read / view.tokens.cache_known_input * 100).toFixed(1)}%` : '未记录'}</span>
              </div>
              {view.tokens.missing_requests > 0 ? <div className='text-muted-foreground'>{view.tokens.missing_requests} 次请求未返回用量</div> : null}
            </CardContent>
          </Card>
          <Card className='gap-3 py-4'>
            <CardHeader className='px-4'><CardTitle className='text-base'>耗时统计</CardTitle></CardHeader>
            <CardContent className='space-y-2 px-4'>
              <div className='text-xl font-semibold tabular-nums'>{duration(timing.active_ms)}</div>
              <SegmentBar label='活跃耗时组成' totalAmount={timing.active_ms} unit='毫秒' segments={[
                { amount: timing.model_wait_ms, color: 'bg-chart-1', label: '模型等待' },
                { amount: timing.generation_ms, color: 'bg-chart-2', label: '模型生成' },
                { amount: timing.model_other_ms, color: 'bg-chart-3', label: '模型未分类' },
                { amount: timing.tools_ms, color: 'bg-chart-4', label: '工具执行' },
                { amount: timing.overlap_ms, color: 'bg-primary', label: '并行执行' },
                { amount: timing.other_ms, color: 'bg-chart-5', label: '其他开销' },
              ]} />
              <div className='grid grid-cols-2 gap-1 text-muted-foreground'>
                <span>模型等待 {duration(timing.model_wait_ms)} · {Math.round(timing.model_wait_ms / Math.max(1, timing.active_ms) * 100)}%</span>
                <span>模型生成 {duration(timing.generation_ms)} · {Math.round(timing.generation_ms / Math.max(1, timing.active_ms) * 100)}%</span>
                <span>模型未分类 {duration(timing.model_other_ms)} · {Math.round(timing.model_other_ms / Math.max(1, timing.active_ms) * 100)}%</span>
                <span>工具执行 {duration(timing.tools_ms)} · {Math.round(timing.tools_ms / Math.max(1, timing.active_ms) * 100)}%</span>
                {timing.overlap_ms > 0 ? <span>并行执行 {duration(timing.overlap_ms)} · {Math.round(timing.overlap_ms / Math.max(1, timing.active_ms) * 100)}%</span> : null}
                <span>其他开销 {duration(timing.other_ms)} · {Math.round(timing.other_ms / Math.max(1, timing.active_ms) * 100)}%</span>
              </div>
            </CardContent>
          </Card>
        </div>
        <Card className='gap-3 py-4'>
          <CardHeader className='flex flex-row items-center justify-between px-4'>
            <CardTitle className='text-base'>当前上下文</CardTitle>
            <span className='text-muted-foreground'>{current?.model ?? '未记录'}</span>
          </CardHeader>
          <CardContent className='space-y-3 px-4'>
            <div className='flex items-baseline gap-2'>
              <strong className='text-xl tabular-nums'>{number(current?.projected_tokens)}</strong>
              <span className='text-muted-foreground'>/ {number(current?.context_window_tokens)} tokens</span>
              <strong className='ml-auto text-base'>{occupancy === null ? '未记录' : `${occupancy}%`}</strong>
            </div>
            {current ? <Segments parts={current.parts ?? []} /> : <div className='text-muted-foreground'>暂无模型请求</div>}
            {current ? <div className='text-muted-foreground'>下一次请求预计 · 上次实际输入 {number(current.input_tokens)} Token</div> : null}
          </CardContent>
        </Card>
        <div className='grid gap-3 lg:grid-cols-2'>
          <Card className='min-w-0 gap-3 py-4'>
            <CardHeader className='flex flex-row flex-wrap items-center justify-between gap-2 px-4'>
              <CardTitle className='text-base'>上下文趋势</CardTitle>
              <div className='flex gap-1'>
                {(['step', 'turn'] as const).map((value) => <Button key={value} size='sm' aria-pressed={trendGroup === value} variant={trendGroup === value ? 'secondary' : 'ghost'} onClick={() => setTrendGroup(value)}>{value === 'step' ? '步骤' : '轮次'}</Button>)}
                {(['total', 'delta'] as const).map((value) => <Button key={value} size='sm' aria-pressed={trendValue === value} variant={trendValue === value ? 'secondary' : 'ghost'} onClick={() => setTrendValue(value)}>{value === 'total' ? '全量' : '增量'}</Button>)}
              </div>
            </CardHeader>
            <CardContent className='min-w-0 space-y-3 px-4'>
              <Trend requests={requests} events={events} group={trendGroup} value={trendValue} selectedId={attemptId} onSelect={setSelectedId} />
              {focusedRequest ? <div className='space-y-1 rounded-md border bg-muted/30 p-2'>
                <div className='font-medium'>第 {focusedRequest.turn_number} 轮 · 第 {focusedRequest.step_number} 步 · {new Date(focusedRequest.started_at).toLocaleTimeString('zh-CN')}</div>
                {focusedRequest.preview ? <div className='truncate text-muted-foreground'>{focusedRequest.preview}</div> : null}
                <div className='flex flex-wrap gap-x-3 gap-y-1 text-muted-foreground'>
                  <span>估算 {number(total(focusedRequest.parts ?? []))}</span>
                  <span>实际输入 {number(focusedRequest.input_tokens)}</span>
                  <span>输出 {number(focusedRequest.output_tokens)}</span>
                  <span>缓存读取 {number(focusedRequest.cache_read_tokens)}</span>
                </div>
              </div> : null}
              {requestsQuery.hasNextPage ? <Button size='sm' variant='outline' onClick={() => void requestsQuery.fetchNextPage()}>加载更早的请求</Button> : null}
            </CardContent>
          </Card>
          <Card className='min-w-0 gap-3 py-4'>
            <CardHeader className='flex flex-row flex-wrap items-center justify-between gap-2 px-4'>
              <CardTitle className='text-base'>上下文浏览器</CardTitle>
              <Select value={selectedId} onValueChange={setSelectedId}>
                <SelectTrigger className='w-44' aria-label='选择上下文请求'><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value='current'>当前</SelectItem>
                  {requests.map((request) => <SelectItem key={request.attempt_id} value={request.attempt_id}>{new Date(request.started_at).toLocaleTimeString('zh-CN')} · {request.model}</SelectItem>)}
                </SelectContent>
              </Select>
            </CardHeader>
            <CardContent className='space-y-3 px-4'>
              <div className='flex flex-wrap gap-2'>
                <Input className='min-w-32 flex-1' aria-label='搜索上下文内容' placeholder='搜索内容' value={search} onChange={(event) => setSearch(event.target.value)} />
                <Select value={category} onValueChange={setCategory}>
                  <SelectTrigger className='w-32' aria-label='上下文类别'><SelectValue /></SelectTrigger>
                  <SelectContent><SelectItem value='all'>全部类别</SelectItem>{categories.map((item) => <SelectItem key={item.id} value={item.id}>{item.name}</SelectItem>)}</SelectContent>
                </Select>
                <Select value={sortBy} onValueChange={setSortBy}>
                  <SelectTrigger className='w-24' aria-label='上下文排序'><SelectValue /></SelectTrigger>
                  <SelectContent><SelectItem value='order'>顺序</SelectItem><SelectItem value='tokens'>Token</SelectItem><SelectItem value='name'>名称</SelectItem></SelectContent>
                </Select>
              </div>
              {browser.isLoading ? <div className='text-muted-foreground'>内容读取中…</div> : browser.isError ? <div className='text-destructive'>内容读取失败</div> : browser.data ? <>
                <Segments parts={browser.data.parts} />
                <div className='text-muted-foreground'>共 {browser.data.parts.length} 项 · 估算 {number(total(browser.data.parts))} Token{previous.data ? ` · 新增或变化 ${browser.data.parts.filter((part) => previousParts.get(part.id) !== part.content).length} 项 · 移除 ${removedParts.length} 项` : ''}</div>
                <div className='space-y-1'>
                  {visibleParts.map((part) => <details key={part.id} className='rounded-md border bg-card'>
                    <summary className='flex cursor-pointer items-center gap-2 px-2 py-2'>
                      <span className={cn('size-2 shrink-0 rounded-[1px]', categories.find((item) => item.id === part.category)?.color ?? 'bg-muted')} />
                      <span className='truncate'>{part.label || categories.find((item) => item.id === part.category)?.name || part.category}</span>
                      {previous.data && previousParts.get(part.id) !== part.content ? <span className='text-primary'>+变更</span> : null}
                      <span className='ml-auto shrink-0 tabular-nums text-muted-foreground'>{number(part.estimated_tokens)}</span>
                    </summary>
                    <pre className='max-h-64 overflow-auto border-t bg-background p-2 font-mono text-sm break-all whitespace-pre-wrap'>{part.content}</pre>
                  </details>)}
                  {removedParts.map((part) => <details key={`removed-${part.id}`} className='rounded-md border bg-muted/30'>
                    <summary className='cursor-pointer px-2 py-2 text-muted-foreground'>− 已移除 · {part.label || part.category} · {number(part.estimated_tokens)}</summary>
                    <pre className='max-h-64 overflow-auto border-t bg-background p-2 font-mono text-sm break-all whitespace-pre-wrap'>{part.content}</pre>
                  </details>)}
                </div>
              </> : <div className='text-muted-foreground'>暂无上下文</div>}
            </CardContent>
          </Card>
        </div>
        <Card className='gap-3 py-4'>
          <CardHeader className='flex flex-row flex-wrap items-center justify-between gap-2 px-4'>
            <CardTitle className='text-base'>上下文事件</CardTitle>
            <div className='flex flex-wrap gap-1'>
              {([
                ['', '全部'], ['CONTEXT_INJECTED', '注入'], ['CONTEXT_COMPACTED', '压缩'],
                ['CONTEXT_PRUNED', '剪枝'], ['MODEL_SWITCHED', '模型'], ['MODE_SWITCHED', '模式'],
              ] as const).map(([kind, label]) => <Button key={kind} size='sm' aria-pressed={eventFilter === kind} variant={eventFilter === kind ? 'secondary' : 'ghost'} onClick={() => setEventFilter(kind)}>{label}</Button>)}
            </div>
          </CardHeader>
          <CardContent className='space-y-1 px-4'>
            {events.map((event) => <div key={event.id} className='flex flex-wrap items-center gap-2 border-b py-2 last:border-0'>
              <span className='rounded-sm bg-muted px-1.5 py-0.5'>{event.kind === 'CONTEXT_INJECTED' ? '注入' : event.kind === 'CONTEXT_PRUNED' ? '剪枝' : event.kind === 'MODEL_SWITCHED' ? '模型切换' : event.kind === 'MODE_SWITCHED' ? '模式切换' : '压缩'}</span>
              <span className='min-w-0 flex-1 truncate'>{event.detail || event.source}</span>
              {event.turn_number > 0 ? <span className='text-muted-foreground'>第 {event.turn_number} 轮 · 第 {event.step_number} 步</span> : null}
              {event.delta_tokens_estimated !== null ? <span className={event.delta_tokens_estimated < 0 ? 'text-chart-3' : 'text-primary'}>{event.delta_tokens_estimated > 0 ? '+' : ''}{number(event.delta_tokens_estimated)}</span> : null}
              <time className='text-muted-foreground'>{new Date(event.created_at).toLocaleTimeString('zh-CN')}</time>
            </div>)}
            {events.length === 0 ? <div className='text-muted-foreground'>暂无上下文事件</div> : null}
            {eventsQuery.hasNextPage ? <Button size='sm' variant='outline' onClick={() => void eventsQuery.fetchNextPage()}>加载更早的事件</Button> : null}
          </CardContent>
        </Card>
      </div>
    </ScrollArea>
  )
}

function Trend({
  requests,
  events,
  group,
  value,
  selectedId,
  onSelect,
}: {
  requests: StudioContextRequest[]
  events: StudioContextEvent[]
  group: 'step' | 'turn'
  value: 'total' | 'delta'
  selectedId?: string
  onSelect: (attemptId: string) => void
}) {
  const ascending = [
    ...requests.filter((request) => request.purpose === 'agent'),
  ].reverse()
  const displayed =
    group === 'turn'
      ? ascending.filter(
          (request, index) =>
            index === ascending.length - 1 ||
            request.turn_id !== ascending[index + 1].turn_id
        )
      : ascending
  const points = displayed.map((request, index) => {
    const estimated = total(request.parts ?? [])
    const previous = total(displayed[index - 1]?.parts ?? [])
    const startedAt = Date.parse(request.started_at)
    const previousAt =
      index > 0 ? Date.parse(displayed[index - 1].started_at) : startedAt
    return {
      index: String(index + 1),
      attemptId: request.attempt_id,
      turn: request.turn_number,
      step: request.step_number,
      model: request.model,
      estimated,
      delta: estimated - previous,
      events:
        index > 0
          ? events.filter((event) => {
              const at = Date.parse(event.created_at)
              return at > previousAt && at <= startedAt
            })
          : [],
      ...totals(request.parts ?? []),
    }
  })
  const selectedPoint = points.find((point) => point.attemptId === selectedId)
  const deltaDomain = [
    Math.min(0, ...points.map((point) => point.delta)),
    Math.max(1, ...points.map((point) => point.delta)),
  ]
  if (displayed.length === 0)
    return (
      <div className='flex h-44 items-center justify-center text-muted-foreground'>
        暂无请求数据
      </div>
    )
  return (
    <div className='overflow-x-auto' aria-label='上下文趋势图'>
      <ChartContainer
        config={trendConfig}
        className='aspect-auto h-64 min-w-full text-sm'
        style={{ width: Math.max(480, points.length * 48) }}
      >
        <BarChart
          accessibilityLayer
          data={points}
          margin={{ top: 20, right: 12, bottom: 4, left: 4 }}
        >
          <CartesianGrid vertical={false} />
          <XAxis
            dataKey='index'
            tickLine={false}
            axisLine={false}
            tickMargin={8}
            height={28}
            tick={{ fontSize: 14 }}
            tickFormatter={(index) => {
              const point = points[Number(index) - 1]
              return point
                ? `${group === 'turn' ? point.turn : `${point.turn}.${point.step}`}${point.events.length > 0 ? ' ×' : ''}`
                : ''
            }}
          />
          <YAxis
            domain={value === 'delta' ? deltaDomain : [0, 'auto']}
            tickLine={false}
            axisLine={false}
            tickMargin={8}
            width={52}
            tick={{ fontSize: 14 }}
            tickFormatter={number}
          />
          <ChartTooltip
            content={
              <ChartTooltipContent
                className='bg-popover text-sm text-popover-foreground shadow-md'
                labelFormatter={(_, payload) => {
                  const point = payload[0]?.payload as
                    | (typeof points)[number]
                    | undefined
                  return point
                    ? `第 ${point.turn} 轮 · 第 ${point.step} 步 · ${point.model} · 估算 ${number(point.estimated)} Token${point.events.length > 0 ? ` · ${point.events.length} 个上下文事件` : ''}`
                    : ''
                }}
              />
            }
          />
          {points
            .filter((point) => point.events.length > 0)
            .map((point) => (
              <ReferenceLine
                key={point.index}
                x={point.index}
                stroke='var(--primary)'
                strokeDasharray='2 4'
                label={{
                  value: '×',
                  position: 'top',
                  fill: 'var(--primary)',
                  fontSize: 14,
                }}
              />
            ))}
          {selectedPoint ? (
            <ReferenceLine x={selectedPoint.index} stroke='var(--foreground)' strokeWidth={2} />
          ) : null}
          {value === 'total' ? (
            categories.map((category) => (
              <Bar
                key={category.id}
                dataKey={category.id}
                stackId='context'
                fill={`var(--color-${category.id})`}
                onClick={(entry) => onSelect(entry.payload.attemptId)}
              />
            ))
          ) : (
            <>
              <ReferenceLine y={0} stroke='var(--border)' />
              <Bar
                dataKey='delta'
                onClick={(entry) => onSelect(entry.payload.attemptId)}
              >
                {points.map((point) => (
                  <Cell
                    key={point.attemptId}
                    fill={point.delta < 0 ? 'var(--destructive)' : 'var(--primary)'}
                  />
                ))}
              </Bar>
            </>
          )}
        </BarChart>
      </ChartContainer>
      <div className='pt-2 text-sm text-muted-foreground'>
        {group === 'turn' ? '轮次' : '步骤'} · {value === 'total' ? '全量' : '增量'} · × 上下文事件
      </div>
    </div>
  )
}
