export function formatDate(d: Date): string {
  const y = d.getFullYear()
  const m = `${d.getMonth() + 1}`.padStart(2, '0')
  const day = `${d.getDate()}`.padStart(2, '0')
  return `${y}-${m}-${day}`
}

export function daysAgo(n: number): string {
  const d = new Date()
  d.setDate(d.getDate() - n)
  return formatDate(d)
}

export const DAILY_PRESETS = [
  { labelKey: 'dashboard.range7d', days: 7 },
  { labelKey: 'dashboard.range30d', days: 30 },
  { labelKey: 'dashboard.range90d', days: 90 },
] as const
