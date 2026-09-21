import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { CheckCircle2, CircleDotDashed, Clock3, XCircle } from 'lucide-react'
import { listStudioRunEvents, listStudioSessionRuns, type StudioRun } from '@/lib/api/studio'
import { ScrollArea } from '@/components/ui/scroll-area'

export function StudioTrace({ sessionId }: { sessionId: string }) {
  const runs = useQuery({ queryKey: ['studio', 'session-runs', sessionId], queryFn: () => listStudioSessionRuns(sessionId), refetchInterval: 1500 })
  const [selectedRunID, setSelectedRunID] = useState<string>()
  const runID = selectedRunID ?? runs.data?.[0]?.id
  const events = useQuery({ queryKey: ['studio', 'run-events', runID], queryFn: () => listStudioRunEvents(runID!), enabled: Boolean(runID), refetchInterval: 1500 })
  return (
    <div className='flex min-h-0 flex-1'>
      <ScrollArea className='w-52 shrink-0 border-e bg-muted/20'>
        <div className='p-3'>
          <p className='px-2 pb-2 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground'>执行记录</p>
          <div className='space-y-1'>
            {runs.data?.map((run) => <button key={run.id} type='button' onClick={() => setSelectedRunID(run.id)} className={`w-full rounded-lg border px-3 py-2.5 text-left transition-colors ${runID === run.id ? 'border-border bg-background' : 'border-transparent hover:bg-background/70'}`}><div className='flex items-center gap-2'><RunIcon run={run} /><span className='text-sm font-medium'>运行 {run.id.slice(0, 6)}</span></div><p className='mt-1 truncate text-xs text-muted-foreground'>{run.status}</p></button>)}
            {!runs.isLoading && runs.data?.length === 0 ? <p className='px-2 py-3 text-xs leading-5 text-muted-foreground'>当前对话还没有执行记录。</p> : null}
          </div>
        </div>
      </ScrollArea>
      <ScrollArea className='min-w-0 flex-1'>
        <div className='mx-auto max-w-2xl space-y-3 p-5'>
          <div><h2 className='text-sm font-semibold'>执行 Trace</h2><p className='mt-1 text-xs text-muted-foreground'>按事件顺序记录模型、工具、审批与资产产出。</p></div>
          {events.data?.map((event) => <article key={event.id} className='rounded-lg border bg-background p-3'><div className='flex items-center justify-between gap-3'><p className='text-sm font-medium'>{eventLabel(event.type)}</p><time className='shrink-0 text-xs text-muted-foreground'>{new Date(event.created_at).toLocaleTimeString()}</time></div><pre className='mt-2 overflow-x-auto whitespace-pre-wrap break-words text-xs leading-5 text-muted-foreground'>{JSON.stringify(event.payload, null, 2)}</pre></article>)}
          {events.isLoading ? <p className='text-sm text-muted-foreground'>正在读取 Trace…</p> : null}
          {!events.isLoading && runID && events.data?.length === 0 ? <p className='text-sm text-muted-foreground'>该运行尚未产生事件。</p> : null}
        </div>
      </ScrollArea>
    </div>
  )
}

function RunIcon({ run }: { run: StudioRun }) {
  if (run.status === 'succeeded') return <CheckCircle2 className='size-3.5 text-success' />
  if (run.status === 'failed' || run.status === 'cancelled') return <XCircle className='size-3.5 text-destructive' />
  if (run.status === 'waiting_approval') return <Clock3 className='size-3.5 text-muted-foreground' />
  return <CircleDotDashed className='size-3.5 text-muted-foreground' />
}

function eventLabel(type: string) {
  return ({ RUN_STARTED: '开始运行', TEXT_MESSAGE_START: '开始输出', TEXT_MESSAGE_CONTENT: '模型输出', TEXT_MESSAGE_END: '结束输出', TOOL_CALL_START: '调用工具', TOOL_CALL_END: '工具返回', ASSET_CREATED: '创建资产', FLOW_UPDATED: '更新资产路线', APPROVAL_REQUIRED: '请求批准', APPROVAL_RESOLVED: '批准已处理', RUN_FINISHED: '运行结束' } as Record<string, string>)[type] ?? type
}
