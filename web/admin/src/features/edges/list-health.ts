export type ListHealthTone = 'ok' | 'warn' | 'bad'

export function listHealthTone(input: {
  enabled: boolean
  edgeOnline: boolean
  comfyRunning: boolean
}): ListHealthTone {
  let n = 0
  if (!input.enabled) n++
  if (!input.edgeOnline) n++
  if (!input.comfyRunning) n++
  if (n === 0) return 'ok'
  if (n === 3) return 'bad'
  return 'warn'
}
