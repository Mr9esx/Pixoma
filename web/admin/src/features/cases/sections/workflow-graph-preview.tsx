import { useMemo, useState } from 'react'
import { Pencil } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { patchCase } from '@/lib/api/cases'
import { queryKeys } from '@/lib/api/query-keys'
import type { CaseRecord } from '@/lib/api/types'
import { parseWorkflow, type WorkflowGraph } from '../lib/workflow-parse'
import {
  deriveBindings,
  type InputFieldDraft,
  type OutputFieldDraft,
} from '../lib/derive'
import {
  EditableInputFields,
  EditableOutputFields,
  InputFieldsTable,
  OutputFieldsTable,
} from './field-cards'
import { useIsWide } from './use-is-wide'
import { WorkflowGraphViewer } from './workflow-import'

type Props = {
  record: CaseRecord
  onSaved?: (next: CaseRecord) => void
}

function toInputDrafts(record: CaseRecord): InputFieldDraft[] {
  const byKey = new Map(record.bindings.inputs.map((b) => [b.key, b]))
  return record.inputs.map((field) => ({
    key: field.key,
    type: field.type,
    required: field.required,
    node_id: byKey.get(field.key)?.node_id ?? '',
    field_path: byKey.get(field.key)?.field_path ?? '',
    description: field.description,
  }))
}

function toOutputDrafts(record: CaseRecord): OutputFieldDraft[] {
  const byKey = new Map(record.bindings.outputs.map((b) => [b.key, b]))
  return record.outputs.map((field) => ({
    key: field.key,
    type: field.type,
    node_id: byKey.get(field.key)?.node_id ?? '',
    index: byKey.get(field.key)?.index ?? 0,
    description: field.description,
  }))
}

