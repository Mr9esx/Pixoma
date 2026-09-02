import { useQuery } from '@tanstack/react-query'
import { queryKeys } from '@/lib/api/query-keys'
import { listTextTemplates } from '@/lib/api/text-templates'
import { cn } from '@/lib/utils'
import { composeListTasksPreview } from './list-tasks-copy'

export { composeListTasksPreview } from './list-tasks-copy'

export function ListTasksPreview({
  channelId,
  className,
}: {
  channelId?: string
  className?: string
}) {
  const q = useQuery({
    queryKey: queryKeys.textTemplates.channel(channelId ?? ''),
    queryFn: () => listTextTemplates(channelId ?? ''),
    enabled: Boolean(channelId),
  })
  const overrides: Record<string, string> = {}
  for (const item of q.data ?? []) {
    overrides[item.key] = item.value || item.default
  }
  return (
    <p
      data-testid='list-tasks-preview'
      className={cn(
        'm-0 rounded-md border border-border bg-background px-3.5 py-3 text-sm whitespace-pre-wrap',
        className
      )}
    >
      {composeListTasksPreview(overrides)}
    </p>
  )
}
