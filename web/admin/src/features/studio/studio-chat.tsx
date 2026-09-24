import { useEffect, useMemo, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  AssistantRuntimeProvider,
  ExportedMessageRepository,
  type AssistantState,
  type ChatModelRunOptions,
  type ThreadHistoryAdapter,
  useAuiState,
  useAui,
} from '@assistant-ui/react'
import {
  fromAgUiMessages,
  useAgUiInterrupts,
  useAgUiRuntime,
  useAgUiSubmitInterruptResponses,
} from '@assistant-ui/react-ag-ui'
import {
  Bot,
  ChevronDown,
  Copy,
  Paperclip,
  ShieldCheck,
  Sparkles,
} from 'lucide-react'
import { StudioWebSocketAgent } from '@/lib/agui-websocket-agent'
import { baseURL } from '@/lib/api/client'
import {
  cancelStudioRun,
  listStudioLibraryAssets,
  type StudioAsset,
  type StudioComposerPart,
  type StudioMessage,
  type StudioModel,
  type StudioPermissionMode,
  type StudioPendingApproval,
  type StudioRun,
  type StudioRunProgress,
  type StudioSkill,
  type StudioTranscript,
} from '@/lib/api/studio'
import { StudioRunConnection } from '@/lib/studio-run-connection'
import { cn } from '@/lib/utils'
import { AlertDescription, AlertTitle } from '@/components/ui/alert'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Confirmation,
  ConfirmationAction,
  ConfirmationActions,
} from '@/components/ai-elements/confirmation'
import {
  Conversation,
  ConversationContent,
  ConversationEmptyState,
  ConversationScrollButton,
} from '@/components/ai-elements/conversation'
import {
  Message,
  MessageAction,
  MessageActions,
  MessageContent,
  MessageResponse,
} from '@/components/ai-elements/message'
import {
  PromptInput,
  PromptInputBody,
  PromptInputButton,
  PromptInputFooter,
  PromptInputSubmit,
  PromptInputTools,
} from '@/components/ai-elements/prompt-input'
import {
  Reasoning,
  ReasoningContent,
  ReasoningTrigger,
} from '@/components/ai-elements/reasoning'
import { Suggestion, Suggestions } from '@/components/ai-elements/suggestion'
import {
  Tool,
  ToolContent,
  ToolHeader,
  ToolInput,
  ToolOutput,
} from '@/components/ai-elements/tool'
import { StudioComposer, type StudioComposerHandle, type StudioReference } from './studio-composer'
import type { StudioComposerValue } from './studio-composer-content'
import { StudioReferenceBadge } from './studio-reference-badge'

type Props = {
  sessionId: string
  messages: StudioMessage[]
  transcript?: StudioTranscript
  latestRun?: StudioRun | null
  runProgress?: StudioRunProgress | null
  pendingApprovals?: StudioPendingApproval[]
  models: StudioModel[]
  modelConfigId?: string
  permissionMode: StudioPermissionMode
  skills: StudioSkill[]
  assets: StudioAsset[]
  selectedSkillIds: string[]
  selectedAssets: SelectedAsset[]
  onModelChange: (id: string) => void
  onPermissionChange: (mode: StudioPermissionMode) => void
  onSkillChange: (ids: string[]) => void
  onAssetChange: (assets: SelectedAsset[]) => void
  onImportLibraryAsset: (asset: SelectedAsset) => Promise<StudioAsset>
  onRunFinished?: () => void
  onRuntimeStateChange?: (running: boolean) => void
}

type SelectedAsset = { assetId: string; assetVersionId: string }

function toAGUIMessages(messages: StudioMessage[]) {
  return messages
    .filter(
      (message) => message.role === 'user' || message.role === 'assistant'
    )
    .map((message) => ({
      id: message.id,
      role: message.role,
      content: message.content
        .filter((part) => part.type === 'text')
        .map((part) => part.text ?? '')
        .join(''),
    }))
}

