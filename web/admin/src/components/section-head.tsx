import { kit } from '@/features/edges/kit-classes'

export function SectionHead({
  title,
  hint,
}: {
  title: string
  hint?: string
}) {
  return (
    <div className='flex flex-col gap-1'>
      <div className='flex items-center gap-3'>
        <h2 className={kit.sectionTitle}>{title}</h2>
        <span className={kit.sectionDash} />
      </div>
      {hint ? <p className='text-xs text-muted-foreground'>{hint}</p> : null}
    </div>
  )
}
