import type { ComfyEdge } from '@/lib/api/types'

export type HardwareSummary = {
  gpuCount: number
  vramBytes: number
  cpuCores: number
  ramBytes: number
}

export function summarizeHardware(edges: ComfyEdge[]): HardwareSummary {
  let gpuCount = 0
  let vramBytes = 0
  let cpuCores = 0
  let ramBytes = 0
  for (const e of edges) {
    if (typeof e.hardware?.cpu_cores === 'number') cpuCores += e.hardware.cpu_cores
    if (typeof e.hardware?.ram_bytes === 'number') ramBytes += e.hardware.ram_bytes
    for (const g of e.hardware?.gpus ?? []) {
      gpuCount++
      if (typeof g.vram_bytes === 'number') vramBytes += g.vram_bytes
    }
  }
  return { gpuCount, vramBytes, cpuCores, ramBytes }
}

export function formatBytes(bytes: number): string {
  const gb = bytes / 1024 ** 3
  if (gb >= 100) return `${Math.round(gb)} GiB`
  if (gb >= 1) return `${gb.toFixed(1).replace(/\.0$/, '')} GiB`
  const mb = bytes / 1024 ** 2
  return `${Math.round(mb)} MiB`
}