function toTranscriptAGUIMessages(
  transcript?: StudioTranscript,
  excludeRunId?: string
) {
  if (!transcript?.messages?.length) return undefined
  return transcript.messages
    .filter(
      (message) =>
        !excludeRunId ||
        message.role === 'user' ||
        message.runId !== excludeRunId
    )
    .map((message) => ({
      id: message.id,
      role: message.role,
      content: message.content,
      ...(message.toolCalls ? { toolCalls: message.toolCalls } : {}),
      ...(message.toolCallId ? { toolCallId: message.toolCallId } : {}),
      ...(message.isError !== undefined ? { isError: message.isError } : {}),
    }))
}

export function StudioChat(props: Props) {
  const [runError, setRunError] = useState<string>()
  const availableModels = props.models.filter(
    (model) => model.enabled && model.agent_enabled
  )
  const selectedModel =
    availableModels.find((model) => model.id === props.modelConfigId) ??
    availableModels.find((model) => model.default) ??
    availableModels[0]
  const modelReady = Boolean(selectedModel)
  const runConfig = useMemo(
    () => ({
      modelConfigId: selectedModel?.id ?? '',
      permissionMode: props.permissionMode,
      selectedSkillIds: props.selectedSkillIds,
      selectedAssets: props.selectedAssets,
    }),
    [
      selectedModel?.id,
      props.permissionMode,
      props.selectedSkillIds,
      props.selectedAssets,
    ]
  )
  const agent = useMemo(() => {
    const endpoint = `${baseURL()}/api/v1/studio/agui/ws`
    const httpURL = new URL(endpoint, window.location.origin)
    httpURL.protocol = httpURL.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsURL = httpURL.toString()
    return new StudioWebSocketAgent({
      url: wsURL,
      threadId: props.sessionId,
      initialMessages: (toTranscriptAGUIMessages(props.transcript) ??
        toAGUIMessages(props.messages)) as never[],
      runConfig,
    })
    // A runtime owns the active connection. Replacing the agent when a
    // transcript query refreshes would silently abandon that connection.
    // Session changes are isolated by StudioWorkspace's keyed mount.
    // Deliberately only depend on sessionId: refreshing transcript data must
    // not replace the live agent instance underneath an active run.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [props.sessionId])

  useEffect(() => {
    agent.updateRunConfig(runConfig)
  }, [agent, runConfig])

  const history = useMemo<ThreadHistoryAdapter>(() => {
    const activeRunId =
      props.latestRun &&
      ['queued', 'running', 'waiting_approval'].includes(props.latestRun.status)
        ? props.latestRun.id
        : undefined
    const activeMessageIDs = new Set(
      props.messages
        .filter(
          (message) => message.run_id === activeRunId && message.role !== 'user'
        )
        .map((message) => message.id)
    )
    const messages = fromAgUiMessages(
      (
        toTranscriptAGUIMessages(props.transcript, activeRunId) ??
        toAGUIMessages(props.messages)
      ).filter((message) => !activeMessageIDs.has(message.id)),
      { showThinking: true }
    )

    const endpoint = `${baseURL()}/api/v1/studio/agui/ws`
    const httpURL = new URL(endpoint, window.location.origin)
    httpURL.protocol = httpURL.protocol === 'https:' ? 'wss:' : 'ws:'
    const connection =
      props.latestRun &&
      (props.latestRun.status === 'queued' ||
        props.latestRun.status === 'running' ||
        props.latestRun.status === 'waiting_approval')
        ? new StudioRunConnection({
            url: httpURL.toString(),
            threadId: props.sessionId,
            studioRunId: props.latestRun.id,
            afterSequence: 0,
          })
        : undefined

    return {
      async load() {
        return {
          ...ExportedMessageRepository.fromArray(messages),
          unstable_resume: Boolean(connection),
        }
      },
      async *resume(options: ChatModelRunOptions) {
        if (!connection) return
        yield* connection.resume(options)
      },
      async append() {
        // The backend persists messages as part of the AG-UI run. History is
        // read from the session endpoint when a thread is opened.
      },
    }
    // The run id/status/sequence are the attach identity; object identity from
    // a Query refresh must not restart history loading by itself.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    props.messages,
    props.transcript,
    props.sessionId,
    props.latestRun?.id,
    props.latestRun?.status,
  ])

  const runtime = useAgUiRuntime({
    agent,
    showThinking: true,
    onError: (error) => setRunError(error.message),
    adapters: { history },
  })

  return (
    <AssistantRuntimeProvider runtime={runtime}>
      <StudioRunCompletionWatcher
        onRunStarted={() => setRunError(undefined)}
        onRunFinished={props.onRunFinished}
        onRuntimeStateChange={props.onRuntimeStateChange}
      />
      <StudioChatSurface
        agent={agent}
        onRunError={setRunError}
        availableModels={availableModels}
        modelReady={modelReady}
        selectedModel={selectedModel}
        runError={runError}
        {...props}
      />
    </AssistantRuntimeProvider>
  )
}

function StudioChatSurface({
  availableModels,
  modelReady,
  selectedModel,
  runError,
  agent,
  onRunError,
  ...props
}: Props & {
  availableModels: StudioModel[]
  modelReady: boolean
  selectedModel?: StudioModel
  runError?: string
  agent: StudioWebSocketAgent
  onRunError: (message: string) => void
}) {
  const aui = useAui()
  const messages = useAuiState((state) => state.thread.messages)
  const isRunning = useAuiState((state) => state.thread.isRunning)
  const isEmpty = useAuiState((state) => state.thread.isEmpty)
  const [answered, setAnswered] = useState(false)
  const composerRef = useRef<StudioComposerHandle>(null)
  const [composerValue, setComposerValue] = useState<StudioComposerValue>({
    text: '', parts: [], selectedSkillIds: [], selectedAssets: [],
  })
  const [liveMessage, setLiveMessage] = useState<{
    previousUserMessageId?: string
    parts: StudioComposerPart[]
  }>()
  const hasPendingAction = useAgUiInterrupts().length > 0
  // interrupt 要等恢复运行的流走完才有，这段时间用会话详情里的运行状态和待批准项撑着，
  // 底部这一行从第一帧就是操作区，聊天输入不会先画出来。用户回答之后以 interrupt 为准。
  const waitingForDecision =
    hasPendingAction ||
    (props.latestRun?.status === 'waiting_approval' && !answered)
  const preloadedActions = answered ? [] : (props.pendingApprovals ?? [])
  const serverRunning =
    props.latestRun?.status === 'queued' ||
    props.latestRun?.status === 'running'
  const runActive =
    isRunning || serverRunning || props.latestRun?.status === 'waiting_approval'
  const isStreaming = isRunning || serverRunning
  const send = (prompt?: string) => {
    const value = prompt === undefined
      ? composerRef.current?.serialize()
      : { text: prompt, parts: [{ type: 'text' as const, text: prompt }], selectedSkillIds: [], selectedAssets: [] }
    if (!modelReady || runActive || !value?.text.trim()) return
    agent.prepareNextRun({
      modelConfigId: selectedModel?.id ?? '',
      permissionMode: props.permissionMode,
      selectedSkillIds: value.selectedSkillIds,
      selectedAssets: value.selectedAssets,
      messageParts: value.parts,
    })
    const composer = aui.thread.composer()
    composer.setText(value.text)
    composer.send()
    setLiveMessage({ previousUserMessageId: latestUserMessageId, parts: value.parts })
    composerRef.current?.clear()
  }

  const transcriptParts = new Map(
    props.transcript?.messages.filter((message) => message.role === 'user' && message.parts).map((message) => [message.id, message.parts!]) ?? []
  )
  const latestUserMessageId = [...messages].reverse().find((message) => message.role === 'user')?.id
  const assistantCopyText = new Map<string, string>()
  let responseText = ''
  let lastAssistantMessageId: string | undefined
  for (const message of messages) {
    if (message.role === 'user') {
      if (lastAssistantMessageId && responseText) {
        assistantCopyText.set(lastAssistantMessageId, responseText)
      }
      responseText = ''
      lastAssistantMessageId = undefined
    } else if (message.role === 'assistant') {
      const text = message.parts.flatMap((part) => part.type === 'text' ? [part.text] : []).join('')
      if (text) responseText += `${responseText ? '\n\n' : ''}${text}`
      lastAssistantMessageId = message.id
    }
  }
  if (lastAssistantMessageId && responseText) {
    assistantCopyText.set(lastAssistantMessageId, responseText)
  }
  const slashItems: StudioReference[] = [
    ...props.skills.filter((skill) => skill.enabled).map((skill) => ({
      kind: 'skill' as const, id: skill.id, label: skill.name,
    })),
    ...props.assets.flatMap((asset) => {
      const version = asset.versions[asset.versions.length - 1]
      return version ? [{
        kind: 'asset' as const, id: asset.id, label: asset.name, versionId: version.id,
      }] : []
    }),
  ]

  return (
    <div className='relative flex min-h-0 flex-1 flex-col'>
      <Conversation className='min-h-0 flex-1'>
        <ConversationContent
          className={cn(
            'mx-auto min-h-full w-full max-w-3xl gap-5 px-5 pt-8',
            waitingForDecision ? 'pb-5' : 'pb-44'
          )}
        >
          {isEmpty ? <StudioWelcome onSelect={send} /> : null}
          {messages.map((message) => (
            <StudioMessage
              key={message.id}
              message={message}
              isRunning={isRunning}
              assistantCopyText={assistantCopyText.get(message.id)}
              referenceParts={transcriptParts.get(message.id) ?? (
                message.id === latestUserMessageId && message.id !== liveMessage?.previousUserMessageId
                  ? liveMessage?.parts
                  : undefined
              )}
            />
          ))}
          {runError ? (
            <div
              role='alert'
              className='max-w-[88%] rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm leading-6 text-destructive'
            >
              {runError}
            </div>
          ) : null}
        </ConversationContent>
        <ConversationScrollButton
          aria-label='跳转至最新消息'
          className={waitingForDecision ? 'bottom-5' : 'bottom-48'}
        />
      </Conversation>
      <StudioActionArea
        preloadedActions={preloadedActions}
        onAnswered={() => setAnswered(true)}
      />
      <div
        data-slot='studio-composer'
        className={cn(
          'pointer-events-none absolute inset-x-0 bottom-0 z-20',
          waitingForDecision && 'invisible'
        )}
      >
        <div
          aria-hidden='true'
          className='h-12 bg-gradient-to-t from-card to-transparent'
        />
        <div className='bg-card'>
          <div className='mx-auto flex w-full max-w-3xl flex-col px-5 pb-5'>
            <div className='pointer-events-auto'>
              <PromptInput
                inputGroupClassName='h-auto overflow-visible bg-background'
                onSubmit={() => send()}
              >
                <PromptInputBody>
                  <StudioComposer
                    ref={composerRef}
                    disabled={!modelReady}
                    slashItems={slashItems}
                    onSubmit={() => send()}
                    onValueChange={(value) => {
                      setComposerValue(value)
                      props.onSkillChange(value.selectedSkillIds)
                      props.onAssetChange(value.selectedAssets)
                    }}
                    placeholder={
                      modelReady
                        ? '描述你想创作的内容，或让 Agent 调用工作流…'
                        : '先在 AI 设置中添加并启用模型'
                    }
                  />
                </PromptInputBody>
                <PromptInputFooter>
                  <PromptInputTools>
                    <SkillPicker
                      skills={props.skills}
                      value={composerValue.selectedSkillIds}
                      onInsert={(skill) => composerRef.current?.insertReference({ kind: 'skill', id: skill.id, label: skill.name })}
                    />
                    <AssetPicker
                      assets={props.assets}
                      value={composerValue.selectedAssets}
                      onInsert={(asset, versionId) => composerRef.current?.insertReference({ kind: 'asset', id: asset.id, label: asset.name, versionId })}
                      onImportLibraryAsset={props.onImportLibraryAsset}
                    />
                    <PermissionPicker
                      value={props.permissionMode}
                      onChange={props.onPermissionChange}
                    />
                  </PromptInputTools>
                  <PromptInputTools>
                    <ModelPicker
                      models={availableModels}
                      value={selectedModel?.id}
                      onChange={props.onModelChange}
                    />
                    <PromptInputSubmit
                      aria-label={isStreaming ? '停止生成' : '发送消息'}
                      disabled={!modelReady || (runActive && !isStreaming)}
                      onStop={() => {
                        const runID =
                          agent.activeStudioRunId() ?? props.latestRun?.id
                        if (!runID) {
                          onRunError('运行正在建立连接，请稍后再试')
                          return
                        }
                        void cancelStudioRun(runID)
                          .then(() => {
                            aui.thread.cancelRun()
                            props.onRunFinished?.()
                          })
                          .catch((error: unknown) => {
                            onRunError(
                              error instanceof Error
                                ? error.message
                                : '停止运行失败'
                            )
                          })
                      }}
                      status={isStreaming ? 'streaming' : undefined}
                    />
                  </PromptInputTools>
                </PromptInputFooter>
              </PromptInput>
            </div>
            <p className='mt-2 text-center text-xs text-muted-foreground'>
              {modelReady
                ? 'Agent 可能会调用模型、Skill、连接器和工作流，请核对重要结果。'
                : '没有可用模型时，无法发起 Agent 对话。'}
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}

function StudioActionArea({
  preloadedActions,
  onAnswered,
}: {
  preloadedActions: StudioAction[]
  onAnswered: () => void
}) {
  const interrupts = useAgUiInterrupts()
  const submitInterruptResponses = useAgUiSubmitInterruptResponses()

  return (
    <StudioActionPanel
      actions={interrupts.map((interrupt) => ({
        id: interrupt.id,
        reason: interrupt.reason,
        message: interrupt.message,
      }))}
      preloadedActions={preloadedActions}
      onRespond={(id, approved) => {
        onAnswered()
        void submitInterruptResponses([
          {
            interruptId: id,
            status: approved ? 'resolved' : 'cancelled',
            ...(approved ? { payload: true } : {}),
          },
        ])
      }}
    />
  )
}

export type StudioAction = {
  id: string
  reason?: string
  message?: string
}

// 待处理事项的类型标题，按 AG-UI 中断原因区分
function approvalTitle(reason?: string) {
  return reason === 'tool_approval' ? '权限审批' : '需要处理'
}

export function StudioActionPanel({
  actions,
  preloadedActions = [],
  onRespond,
}: {
  actions: StudioAction[]
  preloadedActions?: StudioAction[]
  onRespond: (id: string, approved: boolean) => void
}) {
  // interrupt 到了就以它为准，没到之前先用会话详情带过来的待处理项
  const fromRuntime = actions.length > 0
  const visibleActions = fromRuntime ? actions : preloadedActions
  if (visibleActions.length === 0) return null

  return (
    <section aria-label='操作区'>
      <div className='mx-auto flex w-full max-w-3xl flex-col gap-2 px-5 pb-5'>
        {visibleActions.map((action) => (
          <Confirmation
            key={action.id}
            approval={{ id: action.id }}
            state='approval-requested'
            variant='warn'
          >
            <AlertTitle>{approvalTitle(action.reason)}</AlertTitle>
            <AlertDescription>
              {action.message ?? '需要批准后继续执行'}
            </AlertDescription>
            <ConfirmationActions>
              <ConfirmationAction
                variant='outline'
                disabled={!fromRuntime}
                onClick={() => onRespond(action.id, false)}
              >
                拒绝
              </ConfirmationAction>
              <ConfirmationAction
                disabled={!fromRuntime}
                onClick={() => onRespond(action.id, true)}
              >
                批准
              </ConfirmationAction>
            </ConfirmationActions>
          </Confirmation>
        ))}
      </div>
    </section>
  )
}

function StudioRunCompletionWatcher({
  onRunStarted,
  onRunFinished,
  onRuntimeStateChange,
}: {
  onRunStarted?: () => void
  onRunFinished?: () => void
  onRuntimeStateChange?: (running: boolean) => void
}) {
  const isRunning = useAuiState((state) => state.thread.isRunning)
  const wasRunning = useRef(false)

  useEffect(() => {
    if (!wasRunning.current && isRunning) {
      onRunStarted?.()
      onRuntimeStateChange?.(true)
    }
    if (wasRunning.current && !isRunning) {
      onRunFinished?.()
      onRuntimeStateChange?.(false)
    }
    wasRunning.current = isRunning
  }, [isRunning, onRunStarted, onRunFinished, onRuntimeStateChange])
  return null
}

function AssetPicker({
  assets,
  value,
  onInsert,
  onImportLibraryAsset,
}: {
  assets: StudioAsset[]
  value: SelectedAsset[]
  onInsert: (asset: StudioAsset, versionId: string) => void
  onImportLibraryAsset: (asset: SelectedAsset) => Promise<StudioAsset>
}) {
  const libraryAssets = useQuery({
    queryKey: ['studio', 'library', 'assets'],
    queryFn: () => listStudioLibraryAssets(),
  })
  const sessionAssetIDs = new Set(assets.map((asset) => asset.id))
  const assetsByID = new Map<string, StudioAsset>()
  for (const asset of assets) assetsByID.set(asset.id, asset)
  for (const asset of libraryAssets.data ?? []) {
    assetsByID.set(asset.id, asset)
  }
  const insert = async (asset: StudioAsset, fromLibrary = false) => {
    const version = asset.versions[asset.versions.length - 1]
    if (!version) return
    if (fromLibrary) {
      const imported = await onImportLibraryAsset({ assetId: asset.id, assetVersionId: version.id })
      const importedVersion = imported.versions[imported.versions.length - 1]
      if (!importedVersion) return
      onInsert(imported, importedVersion.id)
      return
    }
    onInsert(asset, version.id)
  }
  const unavailableSelections = value.filter((selection) => {
    const asset = assetsByID.get(selection.assetId)
    return (
      !asset ||
      !asset.versions.some((version) => version.id === selection.assetVersionId)
    )
  })
  const currentAssets = assets
  const reusableAssets = (libraryAssets.data ?? []).filter(
    (asset) => !sessionAssetIDs.has(asset.id)
  )

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <PromptInputButton
          aria-label={
            value.length === 0
              ? '选择资产'
              : `选择资产，已选 ${value.length} 项`
          }
          className={cn(value.length > 0 && 'bg-accent text-accent-foreground')}
          size='icon-sm'
          tooltip={value.length === 0 ? '资产' : `资产 · ${value.length}`}
        >
          <Paperclip />
        </PromptInputButton>
      </DropdownMenuTrigger>
      <DropdownMenuContent align='start' className='w-72'>
        <DropdownMenuLabel>本轮使用的资产</DropdownMenuLabel>
        <DropdownMenuSeparator />
        <AssetPickerSection
          label='当前 Session'
          assets={currentAssets}
          onSelect={insert}
          emptyText='当前 Session 还没有资产'
        />
        <DropdownMenuSeparator />
        <AssetPickerSection
          label='资产库'
          assets={reusableAssets}
          onSelect={(asset) => void insert(asset, true)}
          emptyText={
            libraryAssets.isLoading
              ? '正在读取资产库…'
              : '资产库还没有可复用资产'
          }
        />
        {unavailableSelections.length > 0 ? (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem disabled>
              {unavailableSelections.length} 项已选资产不可用
            </DropdownMenuItem>
          </>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

function AssetPickerSection({
  label,
  assets,
  onSelect,
  emptyText,
}: {
  label: string
  assets: StudioAsset[]
  onSelect: (asset: StudioAsset) => void | Promise<void>
  emptyText: string
}) {
  return (
    <>
      <DropdownMenuLabel className='text-xs font-medium text-muted-foreground'>
        {label}
      </DropdownMenuLabel>
      {assets.length === 0 ? (
        <DropdownMenuItem disabled>{emptyText}</DropdownMenuItem>
      ) : (
        assets.map((asset) => {
          const version = asset.versions[asset.versions.length - 1]
          return (
            <DropdownMenuItem
              key={asset.id}
              onSelect={() => void onSelect(asset)}
            >
              <span className='min-w-0 flex-1'>
                <span className='block truncate'>{asset.name}</span>
                <span className='block text-xs text-muted-foreground'>
                  {asset.kind === 'image'
                    ? '图片'
                    : asset.kind === 'document'
                      ? '文档'
                      : '文件'}
                  {version ? ` · v${version.version}` : ''}
                </span>
              </span>
            </DropdownMenuItem>
          )
        })
      )}
    </>
  )
}

function SkillPicker({
  skills,
  value,
  onInsert,
}: {
  skills: StudioSkill[]
  value: string[]
  onInsert: (skill: StudioSkill) => void
}) {
  const enabledSkills = skills.filter((skill) => skill.enabled)
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <PromptInputButton
          aria-label={
            value.length === 0
              ? '选择 Skills'
              : `选择 Skills，已选 ${value.length} 项`
          }
          className={cn(value.length > 0 && 'bg-accent text-accent-foreground')}
          size='icon-sm'
          tooltip={value.length === 0 ? 'Skills' : `Skills · ${value.length}`}
        >
          <Sparkles />
        </PromptInputButton>
      </DropdownMenuTrigger>
      <DropdownMenuContent align='start' className='w-72'>
        <DropdownMenuLabel>本轮使用的 Skills</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {enabledSkills.length === 0 ? (
          <DropdownMenuItem disabled>没有已启用的 Skill</DropdownMenuItem>
        ) : null}
        {enabledSkills.map((skill) => (
          <DropdownMenuItem
            key={skill.id}
            onSelect={() => onInsert(skill)}
          >
            <span className='min-w-0 flex-1'>
              <span className='block truncate'>{skill.name}</span>
              <span className='block truncate text-xs text-muted-foreground'>
                {skill.description}
              </span>
            </span>
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

function StudioWelcome({ onSelect }: { onSelect: (prompt: string) => void }) {
  return (
    <ConversationEmptyState
      className='min-h-full py-16'
      description='和 Agent 一起构思内容、生成资产，或调用现有工作流。'
      icon={<Bot className='size-6' />}
      title='从一个想法开始'
    >
      <span className='flex size-12 items-center justify-center rounded-2xl bg-primary text-primary-foreground'>
        <Bot className='size-6' />
      </span>
      <div className='space-y-2'>
        <h2 className='text-xl font-semibold tracking-tight'>从一个想法开始</h2>
        <p className='max-w-md text-sm leading-6 text-muted-foreground'>
          和 Agent 一起构思内容、生成资产，或调用现有工作流。
        </p>
      </div>
      <Suggestions className='mt-3 max-w-xl'>
        {['为雨夜侦探构思漫画并生成分镜', '根据一张角色图生成三视图'].map(
          (prompt) => (
            <Suggestion key={prompt} onClick={onSelect} suggestion={prompt} />
          )
        )}
      </Suggestions>
    </ConversationEmptyState>
  )
}

type StudioThreadMessage = AssistantState['thread']['messages'][number]

type StudioToolCallPart = Extract<
  StudioThreadMessage['parts'][number],
  { type: 'tool-call' }
>

function StudioMessage({
  message,
  isRunning,
  assistantCopyText,
  referenceParts,
}: {
  message: StudioThreadMessage
  isRunning: boolean
  assistantCopyText?: string
  referenceParts?: StudioComposerPart[]
}) {
  if (message.role !== 'user' && message.role !== 'assistant') return null
  const copyText = message.role === 'user'
    ? message.parts.flatMap((part) => part.type === 'text' ? [part.text] : []).join('')
    : assistantCopyText
  return (
    <Message from={message.role} className={cn('gap-1', message.role === 'assistant' && 'max-w-none')}>
      <MessageContent className={cn('gap-2', message.role === 'assistant' && 'w-full')}>
        {message.role === 'user' && referenceParts ? (
          <span className='whitespace-pre-wrap break-words'>
            {referenceParts.map((part, index) => part.type === 'text' ? (
              <span key={index}>{part.text}</span>
            ) : (
              <StudioReferenceBadge
                key={index}
                kind={part.type === 'skill_ref' ? 'skill' : 'asset'}
                label={part.name}
              />
            ))}
          </span>
        ) : message.parts.map((part, index) => {
          if (part.type === 'text') {
            return (
              <MessageResponse
                isAnimating={isRunning && message.isLast}
                key={`${message.id}-${index}`}
              >
                {part.text}
              </MessageResponse>
            )
          }
          if (part.type === 'reasoning' && part.text.trim()) {
            const streaming = part.status.type === 'running'
            return (
              <Reasoning className='mb-0' isStreaming={streaming} key={`${message.id}-${index}`}>
                <ReasoningTrigger
                  getThinkingMessage={(active) =>
                    active ? '正在思考' : '思考过程'
                  }
                />
                <ReasoningContent className='mt-2'>{part.text}</ReasoningContent>
              </Reasoning>
            )
          }
          if (part.type === 'tool-call') {
            return <StudioToolCall key={`${message.id}-${index}`} part={part} />
          }
          return null
        })}
      </MessageContent>
      {copyText ? (
        <MessageActions className={message.role === 'user' ? 'justify-end' : 'justify-start'}>
          <MessageAction
            label={message.role === 'user' ? '复制发送消息' : '复制返回消息'}
            tooltip='复制消息'
            onClick={() => void navigator.clipboard.writeText(copyText)}
          >
            <Copy className='size-4' />
          </MessageAction>
        </MessageActions>
      ) : null}
    </Message>
  )
}

function StudioToolCall({ part }: { part: StudioToolCallPart }) {
  const state = part.isError
    ? 'output-error'
    : part.result !== undefined
      ? 'output-available'
      : part.status.type === 'running'
        ? 'input-available'
        : 'input-streaming'
  return (
    <Tool className='mb-0' defaultOpen={state === 'output-error'}>
      <ToolHeader
        state={state}
        title={part.toolName}
        toolName={part.toolName}
        type='dynamic-tool'
      />
      <ToolContent className='space-y-2'>
        <ToolInput input={part.args} />
        <ToolOutput
          errorText={
            part.isError ? String(part.result ?? '工具调用失败') : undefined
          }
          output={part.isError ? undefined : part.result}
        />
      </ToolContent>
    </Tool>
  )
}

function ModelPicker({
  models,
  value,
  onChange,
}: {
  models: StudioModel[]
  value?: string
  onChange: (id: string) => void
}) {
  const selected = models.find((model) => model.id === value)
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <PromptInputButton className='max-w-52 font-normal'>
          <span className='truncate'>{selected?.name ?? '未选择模型'}</span>
          <ChevronDown className='size-3.5' />
        </PromptInputButton>
      </DropdownMenuTrigger>
      <DropdownMenuContent align='end' className='w-72'>
        <DropdownMenuLabel>本轮使用的模型</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {models.length === 0 ? (
          <DropdownMenuItem disabled>没有可用模型</DropdownMenuItem>
        ) : null}
        <DropdownMenuRadioGroup value={value} onValueChange={onChange}>
          {models.map((model) => (
            <DropdownMenuRadioItem key={model.id} value={model.id}>
              <span className='min-w-0 flex-1 truncate'>{model.name}</span>
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

const permissionLabels: Record<StudioPermissionMode, string> = {
  request_approval: '请求批准',
  auto_approve: '帮我批准',
  full_access: '完全访问',
}

function PermissionPicker({
  value,
  onChange,
}: {
  value: StudioPermissionMode
  onChange: (mode: StudioPermissionMode) => void
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <PromptInputButton
          aria-label={`Agent 操作权限：${permissionLabels[value]}`}
          className='font-normal'
        >
          <ShieldCheck />
          {permissionLabels[value]}
          <ChevronDown className='size-3.5' />
        </PromptInputButton>
      </DropdownMenuTrigger>
      <DropdownMenuContent align='start' className='w-72'>
        <DropdownMenuLabel>Agent 操作权限</DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuRadioGroup
          value={value}
          onValueChange={(next) => onChange(next as StudioPermissionMode)}
        >
          <DropdownMenuRadioItem value='request_approval'>
            请求批准 · 每次执行前确认
          </DropdownMenuRadioItem>
          <DropdownMenuRadioItem value='auto_approve'>
            帮我批准 · 仅高风险操作确认
          </DropdownMenuRadioItem>
          <DropdownMenuRadioItem value='full_access'>
            完全访问 · 自动执行所有操作
          </DropdownMenuRadioItem>
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
