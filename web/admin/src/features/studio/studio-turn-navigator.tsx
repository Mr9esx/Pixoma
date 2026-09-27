import {
  useCallback,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type RefObject,
} from 'react'
import type { AssistantState } from '@assistant-ui/react'
import {
  defaultRangeExtractor,
  useVirtualizer,
  type Range,
} from '@tanstack/react-virtual'
import { useStickToBottomContext } from 'use-stick-to-bottom'
import styles from './studio-turn-navigator.module.css'

type ThreadMessage = AssistantState['thread']['messages'][number]

type TurnItem = {
  id: string
  turn: number
  prompt: string
  response: string
}

type Props = {
  messages: readonly ThreadMessage[]
  composerRef: RefObject<HTMLDivElement | null>
}

const TURN_SPACING = 10
const RAIL_INSET = 6
const FADE_SIZE = 24

function turnItems(messages: readonly ThreadMessage[]): TurnItem[] {
  const items: TurnItem[] = []
  for (const message of messages) {
    const text = message.parts
      .flatMap((part) => (part.type === 'text' ? [part.text] : []))
      .join('')
      .trim()
    if (message.role === 'user') {
      items.push({
        id: message.id,
        turn: items.length + 1,
        prompt: text.slice(0, 50),
        response: '',
      })
    } else if (message.role === 'assistant' && items.length > 0 && text) {
      const item = items[items.length - 1]
      item.response =
        `${item.response}${item.response ? '\n\n' : ''}${text}`.slice(0, 120)
    }
  }
  return items
}

function preferredBehavior(): ScrollBehavior {
  return matchMedia('(prefers-reduced-motion: reduce)').matches
    ? 'instant'
    : 'smooth'
}

