import { useState, type ReactNode } from 'react'
import { Link } from '@tanstack/react-router'
import { CircleCheck } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { requestFocus, scrollAndFlash } from '@/lib/scroll-focus'
import { cn } from '@/lib/utils'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { SectionHead } from '@/components/section-head'
import type {
  EntityHealth,
  HealthBreakpoint,
  ReferenceItem,
} from './types'

const stateClass: Record<string, string> = {
  ok: 'border-success/25 bg-success/10 text-success',
  warn: 'border-warning/30 bg-warning/10 text-warning',
  pending: 'border-warning/30 bg-warning/10 text-warning',
}

const PAGE_SIZE = 10

// 断点行动点击后要定位闪烁的目标元素 id。
const FOCUS_TARGETS: Record<string, string> = {
  'linkHealth.actionConfigureRouting': 'case-routing-section',
  'linkHealth.actionBindTopic': 'edge-detail',
  'linkHealth.actionManageNodes': 'edge-detail',
  'linkHealth.actionAddEntry': 'channel-menu-section',
  'linkHealth.actionCheckNode': 'link-health-section',
}

type LinkHealthSectionProps = {
  title: string
  health: EntityHealth
  /** 关联列表；不传则不渲染引用区块。 */
  upstream?: { title: string; items: ReferenceItem[] }
  downstream?: { title: string; items: ReferenceItem[] }
  /** 自定义断点行动按钮；返回 null 时回退到默认 Link。 */
  renderAction?: (breakpoint: HealthBreakpoint) => ReactNode | null
}

export function LinkHealthSection({
  title,
  health,
  upstream,
  downstream,
  renderAction,
}: LinkHealthSectionProps) {
  const { t } = useTranslation()
  const blocked = health.breakpoints.length > 0
  return (
    <section
      id='link-health-section'
      data-testid='link-health-section'
      className='flex flex-col gap-4'
    >
      <SectionHead title={title} />
      {health.state === 'ok' && !blocked ? (
        <Alert
          variant='success'
          data-testid='link-health-ok'
          className='flex flex-wrap items-center gap-2'
        >
          <CircleCheck className='size-4 shrink-0' aria-hidden='true' />
          <AlertTitle className='text-sm font-medium'>
            {t('linkHealth.stateOk')}
          </AlertTitle>
          <AlertDescription className='text-sm'>
            {t('linkHealth.stateOkDetail')}
          </AlertDescription>
        </Alert>
      ) : null}
      {blocked ? (
        <div
          className='overflow-hidden rounded-md border'
          data-testid='link-health-breakpoints'
        >
          <ul className='divide-y'>
            {health.breakpoints.map((b, i) => (
              <li
                key={`${b.stage}-${i}`}
                className='flex flex-col gap-1 px-3 py-2.5'
              >
                <div className='flex flex-wrap items-center gap-2'>
                  <span className='text-sm text-warning'>
                    {t(b.key, b.params)}
                  </span>
                  <Badge
                    variant='outline'
                    className={cn(
                      b.fix === 'config'
                        ? 'border-info/30 bg-info/10 text-info-foreground'
                        : 'border-destructive/30 bg-destructive/10 text-destructive'
                    )}
                  >
                    {b.fix === 'config'
                      ? t('linkHealth.fixConfig')
                      : t('linkHealth.fixRuntime')}
                  </Badge>
                  {renderAction?.(b) ?? (
                    <Link
                      to={b.action.to}
                      className='ml-auto text-sm font-medium text-foreground underline underline-offset-2'
                      onClick={(e) => {
                        const focus = FOCUS_TARGETS[b.action.key]
                        if (!focus) return
                        const samePage =
                          b.action.to.replace(/\/$/, '') ===
                          window.location.pathname.replace(/\/$/, '')
                        if (samePage) {
                          e.preventDefault()
                          scrollAndFlash(focus)
                        } else {
                          requestFocus(focus)
                        }
                      }}
                    >
                      {t(b.action.key)} →
                    </Link>
                  )}
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
      {upstream || downstream ? (
        <div className='grid gap-3 sm:grid-cols-2'>
          {upstream ? (
            <ReferenceList title={upstream.title} items={upstream.items} />
          ) : null}
          {downstream ? (
            <ReferenceList title={downstream.title} items={downstream.items} />
          ) : null}
        </div>
      ) : null}
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
                    'min-w-0 truncate text-sm font-medium text-foreground hover:underline'
                  )}
                >
                  {item.name}
                </Link>
                <Badge
                  variant='outline'
                  className={cn('ml-auto shrink-0', stateClass[item.state])}
                >
                  {item.state === 'ok'
                    ? t('linkHealth.stateReady')
                    : item.state === 'warn'
                      ? t('linkHealth.stateWarn')
                      : t('linkHealth.stateBad')}
                </Badge>
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
