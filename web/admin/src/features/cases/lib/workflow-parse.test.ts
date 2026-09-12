import { describe, expect, it } from 'vitest'
import { parseWorkflow } from './workflow-parse'

const API_JSON = JSON.stringify({
  '1': { class_type: 'LoadImage', inputs: { image: 'ref.png' } },
  '2': { class_type: 'CLIPTextEncode', inputs: { text: 'hi' } },
  '3': { class_type: 'SaveImage', inputs: { filename_prefix: 'out' } },
})

const UI_JSON = JSON.stringify({
  nodes: [
    {
      id: 1,
      type: 'LoadImage',
      inputs: [{ name: 'image', link: null }],
      widgets_values: ['ref.png'],
    },
    {
      id: 2,
      type: 'CLIPTextEncode',
      inputs: [{ name: 'text', link: null }],
      widgets_values: ['hi'],
    },
    {
      id: 3,
      type: 'SaveImage',
      inputs: [{ name: 'filename_prefix', link: null }],
      widgets_values: ['out'],
    },
  ],
  links: [],
})

const WIDGET_API_JSON = JSON.stringify({
  '10': {
    class_type: 'LoadImage',
    inputs: { widget_0: 'ref.png', widget_1: 'image' },
  },
})

describe('parseWorkflow', () => {
  it('parses ComfyUI API format as-is', () => {
    const result = parseWorkflow(API_JSON)
    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.graph.nodes).toHaveLength(3)
    expect(result.graph.nodes[0]).toMatchObject({
      id: '1',
      class_type: 'LoadImage',
      outputCount: 1,
    })
    expect(result.graph.nodes[0].inputs[0]).toEqual({
      name: 'image',
      kind: 'image',
      ref: false,
    })
    expect(result.graph.nodes[0].literals).toEqual([['image', 'ref.png']])
    expect(result.graph.nodes[0].links).toEqual([])
  })

  it('keeps connected inputs as node links in API format', () => {
    const raw = JSON.stringify({
      '1': { class_type: 'LoadImage', inputs: { image: 'ref.png' } },
      '2': {
        class_type: 'SaveImage',
        inputs: { images: ['1', 0], filename_prefix: 'out' },
      },
    })
    const result = parseWorkflow(raw)
    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.graph.api['2']).toEqual({
      class_type: 'SaveImage',
      inputs: { images: ['1', 0], filename_prefix: 'out' },
    })
  })

  it('rejects ComfyUI UI format with a clear error', () => {
    const result = parseWorkflow(UI_JSON)
    expect(result.ok).toBe(false)
    if (result.ok) return
    expect(result.error).toBe('cases.importNotApiFormat')
  })

  it('rejects API-format JSON that still carries widget_N keys', () => {
    const result = parseWorkflow(WIDGET_API_JSON)
    expect(result.ok).toBe(false)
    if (result.ok) return
    expect(result.error).toBe('cases.importReasonWidgetPlaceholder')
  })

  it('returns readable errors for invalid input', () => {
    expect(parseWorkflow('not json').ok).toBe(false)
    expect(parseWorkflow(JSON.stringify([1, 2])).ok).toBe(false)
    expect(parseWorkflow(JSON.stringify({ nodes: [{ id: 1 }] })).ok).toBe(false)
    expect(parseWorkflow(JSON.stringify({})).ok).toBe(false)
  })
})
