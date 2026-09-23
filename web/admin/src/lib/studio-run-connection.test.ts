import { describe, expect, it } from 'vitest'
import { StudioRunConnection } from './studio-run-connection'

describe('StudioRunConnection', () => {
  it('reconnects from the last durable event without losing or duplicating streamed text', async () => {
    const requests: Array<Record<string, unknown>> = []
    const sockets = Array.from({ length: 2 }, () => ({
      readyState: 0,
      onopen: null as ((...args: unknown[]) => void) | null,
      onmessage: null as ((...args: unknown[]) => void) | null,
      onerror: null as ((...args: unknown[]) => void) | null,
      onclose: null as ((...args: unknown[]) => void) | null,
      send(data: string) {
        requests.push(JSON.parse(data) as Record<string, unknown>)
      },
      close() {
        this.readyState = 3
      },
    }))
    let calls = 0
    const connection = new StudioRunConnection({
      url: 'wss://studio.test/agui',
      threadId: 'session-1',
      studioRunId: 'run-1',
      socketFactory: () => {
        const index = calls++
        const socket = sockets[index]
        queueMicrotask(() => {
          socket.readyState = 1
          socket.onopen?.()
          const emit = (event: Record<string, unknown>) =>
            socket.onmessage?.({ data: JSON.stringify(event) })
          emit({ type: 'RUN_STARTED', metadata: { studioRunId: 'run-1' } })
          if (index === 0) {
            emit({ type: 'TEXT_MESSAGE_START', messageId: 'm1', sequence: 1 })
            emit({ type: 'TEXT_MESSAGE_CONTENT', messageId: 'm1', delta: '你', sequence: 2 })
            socket.onclose?.()
          } else {
            emit({ type: 'TEXT_MESSAGE_CONTENT', messageId: 'm1', delta: '你', sequence: 2 })
            emit({ type: 'TEXT_MESSAGE_CONTENT', messageId: 'm1', delta: '好', sequence: 3 })
            emit({ type: 'RUN_FINISHED', outcome: { type: 'success' } })
          }
        })
        return socket
      },
    })
    const results = []
    for await (const result of connection.resume({} as never)) results.push(result)

    expect(requests).toHaveLength(2)
    expect(requests[1]).toMatchObject({
      attachRunId: 'run-1', afterSequence: 2, messages: [],
    })
    expect(results.at(-1)).toMatchObject({
      status: { type: 'complete' }, content: [{ type: 'text', text: '你好' }],
    })
  })

  it('attaches to an existing run and aggregates streamed snapshots', async () => {
    let request: Record<string, unknown> | undefined
    const socket = {
      readyState: 0,
      onopen: null as ((...args: unknown[]) => void) | null,
      onmessage: null as ((...args: unknown[]) => void) | null,
      onerror: null as (() => void) | null,
      onclose: null as (() => void) | null,
      send(data: string) {
        request = JSON.parse(data) as Record<string, unknown>
        socket.readyState = 1
        const emit = (event: Record<string, unknown>) =>
          socket.onmessage?.({ data: JSON.stringify(event) })
        queueMicrotask(() => emit({ type: 'RUN_STARTED', runId: 'wire-run' }))
        queueMicrotask(() =>
          emit({ type: 'REASONING_MESSAGE_START', messageId: 'reason-1' })
        )
        queueMicrotask(() =>
          emit({
            type: 'REASONING_MESSAGE_CONTENT',
            messageId: 'reason-1',
            delta: '先思考',
          })
        )
        queueMicrotask(() =>
          emit({ type: 'TEXT_MESSAGE_START', messageId: 'assistant-1' })
        )
        queueMicrotask(() =>
          emit({
            type: 'TEXT_MESSAGE_CONTENT',
            messageId: 'assistant-1',
            delta: '你好',
          })
        )
        queueMicrotask(() =>
          emit({
            type: 'TEXT_MESSAGE_CONTENT',
            messageId: 'assistant-1',
            delta: '，世界',
          })
        )
        queueMicrotask(() =>
          emit({
            type: 'TOOL_CALL_START',
            toolCallId: 'tool-1',
            toolCallName: 'search',
          })
        )
        queueMicrotask(() =>
          emit({
            type: 'TOOL_CALL_ARGS',
            toolCallId: 'tool-1',
            delta: '{"q":"pixoma"}',
          })
        )
        queueMicrotask(() =>
          emit({
            type: 'TOOL_CALL_RESULT',
            toolCallId: 'tool-1',
            content: '命中 1 条',
          })
        )
        queueMicrotask(() =>
          emit({
            type: 'RUN_FINISHED',
            outcome: { type: 'success' },
          })
        )
      },
      close() {
        socket.readyState = 3
      },
    }

    const connection = new StudioRunConnection({
      url: 'wss://studio.test/agui',
      threadId: 'session-1',
      studioRunId: 'run-1',
      afterSequence: 12,
      socketFactory: () => {
        queueMicrotask(() => socket.onopen?.())
        return socket
      },
    })
    const results = []
    for await (const result of connection.resume({} as never)) {
      results.push(result)
    }

    expect(request).toMatchObject({
      threadId: 'session-1',
      attachRunId: 'run-1',
      afterSequence: 12,
      messages: [],
    })
    expect(results.at(-1)).toMatchObject({
      status: { type: 'complete', reason: 'stop' },
      content: [
        { type: 'reasoning', text: '先思考' },
        { type: 'text', text: '你好，世界' },
        {
          type: 'tool-call',
          toolCallId: 'tool-1',
          toolName: 'search',
          argsText: '{"q":"pixoma"}',
          result: '命中 1 条',
        },
      ],
    })
  })
})
