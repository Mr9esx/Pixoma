import { useState } from 'react'
import { AnimatePresence, motion } from 'motion/react'
import { useTranslation } from 'react-i18next'
import { chatMessages } from '@/data/chat'
import { cn } from '@/lib/utils'

export function BotChatDemo() {
  const { t } = useTranslation()
  const [visible, setVisible] = useState(1)
  const shown = chatMessages.slice(0, visible)
  const hasMore = visible < chatMessages.length

  return (
    <div className="flex h-full w-full flex-col justify-between rounded-xl border border-border bg-card">
      <div className="flex flex-1 flex-col gap-3 overflow-hidden p-4">
        <AnimatePresence initial={false}>
          {shown.map((m, i) => (
            <motion.div
              key={`${m.text}-${i}`}
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.25 }}
              className={cn(
                'max-w-[80%] rounded-2xl px-4 py-2.5 text-sm',
                m.role === 'user'
                  ? 'ml-auto bg-primary text-primary-foreground'
                  : 'bg-muted text-foreground',
              )}
            >
              {m.text}
            </motion.div>
          ))}
        </AnimatePresence>
      </div>

      <div className="flex items-center justify-between border-t border-border p-3">
        <span className="text-xs text-muted-foreground">
          {t('features.bot.title')} · Telegram
        </span>
        <button
          type="button"
          disabled={!hasMore}
          onClick={() => setVisible((v) => Math.min(v + 1, chatMessages.length))}
          className="rounded-md border border-border px-3 py-1.5 text-xs font-medium text-foreground transition-colors hover:bg-muted disabled:opacity-50"
        >
          {t('demo.next')}
        </button>
      </div>
    </div>
  )
}
