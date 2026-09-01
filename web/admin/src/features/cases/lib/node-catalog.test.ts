import { describe, expect, it } from 'vitest'
import {
  inputKindFor,
  nodeLabel,
  outputCountFor,
  outputKindFor,
} from './node-catalog'

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

  it('infers loader enums and uses the API value as a fallback', () => {
    expect(inputKindFor('UNETLoader', 'unet_name')).toBe('enum')
    expect(inputKindFor('UNETLoader', 'weight_dtype')).toBe('enum')
    expect(inputKindFor('RandomNode', 'anything', 3)).toBe('number')
    expect(inputKindFor('RandomNode', 'anything', true)).toBe('boolean')
    expect(inputKindFor('RandomNode', 'anything', 'hello')).toBe('string')
  })

  it('defaults output count to 1 for unknown nodes', () => {
    expect(outputCountFor('SaveImage')).toBe(1)
    expect(outputCountFor('SomethingElse')).toBe(1)
  })

  it('infers media output kinds and uses file for unknown nodes', () => {
    expect(outputKindFor('SaveImage')).toBe('image')
    expect(outputKindFor('VHS_VideoCombine')).toBe('video')
    expect(outputKindFor('SaveAudioMP3')).toBe('audio')
    expect(outputKindFor('CLIPTextEncode')).toBe('text')
    expect(outputKindFor('TotallyUnknownNode')).toBe('file')
  })
})
