import { useState } from 'react'
import '@/styles/index.css'
import { expect, it } from 'vitest'
import { render } from 'vitest-browser-react'
import type { StudioFlowNode, StudioWorkflowExecution } from '@/lib/api/studio'
import { StudioFlow } from './studio-flow'
import { StudioSessionTasks } from './studio-session-tasks'

const node: StudioFlowNode = {
  id: 'operation', type: 'operation', title: '生成分镜', position: { x: 100, y: 100 }, sort_order: 0, updated_at: '',
  outputs: [{ key: 'storyboard', type: 'image', name: '分镜图', asset_id: 'asset-storyboard', asset_version_id: 'version' }],
}
const execution: StudioWorkflowExecution = {
  id: 'execution', run_id: 'run', task_id: 'task', workflow_id: '1', operation_node_id: node.id,
  status: 'succeeded', created_at: '2026-10-01T10:00:00Z', completed_at: '2026-10-01T10:01:00Z',
  input_fields: [{ key: 'prompt', type: 'string', description: '画面描述' }],
  output_fields: [{ key: 'storyboard', type: 'image', description: '分镜输出' }],
  inputs: [{ key: 'prompt', value: '森林中的小屋' }],
}

it('流程卡片显示字段并打开输出资产', async () => {
  function Workbench() {
    const [assetId, setAssetId] = useState('')
    return <>
      <div style={{ width: 900, height: 700 }}>
        <StudioFlow nodes={[node]} edges={[]} workflowExecutions={[execution]} onAssetOpen={setAssetId} />
      </div>
      <output>{assetId}</output>
    </>
  }
  const screen = await render(<Workbench />)
  await expect.element(screen.getByText('画面描述')).toBeVisible()
  await expect.element(screen.getByText('森林中的小屋')).toBeVisible()
  await expect.element(screen.getByText('分镜输出')).toBeVisible()
  await screen.getByRole('button', { name: '分镜图', exact: true }).click()
  await expect.element(screen.getByRole('status')).toHaveTextContent('asset-storyboard')
})

it('会话任务按真实任务状态区分执行中与记录', async () => {
  const pending: StudioWorkflowExecution = { ...execution, id: 'pending', task_id: 'pending-task', status: 'submitted', task_status: 'queued', completed_at: undefined }
  const failed: StudioWorkflowExecution = { ...execution, id: 'failed', task_id: 'failed-task', status: 'failed', error_message: '生成失败' }
  const screen = await render(<StudioSessionTasks executions={[pending, execution, failed]} nodes={[node]} />)
  await expect.element(screen.getByRole('region', { name: '执行中的任务' })).toHaveTextContent('排队中')
  await expect.element(screen.getByRole('region', { name: '任务记录' })).toHaveTextContent('已完成')
  await expect.element(screen.getByRole('region', { name: '任务记录' })).toHaveTextContent('生成失败')
  await expect.element(screen.getByText('森林中的小屋').first()).toBeVisible()
})
