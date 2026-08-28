import { useState } from 'react'
import { motion } from 'motion/react'
import { cases } from '@/data/cases'
import { cn } from '@/lib/utils'

export function CaseDirDemo() {
  const [active, setActive] = useState(cases[0].id)
  const selected = cases.find((c) => c.id === active) ?? cases[0]

  return (
    <div className="flex h-full w-full">
      <div className="flex w-1/3 flex-col gap-1 border-r border-border p-3">
        {cases.map((c) => (
          <button
            key={c.id}
            type="button"
            onClick={() => setActive(c.id)}
            className={cn(
              'rounded-md px-3 py-2 text-left text-sm transition-colors',
              c.id === active
                ? 'bg-primary text-primary-foreground'
                : 'text-muted-foreground hover:bg-muted hover:text-foreground',
            )}
          >
            {c.title}
          </button>
        ))}
      </div>

      <div className="flex flex-1 items-center justify-center p-4">
        <motion.div
          key={selected.id}
          initial={{ opacity: 0, x: 8 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ duration: 0.2 }}
          className="text-center"
        >
          <div className="text-lg font-medium text-foreground">{selected.title}</div>
          <p className="mt-1 text-sm text-muted-foreground">{selected.desc}</p>
        </motion.div>
      </div>
    </div>
  )
}
