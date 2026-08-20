import { useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { CodeEditor } from '@/components/code-editor'
import type { WorkflowGraph } from '../lib/workflow-parse'

type Props = {
  value: string
  graph?: WorkflowGraph
  error?: string
  filename?: string
  onChange: (next: string) => void
  onFileName?: (name: string) => void
  disabled?: boolean
}

export function WorkflowImportSection({
  value,
  graph,
  error,
  filename,
  onChange,
  onFileName,
  disabled,
}: Props) {
  const { t } = useTranslation()
  const fileRef = useRef<HTMLInputElement>(null)

  async function onFile(file: File | undefined) {
    if (!file) return
    onChange(await file.text())
    onFileName?.(file.name)
  }

  return (
    <section className='space-y-3' data-testid='case-section-workflow-import'>
      <div className='flex items-center gap-2'>
        <span className='flex size-5 items-center justify-center rounded-full bg-primary text-xs font-semibold text-primary-foreground'>
          1
        </span>
        <h3 className='text-sm font-semibold'>{t('cases.importHeading')}</h3>
        <span className='text-xs text-muted-foreground'>
          {t('cases.importHint')}
        </span>
      </div>

      <div
        role='button'
        tabIndex={0}
        className='cursor-pointer rounded-md border border-dashed p-4 text-center text-sm text-muted-foreground hover:bg-accent/50'
        onDragOver={(e) => e.preventDefault()}
        onDrop={(e) => {
          e.preventDefault()
          if (!disabled) void onFile(e.dataTransfer.files[0])
        }}
        onClick={() => {
          if (!disabled) fileRef.current?.click()
        }}
        onKeyDown={(e) => {
          if (disabled) return
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            fileRef.current?.click()
          }
        }}
      >
        <p>{t('cases.importDropHint')}</p>
        {filename ? (
          <p className='mt-1 text-xs text-emerald-600 dark:text-emerald-400'>
            {t('cases.importFile')}: {filename}
          </p>
        ) : null}
        <input
          ref={fileRef}
          type='file'
          accept='.json,application/json'
          className='hidden'
          disabled={disabled}
          onChange={(e) => {
            void onFile(e.target.files?.[0])
            e.target.value = ''
          }}
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

      <CodeEditor
        title={t('cases.jsonTitle')}
        aria-label={t('cases.jsonTitle')}
        value={value}
        onChange={onChange}
        readOnly={disabled}
        placeholder={t('cases.importDropHint')}
        maxHeight={250}
      />
    </section>
  )
}
