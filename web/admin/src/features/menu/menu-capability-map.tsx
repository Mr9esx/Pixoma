import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { Action, Card, Menu } from '@/lib/api/channel-menu'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  actionIsBroken,
  actionOutcomeLabel,
  mediaKindLabelKey,
  orphanCards,
  pushTrailButton,
  sliceTrail,
  trailFromItem,
  trailFromOrphan,
  workflowEntryCount,
  type MapTrailStep,
} from './lib/menu-flow'
import { MenuMapLayout } from './menu-map-layout'
import type { WorkflowRef } from './node-view'
import { WorkflowInfoCard } from './workflow-info-card'

function outcomeText(
  t: (key: string, opts?: { name?: string }) => string,
  action: Action,
  cards: Card[],
  workflows: WorkflowRef[]
): string {
  const o = actionOutcomeLabel(action, cards, workflows)
  return t(`menu.${o.key}`, { name: o.name ?? '' })
}

function payloadKey(type: Action['type']): string | null {
  switch (type) {
    case 'open_workflow':
      return 'menu.mapPayloadWorkflow'
    case 'send_text':
      return 'menu.mapPayloadText'
    case 'copy_text':
      return 'menu.mapPayloadCopy'
    case 'send_media':
      return 'menu.mapPayloadMedia'
    case 'open_url':
      return 'menu.mapPayloadUrl'
    default:
      return null
  }
}

function MediaThumbs({
  media,
  t,
}: {
  media: { kind: string; url: string; caption?: string }[]
  t: (key: string) => string
}) {
  if (media.length === 0) return null
  return (
    <div
      className={cn(
        'grid gap-px bg-border',
        media.length === 1 ? 'grid-cols-1' : 'grid-cols-2'
      )}
    >
      {media.map((m, i) => (
        <div
          key={`${m.url}-${i}`}
          className='grid aspect-video place-items-center bg-muted text-sm text-muted-foreground'
        >
          {m.url ? (
            <img src={m.url} alt='' className='size-full object-cover' />
          ) : (
            t(`menu.${mediaKindLabelKey(m.kind)}`)
          )}
        </div>
      ))}
    </div>
  )
}

function PathBody({
  step,
  cards,
  workflows,
  t,
  onButton,
}: {
  step: MapTrailStep
  cards: Card[]
  workflows: WorkflowRef[]
  t: (key: string, opts?: { name?: string }) => string
  onButton: (button: Card['buttons'][number]) => void
}) {
  const broken = actionIsBroken(step.action, cards, workflows)
  if (broken) {
    const o = actionOutcomeLabel(step.action, cards, workflows)
    return (
      <div className='rounded-md border border-destructive px-3.5 py-3 text-sm text-destructive'>
        {t(`menu.${o.key}`)}
      </div>
    )
  }

  if (step.action.type === 'open_card' && step.action.card_id) {
    const card = cards.find((c) => c.id === step.action.card_id)
    if (!card) return null
    return (
      <div>
        <p className='mb-2 text-sm text-muted-foreground'>{card.name}</p>
        <div className='overflow-hidden rounded-[10px] border border-border bg-background'>
          <MediaThumbs media={card.media ?? []} t={t} />
          <div className='px-3.5 py-3.5'>
            {card.text ? (
              <p className='mb-3 text-sm leading-relaxed'>{card.text}</p>
            ) : null}
            {card.buttons.length > 0 ? (
              <div className='flex flex-wrap gap-2'>
                {card.buttons.map((b) => (
                  <Button
                    key={b.id}
                    type='button'
                    variant='outline'
                    className='h-11 rounded-md px-3 font-medium hover:border-foreground'
                    onClick={() => onButton(b)}
                  >
                    {b.label}
                  </Button>
                ))}
              </div>
            ) : null}
          </div>
        </div>
      </div>
    )
  }

  const pk = payloadKey(step.action.type)
  if (!pk) return null

  if (step.action.type === 'open_workflow') {
    const workflow = step.action.workflow_id
      ? workflows.find((w) => String(w.id) === step.action.workflow_id)
      : undefined
    const o = actionOutcomeLabel(step.action, cards, workflows)
    return (
      <div className='flex flex-col gap-2'>
        <span className='font-mono text-sm font-semibold tracking-wider text-muted-foreground uppercase'>
          {t(pk)}
        </span>
        {workflow ? (
          <WorkflowInfoCard workflow={workflow} />
        ) : (
          <p className='m-0 text-sm font-semibold'>{o.name}</p>
        )}
      </div>
    )
  }

  if (step.action.type === 'send_text' || step.action.type === 'copy_text') {
    return (
      <div className='flex flex-col gap-2'>
        <span className='font-mono text-sm font-semibold tracking-wider text-muted-foreground uppercase'>
          {t(pk)}
        </span>
        <p className='m-0 rounded-md border border-border bg-background px-3.5 py-3 text-sm whitespace-pre-wrap'>
          {step.action.text}
        </p>
      </div>
    )
  }

  if (step.action.type === 'send_media') {
    return (
      <div className='flex flex-col gap-2'>
        <span className='font-mono text-sm font-semibold tracking-wider text-muted-foreground uppercase'>
          {t(pk)}
        </span>
        <MediaThumbs media={step.action.media ?? []} t={t} />
      </div>
    )
  }

  if (step.action.type === 'open_url') {
    return (
      <div className='flex flex-col gap-2'>
        <span className='font-mono text-sm font-semibold tracking-wider text-muted-foreground uppercase'>
          {t(pk)}
        </span>
        <p className='m-0 font-mono text-sm font-semibold break-all'>
          {step.action.url}
        </p>
      </div>
    )
  }

  return null
}

