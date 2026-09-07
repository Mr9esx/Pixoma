export function opsIdentity(input: {
  userLabel?: string
  caseId?: number | string
}): string {
  const user = input.userLabel?.trim() ?? ''
  const caseId =
    input.caseId === undefined || input.caseId === ''
      ? ''
      : String(input.caseId)
  if (user && caseId) return `${user} · ${caseId}`
  return user || caseId
}

export function lifecycleBadgeClass(status: string): string {
  if (status === 'succeeded') {
    return 'border-success/25 bg-success/10 text-success'
  }
  if (status === 'failed') {
    return 'border-destructive/25 bg-destructive/10 text-destructive'
  }
  return 'border-border bg-muted text-muted-foreground'
}
