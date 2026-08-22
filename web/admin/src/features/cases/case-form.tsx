import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { createCase, patchCase } from '@/lib/api/cases'
import { queryKeys } from '@/lib/api/query-keys'
import type { CaseRecord } from '@/lib/api/types'
import { Button } from '@/components/ui/button'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { emptyCase } from './empty-case'
import {
  deriveBindings,
  deriveInputSchema,
  validateEditor,
  type InputFieldDraft,
  type OutputFieldDraft,
} from './lib/derive'
import { parseWorkflow, type WorkflowGraph } from './lib/workflow-parse'
import { AdvancedSection } from './sections/advanced'
import { BasicsSection } from './sections/basics'
import { InputFieldCard, OutputFieldCard } from './sections/field-cards'
import { PreviewSection } from './sections/preview'
import { WorkflowImportSection } from './sections/workflow-import'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

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

type CreateProps = {
  mode: 'create'
  initial?: undefined
  /** 创建模式的默认名称（向导传入「未命名工作流」）。 */
  initialName?: string
  /** 保存成功后是否跳转到 Case 详情页（向导内嵌时置 false）。 */
  redirectAfterSave?: boolean
  /** 隐藏表单自带的创建/取消按钮（向导内嵌时由向导底部按钮驱动）。 */
  hideActions?: boolean
  /** 只收集校验后的载荷，不落库（向导统一在完成页提交）。 */
  collectOnly?: boolean
  /** 左栏工作流配置（滚动）+ 右栏基础信息（固定），向导 Step 1 使用。 */
  splitPane?: boolean
  /** splitPane 时渲染在左栏顶部的提示。 */
  leftIntro?: React.ReactNode
  /** 创建成功回调（向导内嵌时用于推进步骤）。 */
  onSaved?: (next: CaseRecord) => void
  /** collectOnly 时回调校验后的载荷。 */
  onCollect?: (payload: CaseRecord) => void
  /** 表单 id（向导内嵌时由外部按钮通过 id 触发 requestSubmit）。 */
  formId?: string
}

type EditProps = {
  mode: 'edit'
  initial: CaseRecord
  readOnly?: boolean
  showBasics?: boolean
  showWorkflow?: boolean
  onSaved?: (next: CaseRecord) => void
  onCancel?: () => void
  /** 只收集校验后的载荷，不落库（向导统一在完成页提交）。 */
  collectOnly?: boolean
  /** 左栏工作流配置（滚动）+ 右栏基础信息（固定），向导 Step 1 使用。 */
  splitPane?: boolean
  /** splitPane 时渲染在左栏顶部的提示。 */
  leftIntro?: React.ReactNode
  /** collectOnly 时回调校验后的载荷。 */
  onCollect?: (payload: CaseRecord) => void
  /** 表单 id（向导内嵌时由外部按钮通过 id 触发 requestSubmit）。 */
  formId?: string
}

type Props = CreateProps | EditProps

