import {
  AbstractAgent,
  type AgentConfig,
  type BaseEvent,
  type RunAgentInput,
} from '@ag-ui/client'
import { Observable } from 'rxjs'

type StudioRunConfig = {
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
  private readonly runConfig: StudioRunConfig
  private socket?: WebSocket

  constructor(config: Config) {
    super(config)
    this.url = config.url
    this.runConfig = config.runConfig
  }

  run(input: RunAgentInput): Observable<BaseEvent> {
    const forwardedProps = {
      ...(input.forwardedProps ?? {}),
      runConfig: this.runConfig,
    }
    const request = { ...input, forwardedProps }
    return new Observable<BaseEvent>((subscriber) => {
      const socket = new WebSocket(this.url)
      this.socket = socket
      let closed = false

      socket.onopen = () => {
        if (!closed) socket.send(JSON.stringify(request))
      }
      socket.onmessage = (message) => {
        try {
          subscriber.next(JSON.parse(String(message.data)) as BaseEvent)
        } catch (error) {
          subscriber.error(
            error instanceof Error ? error : new Error('无法解析 Agent 事件')
          )
        }
      }
      socket.onerror = () => {
        subscriber.error(new Error('Agent WebSocket 连接失败'))
      }
      socket.onclose = () => {
        if (!closed) subscriber.complete()
      }

      return () => {
        closed = true
        if (
          socket.readyState === WebSocket.OPEN ||
          socket.readyState === WebSocket.CONNECTING
        ) {
          socket.close()
        }
        if (this.socket === socket) this.socket = undefined
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
