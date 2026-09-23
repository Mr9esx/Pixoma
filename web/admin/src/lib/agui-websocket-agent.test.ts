import { afterEach, describe, expect, it } from 'vitest'
import { StudioWebSocketAgent } from './agui-websocket-agent'

const originalWebSocket = globalThis.WebSocket
afterEach(() => {
  globalThis.WebSocket = originalWebSocket
})

describe('StudioWebSocketAgent', () => {
  it('reattaches a dropped new run with a durable cursor instead of ending it', async () => {
    const requests: Array<Record<string, unknown>> = []
    const sockets: Array<{
      onopen: (() => void) | null
      onmessage: ((event: { data: string }) => void) | null
      onclose: (() => void) | null
      onerror: (() => void) | null
      readyState: number
      send(data: string): void
      close(): void
    }> = []
    class FakeSocket {
      static OPEN = 1
      static CONNECTING = 0
      onopen: (() => void) | null = null
      onmessage: ((event: { data: string }) => void) | null = null
      onclose: (() => void) | null = null
      onerror: (() => void) | null = null
      readyState = 0
      constructor(_url: string) {
        sockets.push(this)
        queueMicrotask(() => {
          this.readyState = 1
          this.onopen?.()
          const emit = (event: Record<string, unknown>) =>
            this.onmessage?.({ data: JSON.stringify(event) })
          emit({ type: 'RUN_STARTED', metadata: { studioRunId: 'run-1' } })
          if (sockets.length === 1) {
            emit({ type: 'TEXT_MESSAGE_CONTENT', messageId: 'm1', delta: '你', sequence: 1 })
            this.onclose?.()
          } else {
            emit({ type: 'TEXT_MESSAGE_CONTENT', messageId: 'm1', delta: '好', sequence: 2 })
            emit({ type: 'RUN_FINISHED', outcome: { type: 'success' } })
          }
        })
      }
      send(data: string) {
        requests.push(JSON.parse(data) as Record<string, unknown>)
      }
      close() {
        this.readyState = 3
      }
    }
    globalThis.WebSocket = FakeSocket as never
    const agent = new StudioWebSocketAgent({
      url: 'wss://studio.test/agui',
      threadId: 'session-1',
      runConfig: { modelConfigId: 'model-1', permissionMode: 'full_access', selectedSkillIds: [], selectedAssets: [] },
    })
    const events = await new Promise<Array<Record<string, unknown>>>((resolve, reject) => {
      const collected: Array<Record<string, unknown>> = []
      agent.run({ threadId: 'session-1', runId: 'wire-1', messages: [] } as never).subscribe({
        next: (event) => collected.push(event as unknown as Record<string, unknown>),
        error: reject,
        complete: () => resolve(collected),
      })
    })
    expect(requests).toHaveLength(2)
    expect(requests[0]).toMatchObject({ requestId: 'wire-1' })
    expect(requests[1]).toMatchObject({ attachRunId: 'run-1', afterSequence: 1, messages: [] })
    expect(events.filter((event) => event.type === 'TEXT_MESSAGE_CONTENT')).toHaveLength(2)
    expect(events.at(-1)?.type).toBe('RUN_FINISHED')
  })
})
