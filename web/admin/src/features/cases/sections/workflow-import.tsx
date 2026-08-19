import { useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import type { WorkflowGraph } from '../lib/workflow-parse'

type Props = {
  value: string
  graph?: WorkflowGraph
  error?: string
  onChange: (next: string) => void
  disabled?: boolean
}

export function WorkflowImportSection({
  value,
  graph,
  error,
  onChange,
  disabled,
}: Props) {
  const { t } = useTranslation()
  const fileRef = useRef<HTMLInputElement>(null)

  async function onFile(file: File | undefined) {
    if (!file) return
    onChange(await file.text())
  }

  return (
    <section className='space-y-3' data-testid='case-section-workflow-import'>
      <div>
        <h3 className='text-sm font-semibold'>{t('cases.importHeading')}</h3>
        <p className='text-xs text-muted-foreground'>{t('cases.importHint')}</p>
      </div>

      <div
        className='rounded-md border border-dashed p-4 text-center text-sm text-muted-foreground'
        onDragOver={(e) => e.preventDefault()}
        onDrop={(e) => {
          e.preventDefault()
          void onFile(e.dataTransfer.files[0])
        }}
      >
        <p>{t('cases.importDropHint')}</p>
        <input
          ref={fileRef}
          type='file'
          accept='.json,application/json'
          className='hidden'
          onChange={(e) => void onFile(e.target.files?.[0])}
        />
      </div>

      {graph ? (
        <div
          data-testid='workflow-import-valid'
          className='rounded-md border border-emerald-600/30 bg-emerald-50 px-3 py-2 text-sm text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-400'
        >
          {t('cases.importValid')} ·{' '}
          {t('cases.importNodesCount', {
            count: graph.nodes.length,
            inputs: graph.nodes.reduce(
              (sum, node) => sum + node.inputs.length,
              0
            ),
            outputs: graph.nodes.reduce(
              (sum, node) => sum + node.outputCount,
              0
            ),
          })}
        </div>
      ) : null}

      {error ? (
        <div
          data-testid='workflow-import-error'
          className='rounded-md border border-red-600/30 bg-red-50 px-3 py-2 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400'
        >
          <p className='font-medium'>{t('cases.importFailed')}</p>
          <p>{error}</p>
        </div>
      ) : null}

      <Label htmlFor='case-workflow-json' className='sr-only'>
        {t('cases.importHeading')}
      </Label>
      <Textarea
        id='case-workflow-json'
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        rows={6}
        className='font-mono text-xs'
        placeholder={t('cases.importDropHint')}
      />
    </section>
  )
}
