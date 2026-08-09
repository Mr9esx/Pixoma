import { useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { MasterDetailShell } from '@/components/master-detail/master-detail-shell'
import { cn } from '@/lib/utils'

/** Temporary placeholder rows for layout B demo; replaced by Task 7 API. */
const PLACEHOLDER_INSTANCES = [
  { id: 'gpu-a', baseUrl: 'http://127.0.0.1:8188' },
  { id: 'gpu-b', baseUrl: 'http://127.0.0.1:8189' },
] as const

export const Route = createFileRoute('/_app/instances/')({
  component: InstancesPage,
})

function InstancesPage() {
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const selected = PLACEHOLDER_INSTANCES.find((item) => item.id === selectedId)

  return (
    <MasterDetailShell
      hasSelection={selectedId != null}
      onBackToList={() => setSelectedId(null)}
      list={
        <ul className='divide-y'>
          {PLACEHOLDER_INSTANCES.map((item) => (
            <li key={item.id}>
              <button
                type='button'
                className={cn(
                  'w-full px-4 py-3 text-left text-sm hover:bg-accent',
                  selectedId === item.id && 'bg-accent'
                )}
                onClick={() => setSelectedId(item.id)}
              >
                <div className='font-medium'>{item.id}</div>
                <div className='text-muted-foreground text-xs'>{item.baseUrl}</div>
              </button>
            </li>
          ))}
        </ul>
      }
      detail={
        selected ? (
          <div className='space-y-2'>
            <h2 className='text-lg font-semibold'>{selected.id}</h2>
            <p className='text-muted-foreground text-sm'>{selected.baseUrl}</p>
          </div>
        ) : null
      }
    />
  )
}
