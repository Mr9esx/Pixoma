import type {
  ChatModelRunOptions,
  ChatModelRunResult,
  MessageStatus,
  ThreadAssistantMessagePart,
} from '@assistant-ui/react'

type AgUiEvent = {
  type: string
  [key: string]: unknown
}

type SocketLike = {
  readyState: number
  send(data: string): void
  close(): void
  onopen: ((...args: unknown[]) => void) | null
  onmessage: ((...args: unknown[]) => void) | null
  onerror: ((...args: unknown[]) => void) | null
  onclose: ((...args: unknown[]) => void) | null
}

type SocketFactory = (url: string) => SocketLike

type TextState = {
  id: string
  text: string
}

type ReasoningState = {
  id: string
  text: string
}

type ToolState = {
  id: string
  name: string
  argsText: string
  result?: string
  isError?: boolean
}

type StudioRunConnectionOptions = {
  url: string
  threadId: string
  studioRunId: string
  afterSequence?: number
  socketFactory?: SocketFactory
}

const OPEN = 1
const CONNECTING = 0

/**
 * Reconnects assistant-ui to an already running Studio run. This is kept
 * separate from StudioWebSocketAgent because history.resume must return
 * assistant-ui snapshots rather than raw AG-UI events.
 */
export class StudioRunConnection {
  private readonly options: StudioRunConnectionOptions
  private readonly socketFactory: SocketFactory

  constructor(options: StudioRunConnectionOptions) {
    this.options = options
    this.socketFactory =
      options.socketFactory ??
      ((url) => new WebSocket(url) as unknown as SocketLike)
  }

  async *resume(
    runOptions: ChatModelRunOptions,
  ): AsyncGenerator<ChatModelRunResult, void, unknown> {
    const aggregator = new StudioRunAggregator()
    let sequence = this.options.afterSequence ?? 0
    let aborted = runOptions.abortSignal?.aborted ?? false
    let closeCurrent = () => {}
    const abort = () => {
      aborted = true
      closeCurrent()
    }
    runOptions.abortSignal?.addEventListener('abort', abort, { once: true })
    let retries = 0
    try {
      while (!aborted && !aggregator.isTerminal) {
        const queue = new AsyncEventQueue<AgUiEvent>()
        const socket = this.socketFactory(this.options.url)
        let closed = false
        closeCurrent = () => {
          if (closed) return
          closed = true
          if (socket.readyState === OPEN || socket.readyState === CONNECTING) {
            socket.close()
          }
          queue.close()
        }
        socket.onopen = () => {
          if (closed) return
          socket.send(JSON.stringify({
            threadId: this.options.threadId,
            runId: crypto.randomUUID(),
            attachRunId: this.options.studioRunId,
            afterSequence: sequence,
            protocolVersion: '1.0',
            messages: [],
            ...(runOptions.runConfig?.custom
              ? { state: runOptions.runConfig.custom }
              : {}),
          }))
        }
        socket.onmessage = (...args) => {
          try {
            const event = args[0] as { data?: unknown } | undefined
            queue.push(JSON.parse(String(event?.data)) as AgUiEvent)
          } catch {
            queue.fail(new Error('无法解析 Studio 续接事件'))
          }
        }
        socket.onerror = () => queue.close()
        socket.onclose = () => queue.close()
        try {
          for await (const event of queue) {
            if (aborted) break
            const eventSequence = Number(event.sequence)
            if (Number.isSafeInteger(eventSequence) && eventSequence > 0) {
              if (eventSequence <= sequence) continue
              sequence = eventSequence
              retries = 0
            }
            yield aggregator.handle(event)
            if (aggregator.isTerminal) break
          }
        } finally {
          closeCurrent()
        }
        if (!aborted && !aggregator.isTerminal) {
          await new Promise((resolve) =>
            setTimeout(resolve, Math.min(100 * 2 ** retries++, 2000))
          )
        }
      }
    } finally {
      runOptions.abortSignal?.removeEventListener('abort', abort)
      closeCurrent()
    }
  }
}

