import { useTranslation } from 'react-i18next'
import type { MenuNode } from '@/lib/api/channel-menu'
import { cn } from '@/lib/utils'
import { buildTgSimulation } from './lib/menu-simulation'

type Props = {
  items: MenuNode[]
  selectedId: string | null
  onSelect: (id: string) => void
}

export function PhoneSimulation({ items, selectedId, onSelect }: Props) {
  const { t } = useTranslation()
  const sim = buildTgSimulation(items)
  const selectedGroup = selectedId ? sim.groups[selectedId] : undefined
  const selectedItem = selectedId
    ? items.find((it) => it.id === selectedId)
    : undefined
  const isCapabilitySelected =
    selectedItem != null &&
    (selectedItem.children?.length ?? 0) === 0 &&
    Boolean(selectedItem.capability_id)

  return (
    <div data-testid='phone-simulation' className='flex flex-col gap-3'>
      <div className='rounded-xl border bg-background p-3'>
        <div className='space-y-2 text-sm'>
          {selectedGroup ? (
            <>
              <p className='font-medium'>{selectedGroup.title}</p>
              <p className='text-muted-foreground'>{selectedGroup.intro}</p>
              <div className='flex flex-wrap gap-2 pt-1'>
                {selectedGroup.buttons.map((b) => (
                  <span
                    key={b.label}
                    className='rounded-md border border-cyan-500/40 bg-cyan-500/10 px-3 py-1.5 text-xs'
                  >
                    {b.label}
                  </span>
                ))}
              </div>
              <p className='pt-1 text-xs text-muted-foreground'>‹ 返回主菜单</p>
            </>
          ) : isCapabilitySelected ? (
            <p className='text-muted-foreground'>
              {selectedItem?.label}：{t('channelMenu.simNote')}
            </p>
          ) : (
            <p className='text-muted-foreground'>欢迎使用，请选择功能 👇</p>
          )}
        </div>
        <div className='mt-3 border-t pt-3'>
          {sim.keyboard.map((row, i) => (
            <div key={i} className='mb-2 flex gap-2 last:mb-0'>
              {row.map((label) => {
                const item = items.find((it) => it.label === label)
                return (
                  <button
                    key={label}
                    type='button'
                    onClick={() => item && onSelect(item.id)}
                    className={cn(
                      'flex-1 rounded-md border px-2 py-2 text-sm',
                      item?.id === selectedId
                        ? 'border-primary bg-primary/10 font-medium'
                        : 'bg-muted/40 hover:bg-muted'
                    )}
                  >
                    {label}
                  </button>
                )
              })}
            </div>
          ))}
        </div>
      </div>
      <p className='text-xs text-muted-foreground'>
        {t('channelMenu.oneLevelHint')} · {t('channelMenu.maxRootHint')}
      </p>
    </div>
  )
}
