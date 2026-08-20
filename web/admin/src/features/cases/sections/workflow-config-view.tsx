import { useTranslation } from 'react-i18next'
import type { CaseRecord } from '@/lib/api/types'

export function WorkflowConfigView({ record }: { record: CaseRecord }) {
  const { t } = useTranslation()
  const nodeCount = Object.keys(record.bindings.workflow ?? {}).length

  return (
    <div className='space-y-6' data-testid='case-workflow-config-view'>
      <div className='space-y-2'>
        <h3 className='text-sm font-semibold'>{t('cases.sectionInputs')}</h3>
        {record.inputs.length === 0 ? (
          <p className='text-xs text-muted-foreground'>{t('cases.noRows')}</p>
        ) : (
          <ul className='space-y-1'>
            {record.inputs.map((f) => (
              <li
                key={f.key}
                className='flex flex-wrap items-center gap-2 text-sm'
              >
                <span className='font-medium'>{f.key}</span>
                <span className='rounded-sm bg-muted px-1.5 py-0.5 text-[10px]'>
                  {f.type}
                </span>
                {f.required ? (
                  <span className='text-[10px] text-amber-600'>
                    {t('cases.fieldRequired')}
                  </span>
                ) : null}
                {f.description ? (
                  <span className='text-xs text-muted-foreground'>
                    · {f.description}
                  </span>
                ) : null}
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className='space-y-2'>
        <h3 className='text-sm font-semibold'>{t('cases.sectionOutputs')}</h3>
        {record.outputs.length === 0 ? (
          <p className='text-xs text-muted-foreground'>{t('cases.noRows')}</p>
        ) : (
          <ul className='space-y-1'>
            {record.outputs.map((f) => (
              <li
                key={f.key}
                className='flex flex-wrap items-center gap-2 text-sm'
              >
                <span className='font-medium'>{f.key}</span>
                <span className='rounded-sm bg-muted px-1.5 py-0.5 text-[10px]'>
                  {f.type}
                </span>
                {f.description ? (
                  <span className='text-xs text-muted-foreground'>
                    · {f.description}
                  </span>
                ) : null}
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className='space-y-2'>
        <h3 className='text-sm font-semibold'>
          {t('cases.sectionBindingInputs')}
        </h3>
        <ul className='space-y-1'>
          {record.bindings.inputs.map((b) => (
            <li
              key={b.key}
              className='flex flex-wrap items-center gap-2 text-sm'
            >
              <span className='font-medium'>{b.key}</span>
              <span className='text-xs text-muted-foreground'>
                → {b.node_id}.{b.field_path}
              </span>
            </li>
          ))}
        </ul>
      </div>

      <div className='space-y-2'>
        <h3 className='text-sm font-semibold'>
          {t('cases.sectionBindingOutputs')}
        </h3>
        <ul className='space-y-1'>
          {record.bindings.outputs.map((b) => (
            <li
              key={b.key}
              className='flex flex-wrap items-center gap-2 text-sm'
            >
              <span className='font-medium'>{b.key}</span>
              <span className='text-xs text-muted-foreground'>
                → {b.node_id}[{b.index ?? 0}]
              </span>
            </li>
          ))}
        </ul>
      </div>

      <div className='space-y-2'>
        <h3 className='text-sm font-semibold'>{t('cases.sectionWorkflow')}</h3>
        <p className='text-xs text-muted-foreground'>
          {t('cases.workflowNodesCount', { count: nodeCount })}
        </p>
        <pre
          aria-readonly='true'
          className='max-h-48 overflow-auto rounded-md border bg-muted/30 p-3 font-mono text-xs'
        >
          {JSON.stringify(record.bindings.workflow ?? {}, null, 2)}
        </pre>
      </div>
    </div>
  )
}
