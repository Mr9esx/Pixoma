import { useTranslation } from 'react-i18next'
import type { MenuNode } from '@/lib/api/channel-menu'
import type { CapabilityBrief } from '@/lib/api/channels'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { ParamsForm } from './params-form'

type Props = {
  node: MenuNode
  capabilities: CapabilityBrief[]
  caseOptions: { id: string; name: string }[]
  rootColumns: number
  onUpdate: (patch: Partial<MenuNode>) => void
  onColumnsChange: (columns: number) => void
  onAddChild: () => void
  onRemove: () => void
}

export function NodeConfigPanel({
  node,
  capabilities,
  caseOptions,
  rootColumns,
  onUpdate,
  onColumnsChange,
  onAddChild,
  onRemove,
}: Props) {
  const { t } = useTranslation()
  const isGroup = (node.children?.length ?? 0) > 0
  const capability = capabilities.find((c) => c.id === node.capability_id)
  const actionValue = isGroup ? 'group' : (node.capability_id ?? 'none')

  function onActionChange(value: string) {
    if (value === 'group') {
      onUpdate({ capability_id: undefined, params: undefined })
      return
    }
    if (value === 'none') {
      onUpdate({ capability_id: undefined, params: undefined })
      return
    }
    const params = { ...(node.params ?? {}) }
    if (value === 'open_case' && !Array.isArray(params.case_ids)) {
      params.case_ids = []
    }
    onUpdate({ capability_id: value, params, children: undefined })
  }

  return (
    <div data-testid='node-config-panel' className='space-y-4'>
      <div className='flex items-start justify-between gap-3'>
        <div>
          <div className='text-xs text-muted-foreground'>
            {t('channelMenu.currentEditing')}
          </div>
          <h2 className='text-lg font-semibold'>
            {node.label.trim() || t('channelMenu.untitled')}
          </h2>
        </div>
        <Button type='button' variant='ghost' size='sm' onClick={onRemove}>
          {t('channelMenu.removeItem')}
        </Button>
      </div>

      <div className='space-y-1.5'>
        <Label>{t('channelMenu.buttonLabel')}</Label>
        <Input
          value={node.label}
          onChange={(e) => onUpdate({ label: e.target.value })}
          autoComplete='off'
        />
      </div>

      <div className='space-y-1.5'>
        <Label>{t('channelMenu.buttonAction')}</Label>
        <Select value={actionValue} onValueChange={onActionChange}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {!isGroup ? (
              <SelectItem value='group'>
                {t('channelMenu.actionShowChildren')}
              </SelectItem>
            ) : null}
            <SelectItem value='none'>
              {t('channelMenu.capabilityNone')}
            </SelectItem>
            {capabilities.map((c) => (
              <SelectItem key={c.id} value={c.id}>
                {c.display_name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {!isGroup && capability ? (
        <ParamsForm
          schema={capability.params_schema}
          value={(node.params as Record<string, unknown>) ?? {}}
          onChange={(params) =>
            onUpdate({ params: { ...(node.params ?? {}), ...params } })
          }
          caseOptions={caseOptions}
        />
      ) : null}

      {isGroup ? (
        <div className='space-y-1.5'>
          <Label>{t('channelMenu.fieldIntro')}</Label>
          <Textarea
            value={node.intro_text ?? ''}
            onChange={(e) => onUpdate({ intro_text: e.target.value })}
            rows={3}
          />
          <div className='rounded-md border p-2'>
            {(node.children ?? []).map((child) => (
              <div key={child.id} className='px-2 py-1.5 text-sm'>
                {child.label.trim() || t('channelMenu.untitled')}
              </div>
            ))}
            <Button
              type='button'
              variant='ghost'
              size='sm'
              onClick={onAddChild}
            >
              {t('channelMenu.addChild')}
            </Button>
          </div>
        </div>
      ) : null}

      <div className='space-y-1.5 border-t pt-3'>
        <Label>{t('channelMenu.mainKeyboard')}</Label>
        <Input
          type='number'
          min={1}
          max={8}
          value={rootColumns}
          onChange={(e) => onColumnsChange(Number(e.target.value) || 2)}
        />
        <p className='text-xs text-muted-foreground'>
          {t('channelMenu.columnsPerRow')}
        </p>
      </div>
    </div>
  )
}
