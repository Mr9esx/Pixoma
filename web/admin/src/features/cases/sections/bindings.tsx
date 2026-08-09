import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { InputBinding, OutputBinding } from '@/lib/api/types'

type Props = {
  inputs: InputBinding[]
  outputs: OutputBinding[]
  onChange: (next: {
    inputs: InputBinding[]
    outputs: OutputBinding[]
  }) => void
  disabled?: boolean
}

function emptyInputBinding(): InputBinding {
  return { key: '', node_id: '', field_path: '' }
}

function emptyOutputBinding(): OutputBinding {
  return { key: '', node_id: '' }
}

export function BindingsSection({
  inputs,
  outputs,
  onChange,
  disabled,
}: Props) {
  const { t } = useTranslation()

  function updateInput(index: number, patch: Partial<InputBinding>) {
    onChange({
      inputs: inputs.map((row, i) => (i === index ? { ...row, ...patch } : row)),
      outputs,
    })
  }

  function updateOutput(index: number, patch: Partial<OutputBinding>) {
    onChange({
      inputs,
      outputs: outputs.map((row, i) =>
        i === index ? { ...row, ...patch } : row,
      ),
    })
  }

  return (
    <section className='space-y-6' data-testid='case-section-bindings'>
      <div className='space-y-3'>
        <div className='flex items-center justify-between gap-2'>
          <h3 className='text-sm font-semibold'>
            {t('cases.sectionBindingInputs')}
          </h3>
          <Button
            type='button'
            size='sm'
            variant='outline'
            disabled={disabled}
            onClick={() =>
              onChange({ inputs: [...inputs, emptyInputBinding()], outputs })
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
                key={`bind-in-${index}`}
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
                    <Label>{t('cases.fieldNodeId')}</Label>
                    <Input
                      value={row.node_id}
                      onChange={(e) =>
                        updateInput(index, { node_id: e.target.value })
                      }
                      disabled={disabled}
                      autoComplete='off'
                    />
                  </div>
                  <div className='space-y-1'>
                    <Label>{t('cases.fieldFieldPath')}</Label>
                    <Input
                      value={row.field_path}
                      onChange={(e) =>
                        updateInput(index, { field_path: e.target.value })
                      }
                      disabled={disabled}
                      autoComplete='off'
                    />
                  </div>
                </div>
                <div className='flex justify-end'>
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
          <h3 className='text-sm font-semibold'>
            {t('cases.sectionBindingOutputs')}
          </h3>
          <Button
            type='button'
            size='sm'
            variant='outline'
            disabled={disabled}
            onClick={() =>
              onChange({
                inputs,
                outputs: [...outputs, emptyOutputBinding()],
              })
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
                key={`bind-out-${index}`}
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
                    <Label>{t('cases.fieldNodeId')}</Label>
                    <Input
                      value={row.node_id}
                      onChange={(e) =>
                        updateOutput(index, { node_id: e.target.value })
                      }
                      disabled={disabled}
                      autoComplete='off'
                    />
                  </div>
                  <div className='space-y-1'>
                    <Label>{t('cases.fieldIndex')}</Label>
                    <Input
                      type='number'
                      value={row.index ?? ''}
                      onChange={(e) => {
                        const raw = e.target.value
                        updateOutput(index, {
                          index:
                            raw === ''
                              ? undefined
                              : Number.parseInt(raw, 10) || 0,
                        })
                      }}
                      disabled={disabled}
                      autoComplete='off'
                    />
                  </div>
                </div>
                <div className='flex justify-end'>
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
