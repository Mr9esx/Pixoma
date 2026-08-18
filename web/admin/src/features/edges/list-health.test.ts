import { describe, expect, it } from 'vitest'
import { listHealthTone } from './list-health'

describe('listHealthTone', () => {
  it('is ok when enabled, node online, and Comfy running', () => {
    expect(
      listHealthTone({
        enabled: true,
        edgeOnline: true,
        comfyRunning: true,
      })
    ).toBe('ok')
  })

  it('is warn when exactly one of the three is bad', () => {
    expect(
      listHealthTone({
        enabled: false,
        edgeOnline: true,
        comfyRunning: true,
      })
    ).toBe('warn')
    expect(
      listHealthTone({
        enabled: true,
        edgeOnline: false,
        comfyRunning: true,
      })
    ).toBe('warn')
    expect(
      listHealthTone({
        enabled: true,
        edgeOnline: true,
        comfyRunning: false,
      })
    ).toBe('warn')
  })

  it('is warn when two of the three are bad', () => {
    expect(
      listHealthTone({
        enabled: false,
        edgeOnline: false,
        comfyRunning: true,
      })
    ).toBe('warn')
  })

  it('is bad when enabled, node, and Comfy are all down', () => {
    expect(
      listHealthTone({
        enabled: false,
        edgeOnline: false,
        comfyRunning: false,
      })
    ).toBe('bad')
  })
})
