import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { Button } from '@/components/ui/button'
import {
  createCase,
  disableCase,
  enableCase,
  patchCase,
} from '@/lib/api/cases'
import { queryKeys } from '@/lib/api/query-keys'
import type { CaseRecord } from '@/lib/api/types'
import { emptyCase } from './empty-case'
import { BasicsSection } from './sections/basics'
import { BindingsSection } from './sections/bindings'
import { InputSchemaSection } from './sections/input-schema'
import { IoFieldsSection } from './sections/io-fields'
import { WorkflowJsonSection } from './sections/workflow-json'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function stringifyObject(value: Record<string, unknown>): string {
  return JSON.stringify(value ?? {}, null, 2)
}

function parseObjectJson(
  raw: string,
  invalidMessage: string,
): { ok: true; value: Record<string, unknown> } | { ok: false; error: string } {
  try {
    const parsed: unknown = JSON.parse(raw)
    if (
      parsed === null ||
      typeof parsed !== 'object' ||
      Array.isArray(parsed)
    ) {
      return { ok: false, error: invalidMessage }
    }
    return { ok: true, value: parsed as Record<string, unknown> }
  } catch (err) {
    return {
      ok: false,
      error: err instanceof Error ? err.message : invalidMessage,
    }
  }
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

  const [draft, setDraft] = useState<CaseRecord>(() =>
    props.mode === 'edit' ? structuredClone(props.initial) : emptyCase(),
  )
  const [workflowText, setWorkflowText] = useState(() =>
    stringifyObject(
      props.mode === 'edit'
        ? props.initial.bindings.workflow
        : emptyCase().bindings.workflow,
    ),
  )
  const [inputSchemaText, setInputSchemaText] = useState(() =>
    stringifyObject(
      props.mode === 'edit'
        ? props.initial.input_schema
        : emptyCase().input_schema,
    ),
  )
  const [workflowError, setWorkflowError] = useState<string | undefined>()
  const [inputSchemaError, setInputSchemaError] = useState<
    string | undefined
  >()

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
      setWorkflowText(stringifyObject(updated.bindings.workflow))
      setInputSchemaText(stringifyObject(updated.input_schema))
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
        updated.enabled
          ? t('cases.enableSuccess')
          : t('cases.disableSuccess'),
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

  function buildPayload(): CaseRecord | null {
    const workflowParsed = parseObjectJson(
      workflowText,
      t('cases.jsonMustBeObject'),
    )
    const schemaParsed = parseObjectJson(
      inputSchemaText,
      t('cases.jsonMustBeObject'),
    )

    setWorkflowError(
      workflowParsed.ok ? undefined : workflowParsed.error,
    )
    setInputSchemaError(
      schemaParsed.ok ? undefined : schemaParsed.error,
    )

    if (!workflowParsed.ok || !schemaParsed.ok) {
      return null
    }

    return {
      ...draft,
      id: draft.id.trim(),
      name: draft.name.trim(),
      description: draft.description?.trim() || undefined,
      preview: draft.preview?.trim() || undefined,
      menu_key: draft.menu_key?.trim() || undefined,
      tags: draft.tags?.length ? draft.tags : undefined,
      categories: draft.categories?.length ? draft.categories : undefined,
      bindings: {
        ...draft.bindings,
        workflow: workflowParsed.value,
      },
      input_schema: schemaParsed.value,
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
    <form
      onSubmit={onSubmit}
      className='space-y-8'
      data-testid='case-form'
    >
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
        disabled={pending}
      />

      <IoFieldsSection
        inputs={draft.inputs}
        outputs={draft.outputs}
        onChange={({ inputs, outputs }) =>
          setDraft((prev) => ({ ...prev, inputs, outputs }))
        }
        disabled={pending}
      />

      <BindingsSection
        inputs={draft.bindings.inputs}
        outputs={draft.bindings.outputs}
        onChange={({ inputs, outputs }) =>
          setDraft((prev) => ({
            ...prev,
            bindings: { ...prev.bindings, inputs, outputs },
          }))
        }
        disabled={pending}
      />

      <WorkflowJsonSection
        value={workflowText}
        onChange={(next) => {
          setWorkflowText(next)
          if (workflowError) setWorkflowError(undefined)
        }}
        error={workflowError}
        disabled={pending}
      />

      <InputSchemaSection
        value={inputSchemaText}
        onChange={(next) => {
          setInputSchemaText(next)
          if (inputSchemaError) setInputSchemaError(undefined)
        }}
        error={inputSchemaError}
        disabled={pending}
      />

      {mutationError ? (
        <ErrorBanner message={errorMessage(mutationError)} />
      ) : null}

      <div className='flex flex-wrap gap-2'>
        <Button type='submit' disabled={pending}>
          {props.mode === 'create' ? t('common.create') : t('common.save')}
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
