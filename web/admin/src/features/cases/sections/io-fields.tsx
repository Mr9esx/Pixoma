import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { CaseInputField, CaseOutputField } from '@/lib/api/types'

type Props = {
  inputs: CaseInputField[]
  outputs: CaseOutputField[]
  onChange: (next: {
    inputs: CaseInputField[]
    outputs: CaseOutputField[]
  }) => void
  disabled?: boolean
}

function emptyInput(): CaseInputField {
  return { key: '', type: 'string', required: false }
}

function emptyOutput(): CaseOutputField {
  return { key: '', type: 'image' }
}

export function IoFieldsSection({
  inputs,
  outputs,
  onChange,
  disabled,
}: Props) {
  const { t } = useTranslation()

  function updateInput(index: number, patch: Partial<CaseInputField>) {
    onChange({
      inputs: inputs.map((row, i) => (i === index ? { ...row, ...patch } : row)),
      outputs,
    })
  }

  function updateOutput(index: number, patch: Partial<CaseOutputField>) {
    onChange({
      inputs,
      outputs: outputs.map((row, i) =>
        i === index ? { ...row, ...patch } : row,
      ),
    })
  }

  return (
    <section className='space-y-6' data-testid='case-section-io-fields'>
      <div className='space-y-3'>
        <div className='flex items-center justify-between gap-2'>
          <h3 className='text-sm font-semibold'>{t('cases.sectionInputs')}</h3>
          <Button
            type='button'
            size='sm'
            variant='outline'
            disabled={disabled}
            onClick={() =>
              onChange({ inputs: [...inputs, emptyInput()], outputs })
            }
          >
            {t('cases.addRow')}
          </Button>
        </div>
        {inputs.length === 0 ? (
          <p className='text-muted-foreground text-xs'>{t('cases.noRows')}</p>
        ) : (
          <ul className='space-y-3'>
            {inputs.map((row, index) => (
              <li
                key={`input-${index}`}
                className='space-y-2 rounded-md border p-3'
              >
                <div className='grid gap-2 sm:grid-cols-3'>
                  <div className='space-y-1'>
                    <Label>{t('cases.fieldKey')}</Label>
                    <Input
                      value={row.key}
                      onChange={(e) =>
                        updateInput(index, { key: e.target.value })
                      }
                      disabled={disabled}
                      autoComplete='off'
                    />
                  </div>
                  <div className='space-y-1'>
                    <Label>{t('cases.fieldType')}</Label>
                    <Input
                      value={row.type}
                      onChange={(e) =>
                        updateInput(index, { type: e.target.value })
                      }
                      disabled={disabled}
                      autoComplete='off'
                    />
                  </div>
                  <div className='space-y-1'>
                    <Label>{t('cases.fieldPreview')}</Label>
                    <Input
                      value={row.preview ?? ''}
                      onChange={(e) =>
                        updateInput(index, { preview: e.target.value })
                      }
                      disabled={disabled}
                      autoComplete='off'
                    />
                  </div>
                </div>
                <div className='space-y-1'>
                  <Label>{t('cases.fieldDescription')}</Label>
                  <Input
                    value={row.description ?? ''}
                    onChange={(e) =>
                      updateInput(index, { description: e.target.value })
                    }
                    disabled={disabled}
                    autoComplete='off'
                  />
                </div>
                <div className='flex flex-wrap items-center justify-between gap-3'>
                  <div className='flex flex-wrap gap-4'>
                    <label className='flex items-center gap-2 text-sm'>
                      <Checkbox
                        checked={row.required}
                        onCheckedChange={(v) =>
                          updateInput(index, { required: v === true })
                        }
                        disabled={disabled}
                      />
                      {t('cases.fieldRequired')}
                    </label>
                    <label className='flex items-center gap-2 text-sm'>
                      <Checkbox
                        checked={Boolean(row.skip_allowed)}
                        onCheckedChange={(v) =>
                          updateInput(index, { skip_allowed: v === true })
                        }
                        disabled={disabled}
                      />
                      {t('cases.fieldSkipAllowed')}
                    </label>
                  </div>
                  <Button
                    type='button'
                    size='sm'
                    variant='ghost'
                    disabled={disabled}
                    onClick={() =>
                      onChange({
                        inputs: inputs.filter((_, i) => i !== index),
                        outputs,
                      })
                    }
                  >
                    {t('cases.removeRow')}
                  </Button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className='space-y-3'>
        <div className='flex items-center justify-between gap-2'>
          <h3 className='text-sm font-semibold'>{t('cases.sectionOutputs')}</h3>
          <Button
            type='button'
            size='sm'
            variant='outline'
            disabled={disabled}
            onClick={() =>
              onChange({ inputs, outputs: [...outputs, emptyOutput()] })
            }
          >
            {t('cases.addRow')}
          </Button>
        </div>
        {outputs.length === 0 ? (
          <p className='text-muted-foreground text-xs'>{t('cases.noRows')}</p>
        ) : (
          <ul className='space-y-3'>
            {outputs.map((row, index) => (
              <li
                key={`output-${index}`}
                className='space-y-2 rounded-md border p-3'
              >
                <div className='grid gap-2 sm:grid-cols-3'>
                  <div className='space-y-1'>
                    <Label>{t('cases.fieldKey')}</Label>
                    <Input
                      value={row.key}
                      onChange={(e) =>
                        updateOutput(index, { key: e.target.value })
                      }
                      disabled={disabled}
                      autoComplete='off'
                    />
                  </div>
                  <div className='space-y-1'>
                    <Label>{t('cases.fieldType')}</Label>
                    <Input
                      value={row.type}
                      onChange={(e) =>
                        updateOutput(index, { type: e.target.value })
                      }
                      disabled={disabled}
                      autoComplete='off'
                    />
                  </div>
                  <div className='space-y-1'>
                    <Label>{t('cases.fieldMediaType')}</Label>
                    <Input
                      value={row.media_type ?? ''}
                      onChange={(e) =>
                        updateOutput(index, { media_type: e.target.value })
                      }
                      disabled={disabled}
                      autoComplete='off'
                    />
                  </div>
                </div>
                <div className='flex flex-wrap items-end justify-between gap-3'>
                  <div className='min-w-0 flex-1 space-y-1'>
                    <Label>{t('cases.fieldDescription')}</Label>
                    <Input
                      value={row.description ?? ''}
                      onChange={(e) =>
                        updateOutput(index, { description: e.target.value })
                      }
                      disabled={disabled}
                      autoComplete='off'
                    />
                  </div>
                  <Button
                    type='button'
                    size='sm'
                    variant='ghost'
                    disabled={disabled}
                    onClick={() =>
                      onChange({
                        inputs,
                        outputs: outputs.filter((_, i) => i !== index),
                      })
                    }
                  >
                    {t('cases.removeRow')}
                  </Button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  )
}
