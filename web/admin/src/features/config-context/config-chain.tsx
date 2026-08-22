import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { ArrowRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { cn } from '@/lib/utils'

export type ChainState = 'ok' | 'warn'

export type ChainHop = {
  key: string
  kind: string
  label: string
  sub: string
  state: ChainState
  to: string
}

export type ChainQA = { q: string; a: string }

export type ChainDetail = {
  conclusion: string
  rows: ChainQA[]
  actionTo?: string
}

export type ConfigChainProps = {
  health: { state: ChainState; text: string }
  hops: ChainHop[]
  details: Record<string, ChainDetail>
}

/**
 * 配置链路：先给整链健康结论，再按环节（入口→工作流→投放→执行）看细节与阻塞，最后给去配置入口。
 */
export function ConfigChain({ health, hops, details }: ConfigChainProps) {
  const { t } = useTranslation()
  const [active, setActive] = useState(hops[0]?.key ?? '')
  const detail = details[active]
  const blocked = hops.filter((h) => h.state === 'warn').length

  return (
    <Card data-testid='config-chain'>
      <CardContent className='space-y-4 p-4'>
        <div
          className={cn(
            'rounded-md border px-3 py-2 text-sm',
            health.state === 'ok'
              ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300'
              : 'border-amber-500/40 bg-amber-500/10 text-amber-700 dark:text-amber-300'
          )}
          role='status'
        >
          {health.text}
        </div>

        <div className='grid grid-cols-2 gap-2 sm:grid-cols-4'>
          {hops.map((hop) => (
            <button
              key={hop.key}
              type='button'
              onClick={() => setActive(hop.key)}
              className={cn(
                'rounded-lg border p-2 text-left transition-colors',
                active === hop.key
                  ? 'border-foreground bg-foreground/5'
                  : 'border-border bg-card hover:border-foreground/40',
                hop.state === 'warn' && 'border-amber-500/60'
              )}
              data-hop={hop.key}
            >
              <div className='text-[11px] text-muted-foreground'>{hop.kind}</div>
              <div className='truncate text-sm font-medium'>{hop.label}</div>
              <div className='truncate text-[11px] text-muted-foreground'>{hop.sub}</div>
            </button>
          ))}
        </div>

        {detail ? (
          <div className='space-y-2'>
            <p className='text-sm'>{detail.conclusion}</p>
            <dl className='space-y-1.5'>
              {detail.rows.map((row) => (
                <div key={row.q} className='flex flex-col gap-0.5 sm:flex-row sm:gap-3'>
                  <dt className='w-28 shrink-0 text-xs text-muted-foreground'>{row.q}</dt>
                  <dd className='text-sm'>{row.a}</dd>
                </div>
              ))}
            </dl>
            {detail.actionTo ? (
              <Button asChild size='sm' variant='outline'>
                <Link to={detail.actionTo}>
                  {t('configChain.goConfig')}
                  <ArrowRight className='size-3.5' />
                </Link>
              </Button>
            ) : null}
          </div>
        ) : null}

        <p className='text-xs text-muted-foreground'>
          {blocked > 0
            ? t('configChain.blockedHops', { n: blocked })
            : t('configChain.chainOk')}
        </p>
      </CardContent>
    </Card>
  )
}
