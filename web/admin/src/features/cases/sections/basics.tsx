import { type RefObject } from 'react'
import { useTranslation } from 'react-i18next'
import type { CaseRecord } from '@/lib/api/types'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { MarkdownTextField } from './markdown-text-field'
import { MultiSelect } from '@/components/ui/multi-select'
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
  /** 创建模式下的名称校验错误文案。 */
  nameError?: string
  /** 名称输入框的 ref，供提交失败时聚焦。 */
  nameRef?: RefObject<HTMLInputElement | null>
}

export function BasicsSection({
  value,
  onChange,
  showEnabled = false,
  disabled,
  nameError,
  nameRef,
}: Props) {
  const { t } = useTranslation()

  function patch(partial: Partial<BasicsSlice>) {
    onChange({ ...value, ...partial })
  }

  return (
    <section className='space-y-4' data-testid='case-section-basics'>
      <h3 className='text-base font-semibold'>{t('cases.sectionBasics')}</h3>

      <div className='space-y-2'>
        <Label htmlFor='case-name'>
          {t('cases.fieldName')}
          <span className='text-destructive' aria-hidden='true'>*</span>
        </Label>
        <Input
          id='case-name'
          ref={nameRef}
          value={value.name}
          onChange={(e) => patch({ name: e.target.value })}
          disabled={disabled}
          required
          autoComplete='off'
          aria-invalid={Boolean(nameError) || undefined}
          aria-describedby={nameError ? 'case-name-error' : undefined}
        />
        {nameError ? (
          <p id='case-name-error' role='alert' className='text-sm text-destructive'>
            {nameError}
          </p>
        ) : null}
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
          <Label>{t('cases.fieldTags')}</Label>
          <MultiSelect
            value={value.tags ?? []}
            onValueChange={(tags) => patch({ tags })}
            disabled={disabled}
            placeholder={t('cases.listPlaceholder')}
            inputPlaceholder={t('cases.listInputPlaceholder')}
            createLabel={t('cases.listAdd')}
            emptyText={t('cases.listEmpty')}
          />
        </div>
        <div className='space-y-2'>
          <Label>{t('cases.fieldCategories')}</Label>
          <MultiSelect
            value={value.categories ?? []}
            onValueChange={(categories) => patch({ categories })}
            disabled={disabled}
            placeholder={t('cases.listPlaceholder')}
            inputPlaceholder={t('cases.listInputPlaceholder')}
            createLabel={t('cases.listAdd')}
            emptyText={t('cases.listEmpty')}
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
