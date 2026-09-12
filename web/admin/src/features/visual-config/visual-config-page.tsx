import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Workflow } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { listCases, patchCase } from '@/lib/api/cases'
import { queryKeys } from '@/lib/api/query-keys'
import type { CaseRecord, RoutingConfig } from '@/lib/api/types'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useCaseReferences } from '@/features/config-context/use-case-references'
import { TaskFlowTable } from '@/features/task-flow/task-flow-table'

/**
 * 可视化配置页：选择工作流后，用规则表编辑其处理流程并保存。
 */
export function VisualConfigPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [caseId, setCaseId] = useState<number | undefined>(undefined)
  const [routing, setRouting] = useState<RoutingConfig | undefined>(undefined)

  const casesQuery = useQuery({
    queryKey: queryKeys.cases.all,
    queryFn: () => listCases(),
  })
  const record: CaseRecord | undefined = casesQuery.data?.find(
    (c) => c.id === caseId
  )

  const { topics, attributes, edges, presence } = useCaseReferences(record)
  const loading = casesQuery.isLoading && !casesQuery.data

  function handleSelect(value: string) {
    const id = Number(value)
    setCaseId(id)
    const next = casesQuery.data?.find((c) => c.id === id)
    setRouting(next?.routing)
  }

  const save = useMutation({
    mutationFn: () => {
      if (!record) throw new Error(t('visualConfig.empty'))
      return patchCase(record.id, { routing })
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.cases.all })
      toast.success(t('visualConfig.saved'))
    },
  })

  return (
    <div className='flex h-full min-h-0 w-full flex-col'>
      <div className='flex shrink-0 flex-wrap items-center justify-between gap-3 border-b border-border bg-background px-6 py-4'>
        <div className='flex min-w-0 items-center gap-3'>
          <Workflow className='size-5 text-muted-foreground' aria-hidden='true' />
          <h1 className='truncate text-lg font-semibold tracking-tight'>
            {t('visualConfig.title')}
          </h1>
        </div>
        <Select
          value={caseId != null ? String(caseId) : undefined}
          onValueChange={handleSelect}
        >
          <SelectTrigger
            size='sm'
            className='h-9 w-64'
            aria-label={t('visualConfig.pickWorkflow')}
          >
            <SelectValue
              placeholder={
                loading ? t('common.loading') : t('visualConfig.pickWorkflow')
              }
            />
          </SelectTrigger>
          <SelectContent>
            {(casesQuery.data ?? []).map((c) => (
              <SelectItem key={c.id} value={String(c.id)}>
                {c.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {record ? (
        <div className='min-h-0 flex-1'>
          <TaskFlowTable
            routing={routing}
            topics={topics}
            attributes={attributes}
            edges={edges}
            presence={presence}
            onChange={setRouting}
            title={record.name}
            className='min-h-0 flex-1 border-0'
            headerActions={
              <Button
                type='button'
                size='sm'
                disabled={save.isPending}
                onClick={() => save.mutate()}
                data-visual-config-save
              >
                {save.isPending ? t('visualConfig.saving') : t('common.save')}
              </Button>
            }
          />
        </div>
      ) : (
        <div className='flex flex-1 items-center justify-center px-6'>
          <div className='flex max-w-md flex-col items-center gap-2 text-center'>
            <Workflow
              className='size-10 text-muted-foreground/60'
              aria-hidden='true'
            />
            <p className='text-sm font-medium'>{t('visualConfig.empty')}</p>
          </div>
        </div>
      )}
    </div>
  )
}
