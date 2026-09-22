import { useEffect, useMemo, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  AssistantRuntimeProvider,
  ComposerPrimitive,
  ExportedMessageRepository,
  MessagePrimitive,
  type ThreadHistoryAdapter,
  ThreadPrimitive,
  useAuiState,
  useMessagePartReasoning,
  useMessagePartText,
  useAui,
} from '@assistant-ui/react'
import { fromAgUiMessages, useAgUiRuntime } from '@assistant-ui/react-ag-ui'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import {
  Bot,
  Check,
  ChevronDown,
  LoaderCircle,
  Paperclip,
  Send,
  ShieldCheck,
  Sparkles,
  Square,
  User,
} from 'lucide-react'
import { baseURL } from '@/lib/api/client'
import { StudioWebSocketAgent } from '@/lib/agui-websocket-agent'
import {
  listStudioLibraryAssets,
  type StudioAsset,
  type StudioMessage,
  type StudioModel,
  type StudioPermissionMode,
  type StudioSkill,
  type StudioTranscript,
} from '@/lib/api/studio'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
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
      initialMessages: (toTranscriptAGUIMessages(props.transcript) ?? toAGUIMessages(props.messages)) as never[],
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
      toTranscriptAGUIMessages(props.transcript) ?? toAGUIMessages(props.messages),
      { showThinking: true },
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
      <ThreadPrimitive.Root className='flex min-h-0 flex-1 flex-col'>
        <ThreadPrimitive.Viewport className='min-h-0 flex-1 overflow-y-auto scroll-smooth'>
          <div className='mx-auto flex min-h-full w-full max-w-3xl flex-col px-5 py-8'>
            <ThreadPrimitive.Empty>
              <StudioWelcome />
            </ThreadPrimitive.Empty>
            <ThreadPrimitive.Messages
              components={{
                UserMessage: StudioUserMessage,
                AssistantMessage: StudioAssistantMessage,
              }}
            />
            {runError ? (
              <div
                role='alert'
                className='mb-6 ml-11 max-w-[88%] rounded-xl border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm leading-6 text-destructive'
              >
                {runError}
              </div>
            ) : null}
            <ThreadPrimitive.If running>
              <div className='mt-1 flex items-center gap-2 pl-11 text-xs font-medium text-muted-foreground'>
                <LoaderCircle className='size-3.5 animate-spin text-primary' />
                <span>正在生成</span>
                <span className='flex gap-0.5' aria-hidden='true'>
                  <span className='size-1 animate-pulse rounded-full bg-primary [animation-delay:-300ms]' />
                  <span className='size-1 animate-pulse rounded-full bg-primary [animation-delay:-150ms]' />
                  <span className='size-1 animate-pulse rounded-full bg-primary' />
                </span>
              </div>
            </ThreadPrimitive.If>
            <div className='min-h-6 flex-1' />
          </div>
        </ThreadPrimitive.Viewport>
        <div className='shrink-0 px-4 pt-2 pb-5'>
          <ComposerPrimitive.Root className='mx-auto w-full max-w-3xl rounded-2xl border bg-card p-2 focus-within:border-ring focus-within:ring-3 focus-within:ring-ring/20'>
            <ComposerPrimitive.Input
              autoFocus
              disabled={!modelReady}
              rows={3}
              placeholder={
                modelReady
                  ? '描述你想创作的内容，或让 Agent 调用工作流…'
                  : '先在 AI 设置中添加并启用模型'
              }
              className='max-h-40 min-h-20 w-full resize-none bg-transparent px-3 py-2 text-sm leading-6 outline-none placeholder:text-muted-foreground'
            />
            <div className='flex flex-wrap items-center justify-between gap-2 px-1 pb-1'>
              <div className='flex flex-wrap items-center gap-1'>
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
              </div>
              <ThreadPrimitive.If running>
                <ComposerPrimitive.Cancel asChild>
                  <Button size='icon' variant='outline' className='rounded-xl' aria-label='停止生成'>
                    <Square className='fill-current' />
                  </Button>
                </ComposerPrimitive.Cancel>
              </ThreadPrimitive.If>
              <ThreadPrimitive.If running={false}>
                <ComposerPrimitive.Send asChild>
                  <Button
                    size='icon'
                    className='rounded-xl'
                    aria-label='发送消息'
                    disabled={!modelReady}
                  >
                    <Send />
                  </Button>
                </ComposerPrimitive.Send>
              </ThreadPrimitive.If>
            </div>
          </ComposerPrimitive.Root>
          <p className='mx-auto mt-2 max-w-3xl text-center text-xs text-muted-foreground'>
            {modelReady
              ? 'Agent 可能会调用模型、Skill、连接器和工作流，请核对重要结果。'
              : '没有可用模型时，无法发起 Agent 对话。'}
          </p>
        </div>
      </ThreadPrimitive.Root>
    </AssistantRuntimeProvider>
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
  const selected = new Map(value.map((item) => [item.assetId, item.assetVersionId]))
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
    onChange([
      ...value.filter((item) => item.assetId !== asset.id),
      next,
    ])
  }
  const unavailableSelections = value.filter((selection) => {
    const asset = assetsByID.get(selection.assetId)
    return !asset || !asset.versions.some((version) => version.id === selection.assetVersionId)
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
          emptyText={libraryAssets.isLoading ? '正在读取资产库…' : '资产库还没有可复用资产'}
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
                  {asset.kind === 'image' ? '图片' : asset.kind === 'document' ? '文档' : '文件'}{version ? ` · v${version.version}` : ''}
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

function StudioWelcome() {
  const aui = useAui()
  return (
    <div className='flex flex-1 flex-col items-center justify-center py-16 text-center'>
      <span className='mb-5 flex size-12 items-center justify-center rounded-2xl bg-primary text-primary-foreground'>
        <Bot className='size-6' />
      </span>
      <h2 className='text-xl font-semibold tracking-tight'>从一个想法开始</h2>
      <p className='mt-2 max-w-md text-sm leading-6 text-muted-foreground'>
        和 Agent
        一起构思内容、生成资产，或让它调用现有工作流。每一项产出都会沉淀在当前
        Session 的资产路线中。
      </p>
      <div className='mt-6 grid w-full max-w-xl gap-2 sm:grid-cols-2'>
        {['为雨夜侦探构思漫画并生成分镜', '根据一张角色图生成三视图'].map(
          (prompt) => (
            <button
              key={prompt}
              type='button'
              onClick={() => {
                const composer = aui.thread.composer()
                composer.setText(prompt)
                composer.send()
              }}
              className='min-h-11 rounded-xl border bg-card px-4 py-3 text-left text-sm transition-colors hover:bg-accent'
            >
              {prompt}
            </button>
          )
        )}
      </div>
    </div>
  )
}

function StudioUserMessage() {
  return (
    <MessagePrimitive.Root className='mb-6 flex justify-end gap-3'>
      <div className='max-w-[82%] rounded-2xl rounded-tr-md bg-primary px-4 py-3 text-sm leading-6 text-primary-foreground'>
        <MessagePrimitive.Parts />
      </div>
      <Avatar className='mt-1 size-8'>
        <AvatarFallback>
          <User className='size-4' />
        </AvatarFallback>
      </Avatar>
    </MessagePrimitive.Root>
  )
}

function StudioAssistantMessage() {
  return (
    <MessagePrimitive.Root className='mb-7 flex gap-3'>
      <Avatar className='mt-1 size-8 border'>
        <AvatarFallback>
          <Bot className='size-4' />
        </AvatarFallback>
      </Avatar>
      <div className='max-w-[88%] min-w-0 pt-1 text-sm leading-7'>
        <MessagePrimitive.Parts
          components={{ Text: StudioMarkdown, Reasoning: StudioReasoning }}
        />
      </div>
    </MessagePrimitive.Root>
  )
}

function StudioMarkdown() {
  const text = useMessagePartText().text
  return (
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      components={{
        h1: ({ children }) => (
          <h1 className='mb-3 text-xl font-semibold leading-tight'>{children}</h1>
        ),
        h2: ({ children }) => (
          <h2 className='mb-2 mt-4 text-lg font-semibold leading-tight'>{children}</h2>
        ),
        h3: ({ children }) => (
          <h3 className='mb-2 mt-3 font-semibold leading-tight'>{children}</h3>
        ),
        p: ({ children }) => <p className='mb-3 last:mb-0'>{children}</p>,
        ul: ({ children }) => (
          <ul className='mb-3 list-disc space-y-1 pl-5'>{children}</ul>
        ),
        ol: ({ children }) => (
          <ol className='mb-3 list-decimal space-y-1 pl-5'>{children}</ol>
        ),
        blockquote: ({ children }) => (
          <blockquote className='mb-3 border-s-2 ps-3 text-muted-foreground'>
            {children}
          </blockquote>
        ),
        pre: ({ children }) => (
          <pre className='mb-3 overflow-x-auto rounded-lg bg-muted p-3 text-xs leading-5'>
            {children}
          </pre>
        ),
        code: ({ children, className }) =>
          className ? (
            <code className='font-mono'>{children}</code>
          ) : (
            <code className='rounded bg-muted px-1.5 py-0.5 font-mono text-[0.9em]'>
              {children}
            </code>
          ),
        a: ({ children, href }) => (
          <a
            href={href}
            target='_blank'
            rel='noreferrer'
            className='text-primary underline underline-offset-4'
          >
            {children}
          </a>
        ),
      }}
    >
      {text}
    </ReactMarkdown>
  )
}

function StudioReasoning() {
  const reasoning = useMessagePartReasoning()
  if (!reasoning.text.trim()) return null
  const running = reasoning.status.type === 'running'
  return (
    <details open={running} className='mb-3 rounded-xl border bg-muted/30 px-3 py-2 text-xs leading-5'>
      <summary className='cursor-pointer select-none font-medium text-muted-foreground'>
        {running ? '正在思考' : '思考过程'}
      </summary>
      <p className='mt-2 whitespace-pre-wrap text-muted-foreground'>{reasoning.text}</p>
    </details>
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
        <Button variant='ghost' size='sm' className='max-w-52 rounded-lg'>
          <span className='truncate'>{selected?.name ?? '未选择模型'}</span>
          <ChevronDown className='size-3.5' />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align='start' className='w-64'>
        <DropdownMenuLabel>本次对话使用的模型</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {models.length === 0 ? (
          <DropdownMenuItem disabled>
            还没有可用模型
          </DropdownMenuItem>
        ) : (
          <DropdownMenuRadioGroup value={value} onValueChange={onChange}>
            {models.map((model) => (
                <DropdownMenuRadioItem key={model.id} value={model.id}>
                  <span className='min-w-0 flex-1 truncate'>{model.name}</span>
                  {model.default ? <Check className='size-3.5' /> : null}
                </DropdownMenuRadioItem>
              ))}
          </DropdownMenuRadioGroup>
        )}
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
