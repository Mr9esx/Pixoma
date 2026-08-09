import { useTranslation } from 'react-i18next'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

type Props = {
  value: string
  onChange: (next: string) => void
  error?: string
  disabled?: boolean
}

export function InputSchemaSection({
  value,
  onChange,
  error,
  disabled,
}: Props) {
  const { t } = useTranslation()

  return (
    <section className='space-y-2' data-testid='case-section-input-schema'>
      <div>
        <h3 className='text-sm font-semibold'>
          {t('cases.sectionInputSchema')}
        </h3>
        <p className='text-muted-foreground text-xs'>
          {t('cases.advancedRawHint')}
        </p>
      </div>
      <Label htmlFor='case-input-schema' className='sr-only'>
        {t('cases.sectionInputSchema')}
      </Label>
      <Textarea
        id='case-input-schema'
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        rows={10}
        className='font-mono text-xs'
        aria-invalid={Boolean(error)}
      />
      {error ? (
        <p className='text-destructive text-xs' role='alert'>
          {error}
        </p>
      ) : null}
    </section>
  )
}
