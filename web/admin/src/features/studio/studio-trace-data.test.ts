import { describe, expect, it } from 'vitest'
import { recordPositions, systemPromptFromRequest } from './studio-trace-data'

describe('systemPromptFromRequest', () => {
  it.each([
    [{ messages: [{ role: 'system', content: '系统指令' }] }, '系统指令'],
    [
      {
        input: [
          {
            role: 'system',
            content: [{ type: 'input_text', text: '系统指令' }],
          },
        ],
      },
      '系统指令',
    ],
    [{ system: '系统指令', messages: [] }, '系统指令'],
  ])(
    'reads the system prompt from a recorded provider request',
    (body, expected) => {
      expect(systemPromptFromRequest(body)).toBe(expected)
    }
  )
})

describe('recordPositions', () => {
  it('gives instantaneous input and tool records their own equal-width segments', () => {
    const records = [
      {
        id: 'user',
        run_id: 'run',
        kind: 'user',
        title: '输入',
        status: 'done',
        started_at: '2026-09-22T10:00:00Z',
      },
      {
        id: 'model',
        run_id: 'run',
        kind: 'model',
        title: '模型',
        status: 'done',
        started_at: '2026-09-22T10:00:01Z',
        ended_at: '2026-09-22T10:00:02Z',
      },
      {
        id: 'tool',
        run_id: 'run',
        kind: 'tool',
        title: '工具',
        status: 'done',
        started_at: '2026-09-22T10:00:03Z',
      },
    ]
    const positions = recordPositions(records, false)
    expect(positions.map(({ start, end }) => [start, end])).toEqual([
      [0, 1 / 3],
      [1 / 3, 2 / 3],
      [2 / 3, 1],
    ])
    expect(recordPositions(records, true)[0]).toMatchObject({
      start: 0,
      end: 0,
    })
  })
})
