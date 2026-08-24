import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { SectionHead } from '@/features/edges/observation-panel'
import type { EntityHealth, ReferenceItem } from './lib/references'

const stateClass: Record<string, string> = {
  ok: 'border-emerald-600/20 bg-emerald-50 text-emerald-700 dark:border-emerald-400/20 dark:bg-emerald-900/30 dark:text-emerald-400',
  warn: 'border-amber-500/40 bg-amber-500/10 text-amber-700 dark:border-amber-400/20 dark:bg-amber-400/10 dark:text-amber-300',
  bad: 'border-red-600/20 bg-red-50 text-red-700 dark:border-red-400/20 dark:bg-red-900/30 dark:text-red-400',
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
    <section
      id='link-health-section'
      data-testid='link-health-section'
      className='space-y-3'
    >
      <SectionHead title={title} hint={t('linkHealth.sectionHint')} />
      {blocked ? (
        <div
          className='overflow-hidden rounded-md border'
          data-testid='link-health-breakpoints'
        >
          <ul className='divide-y'>
            {health.breakpoints.map((b, i) => (
              <li key={`${b.stage}-${i}`} className='space-y-1 px-3 py-2.5'>
                <div className='flex flex-wrap items-center gap-2'>
                  <span className='text-sm text-amber-700 dark:text-amber-300'>
                    {t(b.key, b.params)}
                  </span>
                  <span
                    className={cn(
                      'rounded-sm px-1.5 py-0.5 text-[11px]',
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
                    className='ml-auto text-sm font-medium text-foreground underline underline-offset-2'
                  >
                    {t(b.action.key)} →
                  </Link>
                </div>
                <p
                  className='text-sm text-muted-foreground'
                  data-testid='link-health-guide'
                >
                  {t('linkHealth.howToHandle')}：{t(b.guide)}
                </p>
              </li>
            ))}
          </ul>
        </div>
      ) : null}
      <div className='grid gap-3 sm:grid-cols-2'>
        <ReferenceList title={upstream.title} items={upstream.items} />
        <ReferenceList title={downstream.title} items={downstream.items} />
      </div>
    </section>
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
    <div
      className='rounded-md border bg-muted/20 p-3'
      data-testid='link-health-reference-list'
    >
      <p className='text-xs font-medium text-muted-foreground'>{title}</p>
      {items.length === 0 ? (
        <p className='mt-1.5 text-xs text-muted-foreground'>
          {t('linkHealth.none')}
        </p>
      ) : (
        <ul className='mt-2 flex flex-wrap gap-1.5'>
          {items.map((item) => (
            <li key={`${item.id}:${item.name}`}>
              <Link
                to={item.to}
                className={cn(
                  'inline-flex items-center rounded-md border px-2 py-1 text-xs',
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
