import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { CircleCheck } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { SectionHead } from '@/components/section-head'
import type { EntityHealth, ReferenceItem } from './lib/references'

const stateClass: Record<string, string> = {
  ok: 'border-emerald-600/20 bg-emerald-50 text-emerald-700 dark:border-emerald-400/20 dark:bg-emerald-900/30 dark:text-emerald-400',
  warn: 'border-amber-500/40 bg-amber-500/10 text-amber-700 dark:border-amber-400/20 dark:bg-amber-400/10 dark:text-amber-300',
  bad: 'border-red-600/20 bg-red-50 text-red-700 dark:border-red-400/20 dark:bg-red-900/30 dark:text-red-400',
}

const PAGE_SIZE = 10

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
      className='flex flex-col gap-4'
    >
      <SectionHead title={title} hint={t('linkHealth.sectionHint')} />
      {health.state === 'ok' ? (
        <div
          data-testid='link-health-ok'
          className='flex items-center gap-2 rounded-md border border-emerald-600/20 bg-emerald-50 px-3 py-2.5 text-sm text-emerald-700 dark:border-emerald-400/20 dark:bg-emerald-900/30 dark:text-emerald-400'
        >
          <CircleCheck className='size-4 shrink-0' aria-hidden='true' />
          <span className='font-medium'>{t('linkHealth.stateOk')}</span>
          <span className='text-emerald-700/70 dark:text-emerald-400/70'>
            {t('linkHealth.stateOkDetail')}
          </span>
        </div>
      ) : null}
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
  const [page, setPage] = useState(1)
  const totalPages = Math.max(1, Math.ceil(items.length / PAGE_SIZE))
  const safePage = Math.min(page, totalPages)
  const start = (safePage - 1) * PAGE_SIZE
  const pageItems = items.slice(start, start + PAGE_SIZE)
  return (
    <div
      className='overflow-hidden rounded-md border bg-muted/20'
      data-testid='link-health-reference-list'
    >
      <div className='flex items-center justify-between border-b px-3 py-2'>
        <p className='text-xs font-medium text-muted-foreground'>{title}</p>
        <span className='text-xs text-muted-foreground'>{items.length}</span>
      </div>
      {items.length === 0 ? (
        <p className='px-3 py-2 text-sm text-muted-foreground'>
          {t('linkHealth.none')}
        </p>
      ) : (
        <>
          <ul className='divide-y'>
            {pageItems.map((item) => (
              <li
                key={`${item.id}:${item.name}`}
                className='flex items-center gap-2 px-3 py-2'
              >
                <Link
                  to={item.to}
                  className={cn(
                    'min-w-0 truncate text-sm font-medium text-foreground hover:underline',
                  )}
                >
                  {item.name}
                </Link>
                <span
                  className={cn(
                    'ml-auto shrink-0 rounded-md border px-1.5 py-0.5 text-[11px]',
                    stateClass[item.state],
                  )}
                >
                  {item.state === 'ok'
                    ? t('linkHealth.stateReady')
                    : item.state === 'warn'
                      ? t('linkHealth.stateWarn')
                      : t('linkHealth.stateBad')}
                </span>
              </li>
            ))}
          </ul>
          {totalPages > 1 ? (
            <div className='flex items-center justify-between border-t px-3 py-2'>
              <span className='text-xs text-muted-foreground'>
                {t('linkHealth.page', { page: safePage, total: totalPages })}
              </span>
              <div className='flex gap-2'>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  disabled={safePage <= 1}
                  onClick={() => setPage(safePage - 1)}
                >
                  {t('linkHealth.prev')}
                </Button>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  disabled={safePage >= totalPages}
                  onClick={() => setPage(safePage + 1)}
                >
                  {t('linkHealth.next')}
                </Button>
              </div>
            </div>
          ) : null}
        </>
      )}
    </div>
  )
}
