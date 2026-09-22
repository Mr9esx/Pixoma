import { useEffect, useMemo, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  AssistantRuntimeProvider,
  ExportedMessageRepository,
  type ThreadHistoryAdapter,
  useAuiState,
  useAui,
} from '@assistant-ui/react'
import { fromAgUiMessages, useAgUiRuntime } from '@assistant-ui/react-ag-ui'
import {
  Bot,
  Check,
  ChevronDown,
  Paperclip,
  ShieldCheck,
  Sparkles,
} from 'lucide-react'
import { StudioWebSocketAgent } from '@/lib/agui-websocket-agent'
import { baseURL } from '@/lib/api/client'
import {
  listStudioLibraryAssets,
  type StudioAsset,
  type StudioMessage,
  type StudioModel,
  type StudioPermissionMode,
  type StudioSkill,
  type StudioTranscript,
} from '@/lib/api/studio'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Conversation,
  ConversationContent,
  ConversationEmptyState,
  ConversationScrollButton,
} from '@/components/ai-elements/conversation'
import {
  Message,
  MessageContent,
  MessageResponse,
} from '@/components/ai-elements/message'
import {
  ModelSelector,
  ModelSelectorContent,
  ModelSelectorEmpty,
  ModelSelectorGroup,
  ModelSelectorInput,
  ModelSelectorItem,
  ModelSelectorList,
  ModelSelectorName,
  ModelSelectorTrigger,
} from '@/components/ai-elements/model-selector'
import {
  PromptInput,
  PromptInputBody,
  PromptInputButton,
  PromptInputFooter,
  PromptInputSubmit,
  PromptInputTextarea,
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

type Props = {
  sessionId: string
  messages: StudioMessage[]
  transcript?: StudioTranscript
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

function toTranscriptAGUIMessages(transcript?: StudioTranscript) {
  if (!transcript?.messages?.length) return undefined
  return transcript.messages.map((message) => ({
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
      runConfig: {
        modelConfigId: selectedModel?.id ?? '',
        permissionMode: props.permissionMode,
        selectedSkillIds: props.selectedSkillIds,
        selectedAssets: props.selectedAssets,
      },
    })
  }, [
    props.sessionId,
    selectedModel?.id,
    props.permissionMode,
    props.selectedSkillIds,
    props.selectedAssets,
    props.transcript,
  ])

  const history = useMemo<ThreadHistoryAdapter>(() => {
    const messages = fromAgUiMessages(
      toTranscriptAGUIMessages(props.transcript) ??
        toAGUIMessages(props.messages),
      { showThinking: true }
    )

    return {
      async load() {
        return ExportedMessageRepository.fromArray(messages)
      },
      async append() {
        // The backend persists messages as part of the AG-UI run. History is
        // read from the session endpoint when a thread is opened.
      },
    }
  }, [props.sessionId, props.messages, props.transcript])

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
      />
      <StudioChatSurface
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
  ...props
}: Props & {
  availableModels: StudioModel[]
  modelReady: boolean
  selectedModel?: StudioModel
  runError?: string
}) {
  const aui = useAui()
  const messages = useAuiState((state) => state.thread.messages)
  const isRunning = useAuiState((state) => state.thread.isRunning)
  const isEmpty = useAuiState((state) => state.thread.isEmpty)
  const send = (text: string) => {
    if (!modelReady || isRunning || !text.trim()) return
    const composer = aui.thread.composer()
    composer.setText(text)
    composer.send()
  }

  return (
    <div className='flex min-h-0 flex-1 flex-col'>
      <Conversation className='min-h-0 flex-1'>
        <ConversationContent className='mx-auto min-h-full w-full max-w-3xl px-5 py-8'>
          {isEmpty ? <StudioWelcome onSelect={send} /> : null}
          {messages.map((message) => (
            <StudioMessage
              key={message.id}
              message={message}
              isRunning={isRunning}
            />
          ))}
          {runError ? (
            <div
              role='alert'
              className='max-w-[88%] rounded-xl border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm leading-6 text-destructive'
            >
              {runError}
            </div>
          ) : null}
        </ConversationContent>
        <ConversationScrollButton aria-label='跳转至最新消息' />
      </Conversation>
      <div className='shrink-0 px-4 pt-2 pb-5'>
        <PromptInput
          className='mx-auto max-w-3xl'
          onSubmit={({ text }) => send(text)}
        >
          <PromptInputBody>
            <PromptInputTextarea
              autoFocus
              disabled={!modelReady}
              placeholder={
                modelReady
                  ? '描述你想创作的内容，或让 Agent 调用工作流…'
                  : '先在 AI 设置中添加并启用模型'
              }
            />
          </PromptInputBody>
          <PromptInputFooter>
            <PromptInputTools>
              <ModelPicker
                models={availableModels}
                value={selectedModel?.id}
                onChange={props.onModelChange}
              />
              <SkillPicker
                skills={props.skills}
                value={props.selectedSkillIds}
                onChange={props.onSkillChange}
              />
              <AssetPicker
                assets={props.assets}
                value={props.selectedAssets}
                onChange={props.onAssetChange}
                onImportLibraryAsset={props.onImportLibraryAsset}
              />
              <PermissionPicker
                value={props.permissionMode}
                onChange={props.onPermissionChange}
              />
            </PromptInputTools>
            <PromptInputSubmit
              aria-label={isRunning ? '停止生成' : '发送消息'}
              disabled={!modelReady}
              onStop={() => aui.thread.cancelRun()}
              status={isRunning ? 'streaming' : undefined}
            />
          </PromptInputFooter>
        </PromptInput>
        <p className='mx-auto mt-2 max-w-3xl text-center text-xs text-muted-foreground'>
          {modelReady
            ? 'Agent 可能会调用模型、Skill、连接器和工作流，请核对重要结果。'
            : '没有可用模型时，无法发起 Agent 对话。'}
        </p>
      </div>
    </div>
  )
}

