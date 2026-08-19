import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

type Props = {
  open: boolean
  editMode: boolean
  text: string
  onOpen: () => void
  onEnterEdit: () => void
  onTextChange: (next: string) => void
  disabled?: boolean
}

export function AdvancedSection({
  open,
  editMode,
  text,
  onOpen,
  onEnterEdit,
  onTextChange,
  disabled,
}: Props) {
  const { t } = useTranslation()
  if (!open) {
    return (
      <section data-testid='case-section-advanced' className='flex justify-end'>
        <Button
          type='button'
          size='sm'
          variant='ghost'
          onClick={onOpen}
          disabled={disabled}
        >
          {t('cases.advancedLabel')}
        </Button>
      </section>
    )
  }
  return (
    <section className='space-y-2' data-testid='case-section-advanced'>
      <div className='flex items-center justify-between gap-2'>
        <div>
          <h3 className='text-sm font-semibold'>{t('cases.advancedLabel')}</h3>
          <p className='text-xs text-muted-foreground'>
            {editMode
              ? t('cases.advancedOverwriteWarn')
              : t('cases.advancedReadonlyHint')}
          </p>
        </div>
        {!editMode ? (
          <Button
            type='button'
            size='sm'
            variant='outline'
            onClick={onEnterEdit}
            disabled={disabled}
          >
            {t('cases.advancedEnterEdit')}
          </Button>
        ) : null}
      </div>
      <Label htmlFor='case-advanced-json' className='sr-only'>
        {t('cases.advancedLabel')}
      </Label>
      <Textarea
        id='case-advanced-json'
        value={text}
        onChange={(e) => onTextChange(e.target.value)}
        readOnly={!editMode}
        disabled={disabled}
        rows={12}
        className='font-mono text-xs'
      />
    </section>
  )
}
