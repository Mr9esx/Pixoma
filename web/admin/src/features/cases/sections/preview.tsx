import { useTranslation } from 'react-i18next'
import type { InputBinding, OutputBinding } from '@/lib/api/types'

type Props = {
  bindings: { inputs: InputBinding[]; outputs: OutputBinding[] }
  inputSchema: Record<string, unknown>
}

export function PreviewSection({ bindings, inputSchema }: Props) {
  const { t } = useTranslation()
  const preview = {
    bindings,
    input_schema: inputSchema,
  }
  return (
    <section className='space-y-2' data-testid='case-section-preview'>
      <div>
        <h3 className='text-sm font-semibold'>{t('cases.previewHeading')}</h3>
        <p className='text-xs text-muted-foreground'>
          {t('cases.previewHint')}
        </p>
      </div>
      <pre
        aria-readonly='true'
        className='max-h-48 overflow-auto rounded-md border bg-muted/30 p-3 font-mono text-xs'
      >
        {JSON.stringify(preview, null, 2)}
      </pre>
    </section>
  )
}
