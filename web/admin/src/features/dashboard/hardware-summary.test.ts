import { describe, expect, it } from 'vitest'
import { formatBytes, summarizeHardware } from './hardware-summary'
import type { ComfyEdge } from '@/lib/api/types'

describe('hardware-summary', () => {
  it('sums gpu/vram/cores/ram across edges', () => {
    const edges = [
      {
        id: 'gpu-1',
        name: 'gpu-1',
        enabled: true,
        capabilities: [],
        comfy_version: '',
        created_at: '',
        updated_at: '',
        started_at: null,
        hardware: {
          cpu_cores: 16,
          ram_bytes: 128 * 1024 ** 3,
          gpus: [
            { name: 'A', vram_bytes: 24 * 1024 ** 3 },
            { name: 'B', vram_bytes: 24 * 1024 ** 3 },
          ],
        },
      },
      {
        id: 'gpu-2',
        name: 'gpu-2',
        enabled: true,
        capabilities: [],
        comfy_version: '',
        created_at: '',
        updated_at: '',
        started_at: null,
        hardware: { cpu_cores: 8, ram_bytes: 64 * 1024 ** 3, gpus: [] },
      },
    ] as ComfyEdge[]
    const s = summarizeHardware(edges)
    expect(s).toEqual({
      gpuCount: 2,
      vramBytes: 48 * 1024 ** 3,
      cpuCores: 24,
      ramBytes: 192 * 1024 ** 3,
    })
  })

  it('formats bytes in GiB', () => {
    expect(formatBytes(96 * 1024 ** 3)).toBe('96 GiB')
    expect(formatBytes(12 * 1024 ** 3)).toBe('12 GiB')
  })
})
