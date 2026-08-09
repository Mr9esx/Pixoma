import { useTranslation } from 'react-i18next'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import type { CaseRecord } from '@/lib/api/types'

export type BasicsSlice = Pick<
  CaseRecord,
  | 'id'
  | 'name'
  | 'description'
  | 'preview'
  | 'price'
  | 'tags'
  | 'menu_key'
  | 'categories'
  | 'enabled'
>

type Props = {
  value: BasicsSlice
  onChange: (next: BasicsSlice) => void
  idEditable: boolean
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
  idEditable,
  showEnabled = false,
  disabled,
}: Props) {
  const { t } = useTranslation()

  function patch(partial: Partial<BasicsSlice>) {
    onChange({ ...value, ...partial })
  }

  return (
    <section className='space-y-4' data-testid='case-section-basics'>
      <h3 className='text-sm font-semibold'>{t('cases.sectionBasics')}</h3>

      <div className='grid gap-4 sm:grid-cols-2'>
        <div className='space-y-2'>
          <Label htmlFor='case-id'>{t('cases.fieldId')}</Label>
          <Input
            id='case-id'
            value={value.id}
            onChange={(e) => patch({ id: e.target.value })}
            disabled={!idEditable || disabled}
            required={idEditable}
            autoComplete='off'
          />
        </div>
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
      </div>

      <div className='space-y-2'>
        <Label htmlFor='case-description'>{t('cases.fieldDescription')}</Label>
        <Textarea
          id='case-description'
          value={value.description ?? ''}
          onChange={(e) => patch({ description: e.target.value })}
          disabled={disabled}
          rows={3}
        />
      </div>

      <div className='grid gap-4 sm:grid-cols-2'>
        <div className='space-y-2'>
          <Label htmlFor='case-preview'>{t('cases.fieldPreview')}</Label>
          <Input
            id='case-preview'
            value={value.preview ?? ''}
            onChange={(e) => patch({ preview: e.target.value })}
            disabled={disabled}
            autoComplete='off'
          />
        </div>
        <div className='space-y-2'>
          <Label htmlFor='case-price'>{t('cases.fieldPrice')}</Label>
          <Input
            id='case-price'
            type='number'
            step='any'
            value={Number.isFinite(value.price) ? value.price : 0}
            onChange={(e) =>
              patch({ price: Number.parseFloat(e.target.value) || 0 })
            }
            disabled={disabled}
          />
        </div>
      </div>

      <div className='grid gap-4 sm:grid-cols-2'>
        <div className='space-y-2'>
          <Label htmlFor='case-menu-key'>{t('cases.fieldMenuKey')}</Label>
          <Input
            id='case-menu-key'
            value={value.menu_key ?? ''}
            onChange={(e) => patch({ menu_key: e.target.value })}
            disabled={disabled}
            autoComplete='off'
          />
        </div>
        {showEnabled ? (
          <div className='flex items-center justify-between gap-3 self-end rounded-md border px-3 py-2'>
            <Label htmlFor='case-enabled'>{t('cases.fieldEnabled')}</Label>
            <Switch
              id='case-enabled'
              checked={value.enabled}
              onCheckedChange={(enabled) => patch({ enabled })}
              disabled={disabled}
            />
          </div>
        ) : null}
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
    </section>
  )
}