function StudioRunCompletionWatcher({
  onRunStarted,
  onRunFinished,
}: {
  onRunStarted?: () => void
  onRunFinished?: () => void
}) {
  const isRunning = useAuiState((state) => state.thread.isRunning)
  const wasRunning = useRef(false)

  useEffect(() => {
    if (!wasRunning.current && isRunning) {
      onRunStarted?.()
    }
    if (wasRunning.current && !isRunning) {
      onRunFinished?.()
    }
    wasRunning.current = isRunning
  }, [isRunning, onRunStarted, onRunFinished])
  return null
}

function AssetPicker({
  assets,
  value,
  onChange,
  onImportLibraryAsset,
}: {
  assets: StudioAsset[]
  value: SelectedAsset[]
  onChange: (assets: SelectedAsset[]) => void
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
  const selected = new Map(
    value.map((item) => [item.assetId, item.assetVersionId])
  )
  const toggle = async (asset: StudioAsset, fromLibrary = false) => {
    const version = asset.versions[asset.versions.length - 1]
    if (!version) return
    if (selected.get(asset.id) === version.id) {
      onChange(value.filter((item) => item.assetId !== asset.id))
      return
    }
    let next = { assetId: asset.id, assetVersionId: version.id }
    if (fromLibrary) {
      const imported = await onImportLibraryAsset(next)
      const importedVersion = imported.versions[imported.versions.length - 1]
      if (!importedVersion) return
      next = { assetId: imported.id, assetVersionId: importedVersion.id }
    }
    onChange([...value.filter((item) => item.assetId !== asset.id), next])
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
        <Button variant='ghost' size='sm' className='rounded-lg'>
          <Paperclip />
          {value.length === 0 ? '资产' : `资产 · ${value.length}`}
          <ChevronDown className='size-3.5' />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align='start' className='w-72'>
        <DropdownMenuLabel>本轮使用的资产</DropdownMenuLabel>
        <DropdownMenuSeparator />
        <AssetPickerSection
          label='当前 Session'
          assets={currentAssets}
          selected={selected}
          onToggle={toggle}
          emptyText='当前 Session 还没有资产'
        />
        <DropdownMenuSeparator />
        <AssetPickerSection
          label='资产库'
          assets={reusableAssets}
          selected={selected}
          onToggle={(asset) => void toggle(asset, true)}
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
  selected,
  onToggle,
  emptyText,
}: {
  label: string
  assets: StudioAsset[]
  selected: Map<string, string>
  onToggle: (asset: StudioAsset) => void | Promise<void>
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
            <DropdownMenuCheckboxItem
              key={asset.id}
              checked={selected.get(asset.id) === version?.id}
              onSelect={(event) => event.preventDefault()}
              onCheckedChange={() => void onToggle(asset)}
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
            </DropdownMenuCheckboxItem>
          )
        })
      )}
    </>
  )
}

