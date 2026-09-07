import { Copy } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

export function CopyId({ value }: { value: string }) {
  const { t } = useTranslation()
  if (!value) {
    return (
      <span className='font-mono text-xs text-muted-foreground tabular-nums'>
        —
      </span>
    )
  }

  return (
    <button
      type='button'
      className='inline-flex max-w-full min-w-0 items-center gap-1 font-mono text-xs text-muted-foreground tabular-nums hover:text-foreground'
      aria-label={t('common.copy')}
      onClick={() => {
        void navigator.clipboard.writeText(value).then(
          () => toast.success(t('common.copied')),
          () => toast.error(t('common.errorGeneric'))
        )
      }}
    >
      <span className='min-w-0 truncate'>{value}</span>
      <Copy className='size-3 shrink-0' />
    </button>
  )
}
