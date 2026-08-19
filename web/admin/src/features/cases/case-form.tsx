import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { createCase, disableCase, enableCase, patchCase } from '@/lib/api/cases'
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
}

type EditProps = {
  mode: 'edit'
  initial: CaseRecord
}

type Props = CreateProps | EditProps

export function CaseForm(props: Props) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const initial = props.mode === 'edit' ? props.initial : emptyCase()
  const [draft, setDraft] = useState<CaseRecord>(() => structuredClone(initial))
  const [workflowText, setWorkflowText] = useState(() =>
    stringifyObject(initial.bindings.workflow)
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
      void navigate({
        to: '/cases/$caseId',
        params: { caseId: created.id },
      })
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
      setGraph(parsed.ok ? parsed.graph : undefined)
      setImportError(parsed.ok ? undefined : parsed.error)
      setInputDrafts(toInputDrafts(updated))
      setOutputDrafts(toOutputDrafts(updated))
      setAdvancedText(text)
      toast.success(t('common.successSaved'))
    },
  })

  const toggleMutation = useMutation({
    mutationFn: async () => {
      if (props.mode !== 'edit') {
        throw new Error('toggle requires edit mode')
      }
      return draft.enabled
        ? disableCase(props.initial.id)
        : enableCase(props.initial.id)
    },
    onSuccess: async (updated) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.cases.all })
      await queryClient.invalidateQueries({
        queryKey: queryKeys.cases.detail(updated.id),
      })
      setDraft((prev) => ({ ...prev, enabled: updated.enabled }))
      toast.success(
        updated.enabled ? t('cases.enableSuccess') : t('cases.disableSuccess')
      )
    },
  })

  const pending =
    createMutation.isPending ||
    updateMutation.isPending ||
    toggleMutation.isPending
  const mutationError =
    createMutation.error ??
    updateMutation.error ??
    toggleMutation.error ??
    undefined

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
    if (!graph) {
      setEditorError(t('cases.emptyWorkflowLock'))
      return null
    }
    const validation = validateEditor(inputDrafts, outputDrafts)
    if (
      validation.duplicateKey ||
      validation.unboundRequired ||
      validation.noOutput
    ) {
      setEditorError(
        validation.duplicateKey
          ? t('cases.errDuplicateKey')
          : validation.unboundRequired
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
      id: draft.id.trim(),
      name: draft.name.trim(),
      description: draft.description?.trim() || undefined,
      preview: draft.preview?.trim() || undefined,
      menu_key: draft.menu_key?.trim() || undefined,
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
    }
  }

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    const payload = buildPayload()
    if (!payload) return

    if (props.mode === 'create') {
      if (!payload.id || !payload.name) return
      createMutation.mutate(payload)
      return
    }
    updateMutation.mutate(payload)
  }

  return (
    <form onSubmit={onSubmit} className='space-y-8' data-testid='case-form'>
      {props.mode === 'edit' ? (
        <div className='flex flex-wrap gap-2'>
          <Button
            type='button'
            variant='outline'
            size='sm'
            disabled={pending}
            onClick={() => toggleMutation.mutate()}
          >
            {draft.enabled ? t('cases.disable') : t('cases.enable')}
          </Button>
        </div>
      ) : null}

      <BasicsSection
        value={{
          id: draft.id,
          name: draft.name,
          description: draft.description,
          preview: draft.preview,
          price: draft.price,
          tags: draft.tags,
          menu_key: draft.menu_key,
          categories: draft.categories,
          enabled: draft.enabled,
        }}
        onChange={(basics) => setDraft((prev) => ({ ...prev, ...basics }))}
        idEditable={props.mode === 'create'}
        showEnabled={props.mode === 'create'}
        disabled={pending}
      />

      <WorkflowImportSection
        value={workflowText}
        graph={graph}
        error={importError}
        onChange={onWorkflowTextChange}
        disabled={pending}
      />

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
                  disabled={pending}
                />
              ))}
            </ul>
            <Button
              type='button'
              size='sm'
              variant='outline'
              disabled={pending}
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
                  disabled={pending}
                />
              ))}
            </ul>
            <Button
              type='button'
              size='sm'
              variant='outline'
              disabled={pending}
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

      <PreviewSection
        bindings={deriveBindings(inputDrafts, outputDrafts)}
        inputSchema={deriveInputSchema(inputDrafts)}
      />

      <AdvancedSection
        open={advancedOpen}
        editMode={advancedEdit}
        text={advancedText}
        onOpen={() => setAdvancedOpen(true)}
        onEnterEdit={() => setAdvancedEdit(true)}
        onTextChange={setAdvancedText}
        disabled={pending}
      />

      {mutationError ? (
        <ErrorBanner message={errorMessage(mutationError)} />
      ) : null}
      {editorError ? (
        <p className='text-sm text-destructive' role='alert'>
          {editorError}
        </p>
      ) : null}

      <div className='flex flex-wrap gap-2'>
        <Button type='submit' disabled={pending}>
          {props.mode === 'create'
            ? t('common.create')
            : t('cases.saveWorkflow')}
        </Button>
        {props.mode === 'create' ? (
          <Button
            type='button'
            variant='outline'
            disabled={pending}
            onClick={() => void navigate({ to: '/cases' })}
          >
            {t('common.cancel')}
          </Button>
        ) : null}
      </div>
    </form>
  )
}
