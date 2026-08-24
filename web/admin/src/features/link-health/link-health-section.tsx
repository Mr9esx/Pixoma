import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { Card, CardContent } from '@/components/ui/card'
import { cn } from '@/lib/utils'
import type { EntityHealth, ReferenceItem } from './lib/references'

const stateClass: Record<string, string> = {
  ok: 'border-emerald-500/40 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300',
  warn: 'border-amber-500/40 bg-amber-500/10 text-amber-700 dark:text-amber-300',
  bad: 'border-red-500/40 bg-red-500/10 text-red-700 dark:text-red-300',
}

export type LinkHealthSectionProps = {
  title: string
  health: EntityHealth
  upstream: { title: string; items: ReferenceItem[] }
  downstream: { title: string; items: ReferenceItem[] }
}

export function LinkHealthSection({
  title,
  health,
  upstream,
  downstream,
}: LinkHealthSectionProps) {
  const { t } = useTranslation()
  const blocked = health.breakpoints.length > 0
  return (
    <Card id='link-health-section' data-testid='link-health-section'>
      <CardContent className='space-y-4 p-4'>
        <h3 className='text-sm font-semibold'>{title}</h3>
        {blocked ? (
          <ul
            className='space-y-2 text-xs'
            data-testid='link-health-breakpoints'
          >
            {health.breakpoints.map((b, i) => (
              <li key={`${b.stage}-${i}`} className='space-y-1'>
                <div className='flex flex-wrap items-center gap-2'>
                  <span className='text-amber-700 dark:text-amber-300'>
                    {t(b.key, b.params)}
                  </span>
                  <span
                    className={cn(
                      'rounded-sm px-1.5 py-0.5 text-[10px]',
                      b.fix === 'config'
                        ? 'bg-sky-500/10 text-sky-700 dark:text-sky-300'
                        : 'bg-rose-500/10 text-rose-700 dark:text-rose-300',
                    )}
                  >
                    {b.fix === 'config'
                      ? t('linkHealth.fixConfig')
                      : t('linkHealth.fixRuntime')}
                  </span>
                  <Link
                    to={b.action.to}
                    className='font-medium text-foreground underline underline-offset-2'
                  >
                    {t(b.action.key)} →
                  </Link>
                </div>
                <p
                  className='text-xs text-muted-foreground'
                  data-testid='link-health-guide'
                >
                  {t('linkHealth.howToHandle')}：{t(b.guide)}
                </p>
              </li>
            ))}
          </ul>
        ) : null}
        <div className='grid gap-4 sm:grid-cols-2'>
          <ReferenceList title={upstream.title} items={upstream.items} />
          <ReferenceList title={downstream.title} items={downstream.items} />
        </div>
      </CardContent>
    </Card>
  )
}

function ReferenceList({
  title,
  items,
}: {
  title: string
  items: ReferenceItem[]
}) {
  const { t } = useTranslation()
  return (
    <div className='space-y-2' data-testid='link-health-reference-list'>
      <p className='text-xs font-medium text-muted-foreground'>{title}</p>
      {items.length === 0 ? (
        <p className='text-xs text-muted-foreground'>{t('linkHealth.none')}</p>
      ) : (
        <ul className='flex flex-wrap gap-2'>
          {items.map((item) => (
            <li key={`${item.id}:${item.name}`}>
              <Link
                to={item.to}
                className={cn(
                  'rounded-sm border px-2 py-1 text-xs',
                  stateClass[item.state],
                )}
              >
                {item.name}
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