export function WorkflowGraphPreview({ record, onSaved }: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [inputsOpen, setInputsOpen] = useState(false)
  const [outputsOpen, setOutputsOpen] = useState(false)
  const isWide = useIsWide()

  const wfGraph = useMemo<WorkflowGraph | undefined>(() => {
    const result = parseWorkflow(JSON.stringify(record.bindings.workflow ?? {}))
    return result.ok ? result.graph : undefined
  }, [record])
  const wfNodes = wfGraph?.nodes ?? []
  const wfText = useMemo(
    () => JSON.stringify(record.bindings.workflow ?? {}, null, 2),
    [record]
  )

  // 编辑态 drafts（打开 modal 时初始化）。
  const [inputDrafts, setInputDrafts] = useState<InputFieldDraft[]>(() =>
    toInputDrafts(record)
  )
  const [outputDrafts, setOutputDrafts] = useState<OutputFieldDraft[]>(() =>
    toOutputDrafts(record)
  )

  const saveMutation = useMutation({
    mutationFn: () => {
      const bindings = deriveBindings(inputDrafts, outputDrafts)
      const body: Partial<CaseRecord> = {
        inputs: inputDrafts.map((f) => ({
          key: f.key,
          type: f.type,
          required: f.required,
          description: f.description,
        })),
        outputs: outputDrafts.map((f) => ({
          key: f.key,
          type: f.type,
          description: f.description,
        })),
        bindings: {
          ...record.bindings,
          inputs: bindings.inputs,
          outputs: bindings.outputs,
        },
      }
      return patchCase(record.id, body)
    },
    onSuccess: (next) => {
      queryClient.setQueryData(queryKeys.cases.detail(record.id), next)
      queryClient.invalidateQueries({ queryKey: queryKeys.cases.all })
      onSaved?.(next)
      toast.success(t('cases.saveSuccess'))
      setInputsOpen(false)
      setOutputsOpen(false)
    },
    onError: () => toast.error(t('cases.saveFailed')),
  })

  // 只读展示视图（对齐编辑工作流的表格列）。
  const displayInputs = useMemo(() => toInputDrafts(record), [record])
  const displayOutputs = useMemo(() => toOutputDrafts(record), [record])
  const noop = () => {}

  return (
    <div className='min-w-0 w-full space-y-4' data-testid='case-workflow-graph-preview'>
      {/* ===== 输入（对齐编辑工作流）===== */}
      <section className='space-y-2'>
        <header className='flex flex-wrap items-center justify-between gap-2'>
          <div className='min-w-0'>
            <h3 className='text-sm font-semibold'>{t('cases.inputsHeading')}</h3>
            <p className='text-xs text-muted-foreground'>{t('cases.inputsHint')}</p>
          </div>
          <Button
            type='button'
            variant='outline'
            size='sm'
            className='h-7 gap-1.5 px-2.5 text-xs'
            onClick={() => {
              setInputDrafts(toInputDrafts(record))
              setInputsOpen(true)
            }}
          >
            <Pencil className='size-3' />
            {t('cases.editInputs')}
          </Button>
        </header>
        <InputFieldsTable
          nodes={wfNodes}
          fields={displayInputs}
          onChange={noop}
          onRemove={noop}
          disabled
        />
      </section>

      {/* ===== 输出（对齐编辑工作流）===== */}
      <section className='space-y-2'>
        <header className='flex flex-wrap items-center justify-between gap-2'>
          <div className='min-w-0'>
            <h3 className='text-sm font-semibold'>{t('cases.outputsHeading')}</h3>
            <p className='text-xs text-muted-foreground'>{t('cases.outputsHint')}</p>
          </div>
          <Button
            type='button'
            variant='outline'
            size='sm'
            className='h-7 gap-1.5 px-2.5 text-xs'
            onClick={() => {
              setOutputDrafts(toOutputDrafts(record))
              setOutputsOpen(true)
            }}
          >
            <Pencil className='size-3' />
            {t('cases.editOutputs')}
          </Button>
        </header>
        <OutputFieldsTable
          nodes={wfNodes}
          fields={displayOutputs}
          onChange={noop}
          onRemove={noop}
          disabled
        />
      </section>

      {/* ===== 工作流节点图 / JSON（复用编辑工作流的查看器）===== */}
      {wfGraph && wfGraph.nodes.length > 0 ? (
        <WorkflowGraphViewer
          value={wfText}
          graph={wfGraph}
          onChange={noop}
          readOnly
        />
      ) : (
        <p className='px-1 text-xs text-muted-foreground'>{t('cases.noRows')}</p>
      )}

      {/* ===== 编辑输入 modal（复用编辑工作流的字段列表）===== */}
      <Dialog open={inputsOpen} onOpenChange={setInputsOpen}>
        <DialogContent className='flex max-h-[85vh] flex-col sm:max-w-[822px]'>
          <DialogHeader>
            <DialogTitle>{t('cases.editInputs')}</DialogTitle>
          </DialogHeader>
          <div className='min-h-0 flex-1 overflow-auto'>
            <EditableInputFields
              nodes={wfNodes}
              fields={inputDrafts}
              onChange={(index, next) =>
                setInputDrafts((prev) =>
                  prev.map((p, i) => (i === index ? next : p))
                )
              }
              onRemove={(index) =>
                setInputDrafts((prev) => prev.filter((_, i) => i !== index))
              }
              onAdd={() =>
                setInputDrafts((prev) => [
                  ...prev,
                  {
                    key: '',
                    type: 'string',
                    required: false,
                    node_id: '',
                    field_path: '',
                  },
                ])
              }
              wide={isWide}
            />
          </div>
          <div className='flex justify-end gap-2 border-t pt-3'>
            <Button type='button' variant='outline' onClick={() => setInputsOpen(false)}>
              {t('common.cancel')}
            </Button>
            <Button
              type='button'
              onClick={() => saveMutation.mutate()}
              disabled={saveMutation.isPending}
            >
              {t('common.save')}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* ===== 编辑输出 modal（复用编辑工作流的字段列表）===== */}
      <Dialog open={outputsOpen} onOpenChange={setOutputsOpen}>
        <DialogContent className='flex max-h-[85vh] flex-col sm:max-w-[822px]'>
          <DialogHeader>
            <DialogTitle>{t('cases.editOutputs')}</DialogTitle>
          </DialogHeader>
          <div className='min-h-0 flex-1 overflow-auto'>
            <EditableOutputFields
              nodes={wfNodes}
              fields={outputDrafts}
              onChange={(index, next) =>
                setOutputDrafts((prev) =>
                  prev.map((p, i) => (i === index ? next : p))
                )
              }
              onRemove={(index) =>
                setOutputDrafts((prev) => prev.filter((_, i) => i !== index))
              }
              onAdd={() =>
                setOutputDrafts((prev) => [
                  ...prev,
                  { key: '', type: 'image', node_id: '', index: 0 },
                ])
              }
              wide={isWide}
            />
          </div>
          <div className='flex justify-end gap-2 border-t pt-3'>
            <Button type='button' variant='outline' onClick={() => setOutputsOpen(false)}>
              {t('common.cancel')}
            </Button>
            <Button
              type='button'
              onClick={() => saveMutation.mutate()}
              disabled={saveMutation.isPending}
            >
              {t('common.save')}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