export function CaseForm(props: Props) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const initial = props.mode === 'edit' ? props.initial : emptyCase()
  const readOnly = props.mode === 'edit' && props.readOnly === true
  const showBasics = props.mode !== 'edit' || props.showBasics !== false
  const showWorkflow = props.mode !== 'edit' || props.showWorkflow !== false
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
  const [editorError, setEditorError] = useState<string | undefined>()
  const [advancedOpen, setAdvancedOpen] = useState(false)
  const [advancedEdit, setAdvancedEdit] = useState(false)
  const [advancedText, setAdvancedText] = useState(() =>
    stringifyObject(initial.bindings.workflow)
  )

  const createMutation = useMutation({
    mutationFn: createCase,
    onSuccess: async (created) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.cases.all })
      toast.success(t('cases.createSuccess'))
      const redirectAfterSave = props.mode === 'create' && props.redirectAfterSave !== false
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
      setDraft(structuredClone(updated))
      const text = stringifyObject(updated.bindings.workflow)
      const parsed = parseWorkflow(text)
      setWorkflowText(text)
      setWorkflowFilename(updated.workflow_filename ?? '')
      setGraph(parsed.ok ? parsed.graph : undefined)
      setImportError(parsed.ok ? undefined : parsed.error)
      setInputDrafts(toInputDrafts(updated))
      setOutputDrafts(toOutputDrafts(updated))
      setAdvancedText(text)
      toast.success(t('common.successSaved'))
      if (props.mode === 'edit') props.onSaved?.(updated)
    },
  })

  const pending = createMutation.isPending || updateMutation.isPending
  const mutationError =
    createMutation.error ?? updateMutation.error ?? undefined
  const disabled = pending || readOnly

  function onWorkflowTextChange(next: string) {
    setWorkflowText(next)
    setImportError(undefined)
    if (!next.trim()) {
      setGraph(undefined)
      return
    }
    const result = parseWorkflow(next)
    if (result.ok) {
      setGraph(result.graph)
    } else {
      setGraph(undefined)
      setImportError(result.error)
    }
  }

  function buildPayload(): CaseRecord | null {
    if (!showWorkflow) {
      setEditorError(undefined)
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
      setEditorError(t('cases.emptyWorkflowLock'))
      return null
    }
    const validation = validateEditor(inputDrafts, outputDrafts)
    if (
      validation.duplicateKey ||
      validation.unboundInput ||
      validation.noOutput
    ) {
      setEditorError(
        validation.duplicateKey
          ? t('cases.errDuplicateKey')
          : validation.unboundInput
            ? t('cases.errInputNotBound')
            : t('cases.errNoOutput')
      )
      return null
    }
    setEditorError(undefined)
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
      outputs: outputDrafts.map((f) => ({
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
    const payload = buildPayload()
    if (!payload) return

    if (props.collectOnly) {
      props.onCollect?.(payload)
      return
    }

    if (props.mode === 'create') {
      // id=0 时由后端自动分配（快速配置向导依赖此行为）。
      if (!payload.name) return
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
        price: draft.price,
        tags: draft.tags,
        categories: draft.categories,
        enabled: draft.enabled,
      }}
      onChange={(basics) => setDraft((prev) => ({ ...prev, ...basics }))}
      showEnabled={props.mode === 'create'}
      disabled={disabled}
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
    />
  ) : null

  const inputsSection = showWorkflow ? (
      <section className='space-y-3'>
        <div className='flex items-center gap-2'>
          <span className='flex size-5 items-center justify-center rounded-full bg-primary text-xs font-semibold text-primary-foreground'>
            2
          </span>
          <h3 className='text-sm font-semibold'>{t('cases.inputsHeading')}</h3>
          <span className='text-xs text-muted-foreground'>
            {t('cases.inputsHint')}
          </span>
        </div>
        {graph ? (
          <>
            <ul className='space-y-3'>
              {inputDrafts.map((field, index) => (
                <InputFieldCard
                  key={`input-${index}`}
                  nodes={graph.nodes}
                  value={field}
                  onChange={(next) =>
                    setInputDrafts((prev) =>
                      prev.map((row, i) => (i === index ? next : row))
                    )
                  }
                  onRemove={() =>
                    setInputDrafts((prev) => prev.filter((_, i) => i !== index))
                  }
                  disabled={disabled}
                />
              ))}
            </ul>
            <Button
              type='button'
              size='sm'
              variant='outline'
              disabled={disabled}
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
              {t('cases.addInput')}
            </Button>
          </>
        ) : (
          <p className='rounded-md border border-dashed p-3 text-xs text-muted-foreground'>
            {t('cases.emptyWorkflowLock')}
          </p>
        )}
      </section>
  ) : null

  const outputsSection = showWorkflow ? (
      <section className='space-y-3'>
        <div className='flex items-center gap-2'>
          <span className='flex size-5 items-center justify-center rounded-full bg-primary text-xs font-semibold text-primary-foreground'>
            3
          </span>
          <h3 className='text-sm font-semibold'>{t('cases.outputsHeading')}</h3>
          <span className='text-xs text-muted-foreground'>
            {t('cases.outputsHint')}
          </span>
        </div>
        {graph ? (
          <>
            <ul className='space-y-3'>
              {outputDrafts.map((field, index) => (
                <OutputFieldCard
                  key={`output-${index}`}
                  nodes={graph.nodes}
                  value={field}
                  onChange={(next) =>
                    setOutputDrafts((prev) =>
                      prev.map((row, i) => (i === index ? next : row))
                    )
                  }
                  onRemove={() =>
                    setOutputDrafts((prev) =>
                      prev.filter((_, i) => i !== index)
                    )
                  }
                  disabled={disabled}
                />
              ))}
            </ul>
            <Button
              type='button'
              size='sm'
              variant='outline'
              disabled={disabled}
              onClick={() =>
                setOutputDrafts((prev) => [
                  ...prev,
                  { key: '', type: 'image', node_id: '', index: 0 },
                ])
              }
            >
              {t('cases.addOutput')}
            </Button>
          </>
        ) : (
          <p className='rounded-md border border-dashed p-3 text-xs text-muted-foreground'>
            {t('cases.emptyWorkflowLock')}
          </p>
        )}
      </section>
  ) : null

  const previewSection = showWorkflow ? (
      <PreviewSection
        bindings={deriveBindings(inputDrafts, outputDrafts)}
        inputSchema={deriveInputSchema(inputDrafts)}
      />
  ) : null

  const advancedSection = showWorkflow ? (
        <AdvancedSection
          open={advancedOpen}
          editMode={advancedEdit}
          text={advancedText}
          onOpen={() => setAdvancedOpen(true)}
          onEnterEdit={() => setAdvancedEdit(true)}
          onTextChange={setAdvancedText}
          disabled={disabled}
        />
  ) : null

  const errorBlock = (
    <>
      {mutationError ? (
        <ErrorBanner message={errorMessage(mutationError)} />
      ) : null}
      {editorError ? (
        <p className='text-sm text-destructive' role='alert'>
          {editorError}
        </p>
      ) : null}
    </>
  )

  return (
    <form
      id={props.formId ?? (props.mode === 'edit' ? 'case-edit-form' : undefined)}
      onSubmit={onSubmit}
      className='space-y-8'
      data-testid='case-form'
    >
      {props.splitPane ? (
        <div className='flex items-start gap-6'>
          <div className='min-w-0 flex-1 space-y-6'>
            {props.leftIntro}
            {importSection}
            {inputsSection}
            {outputsSection}
            {errorBlock}
          </div>
          <div className='sticky top-0 w-80 shrink-0'>
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
          {previewSection}
          {advancedSection}
          {errorBlock}
        </>
      )}

      {props.mode === 'edit' || props.hideActions ? null : (
        <div className='flex flex-wrap gap-2'>
          <Button type='submit' disabled={pending}>
            {t('common.create')}
          </Button>
          <Button
            type='button'
            variant='outline'
            disabled={pending}
            onClick={() => void navigate({ to: '/cases' })}
          >
            {t('common.cancel')}
          </Button>
        </div>
      )}
    </form>
  )
}
