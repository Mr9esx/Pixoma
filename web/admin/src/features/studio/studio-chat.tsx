import { useEffect, useMemo, useRef, useState } from 'react'
import type { FileUIPart } from 'ai'
import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
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
  type AgUiInterrupt,
  useAgUiInterrupts,
  useAgUiRuntime,
  useAgUiSubmitInterruptResponses,
} from '@assistant-ui/react-ag-ui'
import {
  ArrowUp,
  Bot,
  Boxes,
  ChevronDown,
  CircleHelp,
  Copy,
  ImagePlus,
  ShieldCheck,
  Sparkles,
  Square,
  Workflow,
  X,
} from 'lucide-react'
import { useStickToBottomContext } from 'use-stick-to-bottom'
import { StudioWebSocketAgent } from '@/lib/agui-websocket-agent'
import { baseURL } from '@/lib/api/client'
import {
  cancelStudioRun,
  listStudioLibraryAssets,
  listStudioAgentWorkflows,
  type StudioAsset,
  type StudioComposerPart,
  type StudioMessage,
  type StudioModel,
  type StudioPermissionMode,
  type StudioPendingApproval,
  type StudioPendingClarification,
  type StudioRun,
  type StudioRunProgress,
  type StudioSkillSummary,
  type StudioTranscript,
  type StudioWorkflowRequest,
  type StudioAgentWorkflow,
} from '@/lib/api/studio'
import { StudioRunConnection } from '@/lib/studio-run-connection'
import { cn } from '@/lib/utils'
import { AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import {
  Confirmation,
  ConfirmationAction,
  ConfirmationActions,
  ConfirmationRequest,
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
  usePromptInputAttachments,
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
import { retainStudioComposerFocus } from './studio-composer-focus'
import { StudioReferenceBadge } from './studio-reference-badge'
import { StudioTurnNavigator } from './studio-turn-navigator'
import { StudioWorkflowCard } from './studio-workflow-card'

type Props = {
  sessionId: string
  messages: StudioMessage[]
  transcript?: StudioTranscript
  latestRun?: StudioRun | null
  runProgress?: StudioRunProgress | null
  pendingApprovals?: StudioPendingApproval[]
  pendingClarifications?: StudioPendingClarification[]
  models: StudioModel[]
  modelConfigId?: string
  permissionMode: StudioPermissionMode
  skills: StudioSkillSummary[]
  assets: StudioAsset[]
  locateMessage?: { id: string; request: number }
  selectedSkillIds: string[]
  selectedAssets: SelectedAsset[]
  onModelChange: (id: string) => void
  onPermissionChange: (mode: StudioPermissionMode) => void
  onSkillChange: (ids: string[]) => void
  onAssetChange: (assets: SelectedAsset[]) => void
  onImportLibraryAsset: (asset: SelectedAsset) => Promise<StudioAsset>
  onUploadAsset?: (file: File) => Promise<StudioAsset>
  onRunFinished?: () => void
  onRuntimeStateChange?: (running: boolean) => void
}

type SelectedAsset = { assetId: string; assetVersionId: string }
type ClarificationDraft = { id: string; selected: string; custom: string }
type ComposerPicker = 'skill' | 'asset' | 'workflow'
type PickerOpenProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

function StudioImageAttachments({ disabled }: { disabled: boolean }) {
  const attachments = usePromptInputAttachments()
  if (attachments.files.length === 0) return null
  return (
    <div className='flex w-full flex-wrap gap-2 px-3 pt-3'>
      {attachments.files.map((file) => (
        <div
          key={file.id}
          className='relative size-28 overflow-visible rounded-lg border bg-muted'
        >
          <img
            src={file.url}
            alt={file.filename ?? '已选择图片'}
            className='size-full rounded-lg object-cover'
          />
          <Button
            type='button'
            variant='secondary'
            size='icon-xs'
            aria-label={`删除图片：${file.filename ?? '图片'}`}
            className='absolute -right-2 -top-2 border'
            disabled={disabled}
            onClick={() => attachments.remove(file.id)}
          >
            <X />
          </Button>
        </div>
      ))}
    </div>
  )
}

function StudioImagePicker({
  disabled,
  unsupported,
}: {
  disabled: boolean
  unsupported: boolean
}) {
  const attachments = usePromptInputAttachments()
  const button = (
    <PromptInputButton
      aria-label='添加图片'
      tooltip={unsupported ? undefined : '添加图片'}
      size='icon-sm'
      className={unsupported ? 'pointer-events-none' : undefined}
      disabled={disabled}
      onClick={() => attachments.openFileDialog()}
    >
      <ImagePlus />
    </PromptInputButton>
  )
  if (!unsupported) return button
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span
          className='inline-flex'
          tabIndex={0}
          aria-disabled='true'
          aria-label='当前模型不支持图片输入'
        >
          {button}
        </span>
      </TooltipTrigger>
      <TooltipContent side='top'>当前模型不支持图片输入</TooltipContent>
    </Tooltip>
  )
}

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
  const [runtimeGeneration, setRuntimeGeneration] = useState(0)
  const [recoveryDraft, setRecoveryDraft] = useState<ClarificationDraft>()
  return (
    <StudioChatRuntime
      key={`${props.sessionId}:${runtimeGeneration}`}
      {...props}
      runError={runError}
      recoveryDraft={recoveryDraft}
      onRunError={setRunError}
      onReconnect={(draft) => {
        if (draft) setRecoveryDraft(draft)
        setRuntimeGeneration((current) => current + 1)
      }}
    />
  )
}

