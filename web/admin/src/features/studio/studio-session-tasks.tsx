import { ListTodo, Workflow } from 'lucide-react'
import type { StudioFlowNode, StudioWorkflowExecution } from '@/lib/api/studio'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'
import { isActiveWorkflowExecution, workflowExecutionTitle } from './studio-session-tasks-data'

function executionStatus(execution: StudioWorkflowExecution) {
  const status = execution.status === 'submitted' ? (execution.task_status ?? 'pending') : execution.status
  switch (status) {
    case 'pending': return '待处理'
    case 'queued': return '排队中'
    case 'running': return '执行中'
    case 'succeeded': return '已完成'
    case 'failed': return '执行失败'
    case 'cancelled': return '已取消'
  }
}

function TaskCard({ execution, nodes }: { execution: StudioWorkflowExecution; nodes: StudioFlowNode[] }) {
  return <Card className='gap-0 py-0 text-sm'>
    <CardHeader className='flex flex-row items-start justify-between gap-3 p-4'>
      <div className='flex min-w-0 items-start gap-2.5'>
        <Workflow className='mt-0.5 size-4 shrink-0 text-muted-foreground' aria-hidden='true' />
        <CardTitle className='min-w-0 text-sm leading-5 break-words'>{workflowExecutionTitle(execution, nodes)}</CardTitle>
      </div>
      <Badge variant={isActiveWorkflowExecution(execution) ? 'secondary' : 'outline'} className='text-sm'>{executionStatus(execution)}</Badge>
    </CardHeader>
    <CardContent className='grid gap-3 px-4 pb-4 text-sm'>
      <div className='grid gap-1 text-muted-foreground'>
        <span>开始于 {new Date(execution.created_at).toLocaleString('zh-CN')}</span>
        {execution.completed_at ? <span>完成于 {new Date(execution.completed_at).toLocaleString('zh-CN')}</span> : null}
        <span className='break-all'>任务 ID：{execution.task_id}</span>
      </div>
      {execution.inputs?.length ? <div className='grid gap-1.5 rounded-md bg-muted p-3'>
        <span className='font-medium'>输入</span>
        {execution.inputs.map((input) => <div key={input.key} className='grid grid-cols-[minmax(0,6rem)_minmax(0,1fr)] gap-2'>
          <span className='text-muted-foreground'>{input.key}</span><span className='min-w-0 break-words'>{input.asset_name || input.value || input.asset_id || '—'}</span>
        </div>)}
      </div> : null}
      {execution.error_message ? <p className='rounded-md bg-muted p-3 text-destructive'>{execution.error_message}</p> : null}
    </CardContent>
  </Card>
}

export function StudioSessionTasks({ executions, nodes }: { executions: StudioWorkflowExecution[]; nodes: StudioFlowNode[] }) {
  const active = executions.filter(isActiveWorkflowExecution)
  const completed = executions.filter((execution) => !isActiveWorkflowExecution(execution)).reverse()
  return <ScrollArea className='h-full min-h-0'><div className='grid gap-6 p-4'>
    {executions.length === 0 ? <div className='flex min-h-48 flex-col items-center justify-center gap-3 text-sm text-muted-foreground'><ListTodo className='size-8' aria-hidden='true' /><span>暂无会话任务</span></div> : null}
    {active.length ? <section className='grid gap-3' aria-label='执行中的任务'><h3 className='text-sm font-medium'>执行中 · {active.length}</h3>{active.map((execution) => <TaskCard key={execution.id} execution={execution} nodes={nodes} />)}</section> : null}
    {completed.length ? <section className='grid gap-3' aria-label='任务记录'><h3 className='text-sm font-medium'>任务记录 · {completed.length}</h3>{completed.map((execution) => <TaskCard key={execution.id} execution={execution} nodes={nodes} />)}</section> : null}
  </div></ScrollArea>
}
