import { describe, expect, it } from 'vitest'
import { inputKindFor, nodeLabel, outputCountFor } from './node-catalog'

describe('node catalog', () => {
  it('labels common nodes in Chinese and falls back to class type', () => {
    expect(nodeLabel('LoadImage')).toBe('加载图片')
    expect(nodeLabel('CLIPTextEncode')).toBe('写提示词')
    expect(nodeLabel('TotallyUnknownNode')).toBe('TotallyUnknownNode')
  })

  it('infers input kinds for known nodes and returns unknown otherwise', () => {
    expect(inputKindFor('LoadImage', 'image')).toBe('image')
    expect(inputKindFor('KSampler', 'seed')).toBe('number')
    expect(inputKindFor('KSampler', 'scheduler')).toBe('enum')
    expect(inputKindFor('CLIPTextEncode', 'text')).toBe('string')
    expect(inputKindFor('RandomNode', 'anything')).toBe('unknown')
  })

  it('defaults output count to 1 for unknown nodes', () => {
    expect(outputCountFor('SaveImage')).toBe(1)
    expect(outputCountFor('SomethingElse')).toBe(1)
  })
})
