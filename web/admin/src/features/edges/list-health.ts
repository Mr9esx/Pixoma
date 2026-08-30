export type ListHealthTone = 'ok' | 'warn'

export function listHealthTone(input: {
  enabled: boolean
  edgeOnline: boolean
  comfyRunning: boolean
}): ListHealthTone {
  const problems = [!input.enabled, !input.edgeOnline, !input.comfyRunning]
  if (problems.every((hasProblem) => !hasProblem)) return 'ok'
  return 'warn'
}
