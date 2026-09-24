import { Paperclip, Sparkles } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

export function StudioReferenceBadge({
  kind,
  label,
}: {
  kind: 'skill' | 'asset'
  label: string
}) {
  return (
    <Badge
      variant='secondary'
      className={cn(
        'mx-0.5 rounded-sm border-0 align-baseline text-foreground',
        kind === 'skill'
          ? 'bg-info/15 [&>svg]:text-info'
          : 'bg-chart-2/15 [&>svg]:text-chart-2'
      )}
    >
      {kind === 'skill' ? <Sparkles /> : <Paperclip />}
      {label}
    </Badge>
  )
}