export function StudioTurnNavigator({ messages, composerRef }: Props) {
  const items = useMemo(() => turnItems(messages), [messages])
  const itemsRef = useRef(items)
  itemsRef.current = items
  const { scrollRef, contentRef, stopScroll } = useStickToBottomContext()
  const scrollerRef = useRef<HTMLDivElement>(null)
  const pointerInsideRef = useRef(false)
  const followRef = useRef<{
    index: number
    count: number
    height: number
  } | null>(null)
  const [activeTurn, setActiveTurn] = useState<number | null>(
    () => items.at(-1)?.turn ?? null
  )
  const [previewTurn, setPreviewTurn] = useState<number | null>(null)
  const [focusedTurn, setFocusedTurn] = useState<number | null>(null)
  const [bandHeight, setBandHeight] = useState(0)
  const previewId = useId()

  useEffect(() => {
    const scroll = scrollRef.current
    if (!scroll) return
    let frame = 0
    const update = () => {
      frame = 0
      const currentItems = itemsRef.current
      const last = currentItems.at(-1)
      const composerHeight =
        composerRef.current?.getBoundingClientRect().height ?? 0
      setBandHeight(Math.max(0, scroll.clientHeight - composerHeight))
      if (!last) {
        setActiveTurn(null)
        return
      }
      if (scroll.scrollHeight - scroll.clientHeight - scroll.scrollTop <= 24) {
        setActiveTurn(last.turn)
        return
      }
      const readingLine =
        scroll.getBoundingClientRect().top +
        Math.min(120, scroll.clientHeight / 4)
      let visible = currentItems[0]
      for (const anchor of scroll.querySelectorAll<HTMLElement>(
        '[data-studio-turn-id]'
      )) {
        if (anchor.getBoundingClientRect().top > readingLine) break
        const item = currentItems.find(
          (candidate) => candidate.id === anchor.dataset.studioTurnId
        )
        if (item) visible = item
      }
      setActiveTurn(visible?.turn ?? last.turn)
    }
    const schedule = () => {
      if (!frame) frame = requestAnimationFrame(update)
    }
    const observer = new ResizeObserver(schedule)
    observer.observe(scroll)
    if (contentRef.current) observer.observe(contentRef.current)
    if (composerRef.current) observer.observe(composerRef.current)
    scroll.addEventListener('scroll', schedule, { passive: true })
    window.addEventListener('resize', schedule)
    schedule()
    return () => {
      cancelAnimationFrame(frame)
      observer.disconnect()
      scroll.removeEventListener('scroll', schedule)
      window.removeEventListener('resize', schedule)
    }
  }, [composerRef, contentRef, items.length, scrollRef])

  const focusedIndex =
    focusedTurn === null
      ? undefined
      : items.findIndex((item) => item.turn === focusedTurn)
  const virtualizer = useVirtualizer({
    count: items.length,
    enabled: items.length > 0,
    getScrollElement: () => scrollerRef.current,
    getItemKey: (index) => items[index]?.id ?? index,
    estimateSize: () => TURN_SPACING,
    overscan: 3,
    paddingStart: RAIL_INSET - TURN_SPACING / 2,
    paddingEnd: RAIL_INSET - TURN_SPACING / 2,
    scrollPaddingStart: FADE_SIZE,
    scrollPaddingEnd: FADE_SIZE,
    initialRect: { width: 28, height: 0 },
    rangeExtractor: (range: Range) => {
      const indexes = defaultRangeExtractor(range)
      if (focusedIndex !== undefined && focusedIndex >= 0) {
        for (
          let index = Math.max(0, focusedIndex - 1);
          index <= Math.min(range.count - 1, focusedIndex + 1);
          index++
        ) {
          if (!indexes.includes(index)) indexes.push(index)
        }
        indexes.sort((left, right) => left - right)
      }
      return indexes
    },
  })
  const viewHeight = virtualizer.scrollRect?.height ?? 0
  const scrollTop = virtualizer.scrollOffset ?? 0
  const activeIndex =
    activeTurn === null
      ? -1
      : items.findIndex((item) => item.turn === activeTurn)

  useEffect(() => {
    if (activeIndex < 0 || viewHeight <= 0 || pointerInsideRef.current) return
    const previous = followRef.current
    if (
      previous?.index === activeIndex &&
      previous.count === items.length &&
      previous.height === viewHeight
    )
      return
    followRef.current = {
      index: activeIndex,
      count: items.length,
      height: viewHeight,
    }
    const center = activeIndex * TURN_SPACING + RAIL_INSET
    if (
      center >= scrollTop + FADE_SIZE &&
      center <= scrollTop + viewHeight - FADE_SIZE
    )
      return
    virtualizer.scrollToIndex(activeIndex, {
      align: 'center',
      behavior:
        previous?.count === items.length && previous.height === viewHeight
          ? preferredBehavior()
          : 'auto',
    })
  }, [activeIndex, items.length, scrollTop, viewHeight, virtualizer])

  const navigate = useCallback(
    (item: TurnItem) => {
      const scroll = scrollRef.current
      const anchor = [
        ...(scroll?.querySelectorAll<HTMLElement>('[data-studio-turn-id]') ??
          []),
      ].find((element) => element.dataset.studioTurnId === item.id)
      if (!scroll || !anchor) return
      stopScroll()
      const top =
        anchor.getBoundingClientRect().top -
        scroll.getBoundingClientRect().top +
        scroll.scrollTop -
        16
      scroll.scrollTo({ top, behavior: preferredBehavior() })
      setActiveTurn(item.turn)
    },
    [scrollRef, stopScroll]
  )

  if (items.length === 0) return null
  const preview = items.find((item) => item.turn === previewTurn)
  const previewPosition = virtualizer
    .getVirtualItems()
    .find((item) => item.index === (previewTurn ?? 0) - 1)
  const scrollerClass = [styles.scroller]
  if (scrollTop > 1) scrollerClass.push(styles.fadeTop)
  if (scrollTop < virtualizer.getTotalSize() - viewHeight - 1)
    scrollerClass.push(styles.fadeBottom)

  return (
    <div className={styles.slot}>
      <nav
        aria-label='轮次导航'
        className={styles.frame}
        style={{ '--turn-rail-band': `${bandHeight}px` } as CSSProperties}
        onPointerEnter={() => {
          pointerInsideRef.current = true
        }}
        onPointerLeave={() => {
          pointerInsideRef.current = false
          if (focusedTurn === null) setPreviewTurn(null)
        }}
      >
        <div
          ref={scrollerRef}
          data-slot='studio-turn-rail-scroll'
          className={scrollerClass.join(' ')}
        >
          <div
            className={styles.marks}
            style={{ height: virtualizer.getTotalSize() }}
          >
            {virtualizer.getVirtualItems().map((virtualItem) => {
              const item = items[virtualItem.index]
              if (!item) return null
              return (
                <button
                  key={virtualItem.key}
                  type='button'
                  data-index={virtualItem.index}
                  className={[
                    styles.mark,
                    item.turn === activeTurn ? styles.active : '',
                    item.turn === previewTurn && item.turn !== activeTurn
                      ? styles.previewMark
                      : '',
                  ]
                    .filter(Boolean)
                    .join(' ')}
                  style={{ transform: `translateY(${virtualItem.start}px)` }}
                  aria-label={`跳转到第 ${item.turn} 轮`}
                  aria-current={item.turn === activeTurn ? 'true' : undefined}
                  aria-describedby={
                    item.turn === previewTurn ? previewId : undefined
                  }
                  onPointerMove={() => {
                    if (focusedTurn === null) setPreviewTurn(item.turn)
                  }}
                  onFocus={() => {
                    setFocusedTurn(item.turn)
                    setPreviewTurn(item.turn)
                  }}
                  onBlur={() => {
                    setFocusedTurn(null)
                    setPreviewTurn(null)
                  }}
                  onClick={() => navigate(item)}
                />
              )
            })}
          </div>
        </div>
        {preview && previewPosition && (
          <div
            id={previewId}
            role='tooltip'
            className={styles.preview}
            style={
              {
                '--turn-preview-center': `${previewPosition.start + previewPosition.size / 2 - scrollTop}px`,
              } as CSSProperties
            }
          >
            <div className={styles.previewPrompt}>
              跳转到第 {preview.turn} 轮
            </div>
            {preview.prompt && (
              <div className={styles.previewResponse}>{preview.prompt}</div>
            )}
            {preview.response && (
              <div className={styles.previewResponse}>{preview.response}</div>
            )}
          </div>
        )}
      </nav>
    </div>
  )
}
