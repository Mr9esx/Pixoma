const cardWrap =
  'overflow-hidden rounded-[8px] border border-border bg-card'

export const kit = {
  pageSection: 'mx-auto flex w-full max-w-7xl flex-col gap-7 px-6 py-7 md:px-8',
  createPage:
    'mx-auto flex min-h-full w-full max-w-7xl flex-col gap-7 px-6 py-7 md:px-8',
  title: 'truncate text-2xl leading-tight font-semibold tracking-tight',
  tagOn:
    'inline-flex items-center border border-success/25 bg-success/10 py-0.5 h-6 rounded-md px-2 text-xs font-medium text-success',
  tagOff:
    'inline-flex items-center border border-border bg-muted py-0.5 h-6 rounded-md px-2 text-xs font-medium text-muted-foreground',
  tagFail:
    'inline-flex items-center border border-destructive/25 bg-destructive/10 py-0.5 h-6 rounded-md px-2 text-xs font-medium text-destructive',
  tagWarn:
    'inline-flex items-center border border-warning/30 bg-warning/10 py-0.5 h-6 rounded-md px-2 text-xs font-medium text-warning',
  desc: 'text-muted-foreground flex max-w-full flex-wrap items-center gap-x-4 gap-y-2 text-sm',
  btnGhost:
    'border border-input bg-background hover:bg-accent rounded-md text-xs h-8 gap-1.5 px-3',
  btnPrimary:
    'bg-primary text-primary-foreground hover:bg-primary/90 rounded-md text-xs h-8 gap-1.5 px-3',
  specsWrap:
    'border-border bg-muted/20 grid grid-cols-2 overflow-hidden border-y xl:grid-cols-[3fr_3fr_1fr_1fr] xl:border-x-0',
  specsCell:
    'border-border/70 min-w-0 border-b p-3.5 even:border-l sm:p-4 xl:border-b-0 xl:border-l xl:first:border-l-0 [&:nth-last-child(-n+2)]:border-b-0',
  specsLabel:
    'text-muted-foreground flex min-w-0 items-center gap-2 text-xs font-medium',
  specsValue: 'min-w-0 truncate text-sm font-semibold',
  specsNote: 'text-muted-foreground min-w-0 truncate text-xs xl:shrink-0',
  metaChip:
    'flex min-w-0 items-center gap-3 border-border/70 bg-muted/25 rounded-full border px-2.5 py-1.5 sm:border-0 sm:bg-transparent sm:px-0 sm:py-0',
  metaChipDivider: 'bg-border hidden h-4 w-px shrink-0 sm:block',
  cardWrap,
  statsWrap: cardWrap,
  statsGrid: 'grid md:grid-cols-3',
  statsCell: [
    'min-w-0 border-b p-3.5 sm:p-4 md:border-b-0 md:border-r',
    'min-w-0 border-b p-3.5 sm:p-4 md:border-b-0 md:border-r',
    'min-w-0 p-3.5 sm:p-4',
  ],
  statsLabel:
    'text-muted-foreground flex items-center gap-2 text-xs font-medium',
  statsValue: 'mt-1.5 text-sm font-semibold sm:mt-2',
  sectionTitle: 'text-[15px] font-semibold',
  sectionDash: 'border-border min-w-0 flex-1 border-t border-dashed',
  tableWrap: 'overflow-hidden rounded-md border',
  th: 'text-muted-foreground h-10 text-left font-medium',
  tagSmOn:
    'inline-flex items-center border border-success/25 bg-success/10 py-0.5 font-semibold h-5 rounded-md px-1.5 text-[11px] text-success',
  tagSmOff:
    'inline-flex items-center border border-border bg-muted py-0.5 font-semibold h-5 rounded-md px-1.5 text-[11px] text-muted-foreground',
  tagSmFail:
    'inline-flex items-center border border-destructive/25 bg-destructive/10 py-0.5 font-semibold h-5 rounded-md px-1.5 text-[11px] text-destructive',
  healthDot: {
    ok: 'size-2 shrink-0 rounded-full bg-success',
    warn: 'size-2 shrink-0 rounded-full bg-warning',
    bad: 'size-2 shrink-0 rounded-full bg-destructive',
  },
} as const