class StudioRunAggregator {
  private readonly text = new Map<string, TextState>()
  private readonly reasoning = new Map<string, ReasoningState>()
  private readonly tools = new Map<string, ToolState>()
  private textOrder: string[] = []
  private reasoningOrder: string[] = []
  private toolOrder: string[] = []
  private partOrder: Array<{ type: 'text' | 'reasoning' | 'tool'; id: string }> = []
  private activeTextId?: string
  private activeReasoningId?: string
  private status: MessageStatus = { type: 'running' }
  private interrupts?: unknown[]
  private started = false
  isTerminal = false

  handle(event: AgUiEvent): ChatModelRunResult {
    switch (event.type) {
      case 'RUN_STARTED':
        if (this.started) return this.snapshot()
        this.started = true
        this.reset()
        this.status = { type: 'running' }
        return this.snapshot()
      case 'TEXT_MESSAGE_START': {
        const id = stringValue(event.messageId) || `text-${this.textOrder.length + 1}`
        if (!this.text.has(id)) {
          this.text.set(id, { id, text: '' })
          this.textOrder.push(id)
          this.partOrder.push({ type: 'text', id })
        }
        this.activeTextId = id
        return this.snapshot()
      }
      case 'TEXT_MESSAGE_CONTENT':
      case 'TEXT_MESSAGE_CHUNK': {
        const id =
          stringValue(event.messageId) ||
          this.activeTextId ||
          this.startAnonymousText()
        const state = this.text.get(id) ?? { id, text: '' }
        state.text += stringValue(event.delta)
        this.text.set(id, state)
        return this.snapshot()
      }
      case 'TEXT_MESSAGE_END':
        return this.snapshot()
      case 'REASONING_START':
      case 'REASONING_MESSAGE_START': {
        const id = stringValue(event.messageId) || 'reasoning-1'
        if (!this.reasoning.has(id)) {
          this.reasoning.set(id, { id, text: '' })
          this.reasoningOrder.push(id)
          this.partOrder.push({ type: 'reasoning', id })
        }
        this.activeReasoningId = id
        return this.snapshot()
      }
      case 'REASONING_MESSAGE_CONTENT':
      case 'THINKING_TEXT_MESSAGE_CONTENT': {
        const id =
          stringValue(event.messageId) ||
          this.activeReasoningId ||
          this.startAnonymousReasoning()
        const state = this.reasoning.get(id) ?? { id, text: '' }
        state.text += stringValue(event.delta)
        this.reasoning.set(id, state)
        return this.snapshot()
      }
      case 'TOOL_CALL_START': {
        const id = stringValue(event.toolCallId)
        if (!id) return this.snapshot()
        if (!this.tools.has(id)) {
          this.tools.set(id, {
            id,
            name: stringValue(event.toolCallName) || 'tool',
            argsText: '',
          })
          this.toolOrder.push(id)
          this.partOrder.push({ type: 'tool', id })
        }
        return this.snapshot()
      }
      case 'TOOL_CALL_ARGS':
      case 'TOOL_CALL_CHUNK': {
        const id = stringValue(event.toolCallId)
        if (!id) return this.snapshot()
        const state = this.tools.get(id) ?? {
          id,
          name: stringValue(event.toolCallName) || 'tool',
          argsText: '',
        }
        state.argsText += stringValue(event.delta)
        this.tools.set(id, state)
        if (!this.toolOrder.includes(id)) {
          this.toolOrder.push(id)
          this.partOrder.push({ type: 'tool', id })
        }
        return this.snapshot()
      }
      case 'TOOL_CALL_RESULT': {
        const id = stringValue(event.toolCallId)
        if (!id) return this.snapshot()
        const state = this.tools.get(id) ?? { id, name: 'tool', argsText: '' }
        state.result = stringValue(event.content)
        if (typeof event.isError === 'boolean') state.isError = event.isError
        this.tools.set(id, state)
        if (!this.toolOrder.includes(id)) {
          this.toolOrder.push(id)
          this.partOrder.push({ type: 'tool', id })
        }
        return this.snapshot()
      }
      case 'RUN_FINISHED': {
        const outcome = event.outcome as { type?: string; interrupts?: unknown[] } | undefined
        this.interrupts = outcome?.interrupts
        if (outcome?.type === 'interrupt') {
          this.status = { type: 'requires-action', reason: 'interrupt' }
        } else if (outcome?.type === 'cancelled') {
          this.status = { type: 'incomplete', reason: 'cancelled' }
        } else {
          this.status = { type: 'complete', reason: 'stop' }
        }
        this.isTerminal = true
        return this.snapshot()
      }
      case 'RUN_CANCELLED':
        this.status = { type: 'incomplete', reason: 'cancelled' }
        this.isTerminal = true
        return this.snapshot()
      case 'RUN_ERROR':
        this.status = {
          type: 'incomplete',
          reason: 'error',
          ...(event.message ? { error: stringValue(event.message) } : {}),
        }
        this.isTerminal = true
        return this.snapshot()
      default:
        return this.snapshot()
    }
  }

