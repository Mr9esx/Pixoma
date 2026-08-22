import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { cn } from '@/lib/utils'

export type ContextState = 'ready' | 'warn' | 'idle'

export type ContextLinkItem = {
  key: string
  label: string
  to: string
  state?: ContextState
  note?: string
}

export type ContextGroup = {
  title: string
  items: ContextLinkItem[]
}

function stateDot(state: ContextState | undefined): string {
  switch (state) {
    case 'ready':
      return 'bg-emerald-500'
    case 'warn':
      return 'bg-amber-500'
    default:
      return 'bg-zinc-400'
  }
}

/**
 * 关联上下文面板：上游/下游引用 + 就绪状态 + 跳转。
 * 统一用于 Case / Topic / 节点 / 菜单详情，降低跨模块配置心智负担。
 */
export function ContextLinks({
  groups,
  className,
}: {
  groups: ContextGroup[]
  className?: string
}) {
  const { t } = useTranslation()
  return (
    <Card className={className} data-testid='context-links'>
      <CardHeader className='pb-2'>
        <CardTitle className='text-sm font-medium'>
          {t('configContext.title')}
        </CardTitle>
      </CardHeader>
      <CardContent className='space-y-4'>
        {groups.map((group) => (
          <div key={group.title} className='space-y-1.5'>
            <p className='text-xs font-medium text-muted-foreground'>
              {group.title}
            </p>
            <ul className='space-y-1'>
              {group.items.map((item) => (
                <li key={item.key} className='flex items-center gap-2 text-sm'>
                  <span
                    className={cn('size-2 shrink-0 rounded-full', stateDot(item.state))}
                    aria-hidden='true'
                  />
                  <Link
                    to={item.to}
                    className='min-w-0 truncate text-foreground hover:underline'
                  >
                    {item.label}
                  </Link>
                  {item.note ? (
                    <span className='truncate text-xs text-muted-foreground'>
                      {item.note}
                    </span>
                  ) : null}
                  <Badge
                    variant='outline'
                    className={cn(
                      'ml-auto shrink-0 text-[11px]',
                      item.state === 'ready' && 'text-emerald-600',
                      item.state === 'warn' && 'text-amber-600'
                    )}
                  >
                    {item.state === 'ready'
                      ? t('configContext.ready')
                      : item.state === 'warn'
                        ? t('configContext.warn')
                        : t('configContext.idle')}
                  </Badge>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </CardContent>
    </Card>
  )
}
