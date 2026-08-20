import { useLayoutEffect, useMemo, useRef, useState } from 'react'
import {
  ArrowRight,
  ArrowUpRight,
  Pencil,
  ChevronRight,
  Plus,
  Workflow,
} from 'lucide-react'
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
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { patchCase } from '@/lib/api/cases'
import { queryKeys } from '@/lib/api/query-keys'
import type { CaseRecord } from '@/lib/api/types'
import { nodeLabel, nodeVisualFor } from '../lib/node-catalog'
import { parseWorkflow, type WorkflowNode } from '../lib/workflow-parse'
import {
  deriveBindings,
  type InputFieldDraft,
  type OutputFieldDraft,
} from '../lib/derive'
import { InputFieldCard, OutputFieldCard, TYPE_LABEL_KEYS } from './field-cards'

type Props = {
  record: CaseRecord
  onSaved?: (next: CaseRecord) => void
}

type RawNode = {
  class_type?: unknown
  inputs?: unknown
}

function workflowNodes(record: CaseRecord): Record<string, RawNode> {
  return (record.bindings.workflow ?? {}) as Record<string, RawNode>
}

function connectedInputs(node: RawNode): Array<[string, string, number]> {
  const out: Array<[string, string, number]> = []
  const inputs = node.inputs
  if (!inputs || typeof inputs !== 'object') return out
  for (const [name, value] of Object.entries(inputs as Record<string, unknown>)) {
    if (Array.isArray(value) && typeof value[0] === 'string') {
      out.push([name, value[0], typeof value[1] === 'number' ? value[1] : 0])
    }
  }
  return out
}

function literalInputs(node: RawNode): Array<[string, string]> {
  const out: Array<[string, string]> = []
  const inputs = node.inputs
  if (!inputs || typeof inputs !== 'object') return out
  for (const [name, value] of Object.entries(inputs as Record<string, unknown>)) {
    if (Array.isArray(value)) continue
    if (value === undefined || value === null) continue
    const s = typeof value === 'string' ? value : JSON.stringify(value)
    out.push([name, s])
  }
  return out
}

type StepDetail =
  | { kind: 'literal'; name: string; value: string }
  | { kind: 'link'; name: string; src: string; slot: number }

