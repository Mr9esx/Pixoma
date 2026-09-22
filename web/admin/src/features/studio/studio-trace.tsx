import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { CheckCircle2, CircleDotDashed, Clock3, XCircle } from 'lucide-react'
import {
  listStudioRunEvents,
  listStudioSessionRuns,
  type StudioRun,
  type StudioRunEvent,
} from '@/lib/api/studio'
import { ScrollArea } from '@/components/ui/scroll-area'

export function StudioTrace({ sessionId }: { sessionId: string }) {
  const runs = useQuery({
    queryKey: ['studio', 'session-runs', sessionId],
    queryFn: () => listStudioSessionRuns(sessionId),
    refetchInterval: 1500,
  })
  const [selectedRunID, setSelectedRunID] = useState<string>()
  const runID = selectedRunID ?? runs.data?.[0]?.id
  const events = useQuery({
    queryKey: ['studio', 'run-events', runID],
    queryFn: () => listStudioRunEvents(runID!),
    enabled: Boolean(runID),
    refetchInterval: 1500,
  })
  const modelAttempts = useMemo(
    () => projectModelAttempts(events.data ?? []),
    [events.data]
  )
  const executionEvents = useMemo(
    () => (events.data ?? []).filter((event) => !isModelTraceEvent(event.type)),
    [events.data]
  )
  return (
    <div className='flex min-h-0 flex-1'>
      <ScrollArea className='w-52 shrink-0 border-e bg-muted/20'>
        <div className='p-3'>
          <p className='px-2 pb-2 text-[11px] font-semibold tracking-wide text-muted-foreground uppercase'>
            执行记录
          </p>
          <div className='space-y-1'>
            {runs.data?.map((run) => (
              <button
                key={run.id}
                type='button'
                onClick={() => setSelectedRunID(run.id)}
                className={`w-full rounded-lg border px-3 py-2.5 text-left transition-colors ${runID === run.id ? 'border-border bg-background' : 'border-transparent hover:bg-background/70'}`}
              >
                <div className='flex items-center gap-2'>
                  <RunIcon run={run} />
                  <span className='text-sm font-medium'>
                    运行 {run.id.slice(0, 6)}
                  </span>
                </div>
                <p className='mt-1 truncate text-xs text-muted-foreground'>
                  {run.status}
                </p>
              </button>
            ))}
            {!runs.isLoading && runs.data?.length === 0 ? (
              <p className='px-2 py-3 text-xs leading-5 text-muted-foreground'>
                当前对话还没有执行记录。
              </p>
            ) : null}
          </div>
        </div>
      </ScrollArea>
      <ScrollArea className='min-w-0 flex-1'>
        <div className='mx-auto max-w-2xl space-y-3 p-5'>
          <div>
            <h2 className='text-sm font-semibold'>执行 Trace</h2>
            <p className='mt-1 text-xs text-muted-foreground'>
              模型请求、工具执行与运行事件。
            </p>
          </div>
          {modelAttempts.map((attempt) => (
            <ModelAttemptCard key={attempt.id} attempt={attempt} />
          ))}
          {executionEvents.map((event) => (
            <article
              key={event.id}
              className='rounded-lg border bg-background p-3'
            >
              <div className='flex items-center justify-between gap-3'>
                <p className='text-sm font-medium'>{eventLabel(event.type)}</p>
                <time className='shrink-0 text-xs text-muted-foreground'>
                  {new Date(event.created_at).toLocaleTimeString()}
                </time>
              </div>
              <pre className='mt-2 overflow-x-auto text-xs leading-5 break-words whitespace-pre-wrap text-muted-foreground'>
                {JSON.stringify(event.payload, null, 2)}
              </pre>
            </article>
          ))}
          {events.isLoading ? (
            <p className='text-sm text-muted-foreground'>正在读取 Trace…</p>
          ) : null}
          {!events.isLoading && runID && events.data?.length === 0 ? (
            <p className='text-sm text-muted-foreground'>
              该运行尚未产生事件。
            </p>
          ) : null}
        </div>
      </ScrollArea>
    </div>
  )
}

type ModelAttempt = {
  id: string
  started?: StudioRunEvent
  firstToken?: StudioRunEvent
  finished?: StudioRunEvent
  failed?: StudioRunEvent
}