function StudioChatRuntime({
  runError,
  recoveryDraft,
  onRunError,
  onReconnect,
  ...props
}: Props & {
  runError?: string
  recoveryDraft?: ClarificationDraft
  onRunError: (message: string | undefined) => void
  onReconnect: (draft?: ClarificationDraft) => void
}) {
  const availableModels = props.models.filter(
    (model) => model.enabled && model.agent_enabled && model.capabilities.tools
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
      [
        'queued',
        'running',
        'waiting_approval',
        'waiting_clarification',
      ].includes(props.latestRun.status)
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
        props.latestRun.status === 'waiting_approval' ||
        props.latestRun.status === 'waiting_clarification')
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
    onError: (error) => onRunError(error.message),
    adapters: { history },
  })

  return (
    <AssistantRuntimeProvider runtime={runtime}>
      <StudioRunCompletionWatcher
        onRunFinished={props.onRunFinished}
        onRuntimeStateChange={props.onRuntimeStateChange}
      />
      <StudioChatSurface
        agent={agent}
        onRunError={onRunError}
        onReconnect={onReconnect}
        availableModels={availableModels}
        modelReady={modelReady}
        selectedModel={selectedModel}
        runError={runError}
        recoveryDraft={recoveryDraft}
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
  recoveryDraft,
  agent,
  onRunError,
  onReconnect,
  ...props
}: Props & {
  availableModels: StudioModel[]
  modelReady: boolean
  selectedModel?: StudioModel
  runError?: string
  recoveryDraft?: ClarificationDraft
  agent: StudioWebSocketAgent
  onRunError: (message: string | undefined) => void
  onReconnect: (draft?: ClarificationDraft) => void
}) {
  const aui = useAui()
  const messages = useAuiState((state) => state.thread.messages)
  const isRunning = useAuiState((state) => state.thread.isRunning)
  const isEmpty = useAuiState((state) => state.thread.isEmpty)
  const [answeredIds, setAnsweredIds] = useState<Set<string>>(() => new Set())
  const [submittedWorkflowIds, setSubmittedWorkflowIds] = useState<Set<string>>(() => new Set())
  const [submittingWorkflowId, setSubmittingWorkflowId] = useState<string>()
  const [submittedClarifications, setSubmittedClarifications] = useState<
    Record<
      string,
      {
        question: string
        options: ClarificationOption[]
        answer: string
        status: 'answered' | 'skipped'
        messageId?: string
      }
    >
  >({})
  const [submittingClarifications, setSubmittingClarifications] = useState<
    Set<string>
  >(() => new Set())
  const [cancellingClarification, setCancellingClarification] = useState(false)
  const composerRef = useRef<StudioComposerHandle>(null)
  const composerOverlayRef = useRef<HTMLDivElement>(null)
  const [composerValue, setComposerValue] = useState<StudioComposerValue>({
    text: '', parts: [], selectedSkillIds: [], selectedAssets: [],
  })
  const [sendingImages, setSendingImages] = useState(false)
  const sendingImagesRef = useRef(false)
  const [activePicker, setActivePicker] = useState<ComposerPicker | null>(null)
  const changePicker = (picker: ComposerPicker, open: boolean) => {
    setActivePicker((current) =>
      open ? picker : current === picker ? null : current
    )
  }
  const [liveMessage, setLiveMessage] = useState<{
    previousUserMessageId?: string
    parts: StudioComposerPart[]
  }>()
  const interrupts = useAgUiInterrupts()
  const submitInterruptResponses = useAgUiSubmitInterruptResponses()
  const fromRuntime = interrupts.length > 0
  const pendingInterrupts: readonly (
    | AgUiInterrupt
    | StudioPendingApproval
    | StudioPendingClarification
  )[] = fromRuntime
    ? interrupts
    : [
        ...(props.pendingApprovals ?? []),
        ...(props.pendingClarifications ?? []),
      ]
  const approvals = pendingInterrupts.filter(
    (interrupt) =>
      interrupt.reason !== 'input_required' && interrupt.reason !== 'workflow_input' && !answeredIds.has(interrupt.id)
  )
  const resolvedHistory = new Set(
    props.transcript?.messages.flatMap((message) =>
      message.clarification && message.clarification.status !== 'pending'
        ? [message.clarification.id]
        : []
    ) ?? []
  )
  const pendingClarifications = pendingInterrupts.filter(
    (interrupt) =>
      interrupt.reason === 'input_required' &&
      !resolvedHistory.has(interrupt.id)
  )
  const pendingWorkflows = pendingInterrupts.filter(
    (interrupt) => interrupt.reason === 'workflow_input' && !resolvedHistory.has(interrupt.id) && !submittedWorkflowIds.has(interrupt.id)
  )
  const activeWorkflowInterrupt = pendingWorkflows[0]
  const activeWorkflow = activeWorkflowInterrupt ? workflowFromInterrupt(activeWorkflowInterrupt) : undefined
  const clarifications = [
    ...pendingClarifications.map((interrupt) => ({
      id: interrupt.id,
      question: interrupt.message ?? '',
      options: clarificationOptions(interrupt),
      answer: submittedClarifications[interrupt.id]?.answer,
      status: submittedClarifications[interrupt.id]
        ? submittedClarifications[interrupt.id].status
        : 'pending' as const,
      messageId: clarificationMessageId(interrupt),
    })),
    ...Object.entries(submittedClarifications)
      .filter(
        ([id]) =>
          !pendingClarifications.some((interrupt) => interrupt.id === id) &&
          !resolvedHistory.has(id)
      )
      .map(([id, value]) => ({ id, ...value })),
  ]
  const approvalPending =
    props.latestRun?.status === 'waiting_approval' || approvals.length > 0
  const clarificationPending = clarifications.some(
    (clarification) => clarification.status === 'pending'
  )
  const workflowPending = Boolean(activeWorkflow)
  const activeClarification = clarifications.find(
    (clarification) => clarification.status === 'pending'
  )
  const serverRunning =
    props.latestRun?.status === 'queued' ||
    props.latestRun?.status === 'running'
  const runActive =
    isRunning || serverRunning || approvalPending ||
    props.latestRun?.status === 'waiting_clarification' ||
    clarificationPending || workflowPending
  const isStreaming = isRunning || serverRunning
  const respondToWorkflow = (inputs?: Record<string, unknown>) => {
    if (!activeWorkflowInterrupt || !fromRuntime) return
    onRunError(undefined)
    const id = activeWorkflowInterrupt.id
    setSubmittedWorkflowIds((current) => new Set(current).add(id))
    setSubmittingWorkflowId(id)
    void submitInterruptResponses([{
      interruptId: id,
      status: inputs ? 'resolved' : 'cancelled',
      ...(inputs ? { payload: { inputs } } : {}),
    }]).catch((error: unknown) => {
      setSubmittedWorkflowIds((current) => {
        const next = new Set(current)
        next.delete(id)
        return next
      })
      reconnectInterrupt()
      onRunError(error instanceof Error ? error.message : '提交工作流失败')
    }).finally(() => setSubmittingWorkflowId(undefined))
  }
  const reconnectInterrupt = (draft?: ClarificationDraft) => {
    props.onRunFinished?.()
    onReconnect(draft)
  }
  const cancelClarificationRun = () => {
    const runId = props.latestRun?.id
    if (!runId || cancellingClarification) return
    setCancellingClarification(true)
    void cancelStudioRun(runId)
      .then(() => {
        onRunError(undefined)
        reconnectInterrupt()
      })
      .catch((error: unknown) => {
        onRunError(error instanceof Error ? error.message : '结束运行失败')
      })
      .finally(() => setCancellingClarification(false))
  }
  const respondToApproval = (id: string, approved: boolean) => {
    onRunError(undefined)
    setAnsweredIds((current) => new Set(current).add(id))
    void submitInterruptResponses([
      {
        interruptId: id,
        status: approved ? 'resolved' : 'cancelled',
        ...(approved ? { payload: true } : {}),
      },
    ]).catch((error: unknown) => {
      setAnsweredIds((current) => {
        const next = new Set(current)
        next.delete(id)
        return next
      })
      reconnectInterrupt()
      onRunError(error instanceof Error ? error.message : '提交审批失败')
    })
  }
  const respondToClarification = (
    id: string,
    selected: string,
    custom: string,
    answer: string
  ) => {
    const clarification = clarifications.find((item) => item.id === id)
    if (!clarification) return
    onRunError(undefined)
    setSubmittedClarifications((current) => ({
      ...current,
      [id]: {
        question: clarification.question,
        options: clarification.options,
        answer,
        status: selected === 'skip' ? 'skipped' : 'answered',
        messageId: clarification.messageId,
      },
    }))
    setSubmittingClarifications((current) => new Set(current).add(id))
    void submitInterruptResponses([
      {
        interruptId: id,
        status: 'resolved',
        payload: { selected, ...(selected === 'other' ? { custom } : {}) },
      },
    ])
      .catch((error: unknown) => {
        setSubmittedClarifications((current) => {
          const next = { ...current }
          delete next[id]
          return next
        })
        reconnectInterrupt({ id, selected, custom })
        onRunError(error instanceof Error ? error.message : selected === 'skip' ? '跳过问题失败' : '提交回答失败')
      })
      .finally(() => {
        setSubmittingClarifications((current) => {
          const next = new Set(current)
          next.delete(id)
          return next
        })
      })
  }
  const send = async (prompt?: string, files: FileUIPart[] = []) => {
    const value = prompt === undefined
      ? composerRef.current?.serialize()
      : { text: prompt, parts: [{ type: 'text' as const, text: prompt }], selectedSkillIds: [], selectedAssets: [] }
    if (
      !modelReady ||
      runActive ||
      sendingImagesRef.current ||
      !value ||
      (!value.text.trim() && files.length === 0)
    ) {
      if (files.length > 0) {
        onRunError('当前无法发送图片，请稍后重试。')
        throw new Error('当前无法发送图片')
      }
      return
    }
    onRunError(undefined)
    const parts = [...value.parts]
    const selectedAssets = [...value.selectedAssets]
    let text = value.text
    if (!text.trim() && files.length > 0) {
      text = '请分析这张图片'
      parts.push({ type: 'text', text })
    }
    if (files.length > 0) {
      if (!selectedModel?.capabilities.vision) {
        onRunError('当前模型未开启图片输入，请切换模型或在 AI 设置中开启。')
        throw new Error('当前模型未开启图片输入')
      }
      if (!props.onUploadAsset) {
        onRunError('当前会话无法上传图片。')
        throw new Error('当前会话无法上传图片')
      }
      sendingImagesRef.current = true
      setSendingImages(true)
      try {
        const images = await Promise.all(
          files.map(async (attachment) => {
            if (!attachment.url || !attachment.filename || !attachment.mediaType) {
              throw new Error('图片内容不完整，请重新选择。')
            }
            const blob = await fetch(attachment.url).then((response) =>
              response.blob()
            )
            return new File([blob], attachment.filename, {
              type: attachment.mediaType,
            })
          })
        )
        if (images.reduce((total, image) => total + image.size, 0) > 20 << 20) {
          throw new Error('图片总大小不能超过 20 MB。')
        }
        for (const image of images) {
          const asset = await props.onUploadAsset(image)
          const version = asset.versions.find(
            (item) => item.version === asset.current_version
          )
          if (!version) throw new Error('上传的图片缺少版本信息。')
          parts.push({ type: 'text', text: '\n' })
          parts.push({
            type: 'asset_ref',
            asset_id: asset.id,
            asset_version_id: version.id,
            name: asset.name,
          })
          text += `\n「${asset.name}」资产`
          selectedAssets.push({ assetId: asset.id, assetVersionId: version.id })
        }
      } catch (error) {
        onRunError(error instanceof Error ? error.message : '上传图片失败。')
        throw error
      } finally {
        sendingImagesRef.current = false
        setSendingImages(false)
      }
    }
    agent.prepareNextRun({
      modelConfigId: selectedModel?.id ?? '',
      permissionMode: props.permissionMode,
      selectedSkillIds: value.selectedSkillIds,
      selectedAssets,
      messageParts: parts,
    })
    const composer = aui.thread.composer()
    composer.setText(text)
    composer.send()
    setLiveMessage({ previousUserMessageId: latestUserMessageId, parts })
    composerRef.current?.clear()
  }

  const transcriptParts = new Map(
    props.transcript?.messages.filter((message) => message.role === 'user' && message.parts).map((message) => [message.id, message.parts!]) ?? []
  )
  const clarificationCards = new Map<string, LiveClarification>()
  for (const message of props.transcript?.messages ?? []) {
    const clarification = message.clarification
    if (!clarification) continue
    clarificationCards.set(clarification.id, {
      id: clarification.id,
      question: clarification.question,
      options: clarification.options.map((label, index) => ({ id: String(index), label })),
      answer: clarification.status === 'answered'
        ? clarification.answer?.trim() || clarification.options.find((_, index) => String(index) === clarification.selected)
        : undefined,
      status: clarification.status,
      workflow: clarification.workflow,
      messageId: message.id,
    })
  }
  for (const clarification of clarifications) {
    const stored = clarificationCards.get(clarification.id)
    clarificationCards.set(clarification.id, {
      ...clarification,
      answer: clarification.answer?.trim() || stored?.answer,
    })
  }
  for (const interrupt of pendingWorkflows) {
    const workflow = workflowFromInterrupt(interrupt)
    clarificationCards.set(interrupt.id, {
      id: interrupt.id, question: workflow.name, options: [], status: 'pending',
      workflow, messageId: clarificationMessageId(interrupt),
    })
  }
  const clarificationParts = new Map<string, Map<number, LiveClarification>>()
  const renderedClarificationIds = new Set<string>()
  const clarificationValues = [...clarificationCards.values()]
  for (const message of messages) {
    if (message.role !== 'assistant') continue
    message.parts.forEach((part, index) => {
      if (part.type !== 'text') return
      const matching = clarificationValues.filter((item) =>
        !renderedClarificationIds.has(item.id) && item.question === part.text
      )
      const clarification = matching.find((item) => item.messageId === message.id) ?? matching[0]
      if (!clarification) return
      const parts = clarificationParts.get(message.id) ?? new Map()
      parts.set(index, clarification)
      clarificationParts.set(message.id, parts)
      renderedClarificationIds.add(clarification.id)
    })
  }
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
      const text = message.parts.flatMap((part, index) =>
        part.type === 'text' && !clarificationParts.get(message.id)?.has(index)
          ? [part.text]
          : []
      ).join('')
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
    <div className='studio-chat-root relative flex min-h-0 flex-1 flex-col'>
      <Conversation className='min-h-0 flex-1'>
        <ConversationContent
          scrollClassName='studio-scrollbar'
          className={cn(
            'mx-auto min-h-full w-full max-w-3xl gap-5 px-5 pt-8',
            clarificationPending || workflowPending ? 'pb-8' : 'pb-44'
          )}
        >
          {isEmpty && approvals.length === 0 && clarifications.length === 0 && pendingWorkflows.length === 0 ? (
            <StudioWelcome onSelect={send} />
          ) : null}
          {messages.map((message) => (
            <StudioMessage
              key={message.id}
              message={message}
              isRunning={isRunning}
              assistantCopyText={assistantCopyText.get(message.id)}
              clarificationParts={clarificationParts.get(message.id)}
              referenceParts={transcriptParts.get(message.id) ?? (
                message.id === latestUserMessageId && message.id !== liveMessage?.previousUserMessageId
                  ? liveMessage?.parts
                  : undefined
              )}
              assets={props.assets}
            />
          ))}
          <StudioConfirmations
            approvals={approvals}
            fromRuntime={fromRuntime}
            onRespond={respondToApproval}
          />
          {clarifications.filter((clarification) =>
            clarification.status === 'pending' &&
            !renderedClarificationIds.has(clarification.id)
          ).map((clarification) => (
            <Message key={clarification.id} from='assistant' className='max-w-none'>
              <MessageContent className='w-full'>
                <StudioClarificationRecord
                  question={clarification.question}
                  answer={clarification.answer}
                  status={clarification.status}
                />
              </MessageContent>
            </Message>
          ))}
          {pendingWorkflows.filter((interrupt) => !renderedClarificationIds.has(interrupt.id)).map((interrupt) => (
            <Message key={interrupt.id} from='assistant' className='max-w-none'>
              <MessageContent className='w-full'>
                <StudioWorkflowRecord workflow={workflowFromInterrupt(interrupt)} status='pending' />
              </MessageContent>
            </Message>
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
        <StudioMessageLocator
          target={props.locateMessage}
          messageCount={messages.length}
        />
        <StudioTurnNavigator messages={messages} composerRef={composerOverlayRef} />
        <ConversationScrollButton
          aria-label='跳转至最新消息'
          className={clarificationPending || workflowPending ? 'bottom-6' : 'bottom-48'}
        />
      </Conversation>
      {activeClarification ? (
        <div className='z-20 shrink-0 bg-card px-5 pb-5 pt-2'>
          <div className='mx-auto w-full max-w-3xl'>
            <StudioClarificationComposer
              key={activeClarification.id}
              id={activeClarification.id}
              question={activeClarification.question}
              options={activeClarification.options}
              initialSelected={recoveryDraft?.id === activeClarification.id ? recoveryDraft.selected : ''}
              initialCustom={recoveryDraft?.id === activeClarification.id ? recoveryDraft.custom : ''}
              disabled={!fromRuntime || submittingClarifications.has(activeClarification.id)}
              submitting={submittingClarifications.has(activeClarification.id)}
              onReconnect={runError && !fromRuntime ? reconnectInterrupt : undefined}
              onCancel={runError && props.latestRun?.status === 'waiting_clarification' ? cancelClarificationRun : undefined}
              cancelling={cancellingClarification}
              onSubmit={(selected, custom, answer) =>
                respondToClarification(activeClarification.id, selected, custom, answer)
              }
            />
          </div>
        </div>
      ) : null}
      {activeWorkflow ? (
        <div className='z-20 shrink-0 bg-card px-5 pb-5 pt-2'>
          <div className='mx-auto w-full max-w-3xl'>
            <StudioWorkflowCard
              key={activeWorkflowInterrupt?.id ?? activeWorkflow.id}
              workflow={activeWorkflow}
              assets={props.assets}
              onUploadAsset={props.onUploadAsset}
              disabled={!fromRuntime || Boolean(submittingWorkflowId)}
              submitting={Boolean(submittingWorkflowId)}
              onSubmit={respondToWorkflow}
              onSkip={() => respondToWorkflow()}
            />
          </div>
        </div>
      ) : null}
      <div
        ref={composerOverlayRef}
        data-slot='studio-composer'
        className={cn(
          'pointer-events-none absolute inset-x-0 bottom-0 z-20',
          (clarificationPending || workflowPending) && 'hidden'
        )}
      >
        <div className='mx-auto w-full max-w-3xl px-5'>
          <div
            aria-hidden='true'
            className='h-6 bg-gradient-to-t from-card to-transparent'
          />
          <div className='flex flex-col bg-card pb-1'>
            <div className='pointer-events-auto'>
              <PromptInput
                aria-busy={sendingImages}
                accept='image/png,image/jpeg,image/webp,image/gif'
                multiple
                maxFiles={4}
                maxFileSize={8 << 20}
                inputGroupClassName='h-auto overflow-visible bg-background'
                onError={(error) =>
                  onRunError(
                    error.code === 'max_file_size'
                      ? '单张图片不能超过 8 MB。'
                      : error.code === 'max_files'
                        ? '最多添加 4 张图片。'
                        : '请选择 PNG、JPEG、WebP 或 GIF 图片。'
                  )
                }
                onSubmit={({ files }) => send(undefined, files)}
              >
                <PromptInputBody>
                  <StudioImageAttachments disabled={sendingImages} />
                  <StudioComposer
                    ref={composerRef}
                    disabled={!modelReady || approvalPending || clarificationPending}
                    slashItems={slashItems}
                    onSubmit={() =>
                      composerOverlayRef.current
                        ?.querySelector('form')
                        ?.requestSubmit()
                    }
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
                <PromptInputFooter className='flex-wrap sm:flex-nowrap'>
                  <PromptInputTools>
                    <StudioImagePicker
                      unsupported={Boolean(
                        selectedModel && !selectedModel.capabilities.vision
                      )}
                      disabled={
                        !modelReady ||
                        !selectedModel?.capabilities.vision ||
                        sendingImages
                      }
                    />
                    <SkillPicker
                      open={activePicker === 'skill'}
                      onOpenChange={(open) => changePicker('skill', open)}
                      skills={props.skills}
                      value={composerValue.selectedSkillIds}
                      onInsert={(skill) => composerRef.current?.insertReference({ kind: 'skill', id: skill.id, label: skill.name })}
                    />
                    <AssetPicker
                      open={activePicker === 'asset'}
                      onOpenChange={(open) => changePicker('asset', open)}
                      assets={props.assets}
                      value={composerValue.selectedAssets}
                      onInsert={(asset, versionId) => composerRef.current?.insertReference({ kind: 'asset', id: asset.id, label: asset.name, versionId })}
                      onImportLibraryAsset={props.onImportLibraryAsset}
                    />
                    <WorkflowPicker
                      open={activePicker === 'workflow'}
                      onOpenChange={(open) => changePicker('workflow', open)}
                      disabled={runActive}
                      onSelect={(workflow) => composerRef.current?.insertReference({ kind: 'workflow', id: workflow.id, label: workflow.name })}
                    />
                    <PermissionPicker
                      value={props.permissionMode}
                      onChange={props.onPermissionChange}
                    />
                  </PromptInputTools>
                  <PromptInputTools className='max-sm:w-full max-sm:justify-between'>
                    <ModelPicker
                      models={availableModels}
                      value={selectedModel?.id}
                      onChange={props.onModelChange}
                    />
                    <PromptInputSubmit
                      className='group/send h-8 min-h-8 min-w-0 justify-center bg-transparent p-0 text-transparent shadow-none hover:bg-transparent active:bg-transparent'
                      size='sm'
                      aria-label={
                        sendingImages
                          ? '正在上传图片'
                          : isStreaming
                            ? '停止生成'
                            : '发送消息'
                      }
                      disabled={!modelReady || sendingImages || (runActive && !isStreaming)}
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
                    >
                      <span className='inline-flex h-8 items-center gap-1.5 rounded-md bg-primary px-2.5 text-primary-foreground transition-colors group-hover/send:bg-primary/90 group-active/send:bg-primary/80'>
                        {sendingImages ? (
                          <span>正在上传图片…</span>
                        ) : isStreaming ? (
                          <>
                            <span>停止</span>
                            <Square data-icon='inline-end' />
                          </>
                        ) : (
                          <>
                            <span>发送</span>
                            <ArrowUp data-icon='inline-end' />
                          </>
                        )}
                      </span>
                    </PromptInputSubmit>
                  </PromptInputTools>
                </PromptInputFooter>
              </PromptInput>
            </div>
            <p className='mt-1 text-center text-[10px] text-muted-foreground/70'>
              内容由 AI 生成，请仔细甄别
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}

function StudioConfirmations({
  approvals,
  fromRuntime,
  onRespond,
}: {
  approvals: readonly StudioPendingApproval[]
  fromRuntime: boolean
  onRespond: (id: string, approved: boolean) => void
}) {
  return (
    <>
      {approvals.map((action) => (
        <Message key={action.id} from='assistant' className='max-w-none'>
          <MessageContent className='w-full'>
            <Confirmation
              approval={{ id: action.id }}
              state='approval-requested'
            >
              <ConfirmationRequest>
                <AlertDescription>
                  {action.message ?? '该工具请求执行操作。是否批准？'}
                </AlertDescription>
              </ConfirmationRequest>
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
          </MessageContent>
        </Message>
      ))}
    </>
  )
}

type ClarificationOption = { id: string; label: string }
type LiveClarification = {
  id: string
  question: string
  options: ClarificationOption[]
  answer?: string
  status: 'pending' | 'answered' | 'skipped'
  messageId?: string
  workflow?: StudioWorkflowRequest
}

function workflowFromInterrupt(interrupt: AgUiInterrupt | StudioPendingApproval | StudioPendingClarification): StudioWorkflowRequest {
  const metadata = 'metadata' in interrupt ? interrupt.metadata : undefined
  const workflow = metadata?.workflow
  if (!workflow || typeof workflow !== 'object' || !('id' in workflow) || !('name' in workflow) || !('input_schema' in workflow)) {
    throw new Error('工作流输入格式不正确')
  }
  return workflow as StudioWorkflowRequest
}

function clarificationMessageId(
  interrupt: AgUiInterrupt | StudioPendingApproval | StudioPendingClarification
): string | undefined {
  const metadata = 'metadata' in interrupt ? interrupt.metadata : undefined
  return typeof metadata?.messageId === 'string' ? metadata.messageId : undefined
}

function clarificationOptions(
  interrupt: AgUiInterrupt | StudioPendingApproval | StudioPendingClarification
): ClarificationOption[] {
  const metadata = 'metadata' in interrupt ? interrupt.metadata : undefined
  const options = metadata?.options
  if (
    !Array.isArray(options) ||
    !options.every(
      (option: unknown) =>
        typeof option === 'object' &&
        option !== null &&
        'id' in option &&
        typeof option.id === 'string' &&
        'label' in option &&
        typeof option.label === 'string'
    )
  ) {
    throw new Error('澄清问题的选项格式不正确')
  }
  return options as ClarificationOption[]
}

function StudioClarificationRecord({
  question,
  answer,
  status,
}: {
  question: string
  answer?: string
  status: LiveClarification['status']
}) {
  return (
    <div className='flex flex-col gap-2'>
      <div className='flex items-center gap-2 text-sm text-muted-foreground'>
        <CircleHelp className='size-4' />
        <span>提问{status === 'pending' ? ' · 等待回答' : ''}</span>
      </div>
      {status !== 'pending' ? (
        <div className='flex flex-col gap-1 rounded-lg border bg-muted px-4 py-3'>
          <p className='whitespace-pre-wrap text-xs text-muted-foreground'>{question}</p>
          <p className='whitespace-pre-wrap text-sm'>{status === 'skipped' ? '已跳过' : answer}</p>
        </div>
      ) : null}
    </div>
  )
}

function StudioWorkflowRecord({ workflow, status }: { workflow: StudioWorkflowRequest; status: LiveClarification['status'] }) {
  return (
    <div className='flex items-center gap-2 rounded-lg border bg-muted px-4 py-3 text-sm'>
      <Workflow className='size-4 text-muted-foreground' />
      <span className='flex-1'>{workflow.name}</span>
      <span className='text-muted-foreground'>{status === 'pending' ? '等待输入' : status === 'skipped' ? '已跳过' : '已提交'}</span>
    </div>
  )
}

function StudioClarificationComposer({
  id,
  question,
  options,
  initialSelected,
  initialCustom,
  disabled,
  submitting = false,
  onReconnect,
  onCancel,
  cancelling = false,
  onSubmit,
}: {
  id: string
  question: string
  options: ClarificationOption[]
  initialSelected: string
  initialCustom: string
  disabled: boolean
  submitting?: boolean
  onReconnect?: () => void
  onCancel?: () => void
  cancelling?: boolean
  onSubmit: (selected: string, custom: string, answer: string) => void
}) {
  const [selected, setSelected] = useState(initialSelected)
  const [custom, setCustom] = useState(initialCustom)
  const chosenAnswer = selected === 'other'
    ? custom.trim()
    : options.find((option) => option.id === selected)?.label ?? ''

  return (
    <Card className='max-h-[min(60vh,32rem)] gap-0 overflow-hidden py-0'>
      <CardHeader className='shrink-0 px-5 pb-3 pt-5'>
        <CardTitle role='heading' aria-level={2} className='whitespace-pre-wrap text-base leading-6'>
          {question}
        </CardTitle>
      </CardHeader>
      <CardContent className='min-h-0 overflow-y-auto px-3'>
        <RadioGroup
          aria-label={question}
          value={selected}
          onValueChange={setSelected}
          disabled={disabled}
          className='gap-1'
        >
          {options.map((option) => (
            <label
              key={option.id}
              htmlFor={`${id}-${option.id}`}
              className={cn(
                'flex min-h-10 items-center gap-3 rounded-md px-3 py-2 text-sm leading-6 transition-colors',
                disabled ? 'cursor-default' : 'cursor-pointer hover:bg-muted',
                selected === option.id && 'bg-muted'
              )}
            >
              <RadioGroupItem value={option.id} id={`${id}-${option.id}`} />
              <span className='flex-1'>{option.label}</span>
            </label>
          ))}
          <div className={cn(
            'flex min-h-10 items-center gap-3 rounded-md px-3 py-2 transition-colors',
            !disabled && 'hover:bg-muted focus-within:bg-muted',
            selected === 'other' && 'bg-muted'
          )}>
            <RadioGroupItem value='other' id={`${id}-other`} />
            <label htmlFor={`${id}-other`} className={cn('text-sm', !disabled && 'cursor-pointer')}>其他</label>
            <Input
              aria-label='其他回答'
              value={custom}
              disabled={disabled}
              maxLength={1000}
              placeholder='输入你的回答'
              className='h-7 min-w-0 flex-1 border-0 bg-transparent px-0 shadow-none focus-visible:ring-0'
              onFocus={() => setSelected('other')}
              onChange={(event) => {
                setCustom(event.target.value)
                setSelected('other')
              }}
            />
          </div>
        </RadioGroup>
      </CardContent>
      <CardFooter className='shrink-0 justify-end gap-2 px-5 pb-4 pt-3'>
        {onCancel ? (
          <Button variant='ghost' disabled={cancelling} onClick={onCancel}>
            {cancelling ? '结束中…' : '结束本次运行'}
          </Button>
        ) : null}
        {onReconnect ? (
          <Button variant='ghost' onClick={onReconnect}>重新连接</Button>
        ) : null}
        <Button
          variant='ghost'
          disabled={disabled}
          onClick={() => onSubmit('skip', '', '')}
        >
          跳过本题
        </Button>
        <Button
          disabled={disabled || chosenAnswer === ''}
          onClick={() => onSubmit(selected, selected === 'other' ? custom.trim() : '', chosenAnswer)}
        >
          {submitting ? '提交中…' : '提交回答'}
        </Button>
      </CardFooter>
    </Card>
  )
}

function StudioRunCompletionWatcher({
  onRunFinished,
  onRuntimeStateChange,
}: {
  onRunFinished?: () => void
  onRuntimeStateChange?: (running: boolean) => void
}) {
  const isRunning = useAuiState((state) => state.thread.isRunning)
  const wasRunning = useRef(false)

  useEffect(() => {
    if (!wasRunning.current && isRunning) {
      onRuntimeStateChange?.(true)
    }
    if (wasRunning.current && !isRunning) {
      onRunFinished?.()
      onRuntimeStateChange?.(false)
    }
    wasRunning.current = isRunning
  }, [isRunning, onRunFinished, onRuntimeStateChange])
  return null
}

function AssetPicker({
  open,
  onOpenChange,
  assets,
  value,
  onInsert,
  onImportLibraryAsset,
}: PickerOpenProps & {
  assets: StudioAsset[]
  value: SelectedAsset[]
  onInsert: (asset: StudioAsset, versionId: string) => void
  onImportLibraryAsset: (asset: SelectedAsset) => Promise<StudioAsset>
}) {
  const libraryAssets = useInfiniteQuery({
    queryKey: ['studio', 'library', 'assets'],
    queryFn: ({ pageParam }) =>
      listStudioLibraryAssets({ page: pageParam, limit: 100 }),
    initialPageParam: 1,
    getNextPageParam: (lastPage, pages) =>
      lastPage.assets.length > 0 &&
      pages.reduce((count, page) => count + page.assets.length, 0) <
        lastPage.total
        ? pages.length + 1
        : undefined,
  })
  const libraryAssetItems =
    libraryAssets.data?.pages.flatMap((page) => page.assets) ?? []
  const sessionAssetIDs = new Set(assets.map((asset) => asset.id))
  const assetsByID = new Map<string, StudioAsset>()
  for (const asset of assets) assetsByID.set(asset.id, asset)
  for (const asset of libraryAssetItems) {
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
    if (!asset) return libraryAssets.isSuccess && !libraryAssets.hasNextPage
    return !asset.versions.some(
      (version) => version.id === selection.assetVersionId
    )
  })
  const currentAssets = assets
  const reusableAssets = libraryAssetItems.filter(
    (asset) => !sessionAssetIDs.has(asset.id)
  )
  const choose = (asset: StudioAsset, fromLibrary = false) => {
    onOpenChange(false)
    void insert(asset, fromLibrary)
  }

  return (
    <Popover open={open} onOpenChange={onOpenChange}>
      <PopoverTrigger asChild>
        <PromptInputButton
          aria-label={
            value.length === 0
              ? '选择资产'
              : `选择资产，已选 ${value.length} 项`
          }
          className={cn(
            value.length > 0 && 'bg-accent text-accent-foreground'
          )}
          size='icon-sm'
          tooltip={value.length === 0 ? '资产' : `资产 · ${value.length}`}
        >
          <Boxes />
        </PromptInputButton>
      </PopoverTrigger>
      <PopoverContent align='start' className='w-72 p-1' onCloseAutoFocus={retainStudioComposerFocus}>
        <Tabs defaultValue='session'>
          <TabsContent value='session' className='h-48 flex-none overflow-y-auto'>
            <AssetPickerSection
              assets={currentAssets}
              onSelect={choose}
              emptyText='会话内还没有资产'
            />
          </TabsContent>
          <TabsContent value='global' className='h-48 flex-none overflow-y-auto'>
            <AssetPickerSection
              assets={reusableAssets}
              onSelect={(asset) => choose(asset, true)}
              emptyText={
                libraryAssets.isLoading
                  ? '正在读取全局资产…'
                  : libraryAssets.isError
                    ? '全局资产读取失败'
                    : '全局还没有可用资产'
              }
            />
            {libraryAssets.hasNextPage ? (
              <Button
                type='button'
                variant='ghost'
                className='h-auto w-full justify-start px-2 py-1.5 font-normal'
                disabled={libraryAssets.isFetchingNextPage}
                onClick={() => void libraryAssets.fetchNextPage()}
              >
                {libraryAssets.isFetchingNextPage
                  ? '正在读取更多资产…'
                  : libraryAssets.isFetchNextPageError
                    ? '读取失败，重试加载'
                    : '加载更多资产'}
              </Button>
            ) : null}
          </TabsContent>
          {unavailableSelections.length > 0 ? (
            <p className='border-t px-2 py-2 text-xs text-muted-foreground'>
              {unavailableSelections.length} 项已选资产不可用
            </p>
          ) : null}
          <TabsList className='h-8 w-full'>
            <TabsTrigger value='session' className='text-xs'>会话内</TabsTrigger>
            <TabsTrigger value='global' className='text-xs'>全局</TabsTrigger>
          </TabsList>
        </Tabs>
      </PopoverContent>
    </Popover>
  )
}

function AssetPickerSection({
  assets,
  onSelect,
  emptyText,
}: {
  assets: StudioAsset[]
  onSelect: (asset: StudioAsset) => void
  emptyText: string
}) {
  return (
    <>
      {assets.length === 0 ? (
        <p className='px-2 py-1.5 text-sm text-muted-foreground'>{emptyText}</p>
      ) : (
        assets.map((asset) => {
          const version = asset.versions[asset.versions.length - 1]
          return (
            <Button
              key={asset.id}
              type='button'
              variant='ghost'
              className='h-auto w-full justify-start px-2 py-1.5 text-left font-normal'
              onClick={() => onSelect(asset)}
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
            </Button>
          )
        })
      )}
    </>
  )
}

function SkillPicker({
  open,
  onOpenChange,
  skills,
  value,
  onInsert,
}: PickerOpenProps & {
  skills: StudioSkillSummary[]
  value: string[]
  onInsert: (skill: StudioSkillSummary) => void
}) {
  const enabledSkills = skills.filter((skill) => skill.enabled)
  return (
    <DropdownMenu open={open} onOpenChange={onOpenChange} modal={false}>
      <DropdownMenuTrigger asChild>
        <PromptInputButton
          aria-label={
            value.length === 0
              ? '选择技能'
              : `选择技能，已选 ${value.length} 项`
          }
          className={cn(
            value.length > 0 && 'bg-accent text-accent-foreground'
          )}
          size='icon-sm'
          tooltip={value.length === 0 ? '技能' : `技能 · ${value.length}`}
        >
          <Sparkles />
        </PromptInputButton>
      </DropdownMenuTrigger>
      <DropdownMenuContent align='start' className='w-72' onCloseAutoFocus={retainStudioComposerFocus}>
        {enabledSkills.length === 0 ? (
          <DropdownMenuItem disabled>没有已启用的技能</DropdownMenuItem>
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

function WorkflowPicker({
  open,
  onOpenChange,
  disabled,
  onSelect,
}: PickerOpenProps & {
  disabled: boolean
  onSelect: (workflow: StudioAgentWorkflow) => void
}) {
  const query = useQuery({
    queryKey: ['studio', 'agent-workflows'],
    queryFn: listStudioAgentWorkflows,
    enabled: open,
  })
  const available = (query.data ?? []).filter((workflow) => workflow.workflow_enabled && workflow.agent_enabled)
  return (
    <DropdownMenu open={open} onOpenChange={onOpenChange} modal={false}>
      <DropdownMenuTrigger asChild>
        <PromptInputButton aria-label='选择工作流' size='icon-sm' tooltip='工作流' disabled={disabled}>
          <Workflow />
        </PromptInputButton>
      </DropdownMenuTrigger>
      <DropdownMenuContent align='start' className='w-72' onCloseAutoFocus={retainStudioComposerFocus}>
        {query.isPending ? <DropdownMenuItem disabled>正在读取工作流…</DropdownMenuItem> : null}
        {query.isError ? <DropdownMenuItem disabled>读取工作流失败</DropdownMenuItem> : null}
        {query.isSuccess && available.length === 0 ? <DropdownMenuItem disabled>没有可用工作流</DropdownMenuItem> : null}
        {available.map((workflow) => (
          <DropdownMenuItem key={workflow.id} onSelect={() => onSelect(workflow)}>
            <span className='min-w-0 flex-1'>
              <span className='block truncate'>{workflow.name}</span>
              {workflow.description ? <span className='block truncate text-xs text-muted-foreground'>{workflow.description}</span> : null}
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

function StudioMessageLocator({
  target,
  messageCount,
}: {
  target?: { id: string; request: number }
  messageCount: number
}) {
  const { scrollRef, stopScroll } = useStickToBottomContext()
  const handledRequest = useRef<number>(undefined)

  useEffect(() => {
    if (!target || handledRequest.current === target.request) return
    const frame = requestAnimationFrame(() => {
      const scroll = scrollRef.current
      const anchor = Array.from(
        scroll?.querySelectorAll<HTMLElement>('[data-studio-turn-id]') ?? []
      ).find((element) => element.dataset.studioTurnId === target.id)
      if (!scroll || !anchor) return
      stopScroll()
      const top =
        anchor.getBoundingClientRect().top -
        scroll.getBoundingClientRect().top +
        scroll.scrollTop -
        16
      scroll.scrollTo({
        top,
        behavior: matchMedia('(prefers-reduced-motion: reduce)').matches
          ? 'instant'
          : 'smooth',
      })
      handledRequest.current = target.request
    })
    return () => cancelAnimationFrame(frame)
  }, [messageCount, scrollRef, stopScroll, target])

  return null
}

function StudioMessage({
  message,
  isRunning,
  assistantCopyText,
  referenceParts,
  clarificationParts,
  assets,
}: {
  message: StudioThreadMessage
  isRunning: boolean
  assistantCopyText?: string
  referenceParts?: StudioComposerPart[]
  clarificationParts?: Map<number, LiveClarification>
  assets: StudioAsset[]
}) {
  if (message.role !== 'user' && message.role !== 'assistant') return null
  const copyText = message.role === 'user'
    ? message.parts.flatMap((part) => part.type === 'text' ? [part.text] : []).join('')
    : assistantCopyText
  return (
    <Message
      from={message.role}
      data-studio-turn-id={message.role === 'user' ? message.id : undefined}
      className={cn('gap-1', message.role === 'assistant' && 'max-w-none')}
    >
      <MessageContent
        className={cn(
          message.role === 'assistant' ? 'w-full gap-4' : 'gap-2'
        )}
      >
        {message.role === 'user' && referenceParts?.map((part) => {
          if (part.type !== 'asset_ref') return null
          const asset = assets.find((item) => item.id === part.asset_id)
          if (asset?.kind !== 'image') return null
          const version = asset.versions.find((item) => item.id === part.asset_version_id)
          if (!version) return null
          return <img key={part.asset_id} src={`${baseURL()}${version.content_url}`} alt={asset.name} className='max-h-48 max-w-64 rounded-lg border object-contain' />
        })}
        {message.role === 'user' && referenceParts ? (
          <span className='whitespace-pre-wrap break-words'>
            {referenceParts.map((part, index) => part.type === 'text' ? (
              <span key={index}>{part.text}</span>
            ) : (
              <StudioReferenceBadge
                key={index}
                kind={part.type === 'skill_ref' ? 'skill' : part.type === 'workflow_ref' ? 'workflow' : 'asset'}
                label={part.name}
              />
            ))}
          </span>
        ) : message.parts.map((part, index) => {
          if (part.type === 'text') {
            const clarification = clarificationParts?.get(index)
            if (clarification) {
              return (
                clarification.workflow ? (
                  <StudioWorkflowRecord key={`${message.id}-${index}`} workflow={clarification.workflow} status={clarification.status} />
                ) : (
                  <StudioClarificationRecord
                    key={`${message.id}-${index}`}
                    question={clarification.question}
                    answer={clarification.answer}
                    status={clarification.status}
                  />
                )
              )
            }
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
        <PromptInputButton
          aria-label={`选择模型：${selected?.name ?? '未选择模型'}`}
          className='max-w-52 font-normal'
        >
          <span className='truncate'>{selected?.name ?? '未选择模型'}</span>
          <ChevronDown className='size-3.5' />
        </PromptInputButton>
      </DropdownMenuTrigger>
      <DropdownMenuContent align='end' className='w-72'>
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
