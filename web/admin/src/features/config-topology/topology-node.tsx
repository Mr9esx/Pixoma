import { Handle, Position, type Node, type NodeProps } from '@xyflow/react'
import { Boxes, Radio, Server, Tags } from 'lucide-react'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { StatusDot } from '@/components/status-dot'
import { cn } from '@/lib/utils'
import type { GraphNode } from './lib/build-link-graph'

export type TopologyNodeType = Node<
  GraphNode & { highlighted: boolean; dimmed: boolean; isFocus: boolean },
  'topology'
>

const kindKey: Record<GraphNode['kind'], string> = {
  platform: 'topology.kindPlatform',
  case: 'topology.kindCase',
  topic: 'topology.kindTopic',
  edge: 'topology.kindEdge',
}

const kindIcon: Record<GraphNode['kind'], typeof Radio> = {
  platform: Radio,
  case: Boxes,
  topic: Tags,
  edge: Server,
}

export function TopologyNode({ data }: NodeProps<TopologyNodeType>) {
  const { t } = useTranslation()
  const Icon = kindIcon[data.kind]
  return (
    <div
      className={cn(
        'rounded-md border bg-card px-3 py-2',
        data.isFocus && 'border-primary bg-muted/60',
        data.dimmed && 'opacity-30'
      )}
    >
      <Handle type='target' position={Position.Left} className='opacity-0' />
      <div className='flex flex-col gap-1'>
        <div className='flex items-center gap-1.5'>
          <StatusDot
            problems={data.health === 'ok' ? 0 : 1}
            label={data.health === 'ok' ? t('linkHealth.stateOk') : t('linkHealth.stateWarn')}
          />
          <Icon className='size-3.5 shrink-0 text-muted-foreground' aria-hidden />
          <span className='text-xs text-muted-foreground'>
            {t(kindKey[data.kind])}
          </span>
        </div>
        <p className='max-w-[140px] truncate text-sm font-medium'>{data.name}</p>
        {data.highlighted ? (
          <Link
            to={data.to}
            className='nodrag text-xs font-medium underline underline-offset-2'
          >
            {t('topology.openDetail')}
          </Link>
        ) : null}
      </div>
      <Handle type='source' position={Position.Right} className='opacity-0' />
    </div>
  )
}