/** 只有内容被截断省略时才显示 tooltip。 */
function TruncatedValue({
  value,
  className,
}: {
  value: string
  className?: string
}) {
  const ref = useRef<HTMLElement>(null)
  const [overflow, setOverflow] = useState(false)
  useLayoutEffect(() => {
    const el = ref.current
    setOverflow(!!el && el.scrollWidth > el.clientWidth)
  })
  const dd = (
    <dd ref={ref} className={className}>
      {value}
    </dd>
  )
  if (!overflow) return dd
  return (
    <Tooltip>
      <TooltipTrigger asChild>{dd}</TooltipTrigger>
      <TooltipContent side='top' className='max-w-80'>
        {value}
      </TooltipContent>
    </Tooltip>
  )
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

  // 工作流节点（供字段编辑选节点）。
  const wfNodes = useMemo<WorkflowNode[]>(() => {
    const result = parseWorkflow(JSON.stringify(record.bindings.workflow ?? {}))
    return result.ok ? result.graph.nodes : []
  }, [record])

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

  // 处理流程（横向）。
  const steps = useMemo(() => {
    const workflow = workflowNodes(record)
    const ids = Object.keys(workflow)
    const layer = new Map<string, number>(ids.map((id) => [id, 0]))
    const incoming = new Map<string, string[]>()
    for (const id of ids) incoming.set(id, [])
    for (const id of ids) {
      for (const [, src] of connectedInputs(workflow[id])) {
        if (!workflow[src]) continue
        incoming.get(id)?.push(src)
      }
    }
    for (let pass = 0; pass < ids.length; pass += 1) {
      let changed = false
      for (const id of ids) {
        for (const src of incoming.get(id) ?? []) {
          const next = (layer.get(src) ?? 0) + 1
          if (next > (layer.get(id) ?? 0)) {
            layer.set(id, next)
            changed = true
          }
        }
      }
      if (!changed) break
    }
    const ordered = [...ids].sort(
      (a, b) => (layer.get(a) ?? 0) - (layer.get(b) ?? 0) || a.localeCompare(b)
    )
    return ordered.map((id) => {
      const classType = String(workflow[id]?.class_type ?? id)
      const { Icon, className } = nodeVisualFor(classType)
      const details: StepDetail[] = [
        ...literalInputs(workflow[id]).map(([name, value]) => ({
          kind: 'literal' as const,
          name,
          value,
        })),
        ...connectedInputs(workflow[id]).map(([name, src, slot]) => ({
          kind: 'link' as const,
          name,
          src,
          slot,
        })),
      ]
      return { id, classType, Icon, className, details }
    })
  }, [record])

  // 输入/输出卡片：关联到哪个处理节点。
  const inputCards = useMemo(
    () =>
      record.inputs.map((f) => {
        const bind = record.bindings.inputs.find((b) => b.key === f.key)
        const target = bind ? steps.find((s) => s.id === bind.node_id) : undefined
        return {
          field: f,
          targetLabel: target ? nodeLabel(target.classType) : bind?.node_id,
          fieldPath: bind?.field_path,
        }
      }),
    [record, steps]
  )
  const outputCards = useMemo(
    () =>
      record.outputs.map((f) => {
        const bind = record.bindings.outputs.find((b) => b.key === f.key)
        const src = bind ? steps.find((s) => s.id === bind.node_id) : undefined
        return {
          field: f,
          srcLabel: src ? nodeLabel(src.classType) : bind?.node_id,
          index: bind?.index ?? 0,
        }
      }),
    [record, steps]
  )

  return (
    <div className='min-w-0 w-full space-y-4' data-testid='case-workflow-graph-preview'>
      {/* ===== 输入 + 输出（并排）===== */}
      <div className='grid grid-cols-1 gap-4 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]'>
        {/* 输入卡片 */}
        <section className='bg-card flex min-w-0 flex-col overflow-hidden rounded-lg border'>
          <header className='flex min-h-12 shrink-0 flex-wrap items-center justify-between gap-2 border-b px-4 py-2.5'>
            <h3 className='flex items-center gap-2 text-sm font-medium'>
              <span className='size-2 rounded-full bg-amber-500' />
              {t('cases.sectionInputs')}
            </h3>
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
          <div className='flex min-w-0 flex-col gap-2 p-3'>
            {inputCards.length === 0 ? (
              <p className='px-1 text-xs text-muted-foreground'>{t('cases.noRows')}</p>
            ) : (
              <div className='divide-y rounded-md border bg-muted/30'>
                {inputCards.map(({ field, targetLabel, fieldPath }) => (
                  <div
                    key={field.key}
                    className='flex flex-wrap items-center justify-between gap-x-3 gap-y-1 px-3 py-2.5'
                  >
                    <div className='min-w-0 flex-1 basis-40'>
                      <div className='flex flex-wrap items-center gap-1.5'>
                        <span className='text-sm font-medium'>{field.key}</span>
                        {field.required ? (
                          <span className='rounded-sm border bg-muted px-1.5 py-px font-mono text-[10px] text-amber-600'>
                            {t('cases.fieldRequired')}
                          </span>
                        ) : null}
                      </div>
                      {field.description ? (
                        <p className='mt-0.5 text-xs text-muted-foreground'>{field.description}</p>
                      ) : null}
                    </div>
                    <div className='flex flex-wrap items-center gap-2'>
                      <span className='rounded-sm border bg-muted px-2 py-0.5 font-mono text-xs text-foreground'>
                        {t(TYPE_LABEL_KEYS[field.type] ?? field.type)}
                      </span>
                      {targetLabel ? (
                        <span className='flex items-center gap-1 text-xs text-muted-foreground'>
                          <ArrowRight className='size-3 shrink-0' />
                          <span>{targetLabel}</span>
                          {fieldPath ? <span className='font-mono'>{`.${fieldPath}`}</span> : null}
                        </span>
                      ) : null}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </section>

        {/* 输出卡片 */}
        <section className='bg-card flex min-w-0 flex-col overflow-hidden rounded-lg border'>
          <header className='flex min-h-12 shrink-0 flex-wrap items-center justify-between gap-2 border-b px-4 py-2.5'>
            <h3 className='flex items-center gap-2 text-sm font-medium'>
              <span className='size-2 rounded-full bg-emerald-500' />
              {t('cases.sectionOutputs')}
            </h3>
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
          <div className='flex min-w-0 flex-col gap-2 p-3'>
            {outputCards.length === 0 ? (
              <p className='px-1 text-xs text-muted-foreground'>{t('cases.noRows')}</p>
            ) : (
              <div className='divide-y rounded-md border bg-muted/30'>
                {outputCards.map(({ field, srcLabel, index }) => (
                  <div
                    key={field.key}
                    className='flex flex-wrap items-center justify-between gap-x-3 gap-y-1 px-3 py-2.5'
                  >
                    <div className='min-w-0 flex-1 basis-40'>
                      <div className='text-sm font-medium'>{field.key}</div>
                      {field.description ? (
                        <p className='mt-0.5 text-xs text-muted-foreground'>{field.description}</p>
                      ) : null}
                    </div>
                    <div className='flex flex-wrap items-center gap-2'>
                      <span className='rounded-sm border bg-muted px-2 py-0.5 font-mono text-xs text-foreground'>
                        {t(TYPE_LABEL_KEYS[field.type] ?? field.type)}
                      </span>
                      {srcLabel ? (
                        <span className='flex items-center gap-1 text-xs text-muted-foreground'>
                          <ArrowUpRight className='size-3 shrink-0' />
                          <span>{srcLabel}</span>
                          <span className='font-mono'>[{index}]</span>
                        </span>
                      ) : null}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </section>
      </div>

      {/* ===== 处理流程（横向滚动）===== */}
      <section className='bg-card flex min-w-0 flex-col rounded-lg border'>
        <header className='flex min-h-12 shrink-0 items-center gap-2.5 border-b px-4 py-2.5'>
          <span className='grid size-7 shrink-0 place-items-center rounded-md bg-sky-50 text-sky-600 dark:bg-sky-950/50 dark:text-sky-400'>
            <Workflow className='size-4' />
          </span>
          <h3 className='shrink-0 whitespace-nowrap text-sm font-semibold'>
            {t('cases.sectionProcess')}
          </h3>
        </header>
        <div className='min-w-0 p-3'>
          {steps.length === 0 ? (
            <p className='px-1 text-xs text-muted-foreground'>{t('cases.noRows')}</p>
          ) : (
            <div className='overflow-x-auto pb-2'>
              <div className='flex min-w-max items-stretch gap-1.5'>
                {steps.map((step, i) => (
                  <div key={step.id} className='flex items-center'>
                    <div className='flex w-56 shrink-0 flex-col overflow-hidden rounded-md border bg-muted/30'>
                      <div className='flex items-center gap-2 border-b border-border px-3 py-2'>
                        <span className={`grid size-7 shrink-0 place-items-center rounded-md ${step.className}`}>
                          <step.Icon className='size-4' />
                        </span>
                        <span className='min-w-0 truncate text-sm font-medium'>
                          {nodeLabel(step.classType)}
                        </span>
                      </div>
                      {step.details.length > 0 ? (
                        <TooltipProvider>
                          <dl className='flex flex-col gap-2 px-3 py-2.5'>
                            {step.details.map((d) =>
                              d.kind === 'link' ? (
                                <div
                                  key={`link-${d.name}`}
                                  className='flex min-w-0 items-center justify-between gap-3'
                                >
                                  <dt className='truncate text-sm text-muted-foreground'>{d.name}</dt>
                                  <dd className='shrink-0 font-mono text-sm text-muted-foreground'>
                                    → #{d.src}[{d.slot}]
                                  </dd>
                                </div>
                              ) : (
                                <div
                                  key={`lit-${d.name}`}
                                  className='flex min-w-0 items-center justify-between gap-3'
                                >
                                  <dt className='shrink-0 text-sm text-muted-foreground'>{d.name}</dt>
                                  <TruncatedValue
                                    value={d.value}
                                    className='min-w-0 truncate font-mono text-sm'
                                  />
                                </div>
                              )
                            )}
                          </dl>
                        </TooltipProvider>
                      ) : (
                        <div className='px-3 py-2.5 text-xs text-muted-foreground'>—</div>
                      )}
                      <div className='border-t border-border px-3 py-1.5 font-mono text-xs text-muted-foreground'>
                        #{step.id} · {step.classType}
                      </div>
                    </div>
                      {i < steps.length - 1 ? (
                        <ChevronRight className='mx-0.5 size-4 shrink-0 text-muted-foreground/50' />
                      ) : null}
                    </div>
                  ))}
              </div>
            </div>
          )}
        </div>
      </section>

      {/* ===== 编辑输入 modal（真正可编辑）===== */}
      <Dialog open={inputsOpen} onOpenChange={setInputsOpen}>
        <DialogContent className='flex max-h-[85vh] flex-col sm:max-w-2xl'>
          <DialogHeader>
            <DialogTitle>{t('cases.editInputs')}</DialogTitle>
          </DialogHeader>
          <div className='min-h-0 flex-1 overflow-auto'>
            <ul className='space-y-3'>
              {inputDrafts.map((draft, i) => (
                <InputFieldCard
                  key={`${draft.key}-${i}`}
                  nodes={wfNodes}
                  value={draft}
                  onChange={(next) =>
                    setInputDrafts((prev) =>
                      prev.map((p, idx) => (idx === i ? next : p))
                    )
                  }
                  onRemove={() =>
                    setInputDrafts((prev) => prev.filter((_, idx) => idx !== i))
                  }
                />
              ))}
            </ul>
            <Button
              type='button'
              variant='outline'
              size='sm'
              className='mt-3 gap-1.5'
              onClick={() =>
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
            >
              <Plus className='size-3.5' />
              {t('cases.addInput')}
            </Button>
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

      {/* ===== 编辑输出 modal（真正可编辑）===== */}
      <Dialog open={outputsOpen} onOpenChange={setOutputsOpen}>
        <DialogContent className='flex max-h-[85vh] flex-col sm:max-w-2xl'>
          <DialogHeader>
            <DialogTitle>{t('cases.editOutputs')}</DialogTitle>
          </DialogHeader>
          <div className='min-h-0 flex-1 overflow-auto'>
            <ul className='space-y-3'>
              {outputDrafts.map((draft, i) => (
                <OutputFieldCard
                  key={`${draft.key}-${i}`}
                  nodes={wfNodes}
                  value={draft}
                  onChange={(next) =>
                    setOutputDrafts((prev) =>
                      prev.map((p, idx) => (idx === i ? next : p))
                    )
                  }
                  onRemove={() =>
                    setOutputDrafts((prev) => prev.filter((_, idx) => idx !== i))
                  }
                />
              ))}
            </ul>
            <Button
              type='button'
              variant='outline'
              size='sm'
              className='mt-3 gap-1.5'
              onClick={() =>
                setOutputDrafts((prev) => [
                  ...prev,
                  { key: '', type: 'image', node_id: '', index: 0 },
                ])
              }
            >
              <Plus className='size-3.5' />
              {t('cases.addOutput')}
            </Button>
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
