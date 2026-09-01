import { cn } from '@/lib/utils'

type Props = {
  className?: string
}

export function PixomaLoading({ className }: Props) {
  return (
    <img
      alt=''
      aria-hidden
      src='/images/pixoma-loading.gif?v=3'
      className={cn('size-12', className)}
    />
  )
}