export function MenuCapabilityMap({
  menu,
  cards,
  workflows,
  onEdit,
}: {
  menu: Menu
  cards: Card[]
  workflows: WorkflowRef[]
  onEdit: () => void
}) {
  const { t } = useTranslation()
  const [trail, setTrail] = useState<MapTrailStep[]>([])
  const cols = menu.columns >= 1 && menu.columns <= 8 ? menu.columns : 2
  const orphans = orphanCards(menu, cards)
  const last = trail[trail.length - 1]
  const selectedItemId = trail[0]?.kind === 'item' ? trail[0].id : null
  const selectedOrphanId = trail[0]?.kind === 'orphan' ? trail[0].id : null
  const crumbRoot =
    trail[0]?.kind === 'orphan'
      ? t('menu.mapCrumbOrphan')
      : t('menu.mainKeyboard')

  return (
    <MenuMapLayout
      testId='menu-capability-map'
      toolbar={
        <>
          <div className='flex min-w-0 flex-wrap items-center gap-3'>
            <div className='flex gap-3.5 font-mono text-sm text-muted-foreground tabular-nums'>
              <span>{t('menu.mapCountKeys', { n: menu.items.length })}</span>
              <span>{t('menu.mapCountCards', { n: cards.length })}</span>
              <span>
                {t('menu.mapCountWorkflows', {
                  n: workflowEntryCount(menu, cards),
                })}
              </span>
            </div>
          </div>
          <Button
            type='button'
            size='sm'
            data-testid='edit-menu'
            onClick={onEdit}
          >
            {t('menu.editMenu')}
          </Button>
        </>
      }
      keyboard={
        <>
          <p className='mb-3 font-mono text-sm font-semibold tracking-wider text-muted-foreground uppercase'>
            {t('menu.mainKeyboard')}
          </p>
          <div
            data-testid='map-keyboard'
            className='grid gap-2'
            style={{ gridTemplateColumns: `repeat(${cols}, minmax(0, 1fr))` }}
          >
            {menu.items.length === 0 ? (
              <p className='m-0 text-sm text-muted-foreground'>
                {t('menu.mapEmpty')}
              </p>
            ) : (
              menu.items.map((it) => {
                const broken = actionIsBroken(it.action, cards, workflows)
                return (
                  <Button
                    key={it.id}
                    type='button'
                    data-testid='map-key'
                    aria-pressed={it.id === selectedItemId}
                    variant='outline'
                    className={cn(
                      'flex h-11 flex-col items-start justify-center gap-0.5 rounded-md px-3 py-2 text-left',
                      'hover:border-foreground/35',
                      it.id === selectedItemId && 'border-foreground bg-muted',
                      broken && 'border-destructive'
                    )}
                    onClick={() => setTrail(trailFromItem(it))}
                  >
                    <span className='text-sm leading-snug font-semibold'>
                      {it.label}
                    </span>
                    <span
                      className={cn(
                        'w-full truncate text-sm leading-snug text-muted-foreground',
                        broken && 'text-destructive'
                      )}
                    >
                      {outcomeText(t, it.action, cards, workflows)}
                    </span>
                  </Button>
                )
              })
            )}
          </div>
        </>
      }
      path={
        <>
          <p className='mb-3 font-mono text-sm font-semibold tracking-wider text-muted-foreground uppercase'>
            {t('menu.mapPath')}
          </p>
          {last ? (
            <>
              <div className='mb-3.5 flex flex-wrap items-center gap-1 text-sm text-muted-foreground'>
                <span>{crumbRoot}</span>
                {trail.map((step, i) => (
                  <span
                    key={`${step.kind}-${step.id}-${i}`}
                    className='contents'
                  >
                    <span className='opacity-50'>/</span>
                    {i === trail.length - 1 ? (
                      <span className='font-medium text-foreground'>
                        {step.label}
                      </span>
                    ) : (
                      <Button
                        type='button'
                        variant='ghost'
                        size='sm'
                        className='h-auto rounded-none bg-transparent p-0 text-sm text-muted-foreground hover:bg-transparent hover:text-foreground'
                        onClick={() => setTrail((cur) => sliceTrail(cur, i))}
                      >
                        {step.label}
                      </Button>
                    )}
                  </span>
                ))}
              </div>
              <PathBody
                step={last}
                cards={cards}
                workflows={workflows}
                t={t}
                onButton={(b) => setTrail((cur) => pushTrailButton(cur, b))}
              />
            </>
          ) : null}
        </>
      }
      orphans={
        orphans.length > 0 ? (
          <div
            data-testid='map-orphans'
            className='border-t border-border px-4 py-3 pb-4'
          >
            <p className='mb-3 font-mono text-sm font-semibold tracking-wider text-muted-foreground uppercase'>
              {t('menu.mapOrphans', { n: orphans.length })}
            </p>
            <div className='flex flex-wrap gap-2'>
              {orphans.map((c) => (
                <Button
                  key={c.id}
                  type='button'
                  aria-pressed={c.id === selectedOrphanId}
                  variant='outline'
                  className={cn(
                    'h-11 rounded-md border-dashed px-3 text-sm',
                    'hover:border-solid hover:border-foreground',
                    c.id === selectedOrphanId &&
                      'border-solid border-foreground bg-muted'
                  )}
                  onClick={() => setTrail(trailFromOrphan(c))}
                >
                  {c.name}
                </Button>
              ))}
            </div>
          </div>
        ) : null
      }
    />
  )
}
