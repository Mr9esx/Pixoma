import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { Sparkles, TriangleAlert } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { createCase, patchCase } from '@/lib/api/cases'
import { queryKeys } from '@/lib/api/query-keys'
import type { CaseRecord } from '@/lib/api/types'
import { cn } from '@/lib/utils'
import { Alert, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useCaseReferences } from '@/features/config-context/use-case-references'
import { validateRouting } from '@/features/task-flow/lib/validate'
import { TaskFlowTable } from '@/features/task-flow/task-flow-table'
import { emptyCase } from './empty-case'
import {
  deriveBindings,
  deriveInputSchema,
  normalizeOutputTypes,
  validateEditor,
  type InputFieldDraft,
  type OutputFieldDraft,
} from './lib/derive'
import { outputKindFor } from './lib/node-catalog'
import {
  parseWorkflow,
  type WorkflowGraph,
  type WorkflowNode,
} from './lib/workflow-parse'
import { BasicsSection } from './sections/basics'
import {
  EditableInputFields,
  EditableOutputFields,
} from './sections/field-cards'
import { useIsWide } from './sections/use-is-wide'
import { WorkflowImportSection } from './sections/workflow-import'

function stringifyObject(value: Record<string, unknown>): string {
  return JSON.stringify(value ?? {}, null, 2)
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

/** 一键生成：图中全部字面量参数 → 输入草稿。 */
function generateInputCandidates(nodes: WorkflowNode[]): InputFieldDraft[] {
  const out: InputFieldDraft[] = []
  const seen = new Set<string>()
  for (const n of nodes) {
    for (const input of n.inputs) {
      if (input.ref) continue
      let key = input.name
      let i = 2
      while (seen.has(key)) key = `${input.name}_${i++}`
      seen.add(key)
      const kind = input.kind
      out.push({
        key,
        type: kind === 'unknown' ? 'string' : kind,
        required: false,
        node_id: n.id,
        field_path: input.name,
      })
    }
  }
  return out
}

/** 一键生成：图中全部输出槽位 → 输出草稿。 */
function generateOutputCandidates(nodes: WorkflowNode[]): OutputFieldDraft[] {
  const out: OutputFieldDraft[] = []
  const seen = new Set<string>()
  for (const n of nodes) {
    for (let i = 0; i < n.outputCount; i += 1) {
      const base = `${n.id}[${i}]`
      let key = base
      let k = 2
      while (seen.has(key)) key = `${base}_${k++}`
      seen.add(key)
      out.push({
        key,
        type: outputKindFor(n.class_type),
        node_id: n.id,
        index: i,
      })
    }
  }
  return out
}

/** 定位需要行级红色标注的输入行：重复 key 的冲突行 / 未绑定节点与参数的字段。 */
function inputFieldErrors(inputs: InputFieldDraft[]): Record<number, boolean> {
  const errors: Record<number, boolean> = {}
  const seen = new Map<string, number>()
  inputs.forEach((field, index) => {
    const key = field.key.trim()
    if (!key) return
    if (seen.has(key)) {
      errors[index] = true
    } else {
      seen.set(key, index)
    }
    if (!field.node_id || !field.field_path) errors[index] = true
  })
  return errors
}

type CreateProps = {
  mode: 'create'
  initial?: undefined
  /** 创建模式的默认名称。 */
  initialName?: string
  /** 保存成功后是否跳转到 Case 详情页。 */
  redirectAfterSave?: boolean
  /** 隐藏表单自带的创建/取消按钮（向导内嵌时由向导底部按钮驱动）。 */
  hideActions?: boolean
  /** 只收集校验后的载荷，不落库。 */
  collectOnly?: boolean
  /** 左栏工作流配置 + 右栏基础信息（固定），向导 Step 1 使用。 */
  splitPane?: boolean
  /** splitPane 时渲染在左栏顶部的提示。 */
  leftIntro?: React.ReactNode
  /** 新建时把步骤标题用左侧时间线竖线串起来。 */
  stepRail?: boolean
  /** 向导内隐藏处理流程表。 */
  hideProcessing?: boolean
  /** 创建成功回调。 */
  onSaved?: (next: CaseRecord) => void
  /** 提交中状态变化回调。 */
  onPendingChange?: (pending: boolean) => void
  /** 是否有未保存修改。 */
  onDirtyChange?: (dirty: boolean) => void
  /** collectOnly 时回调校验后的载荷。 */
  onCollect?: (payload: CaseRecord) => void
  /** 工作流导入校验结果变化回调。 */
  onWorkflowImportedChange?: (imported: boolean) => void
  /** 表单 id。 */
  formId?: string
  /** 渲染在表单底部的自定义操作区（在 <form> 内部）。 */
  footer?: React.ReactNode
}

type EditProps = {
  mode: 'edit'
  initial: CaseRecord
  readOnly?: boolean
  showBasics?: boolean
  showWorkflow?: boolean
  onSaved?: (next: CaseRecord) => void
  onCancel?: () => void
  onPendingChange?: (pending: boolean) => void
  onDirtyChange?: (dirty: boolean) => void
  collectOnly?: boolean
  splitPane?: boolean
  leftIntro?: React.ReactNode
  stepRail?: boolean
  hideProcessing?: boolean
  onCollect?: (payload: CaseRecord) => void
  /** 工作流导入校验结果变化回调。 */
  onWorkflowImportedChange?: (imported: boolean) => void
  formId?: string
  footer?: React.ReactNode
}

export type WorkflowEditorProps = CreateProps | EditProps

/**
 * 统一工作流编辑组件：导入 JSON + 基础信息 + 输入 + 输出。
 * 独立编辑页（CaseForm）与 Quick Config 第一步共用，改一次两端都生效。
 */
export function WorkflowEditor(props: WorkflowEditorProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const isWide = useIsWide()

  const initial = props.mode === 'edit' ? props.initial : emptyCase()
  const readOnly = props.mode === 'edit' && props.readOnly === true
  const showBasics = props.mode !== 'edit' || props.showBasics !== false
  const showWorkflow = props.mode !== 'edit' || props.showWorkflow !== false
  const hideProcessing = props.hideProcessing === true
  const railRef = useRef<HTMLDivElement>(null)
  const [rail, setRail] = useState<{ top: number; height: number } | null>(null)
  useLayoutEffect(() => {
    if (!props.stepRail) {
      return
    }
    const root = railRef.current
    if (!root) return
    const update = () => {
      const pick = (sel: string) => root.querySelector<HTMLElement>(sel)
      const imp = pick(
        '[data-testid="case-section-workflow-import"] [class*="rounded-full"]'
      )
      const out = pick(
        hideProcessing
          ? '[data-testid="case-section-outputs"] [class*="rounded-full"]'
          : '[data-testid="case-section-processing"] [class*="rounded-full"]'
      )
      if (!imp || !out) {
        setRail(null)
        return
      }
      const base = root.getBoundingClientRect().top
      const center = (el: HTMLElement) =>
        el.getBoundingClientRect().top - base + el.offsetHeight / 2
      const top = center(imp)
      const height = center(out) - top
      setRail(top >= 0 && height > 0 ? { top, height } : null)
    }
    update()
    const ro = new ResizeObserver(update)
    ro.observe(root)
    return () => ro.disconnect()
  }, [props.stepRail, hideProcessing])
  const [draft, setDraft] = useState<CaseRecord>(() => ({
    ...structuredClone(initial),
    ...(props.mode === 'create' && props.initialName
      ? { name: props.initialName }
      : {}),
  }))
  const [workflowText, setWorkflowText] = useState(() =>
    stringifyObject(initial.bindings.workflow)
  )
  const [workflowFilename, setWorkflowFilename] = useState(
    initial.workflow_filename ?? ''
  )
  const [graph, setGraph] = useState<WorkflowGraph | undefined>(() => {
    if (props.mode !== 'edit') return undefined
    const result = parseWorkflow(
      stringifyObject(props.initial.bindings.workflow)
    )
    return result.ok ? result.graph : undefined
  })
  const [importError, setImportError] = useState<string | undefined>(() => {
    if (props.mode !== 'edit') return undefined
    const result = parseWorkflow(
      stringifyObject(props.initial.bindings.workflow)
    )
    return result.ok ? undefined : result.error
  })
  const [inputDrafts, setInputDrafts] = useState<InputFieldDraft[]>(() =>
    props.mode === 'edit' ? toInputDrafts(props.initial) : []
  )
  const [outputDrafts, setOutputDrafts] = useState<OutputFieldDraft[]>(() =>
    props.mode === 'edit' ? toOutputDrafts(props.initial) : []
  )
  const [nameError, setNameError] = useState<string | undefined>()
  const [showFieldErrors, setShowFieldErrors] = useState(false)
  const nameRef = useRef<HTMLInputElement | null>(null)
  const { topics, attributes, edges, presence } = useCaseReferences(
    props.mode === 'edit' ? props.initial : undefined
  )
  const routingValidation = useMemo(
    () => validateRouting(draft.routing, topics, attributes),
    [draft.routing, topics, attributes]
  )
  const editorValidation = useMemo(
    () => validateEditor(inputDrafts, outputDrafts),
    [inputDrafts, outputDrafts]
  )

  const pristineRef = useRef<{
    draft: string
    workflowText: string
    workflowFilename: string
    inputs: string
    outputs: string
  } | null>(null)
  if (pristineRef.current === null) {
    pristineRef.current = {
      draft: JSON.stringify(draft),
      workflowText,
      workflowFilename,
      inputs: JSON.stringify(inputDrafts),
      outputs: JSON.stringify(outputDrafts),
    }
  }

  const createMutation = useMutation({
    mutationFn: createCase,
    onSuccess: async (created) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.cases.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.linkHealth })
      toast.success(t('cases.createSuccess'))
      const redirectAfterSave =
        props.mode === 'create' && props.redirectAfterSave !== false
      if (redirectAfterSave) {
        void navigate({
          to: '/cases/$caseId',
          params: { caseId: String(created.id) },
        })
      }
      if (props.mode === 'create') props.onSaved?.(created)
    },
  })

  const updateMutation = useMutation({
    mutationFn: (body: CaseRecord) => {
      if (props.mode !== 'edit') {
        throw new Error('update requires edit mode')
      }
      return patchCase(props.initial.id, body)
    },
    onSuccess: async (updated) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.cases.all })
      await queryClient.invalidateQueries({
        queryKey: queryKeys.cases.detail(updated.id),
      })
      void queryClient.invalidateQueries({ queryKey: queryKeys.linkHealth })
      setDraft(structuredClone(updated))
      const text = stringifyObject(updated.bindings.workflow)
      const parsed = parseWorkflow(text)
      setWorkflowText(text)
      setWorkflowFilename(updated.workflow_filename ?? '')
      setGraph(parsed.ok ? parsed.graph : undefined)
      setImportError(parsed.ok ? undefined : parsed.error)
      setInputDrafts(toInputDrafts(updated))
      setOutputDrafts(toOutputDrafts(updated))
      toast.success(t('common.successSaved'))
      if (props.mode === 'edit') props.onSaved?.(updated)
    },
  })

  const pending = createMutation.isPending || updateMutation.isPending
  const disabled = pending || readOnly

  useEffect(() => {
    props.onPendingChange?.(pending)
  }, [pending, props.onPendingChange])

  useEffect(() => {
    const p = pristineRef.current
    if (!p) return
    const dirty =
      JSON.stringify(draft) !== p.draft ||
      workflowText !== p.workflowText ||
      workflowFilename !== p.workflowFilename ||
      JSON.stringify(inputDrafts) !== p.inputs ||
      JSON.stringify(outputDrafts) !== p.outputs
    props.onDirtyChange?.(dirty)
  }, [
    draft,
    workflowText,
    workflowFilename,
    inputDrafts,
    outputDrafts,
    props.onDirtyChange,
  ])

  function onWorkflowTextChange(next: string) {
    setWorkflowText(next)
    setImportError(undefined)
    if (!next.trim()) {
      setGraph(undefined)
      props.onWorkflowImportedChange?.(false)
      return
    }
    const result = parseWorkflow(next)
    if (result.ok) {
      setGraph(result.graph)
      props.onWorkflowImportedChange?.(true)
    } else {
      setGraph(undefined)
      setImportError(result.error)
      props.onWorkflowImportedChange?.(false)
    }
  }

  function generateInputs() {
    if (!graph) return
    const existing = new Set(
      inputDrafts.map((f) => `${f.node_id}\u0000${f.field_path}`)
    )
    const add = generateInputCandidates(graph.nodes).filter(
      (c) => !existing.has(`${c.node_id}\u0000${c.field_path}`)
    )
    if (add.length) {
      setInputDrafts((prev) => [...prev, ...add])
    }
  }

  function generateOutputs() {
    if (!graph) return
    const existing = new Set(
      outputDrafts.map((f) => `${f.node_id}\u0000${f.index ?? 0}`)
    )
    const add = generateOutputCandidates(graph.nodes).filter(
      (c) => !existing.has(`${c.node_id}\u0000${c.index ?? 0}`)
    )
    if (add.length) {
      setOutputDrafts((prev) => [...prev, ...add])
    }
  }

  function buildPayload(): CaseRecord | null {
    if (!showWorkflow) {
      return {
        ...draft,
        id: draft.id,
        name: draft.name.trim(),
        description: draft.description?.trim() || undefined,
        preview: draft.preview?.trim() || undefined,
        tags: draft.tags?.length ? draft.tags : undefined,
        categories: draft.categories?.length ? draft.categories : undefined,
      }
    }
    if (!graph) {
      setShowFieldErrors(true)
      return null
    }
    if (!hideProcessing && !routingValidation.valid) {
      return null
    }
    const validation = editorValidation
    if (
      validation.duplicateKey ||
      validation.unboundInput ||
      validation.noOutput
    ) {
      setShowFieldErrors(true)
      return null
    }
    setShowFieldErrors(false)
    const bindings = deriveBindings(inputDrafts, outputDrafts)
    const inputSchema = deriveInputSchema(inputDrafts)
    return {
      ...draft,
      id: draft.id,
      name: draft.name.trim(),
      description: draft.description?.trim() || undefined,
      preview: draft.preview?.trim() || undefined,
      tags: draft.tags?.length ? draft.tags : undefined,
      categories: draft.categories?.length ? draft.categories : undefined,
      inputs: inputDrafts.map((f) => ({
        key: f.key.trim(),
        type: f.type,
        required: f.required,
        description: f.description?.trim() || undefined,
      })),
      outputs: normalizeOutputTypes(outputDrafts, graph.nodes).map((f) => ({
        key: f.key.trim(),
        type: f.type,
        description: f.description?.trim() || undefined,
      })),
      bindings: {
        workflow: graph.api,
        ...bindings,
      },
      input_schema: inputSchema,
      workflow_filename: workflowFilename.trim() || undefined,
    }
  }

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (props.mode === 'create' && !draft.name.trim()) {
      setNameError(t('cases.errNameRequired'))
      nameRef.current?.focus()
      return
    }
    const payload = buildPayload()
    if (!payload) return

    if (props.collectOnly) {
      props.onCollect?.(payload)
      return
    }

    if (props.mode === 'create') {
      createMutation.mutate(payload)
      return
    }
    updateMutation.mutate(payload)
  }

  const basicsSection = showBasics ? (
    <BasicsSection
      value={{
        name: draft.name,
        description: draft.description,
        preview: draft.preview,
        tags: draft.tags,
        categories: draft.categories,
        enabled: draft.enabled,
      }}
      onChange={(basics) => {
        if (basics.name !== undefined) setNameError(undefined)
        setDraft((prev) => ({ ...prev, ...basics }))
      }}
      showEnabled={props.mode === 'create'}
      disabled={disabled}
      nameError={nameError}
      nameRef={nameRef}
    />
  ) : null

  const importSection = showWorkflow ? (
    <WorkflowImportSection
      value={workflowText}
      graph={graph}
      error={importError}
      filename={workflowFilename || undefined}
      onChange={onWorkflowTextChange}
      onFileName={setWorkflowFilename}
      disabled={disabled}
      stepRail={props.stepRail}
    />
  ) : null

  const inputsSection = showWorkflow ? (
    <section
      className={cn('relative flex flex-col gap-3', props.stepRail && 'pl-7')}
      data-testid='case-section-inputs'
    >
      <div className='flex items-center gap-2'>
        <span
          className={cn(
            'flex size-5 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-semibold text-primary-foreground',
            props.stepRail && 'absolute top-0.5 left-0'
          )}
        >
          2
        </span>
        <div className='min-w-0 flex-1'>
          <div className='flex min-w-0 flex-wrap items-center gap-1.5'>
            <Badge
              variant='default'
              className='h-auto shrink-0 px-1.5 py-0.5 text-xs leading-none'
            >
              {t('cases.inputPill')}
            </Badge>
            <h3 className='text-sm font-semibold'>
              {t('cases.inputsHeading')}
            </h3>
          </div>
          <p className='text-xs text-muted-foreground'>
            {t('cases.inputsHint')}
          </p>
        </div>
        {graph ? (
          <Button
            type='button'
            size='sm'
            variant='ghost'
            disabled={disabled}
            onClick={generateInputs}
            className='h-8 shrink-0 gap-1 text-xs'
          >
            <Sparkles className='size-3.5' />
            {t('cases.autoGenerateInputs')}
          </Button>
        ) : null}
      </div>
      {showFieldErrors &&
      (editorValidation.duplicateKey || editorValidation.unboundInput) ? (
        <Alert variant='destructive'>
          <TriangleAlert aria-hidden='true' />
          <AlertTitle>
            {editorValidation.duplicateKey
              ? t('cases.errDuplicateKey')
              : t('cases.errInputNotBound')}
          </AlertTitle>
        </Alert>
      ) : null}
      <EditableInputFields
        nodes={graph?.nodes ?? []}
        fields={inputDrafts}
        onChange={(index, next) =>
          setInputDrafts((prev) =>
            prev.map((row, i) => (i === index ? next : row))
          )
        }
        onRemove={(index) =>
          setInputDrafts((prev) => prev.filter((_, i) => i !== index))
        }
        onReorder={graph ? (next) => setInputDrafts(next) : undefined}
        onAdd={() =>
          setInputDrafts((prev) => [
            ...prev,
            {
              key: t('common.untitled'),
              type: 'string',
              required: false,
              node_id: '',
              field_path: '',
            },
          ])
        }
        wide={isWide || (!graph && inputDrafts.length === 0)}
        disabled={disabled || !graph}
        lockReason={graph ? undefined : t('cases.emptyWorkflowLock')}
        fieldErrors={
          showFieldErrors ? inputFieldErrors(inputDrafts) : undefined
        }
      />
    </section>
  ) : null

  const outputsSection = showWorkflow ? (
    <section
      className={cn('relative flex flex-col gap-3', props.stepRail && 'pl-7')}
      data-testid='case-section-outputs'
    >
      <div className='flex items-center gap-2'>
        <span
          className={cn(
            'flex size-5 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-semibold text-primary-foreground',
            props.stepRail && 'absolute top-0.5 left-0'
          )}
        >
          3
        </span>
        <div className='min-w-0 flex-1'>
          <div className='flex min-w-0 flex-wrap items-center gap-1.5'>
            <Badge
              variant='default'
              className='h-auto shrink-0 px-1.5 py-0.5 text-xs leading-none'
            >
              {t('cases.outputPill')}
            </Badge>
            <h3 className='text-sm font-semibold'>
              {t('cases.outputsHeading')}
            </h3>
          </div>
          <p className='text-xs text-muted-foreground'>
            {t('cases.outputsHint')}
          </p>
        </div>
        {graph ? (
          <Button
            type='button'
            size='sm'
            variant='ghost'
            disabled={disabled}
            onClick={generateOutputs}
            className='h-8 shrink-0 gap-1 text-xs'
          >
            <Sparkles className='size-3.5' />
            {t('cases.autoGenerateOutputs')}
          </Button>
        ) : null}
      </div>
      {showFieldErrors && editorValidation.noOutput ? (
        <Alert variant='destructive'>
          <TriangleAlert aria-hidden='true' />
          <AlertTitle>{t('cases.errNoOutput')}</AlertTitle>
        </Alert>
      ) : null}
      <EditableOutputFields
        nodes={graph?.nodes ?? []}
        fields={outputDrafts}
        onChange={(index, next) =>
          setOutputDrafts((prev) =>
            prev.map((row, i) => (i === index ? next : row))
          )
        }
        onRemove={(index) =>
          setOutputDrafts((prev) => prev.filter((_, i) => i !== index))
        }
        onAdd={() =>
          setOutputDrafts((prev) => [
            ...prev,
            {
              key: t('common.untitled'),
              type: 'image',
              node_id: '',
              index: 0,
            },
          ])
        }
        wide={isWide || (!graph && outputDrafts.length === 0)}
        disabled={disabled || !graph}
        lockReason={graph ? undefined : t('cases.emptyWorkflowLock')}
      />
    </section>
  ) : null

  const processingSection =
    showWorkflow && !hideProcessing ? (
      <section
        className={cn('relative flex flex-col gap-3', props.stepRail && 'pl-7')}
        data-testid='case-section-processing'
      >
        <div className='flex items-center gap-2'>
          <span
            className={cn(
              'flex size-5 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-semibold text-primary-foreground',
              props.stepRail && 'absolute top-0.5 left-0'
            )}
          >
            4
          </span>
          <div className='min-w-0 flex-1'>
            <div className='flex min-w-0 flex-wrap items-center gap-1.5'>
              <Badge
                variant='default'
                className='h-auto shrink-0 px-1.5 py-0.5 text-xs leading-none'
              >
                {t('cases.processingPill')}
              </Badge>
              <h3 className='text-sm font-semibold'>
                {t('cases.processingHeading')}
              </h3>
            </div>
            <p className='text-xs text-muted-foreground'>
              {t('cases.processingHint')}
            </p>
          </div>
        </div>
        {showFieldErrors && !hideProcessing && !routingValidation.valid ? (
          <Alert variant='destructive'>
            <TriangleAlert aria-hidden='true' />
            <AlertTitle>{t('cases.errRoutingInvalid')}</AlertTitle>
          </Alert>
        ) : null}
        {graph ? (
          <TaskFlowTable
            routing={draft.routing}
            topics={topics}
            attributes={attributes}
            edges={edges}
            presence={presence}
            onChange={(next) =>
              setDraft((prev) => ({ ...prev, routing: next }))
            }
            showHeader={false}
            readOnly={disabled}
          />
        ) : null}
      </section>
    ) : null

  return (
    <form
      id={
        props.formId ?? (props.mode === 'edit' ? 'case-edit-form' : undefined)
      }
      onSubmit={onSubmit}
      className='flex flex-col gap-8'
      noValidate
      data-testid='case-form'
    >
      {props.splitPane ? (
        <div className='flex flex-col gap-6 lg:flex-row lg:items-start'>
          <div
            ref={railRef}
            className='relative flex min-w-0 flex-1 flex-col gap-6'
          >
            {props.stepRail && rail ? (
              <div
                aria-hidden='true'
                className='pointer-events-none absolute w-px bg-border'
                style={{ left: '9.5px', top: rail.top, height: rail.height }}
              />
            ) : null}
            {props.leftIntro}
            {importSection}
            {inputsSection}
            {outputsSection}
            {processingSection}
          </div>
          <div className='w-full shrink-0 lg:w-80 xl:w-[25rem]'>
            <div className='rounded-xl border border-border bg-card p-5'>
              {basicsSection}
            </div>
          </div>
        </div>
      ) : (
        <>
          {basicsSection}
          {importSection}
          {inputsSection}
          {outputsSection}
          {processingSection}
        </>
      )}
      {props.footer}
    </form>
  )
}
