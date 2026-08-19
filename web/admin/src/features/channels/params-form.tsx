import { useTranslation } from 'react-i18next'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

type Props = {
  schema: Record<string, unknown>
  value: Record<string, unknown>
  onChange: (next: Record<string, unknown>) => void
  caseOptions?: { id: string; name: string }[]
  disabled?: boolean
}

function xAdminWidget(schema: Record<string, unknown>): string | undefined {
  const ext = schema['x-admin']
  if (ext && typeof ext === 'object') {
    const widget = (ext as Record<string, unknown>)['widget']
    if (typeof widget === 'string') return widget
  }
  return undefined
}

export function ParamsForm({
  schema,
  value,
  onChange,
  caseOptions,
  disabled,
}: Props) {
  const { t } = useTranslation()
  const properties = (schema['properties'] ?? {}) as Record<
    string,
    Record<string, unknown>
  >
  const required = Array.isArray(schema['required'])
    ? (schema['required'] as string[])
    : []

  function patch(key: string, next: unknown) {
    onChange({ ...value, [key]: next })
  }

  return (
    <div data-testid='capability-params-form' className='space-y-3'>
      {Object.entries(properties).map(([key, propSchema]) => {
        const widget = xAdminWidget(propSchema)
        const label = key
        const isRequired = required.includes(key)
        const current = value[key]

        if (widget === 'workflow_picker') {
          const selected = Array.isArray(current) ? (current as string[]) : []
          return (
            <div key={key} className='space-y-1.5'>
              <Label>{t('channelMenu.paramSelectWorkflows')}</Label>
              <div className='max-h-48 space-y-1.5 overflow-auto rounded-md border p-2'>
                {(caseOptions ?? []).map((c) => (
                  <label key={c.id} className='flex items-center gap-2 text-sm'>
                    <Checkbox
                      checked={selected.includes(c.id)}
                      onCheckedChange={(v) =>
                        patch(
                          key,
                          v === true
                            ? [...selected, c.id]
                            : selected.filter((id) => id !== c.id)
                        )
                      }
                      disabled={disabled}
                    />
                    <span>{c.name}</span>
                  </label>
                ))}
              </div>
            </div>
          )
        }

        if (widget === 'text') {
          return (
            <div key={key} className='space-y-1.5'>
              <Label>
                {label}
                {isRequired ? ' *' : ''}
              </Label>
              <Textarea
                value={typeof current === 'string' ? current : ''}
                onChange={(e) => patch(key, e.target.value)}
                disabled={disabled}
                rows={3}
              />
            </div>
          )
        }

        if (propSchema['type'] === 'number') {
          return (
            <div key={key} className='space-y-1.5'>
              <Label>{label}</Label>
              <Input
                type='number'
                value={typeof current === 'number' ? current : ''}
                onChange={(e) => patch(key, Number(e.target.value) || 0)}
                disabled={disabled}
              />
            </div>
          )
        }

        if (propSchema['type'] === 'boolean') {
          return (
            <div
              key={key}
              className='flex items-center justify-between gap-3 rounded-md border px-3 py-2'
            >
              <Label>{label}</Label>
              <Switch
                checked={current === true}
                onCheckedChange={(v) => patch(key, v)}
                disabled={disabled}
              />
            </div>
          )
        }

        if (propSchema['type'] === 'array') {
          const items = Array.isArray(current) ? (current as string[]) : []
          return (
            <div key={key} className='space-y-1.5'>
              <Label>{label}</Label>
              <Textarea
                value={items.join('\n')}
                onChange={(e) =>
                  patch(
                    key,
                    e.target.value
                      .split('\n')
                      .map((s) => s.trim())
                      .filter(Boolean)
                  )
                }
                disabled={disabled}
                rows={3}
              />
            </div>
          )
        }

        return (
          <div key={key} className='space-y-1.5'>
            <Label>{label}</Label>
            <Input
              value={typeof current === 'string' ? current : ''}
              onChange={(e) => patch(key, e.target.value)}
              disabled={disabled}
            />
          </div>
        )
      })}
    </div>
  )
}
