export function formatDateTime(value: string | null | undefined): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime()) || date.getFullYear() <= 1) return '—'
  return new Intl.DateTimeFormat(undefined, {
    timeZone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).format(date)
}

type UserLabelFields = {
  username?: string | null
  first_name?: string | null
  last_name?: string | null
}

export function formatUserLabel(
  user: UserLabelFields | null | undefined,
  fallback?: string
): string {
  const name =
    user?.username ||
    [user?.first_name, user?.last_name].filter(Boolean).join(' ').trim()
  return name || fallback || ''
}