function SkillPicker({
  skills,
  value,
  onChange,
}: {
  skills: StudioSkill[]
  value: string[]
  onChange: (ids: string[]) => void
}) {
  const selected = new Set(value)
  const enabledSkills = skills.filter((skill) => skill.enabled)
  const toggle = (id: string) =>
    onChange(
      selected.has(id)
        ? value.filter((selectedID) => selectedID !== id)
        : [...value, id]
    )
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant='ghost' size='sm' className='rounded-lg'>
          <Sparkles />
          {value.length === 0 ? 'Skills' : `Skills · ${value.length}`}
          <ChevronDown className='size-3.5' />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align='start' className='w-72'>
        <DropdownMenuLabel>本轮使用的 Skills</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {enabledSkills.length === 0 ? (
          <DropdownMenuItem disabled>没有已启用的 Skill</DropdownMenuItem>
        ) : null}
        {enabledSkills.map((skill) => (
          <DropdownMenuCheckboxItem
            key={skill.id}
            checked={selected.has(skill.id)}
            onSelect={(event) => event.preventDefault()}
            onCheckedChange={() => toggle(skill.id)}
          >
            <span className='min-w-0 flex-1'>
              <span className='block truncate'>{skill.name}</span>
              <span className='block truncate text-xs text-muted-foreground'>
                {skill.description}
              </span>
            </span>
          </DropdownMenuCheckboxItem>
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

function StudioMessage({
  message,
  isRunning,
}: {
  message: ReturnType<typeof useAuiState>['thread']['messages'][number]
  isRunning: boolean
}) {
  if (message.role !== 'user' && message.role !== 'assistant') return null
  return (
    <Message from={message.role}>
      <MessageContent>
        {message.parts.map((part, index) => {
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
              <Reasoning isStreaming={streaming} key={`${message.id}-${index}`}>
                <ReasoningTrigger
                  getThinkingMessage={(active) =>
                    active ? '正在思考' : '思考过程'
                  }
                />
                <ReasoningContent>{part.text}</ReasoningContent>
              </Reasoning>
            )
          }
          if (part.type === 'tool-call') {
            return <StudioToolCall key={`${message.id}-${index}`} part={part} />
          }
          return null
        })}
      </MessageContent>
    </Message>
  )
}

function StudioToolCall({
  part,
}: {
  part: Extract<
    ReturnType<
      typeof useAuiState
    >['thread']['messages'][number]['parts'][number],
    { type: 'tool-call' }
  >
}) {
  const state = part.isError
    ? 'output-error'
    : part.result !== undefined
      ? 'output-available'
      : part.status.type === 'running'
        ? 'input-available'
        : 'input-streaming'
  return (
    <Tool defaultOpen={state === 'output-error'}>
      <ToolHeader
        state={state}
        title={part.toolName}
        toolName={part.toolName}
        type='dynamic-tool'
      />
      <ToolContent>
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
    <ModelSelector>
      <ModelSelectorTrigger asChild>
        <PromptInputButton className='max-w-52'>
          <span className='truncate'>{selected?.name ?? '未选择模型'}</span>
          <ChevronDown className='size-3.5' />
        </PromptInputButton>
      </ModelSelectorTrigger>
      <ModelSelectorContent title='选择模型'>
        <ModelSelectorInput placeholder='筛选模型' />
        <ModelSelectorList>
          <ModelSelectorEmpty>没有可用模型</ModelSelectorEmpty>
          <ModelSelectorGroup heading='可用模型'>
            {models.map((model) => (
              <ModelSelectorItem
                key={model.id}
                onSelect={() => onChange(model.id)}
                value={model.name}
              >
                <ModelSelectorName>{model.name}</ModelSelectorName>
                {model.id === value ? <Check className='size-3.5' /> : null}
              </ModelSelectorItem>
            ))}
          </ModelSelectorGroup>
        </ModelSelectorList>
      </ModelSelectorContent>
    </ModelSelector>
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
        <Button variant='ghost' size='sm' className='rounded-lg'>
          <ShieldCheck />
          {permissionLabels[value]}
          <ChevronDown className='size-3.5' />
        </Button>
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
