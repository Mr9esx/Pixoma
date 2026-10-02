import { useMemo, useState } from 'react'
import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { ArrowDownWideNarrow } from 'lucide-react'
import { Bar, BarChart, Cell, ReferenceDot, ReferenceLine, XAxis, YAxis } from 'recharts'
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
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/accordion'
import { DropdownMenu, DropdownMenuContent, DropdownMenuLabel, DropdownMenuRadioGroup, DropdownMenuRadioItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { IconButtonTooltip } from '@/components/ui/icon-button-tooltip'
import { FilterSegment } from '@/components/filters/filter-segment'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ChartContainer, ChartTooltip, type ChartConfig } from '@/components/ui/chart'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectGroup,
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
    <div className='flex h-1.5 overflow-hidden rounded-sm bg-muted' aria-label={label}>
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
      <div className='flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground'>
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
  const [selectedPartId, setSelectedPartId] = useState<string | null>(null)
  const agentRequests = useMemo(() => requests.filter((request) => request.purpose === 'agent'), [requests])
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
  const index = agentRequests.findIndex((request) => request.attempt_id === attemptId)
  const previousId = index >= 0 ? agentRequests[index + 1]?.attempt_id : undefined
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
  const focusedRequest = agentRequests.find((request) => request.attempt_id === attemptId)
  const occupancy = current?.projected_tokens !== undefined && current.context_window_tokens > 0
    ? Math.round(current.projected_tokens / current.context_window_tokens * 100)
    : null
  const tokenTotal = view.tokens.input + view.tokens.output
  const timing = view.timing
  const sortLabel = sortBy === 'order' ? '原始顺序' : sortBy === 'tokens' ? 'Token 从多到少' : '名称排序'
  const allBrowserParts = [...visibleParts.map((part) => ({ part, removed: false })), ...removedParts.map((part) => ({ part, removed: true }))]

  return (
    <ScrollArea className='min-h-0 flex-1 bg-card text-sm' data-slot='studio-context'>
      <div className='@container/context mx-auto flex w-full max-w-[1440px] flex-col gap-5 px-4 py-4 sm:px-5'>
        <section className='flex flex-col gap-2'>
          <h2 className='text-sm font-semibold'>当前上下文</h2>
          <Card className='min-w-0 gap-0 overflow-hidden py-0'>
            <CardHeader className='flex flex-row flex-wrap items-baseline justify-between gap-2 border-b px-4 py-3 pb-3!'>
              <CardTitle className='min-w-0 text-sm'>{current?.model ?? '暂无模型请求'}</CardTitle>
              <span className='font-mono text-xs tabular-nums text-muted-foreground'>{number(current?.projected_tokens)} / {number(current?.context_window_tokens)} Token</span>
            </CardHeader>
            <CardContent className='flex flex-col gap-3 px-4 py-4'>
              <div className='flex items-baseline justify-between gap-3'><span className='text-muted-foreground'>窗口占用</span><strong className='font-mono text-lg tabular-nums'>{occupancy === null ? '未记录' : `${occupancy}%`}</strong></div>
              <div role='meter' aria-label='上下文窗口占用' aria-valuemin={0} aria-valuemax={100} aria-valuenow={occupancy === null ? undefined : Math.min(100, Math.max(0, occupancy))} aria-valuetext={occupancy === null ? '未记录' : undefined} className='h-1 overflow-hidden rounded-sm bg-muted'>
                <div className='h-full bg-primary' style={{ width: `${Math.min(100, Math.max(0, occupancy ?? 0))}%` }} />
              </div>
              {current ? <div className='flex flex-col gap-3 border-t pt-3'><span className='font-mono text-xs tabular-nums text-muted-foreground'>上次实际输入 {number(current.input_tokens)} Token</span><Segments parts={current.parts ?? []} /></div> : null}
            </CardContent>
            <dl className='grid grid-cols-2 gap-px border-t bg-border @min-[36rem]/context:grid-cols-3 @min-[64rem]/context:grid-cols-6'>
              {[
                ['轮次', view.turns], ['步骤', view.steps], ['工具调用', view.tool_calls],
                ['注入', view.injections], ['压缩', view.compactions], ['剪枝', view.prunes],
              ].map(([label, value]) => (
                <div key={label} className='flex min-w-0 flex-col gap-1 bg-card px-4 py-3'>
                  <dt className='text-xs text-muted-foreground'>{label}</dt>
                  <dd className='font-mono text-lg font-semibold tabular-nums'>{number(Number(value))}</dd>
                </div>
              ))}
            </dl>
          </Card>
        </section>
        <section className='flex flex-col gap-2'>
          <h2 className='text-sm font-semibold'>会话用量</h2>
          <div className='grid gap-3 @min-[48rem]/context:grid-cols-2'>
          <Card className='gap-3 py-4'>
            <CardHeader className='px-4'><CardTitle className='text-sm'>Token 用量</CardTitle></CardHeader>
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
            <CardHeader className='px-4'><CardTitle className='text-sm'>活跃耗时</CardTitle></CardHeader>
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
        </section>
        <section className='flex flex-col gap-2'>
          <h2 className='text-sm font-semibold'>上下文浏览器</h2>
          <Card className='min-w-0 gap-0 overflow-hidden py-0'>
            <div className='grid min-w-0 @min-[56rem]/context:grid-cols-2'>
            <div className='flex min-w-0 flex-col gap-3 border-b px-4 py-4 @min-[56rem]/context:border-r @min-[56rem]/context:border-b-0'>
            <div className='flex flex-row flex-wrap items-center justify-between gap-2'>
              <h3 className='text-sm font-medium'>上下文趋势</h3>
              <div className='flex gap-1'>
                {(['step', 'turn'] as const).map((value) => <Button key={value} size='sm' aria-pressed={trendGroup === value} variant={trendGroup === value ? 'secondary' : 'ghost'} onClick={() => setTrendGroup(value)}>{value === 'step' ? '步骤' : '轮次'}</Button>)}
                {(['total', 'delta'] as const).map((value) => <Button key={value} size='sm' aria-pressed={trendValue === value} variant={trendValue === value ? 'secondary' : 'ghost'} onClick={() => setTrendValue(value)}>{value === 'total' ? '全量' : '增量'}</Button>)}
              </div>
            </div>
            <div className='min-w-0 space-y-3'>
              <Trend requests={agentRequests} events={events} group={trendGroup} value={trendValue} selectedId={attemptId} onSelect={setSelectedId} />
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
            </div>
            </div>
            <div className='flex min-w-0 flex-col gap-3 px-4 py-4'>
              <div className='grid grid-cols-[minmax(0,10rem)_minmax(0,1fr)_auto] items-center gap-2'>
              <Select value={selectedId} onValueChange={setSelectedId}>
                <SelectTrigger className='w-full' aria-label='选择上下文请求'><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                  <SelectItem value='current'>当前</SelectItem>
                  {agentRequests.map((request) => <SelectItem key={request.attempt_id} value={request.attempt_id}>{new Date(request.started_at).toLocaleTimeString('zh-CN')} · {request.model}</SelectItem>)}
                  </SelectGroup>
                </SelectContent>
              </Select>
                <Input className='min-w-32 flex-1' aria-label='搜索上下文内容' placeholder='搜索内容' value={search} onChange={(event) => setSearch(event.target.value)} />
                <DropdownMenu>
                  <IconButtonTooltip label={`排序：${sortLabel}`}><DropdownMenuTrigger asChild><Button variant='ghost' size='icon' aria-label={`排序上下文：${sortLabel}`}><ArrowDownWideNarrow /></Button></DropdownMenuTrigger></IconButtonTooltip>
                  <DropdownMenuContent align='end'><DropdownMenuLabel>排序方式</DropdownMenuLabel><DropdownMenuRadioGroup value={sortBy} onValueChange={setSortBy}><DropdownMenuRadioItem value='order'>原始顺序</DropdownMenuRadioItem><DropdownMenuRadioItem value='tokens'>Token 从多到少</DropdownMenuRadioItem><DropdownMenuRadioItem value='name'>名称排序</DropdownMenuRadioItem></DropdownMenuRadioGroup></DropdownMenuContent>
                </DropdownMenu>
              </div>
              <FilterSegment aria-label='上下文类别' value={category} options={[{ value: 'all', label: '全部' }, ...categories.map((item) => ({ value: item.id, label: item.name }))]} onValueChange={setCategory} />
              {browser.isLoading ? <div className='text-muted-foreground'>内容读取中…</div> : browser.isError ? <div className='text-destructive'>内容读取失败</div> : browser.data ? <>
                <Segments parts={browser.data.parts} />
                <div className='text-muted-foreground'>共 {browser.data.parts.length} 项 · 估算 {number(total(browser.data.parts))} Token{previous.data ? ` · 新增或变化 ${browser.data.parts.filter((part) => previousParts.get(part.id) !== part.content).length} 项 · 移除 ${removedParts.length} 项` : ''}</div>
                {browser.data.content_available === false ? <div className='text-xs text-muted-foreground'>此请求保留了上下文统计，原始内容未保存。</div> : null}
                <Accordion type='single' collapsible value={selectedPartId ?? ''} onValueChange={(value) => setSelectedPartId(value || null)} className='flex min-w-0 max-h-96 flex-col gap-1 overflow-y-auto' role='list' aria-label='上下文内容项'>
                  {allBrowserParts.map(({ part, removed }) => {
                    const itemCategory = categories.find((item) => item.id === part.category)
                    const label = part.label || itemCategory?.name || part.category
                    return <AccordionItem key={(removed ? 'removed:' : '') + part.id} value={(removed ? 'removed:' : '') + part.id} role='listitem' className='rounded-md border-b-0 data-[state=open]:bg-muted'>
                      <AccordionTrigger className='min-h-11 w-full min-w-0 items-center gap-2 px-2 py-2 font-normal hover:bg-muted hover:no-underline'>
                        <span className={cn('size-2 shrink-0 rounded-[1px]', itemCategory?.color ?? 'bg-muted-foreground')} />
                        <span className='min-w-0 flex-1'><span className='block truncate'>{label}</span>{label !== (itemCategory?.name ?? part.category) ? <span className='block truncate text-xs text-muted-foreground'>{itemCategory?.name ?? part.category}</span> : null}</span>
                        {removed ? <span className='shrink-0 text-xs text-muted-foreground'>已移除</span> : previous.data && previousParts.get(part.id) !== part.content ? <span className='shrink-0 text-xs text-muted-foreground'>有变化</span> : null}
                        <span className='shrink-0 font-mono text-xs tabular-nums text-muted-foreground'>{number(part.estimated_tokens)} Token</span>
                      </AccordionTrigger>
                      {browser.data!.content_available !== false ? <AccordionContent className='px-6 pb-3'><pre className='m-0 max-h-36 overflow-auto font-mono text-sm leading-5 break-all whitespace-pre-wrap'>{part.content ?? ''}</pre></AccordionContent> : null}
                    </AccordionItem>
                  })}
                  {allBrowserParts.length === 0 ? <div className='py-6 text-center text-muted-foreground'>暂无上下文内容</div> : null}
                </Accordion>
              </> : <div className='text-muted-foreground'>暂无上下文</div>}
            </div>
            </div>
          </Card>
        </section>
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
      turnId: request.turn_id,
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
  const selectedRequest = requests.find((request) => request.attempt_id === selectedId)
  const selectedPoint = points.find((point) => point.attemptId === selectedId) ??
    (group === 'turn' && selectedRequest
      ? points.find((point) => point.turnId === selectedRequest.turn_id)
      : undefined)
  const deltaDomain = [
    Math.min(0, ...points.map((point) => point.delta)),
    Math.max(1, ...points.map((point) => point.delta)),
  ]
  if (displayed.length === 0)
    return (
      <div className='flex h-44 items-center justify-center text-muted-foreground @min-[56rem]/context:h-auto @min-[56rem]/context:flex-1'>
        暂无请求数据
      </div>
    )
  return (
    <div className='min-h-0 min-w-0 @min-[56rem]/context:flex-1' aria-label='上下文趋势图'>
      <ChartContainer
        config={trendConfig}
        className='aspect-auto h-56 w-full text-xs @min-[56rem]/context:h-full'
      >
        <BarChart
          accessibilityLayer
          data={points}
          maxBarSize={32}
          margin={{ top: 20, right: 12, bottom: 4, left: 4 }}
        >
          <XAxis
            dataKey='index'
            tickLine={false}
            axisLine={false}
            tickMargin={8}
            height={28}
            minTickGap={16}
            tick={{ fontSize: 12 }}
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
            tick={{ fontSize: 12 }}
            tickFormatter={number}
          />
          <ChartTooltip
            cursor={false}
            shared={false}
            content={({ active, payload }) => {
              if (!active || !payload?.length) return null
              const item = payload[0]
              const point = item.payload as (typeof points)[number]
              const label = item.dataKey === 'delta' ? '变化' : categories.find((category) => category.id === item.dataKey)?.name ?? ''
              return <div className='rounded-md border bg-popover px-2 py-1.5 text-xs text-popover-foreground shadow-md'>
                <div>第 {point.turn} 轮 · 第 {point.step} 步</div>
                <div className='font-mono tabular-nums'>{label} {number(Number(item.value))} Token</div>
              </div>
            }}
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
          {value === 'total' ? (
            categories.map((category) => (
              <Bar
                key={category.id}
                dataKey={category.id}
                stackId='context'
                fill={`var(--color-${category.id})`}
                activeBar={false}
                onClick={(entry) => onSelect(entry.payload.attemptId)}
              >
                {points.map((point) => <Cell key={point.attemptId} opacity={selectedPoint && point.attemptId !== selectedPoint.attemptId ? 0.5 : 1} />)}
              </Bar>
            ))
          ) : (
            <>
              <ReferenceLine y={0} stroke='var(--border)' />
              <Bar
                dataKey='delta'
                activeBar={false}
                onClick={(entry) => onSelect(entry.payload.attemptId)}
              >
                {points.map((point) => (
                  <Cell
                    key={point.attemptId}
                    fill={point.delta < 0 ? 'var(--destructive)' : 'var(--primary)'}
                    opacity={selectedPoint && point.attemptId !== selectedPoint.attemptId ? 0.5 : 1}
                  />
                ))}
              </Bar>
            </>
          )}
          {selectedPoint ? <ReferenceDot x={selectedPoint.index} y={value === 'total' ? selectedPoint.estimated : selectedPoint.delta} r={4} fill='var(--foreground)' stroke='var(--card)' strokeWidth={2} ifOverflow='visible' /> : null}
        </BarChart>
      </ChartContainer>
    </div>
  )
}
