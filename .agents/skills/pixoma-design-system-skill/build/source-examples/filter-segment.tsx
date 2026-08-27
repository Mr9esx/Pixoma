import { ChevronLeft, ChevronRight } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import { cn } from '@/lib/utils'

export type FilterSegmentOption<T extends string> = {
  value: T
  label: string
}

type Props<T extends string> = {
  value: T
  options: FilterSegmentOption<T>[]
  onValueChange: (value: T) => void
  'aria-label': string
  'data-testid'?: string
  className?: string
}

const SCROLL_EDGE = 2
const PAGE_MIN = 80
/** Fewer than this many options → equal-width fill; otherwise horizontal scroll. */
const FILL_BELOW = 5

function measure(el: HTMLElement) {
  const maxScroll = Math.max(0, el.scrollWidth - el.clientWidth)
  return {
    maxScroll,
    canScrollLeft: el.scrollLeft > SCROLL_EDGE,
    canScrollRight: el.scrollLeft < maxScroll - SCROLL_EDGE,
  }
}

/**
 * Compact horizontal segment for enum filters in Master–Detail list panels.
 * <5 options fill the row evenly; ≥5 use in-segment chevrons (no scrollbar).
 */
export function FilterSegment<T extends string>({
  value,
  options,
  onValueChange,
  'aria-label': ariaLabel,
  'data-testid': testId,
  className,
}: Props<T>) {
  const scrollerRef = useRef<HTMLDivElement>(null)
  const activeRef = useRef<HTMLButtonElement>(null)
  const [canScrollLeft, setCanScrollLeft] = useState(false)
  const [canScrollRight, setCanScrollRight] = useState(false)
  const fill = options.length < FILL_BELOW

  const syncChevrons = useCallback(() => {
    const el = scrollerRef.current
    if (!el) return
    if (fill) {
      setCanScrollLeft(false)
      setCanScrollRight(false)
      return
    }
    const { canScrollLeft: left, canScrollRight: right } = measure(el)
    setCanScrollLeft(left)
    setCanScrollRight(right)
  }, [fill])

  useEffect(() => {
    const el = scrollerRef.current
    if (!el) return
    syncChevrons()
    if (fill) return
    el.addEventListener('scroll', syncChevrons, { passive: true })
    const ro = new ResizeObserver(syncChevrons)
    ro.observe(el)
    return () => {
      el.removeEventListener('scroll', syncChevrons)
      ro.disconnect()
    }
  }, [syncChevrons, options.length, fill])

  // Only when the selected value changes — never while the user is paging.
  useEffect(() => {
    if (fill) return
    const scroller = scrollerRef.current
    const active = activeRef.current
    if (!scroller || !active) return
    active.scrollIntoView({
      behavior: 'smooth',
      inline: 'nearest',
      block: 'nearest',
    })
  }, [value, fill])

  const scrollPage = (dir: -1 | 1) => {
    const el = scrollerRef.current
    if (!el || fill) return
    const step = Math.max(el.clientWidth * 0.75, PAGE_MIN) * dir
    el.scrollBy({ left: step, behavior: 'smooth' })
  }

  return (
    <div
      role='group'
      aria-label={ariaLabel}
      data-testid={testId}
      className={cn(
        'relative flex items-center overflow-hidden rounded-md border',
        className,
      )}
    >
      {!fill ? (
        <button
          type='button'
          aria-label='Scroll filters left'
          aria-hidden={!canScrollLeft}
          tabIndex={canScrollLeft ? 0 : -1}
          data-testid={testId ? `${testId}-prev` : undefined}
          className={cn(
            'bg-background/90 text-muted-foreground hover:text-foreground absolute top-1/2 left-1.5 z-10 flex size-5 -translate-y-1/2 items-center justify-center rounded-full border backdrop-blur-sm transition-opacity duration-300',
            canScrollLeft
              ? 'opacity-100'
              : 'pointer-events-none opacity-0',
          )}
          onClick={() => scrollPage(-1)}
        >
          <ChevronLeft className='size-3' />
        </button>
      ) : null}

      <div
        ref={scrollerRef}
        className={cn(
          'flex min-w-0 flex-1 gap-1 p-0.5',
          fill ? 'w-full' : 'overflow-x-auto',
          !fill &&
            '[scrollbar-width:none] [-ms-overflow-style:none] [&::-webkit-scrollbar]:hidden',
        )}
      >
        {options.map((opt) => {
          const active = value === opt.value
          return (
            <button
              key={opt.value}
              ref={active ? activeRef : undefined}
              type='button'
              aria-pressed={active}
              className={cn(
                'rounded-sm px-2 py-1 text-xs whitespace-nowrap transition-colors',
                fill ? 'min-w-0 flex-1 text-center' : 'shrink-0',
                active
                  ? 'bg-accent text-accent-foreground font-medium'
                  : 'text-muted-foreground hover:text-foreground',
              )}
              onClick={() => onValueChange(opt.value)}
            >
              {opt.label}
            </button>
          )
        })}
      </div>

      {!fill ? (
        <button
          type='button'
          aria-label='Scroll filters right'
          aria-hidden={!canScrollRight}
          tabIndex={canScrollRight ? 0 : -1}
          data-testid={testId ? `${testId}-next` : undefined}
          className={cn(
            'bg-background/90 text-muted-foreground hover:text-foreground absolute top-1/2 right-1.5 z-10 flex size-5 -translate-y-1/2 items-center justify-center rounded-full border backdrop-blur-sm transition-opacity duration-300',
            canScrollRight
              ? 'opacity-100'
              : 'pointer-events-none opacity-0',
          )}
          onClick={() => scrollPage(1)}
        >
          <ChevronRight className='size-3' />
        </button>
      ) : null}
    </div>
  )
}
