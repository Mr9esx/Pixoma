export const kit = {
  pageSection: 'mx-auto flex w-full max-w-7xl flex-col gap-7 px-6 py-7 md:px-8',
  header: 'flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between',
  title: 'truncate text-2xl leading-tight font-semibold tracking-tight',
  tagOn:
    'inline-flex items-center border py-0.5 h-6 rounded-md border-emerald-600/20 bg-emerald-50 px-2 text-xs font-medium text-emerald-700 shadow-none dark:border-emerald-400/20 dark:bg-emerald-900/30 dark:text-emerald-400',
  tagOff:
    'inline-flex items-center border py-0.5 h-6 rounded-md border-zinc-300 bg-zinc-50 px-2 text-xs font-medium text-zinc-700 shadow-none dark:border-zinc-700 dark:bg-zinc-900/60 dark:text-zinc-300',
  desc: 'text-muted-foreground flex max-w-full flex-wrap items-center gap-x-4 gap-y-2 text-sm',
  btnGhost:
    'border border-input bg-background shadow-xs hover:bg-accent rounded-md text-xs h-8 gap-1.5 px-3',
  btnPrimary:
    'bg-primary text-primary-foreground shadow-sm hover:bg-primary/90 rounded-md text-xs h-8 gap-1.5 px-3',
  rule: 'bg-border shrink-0 h-[1px] w-full',
  dl: 'grid gap-x-20 gap-y-4 text-sm md:grid-cols-2',
  field: 'grid grid-cols-[7rem_minmax(0,1fr)] items-start gap-1',
  dt: 'text-muted-foreground flex items-center gap-3 font-medium',
  dd: 'min-w-0 truncate font-medium',
  statsWrap:
    'text-card-foreground border bg-muted/15 rounded-lg p-1 shadow-none',
  statsGrid: 'grid gap-1 md:grid-cols-3',
  statsCell: 'bg-background rounded-md border px-4 py-3',
  statsLabel:
    'text-muted-foreground flex items-center gap-2 text-xs font-medium',
  statsValue: 'mt-5 text-2xl font-semibold tracking-tight',
  sectionTitle: 'text-[15px] font-semibold',
  sectionDash: 'border-border min-w-0 flex-1 border-t border-dashed',
  tableWrap: 'overflow-hidden rounded-md border',
  th: 'text-muted-foreground h-10 text-left font-medium',
  tagSmOn:
    'inline-flex items-center border py-0.5 font-semibold h-5 rounded-md px-1.5 text-[11px] shadow-none border-emerald-600/20 bg-emerald-50 text-emerald-700 dark:border-emerald-400/20 dark:bg-emerald-900/30 dark:text-emerald-400',
  tagSmOff:
    'inline-flex items-center border py-0.5 font-semibold h-5 rounded-md px-1.5 text-[11px] shadow-none border-zinc-300 bg-zinc-50 text-zinc-700 dark:border-zinc-700 dark:bg-zinc-900/60 dark:text-zinc-300',
  tagSmFail:
    'inline-flex items-center border py-0.5 font-semibold h-5 rounded-md px-1.5 text-[11px] shadow-none border-red-600/20 bg-red-50 text-red-700 dark:border-red-400/20 dark:bg-red-900/30 dark:text-red-400',
  healthDot: {
    ok: 'size-2 shrink-0 rounded-full bg-emerald-500',
    warn: 'size-2 shrink-0 rounded-full bg-amber-400',
    bad: 'size-2 shrink-0 rounded-full bg-red-500',
  },
} as const