  private snapshot(): ChatModelRunResult {
    const content: ThreadAssistantMessagePart[] = []
    for (const entry of this.partOrder) {
      if (entry.type === 'text') {
        const part = this.text.get(entry.id)
        if (part) content.push({ type: 'text', text: part.text })
      } else if (entry.type === 'reasoning') {
        const part = this.reasoning.get(entry.id)
        if (part) content.push({ type: 'reasoning', text: part.text })
      } else {
        const part = this.tools.get(entry.id)
        if (!part) continue
        content.push({
          type: 'tool-call',
          toolCallId: part.id,
          toolName: part.name,
          argsText: part.argsText,
          args: parseArgs(part.argsText) as never,
          ...(part.result !== undefined ? { result: part.result } : {}),
          ...(part.isError !== undefined ? { isError: part.isError } : {}),
        })
      }
    }
    return {
      content,
      status: this.status,
      ...(this.interrupts
        ? {
            metadata: {
              custom: {
                agui: { interrupts: this.interrupts },
              },
            },
          }
        : {}),
    }
  }

  private reset() {
    this.text.clear()
    this.reasoning.clear()
    this.tools.clear()
    this.textOrder = []
    this.reasoningOrder = []
    this.toolOrder = []
    this.partOrder = []
    this.activeTextId = undefined
    this.activeReasoningId = undefined
    this.interrupts = undefined
    this.isTerminal = false
  }

  private startAnonymousText() {
    const id = `text-${this.textOrder.length + 1}`
    this.text.set(id, { id, text: '' })
    this.textOrder.push(id)
    this.partOrder.push({ type: 'text', id })
    this.activeTextId = id
    return id
  }

  private startAnonymousReasoning() {
    const id = `reasoning-${this.reasoningOrder.length + 1}`
    this.reasoning.set(id, { id, text: '' })
    this.reasoningOrder.push(id)
    this.partOrder.push({ type: 'reasoning', id })
    this.activeReasoningId = id
    return id
  }
}

class AsyncEventQueue<T> implements AsyncIterable<T> {
  private values: T[] = []
  private waiters: Array<{
    resolve: (result: IteratorResult<T>) => void
    reject: (error: Error) => void
  }> = []
  private error?: Error
  private done = false

  push(value: T) {
    const waiter = this.waiters.shift()
    if (waiter) waiter.resolve({ value, done: false })
    else this.values.push(value)
  }

  close() {
    this.done = true
    while (this.waiters.length) {
      this.waiters.shift()?.resolve({ value: undefined as never, done: true })
    }
  }

  fail(error: Error) {
    this.error = error
    this.done = true
    while (this.waiters.length) this.waiters.shift()?.reject(error)
  }

  [Symbol.asyncIterator](): AsyncIterator<T> {
    return {
      next: async () => {
        if (this.error) throw this.error
        const value = this.values.shift()
        if (value !== undefined) return { value, done: false }
        if (this.done) return { value: undefined as never, done: true }
        return new Promise<IteratorResult<T>>((resolve, reject) =>
          this.waiters.push({ resolve, reject })
        )
      },
    }
  }
}

function stringValue(value: unknown): string {
  return typeof value === 'string' ? value : ''
}

function parseArgs(value: string): Record<string, unknown> {
  if (!value.trim()) return {}
  try {
    const parsed: unknown = JSON.parse(value)
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
      ? (parsed as Record<string, unknown>)
      : {}
  } catch {
    return {}
  }
}
