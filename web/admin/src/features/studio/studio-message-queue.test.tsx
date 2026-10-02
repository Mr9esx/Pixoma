import { createRef, useState } from 'react'
import { expect, it } from 'vitest'
import { render } from 'vitest-browser-react'
import type { StudioRunConfig } from '@/lib/agui-websocket-agent'
import { StudioMessageQueue } from './studio-chat'
import { StudioComposer, type StudioComposerHandle } from './studio-composer'
import { serializeStudioComposer, studioComposerContent } from './studio-composer-content'

it('编辑待发送消息时恢复引用，并保留其他消息的顺序和运行配置', async () => {
  const composer = createRef<StudioComposerHandle>()
  const first = serializeStudioComposer(studioComposerContent([
    { type: 'text', text: '按照' },
    { type: 'skill_ref', skill_id: 'skill-1', name: '分镜草稿' },
    { type: 'asset_ref', asset_id: 'asset-1', asset_version_id: 'version-1', name: '产品照片' },
    { type: 'workflow_ref', workflow_id: 'workflow-1', name: '角色三视图' },
  ]))
  const second = serializeStudioComposer(studioComposerContent([{ type: 'text', text: '补充背景' }]))
  const runConfig: StudioRunConfig = { modelConfigId: 'model-1', locale: 'zh', permissionMode: 'request_approval', selectedSkillIds: first.selectedSkillIds, selectedAssets: first.selectedAssets, messageParts: first.parts }
  const secondConfig: StudioRunConfig = { ...runConfig, modelConfigId: 'model-2', permissionMode: 'full_access', messageParts: second.parts }
  function QueueEditor() {
    const [messages, setMessages] = useState([
      { id: 'first', value: first, runConfig },
      { id: 'second', value: second, runConfig: secondConfig },
    ])
    const [sentConfig, setSentConfig] = useState<StudioRunConfig>()
    return <>
      <StudioMessageQueue messages={messages} availableModelIds={['model-1', 'model-2']}
        onRemove={(id) => setMessages((items) => items.filter((item) => item.id !== id))}
        onEdit={(message) => { setMessages((items) => items.filter((item) => item.id !== message.id)); composer.current?.setValue(message.value) }}
        onSendNow={(message) => { setSentConfig(message.runConfig); setMessages((items) => items.filter((item) => item.id !== message.id)) }}
      />
      <StudioComposer ref={composer} placeholder='输入消息' />
      {sentConfig ? <output>{sentConfig.modelConfigId} · {sentConfig.permissionMode}</output> : null}
    </>
  }
  const screen = await render(<QueueEditor />)
  await screen.getByRole('button', { name: `更多操作：${first.text}` }).click()
  await screen.getByRole('menuitem', { name: '编辑消息' }).click()
  await expect.element(screen.getByText('分镜草稿')).toBeVisible()
  await expect.element(screen.getByText('产品照片')).toBeVisible()
  await expect.element(screen.getByText('角色三视图')).toBeVisible()
  expect(composer.current?.serialize()).toEqual(first)
  await expect.element(screen.getByRole('listitem')).toHaveTextContent('补充背景')
  await screen.getByRole('button', { name: '立即发送：补充背景' }).click()
  await expect.element(screen.getByRole('status')).toHaveTextContent('model-2 · full_access')
  await expect.element(screen.getByRole('list', { name: '待发送消息' })).not.toBeInTheDocument()
})
