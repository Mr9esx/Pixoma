import {
  AbstractAgent,
  type AgentConfig,
  type BaseEvent,
  type RunAgentInput,
} from '@ag-ui/client'
import { Observable } from 'rxjs'

export type StudioRunConfig = {
  modelConfigId: string
  permissionMode: string
  selectedSkillIds: string[]
  selectedAssets: Array<{ assetId: string; assetVersionId: string }>
}

type Config = AgentConfig & {
  url: string
  runConfig: StudioRunConfig
}

/** AG-UI transport for the Studio run stream. Cookies are sent by the browser
 * during the same-origin WebSocket handshake, so the session token never goes
 * into a query string or a loggable URL.
 */
export class StudioWebSocketAgent extends AbstractAgent {
  private readonly url: string
  private runConfig: StudioRunConfig
  private socket?: WebSocket
  private studioRunId?: string

  activeStudioRunId(): string | undefined {
    return this.studioRunId
  }

  constructor(config: Config) {
    super(config)
    this.url = config.url
    this.runConfig = config.runConfig
  }

  updateRunConfig(runConfig: StudioRunConfig): void {
    this.runConfig = runConfig
  }

  run(input: RunAgentInput): Observable<BaseEvent> {
    const forwardedProps = {
      ...(input.forwardedProps ?? {}),
      runConfig: this.runConfig,
    }
    const request = { ...input, requestId: input.runId, forwardedProps }
    return new Observable<BaseEvent>((subscriber) => {
      this.studioRunId = undefined
      let lastSequence = 0
      let started = false
      let ended = false
      let retries = 0
      let reconnectTimer: ReturnType<typeof setTimeout> | undefined
      const connect = () => {
        if (ended) return
        const socket = new WebSocket(this.url)
        this.socket = socket
        let disconnected = false
        const reconnect = () => {
          if (ended || disconnected) return
          disconnected = true
          if (this.socket === socket) this.socket = undefined
          reconnectTimer = setTimeout(connect, Math.min(100 * 2 ** retries++, 2000))
        }
        socket.onopen = () => {
          if (ended) return
          socket.send(JSON.stringify(this.studioRunId
            ? {
                threadId: input.threadId,
                runId: input.runId,
                attachRunId: this.studioRunId,
                afterSequence: lastSequence,
                messages: [],
                protocolVersion: '1.0',
              }
            : request))
        }
        socket.onmessage = (message) => {
          try {
            const event = JSON.parse(String(message.data)) as BaseEvent & {
              sequence?: number
              metadata?: { studioRunId?: string }
            }
            if (event.type === 'RUN_STARTED') {
              if (event.metadata?.studioRunId) this.studioRunId = event.metadata.studioRunId
              if (started) return
              started = true
            }
            if (typeof event.sequence === 'number' && event.sequence > 0) {
              if (event.sequence <= lastSequence) return
              lastSequence = event.sequence
              retries = 0
            }
            subscriber.next(event)
            if (event.type === 'RUN_FINISHED' || event.type === 'RUN_ERROR') {
              ended = true
              subscriber.complete()
            }
          } catch (error) {
            ended = true
            subscriber.error(error instanceof Error ? error : new Error('无法解析 Agent 事件'))
          }
        }
        socket.onerror = () => {
          reconnect()
          socket.close()
        }
        socket.onclose = reconnect
      }
      connect()

      return () => {
        ended = true
        if (reconnectTimer) clearTimeout(reconnectTimer)
        const socket = this.socket
        if (
          socket && (socket.readyState === WebSocket.OPEN ||
          socket.readyState === WebSocket.CONNECTING)
        ) {
          socket.close()
        }
        this.socket = undefined
      }
    })
  }

  override abortRun(): void {
    this.socket?.close()
    this.socket = undefined
  }

  override clone(): StudioWebSocketAgent {
    return new StudioWebSocketAgent({
      url: this.url,
      runConfig: this.runConfig,
      agentId: this.agentId,
      description: this.description,
      threadId: this.threadId,
      initialMessages: this.messages,
      initialState: this.state,
    })
  }
}