function ModelAttemptCard({ attempt }: { attempt: ModelAttempt }) {
  const source =
    attempt.started ?? attempt.firstToken ?? attempt.finished ?? attempt.failed
  const payload = source?.payload ?? {}
  const elapsed =
    attempt.finished?.payload.elapsed_ms ?? attempt.failed?.payload.elapsed_ms
  const firstToken = attempt.firstToken?.payload.elapsed_ms
  const requestBody = attempt.started?.payload.request_body
  const outcome = attempt.failed ? '失败' : attempt.finished ? '完成' : '进行中'
  return (
    <article className='rounded-lg border bg-background p-3'>
      <div className='flex items-center justify-between gap-3'>
        <div>
          <p className='text-sm font-medium'>
            模型请求 · {stringValue(payload.model) || '未命名模型'}
          </p>
          <p className='mt-1 text-xs text-muted-foreground'>
            {purposeLabel(stringValue(payload.purpose))} · {outcome}
          </p>
        </div>
        <time className='shrink-0 text-xs text-muted-foreground'>
          {source ? new Date(source.created_at).toLocaleTimeString() : ''}
        </time>
      </div>
      <dl className='mt-3 grid grid-cols-2 gap-x-4 gap-y-2 text-xs sm:grid-cols-4'>
        <TraceMetric
          label='首 Token'
          value={firstToken === undefined ? '—' : `${firstToken} ms`}
        />
        <TraceMetric
          label='耗时'
          value={elapsed === undefined ? '—' : `${elapsed} ms`}
        />
        <TraceMetric
          label='输入'
          value={tokenValue(attempt.finished?.payload.input_tokens)}
        />
        <TraceMetric
          label='输出'
          value={tokenValue(attempt.finished?.payload.output_tokens)}
        />
      </dl>
      {attempt.failed && stringValue(attempt.failed.payload.error) ? (
        <p className='mt-3 text-xs text-destructive'>
          {stringValue(attempt.failed.payload.error)}
        </p>
      ) : null}
      {requestBody ? (
        <details className='mt-3 border-t pt-3'>
          <summary className='cursor-pointer text-xs font-medium text-muted-foreground'>
            请求提示词与工具定义
          </summary>
          <pre className='mt-2 overflow-x-auto text-xs leading-5 break-words whitespace-pre-wrap text-muted-foreground'>
            {JSON.stringify(requestBody, null, 2)}
          </pre>
        </details>
      ) : null}
    </article>
  )
}

function TraceMetric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className='text-muted-foreground'>{label}</dt>
      <dd className='mt-0.5 font-medium'>{value}</dd>
    </div>
  )
}

function projectModelAttempts(events: StudioRunEvent[]): ModelAttempt[] {
  const attempts = new Map<string, ModelAttempt>()
  for (const event of events) {
    if (!isModelTraceEvent(event.type)) continue
    const attemptID = stringValue(event.payload.attempt_id) || event.id
    const attempt = attempts.get(attemptID) ?? { id: attemptID }
    if (event.type === 'MODEL_REQUEST_STARTED') attempt.started = event
    if (event.type === 'MODEL_FIRST_TOKEN') attempt.firstToken = event
    if (event.type === 'MODEL_REQUEST_FINISHED') attempt.finished = event
    if (event.type === 'MODEL_REQUEST_FAILED') attempt.failed = event
    attempts.set(attemptID, attempt)
  }
  return [...attempts.values()]
}

function isModelTraceEvent(type: string) {
  return (
    type === 'MODEL_REQUEST_STARTED' ||
    type === 'MODEL_FIRST_TOKEN' ||
    type === 'MODEL_REQUEST_FINISHED' ||
    type === 'MODEL_REQUEST_FAILED'
  )
}

function stringValue(value: unknown) {
  return typeof value === 'string' ? value : ''
}

function tokenValue(value: unknown) {
  return typeof value === 'number' && value > 0 ? value.toLocaleString() : '—'
}

function purposeLabel(value: string) {
  if (value === 'context_summary') return '上下文压缩'
  return 'Agent 推理'
}

function RunIcon({ run }: { run: StudioRun }) {
  if (run.status === 'succeeded')
    return <CheckCircle2 className='size-3.5 text-success' />
  if (run.status === 'failed' || run.status === 'cancelled')
    return <XCircle className='size-3.5 text-destructive' />
  if (run.status === 'waiting_approval')
    return <Clock3 className='size-3.5 text-muted-foreground' />
  return <CircleDotDashed className='size-3.5 text-muted-foreground' />
}

function eventLabel(type: string) {
  return (
    (
      {
        RUN_STARTED: '开始运行',
        TEXT_MESSAGE_START: '开始输出',
        TEXT_MESSAGE_CONTENT: '模型输出',
        TEXT_MESSAGE_END: '结束输出',
        REASONING_MESSAGE_START: '开始思考',
        REASONING_MESSAGE_CONTENT: '思考内容',
        REASONING_MESSAGE_END: '结束思考',
        TOOL_CALL_START: '调用工具',
        TOOL_CALL_ARGS: '工具参数',
        TOOL_CALL_RESULT: '工具结果',
        TOOL_CALL_END: '结束工具调用',
        ASSET_CREATED: '创建资产',
        FLOW_UPDATED: '更新资产路线',
        APPROVAL_REQUIRED: '请求批准',
        APPROVAL_RESOLVED: '批准已处理',
        RUN_FINISHED: '运行结束',
      } as Record<string, string>
    )[type] ?? type
  )
}
