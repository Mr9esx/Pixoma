import { useTranslation } from 'react-i18next'
import type { CaseRecord } from '@/lib/api/types'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { MarkdownTextField } from './markdown-text-field'
import { MediaPreviewField } from './media-preview-field'

export type BasicsSlice = Pick<
  CaseRecord,
  'name' | 'description' | 'preview' | 'tags' | 'categories' | 'enabled'
>

type Props = {
  value: BasicsSlice
  onChange: (next: BasicsSlice) => void
  /** Create-only: edit mode uses topbar Enable/Disable API instead. */
  showEnabled?: boolean
  disabled?: boolean
}

function joinList(v?: string[]): string {
  return (v ?? []).join(', ')
}

function splitList(raw: string): string[] {
  return raw
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}

export function BasicsSection({
  value,
  onChange,
  showEnabled = false,
  disabled,
}: Props) {
  const { t } = useTranslation()

  function patch(partial: Partial<BasicsSlice>) {
    onChange({ ...value, ...partial })
  }

  return (
    <section className='space-y-4' data-testid='case-section-basics'>
      <h3 className='text-base font-semibold'>{t('cases.sectionBasics')}</h3>

      <div className='space-y-2'>
        <Label htmlFor='case-name'>{t('cases.fieldName')}</Label>
        <Input
          id='case-name'
          value={value.name}
          onChange={(e) => patch({ name: e.target.value })}
          disabled={disabled}
          required
          autoComplete='off'
        />
      </div>

      <div className='space-y-2'>
        <Label htmlFor='case-description'>{t('cases.fieldDescription')}</Label>
        <MarkdownTextField
          id='case-description'
          value={value.description ?? ''}
          onChange={(next) => patch({ description: next })}
          disabled={disabled}
        />
      </div>

      <div className='space-y-2'>
        <Label htmlFor='case-preview'>{t('cases.fieldPreview')}</Label>
        <MediaPreviewField
          value={value.preview}
          onChange={(next) => patch({ preview: next })}
          disabled={disabled}
        />
      </div>


      <div className='grid gap-4 sm:grid-cols-2'>
        <div className='space-y-2'>
          <Label htmlFor='case-tags'>{t('cases.fieldTags')}</Label>
          <Input
            id='case-tags'
            value={joinList(value.tags)}
            onChange={(e) => patch({ tags: splitList(e.target.value) })}
            disabled={disabled}
            placeholder={t('cases.listPlaceholder')}
            autoComplete='off'
          />
        </div>
        <div className='space-y-2'>
          <Label htmlFor='case-categories'>{t('cases.fieldCategories')}</Label>
          <Input
            id='case-categories'
            value={joinList(value.categories)}
            onChange={(e) => patch({ categories: splitList(e.target.value) })}
            disabled={disabled}
            placeholder={t('cases.listPlaceholder')}
            autoComplete='off'
          />
        </div>
      </div>
      {showEnabled ? (
        <div className='flex items-center justify-between gap-3'>
          <Label htmlFor='case-enabled'>{t('cases.fieldEnabled')}</Label>
          <Switch
            id='case-enabled'
            checked={value.enabled}
            onCheckedChange={(enabled) => patch({ enabled })}
            disabled={disabled}
          />
        </div>
      ) : null}
    </section>
  )
}
