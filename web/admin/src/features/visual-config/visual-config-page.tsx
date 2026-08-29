import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Workflow } from 'lucide-react'
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
import { TaskFlowEditor } from '@/features/task-flow/task-flow-editor'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

/**
 * 可视化配置页：选择工作流后，用可视化画布编辑其路由规则并保存。
 * 复用 TaskFlowEditor（校验顶栏 + 画布 + Topic 池），数据取自真实 API。
 */
export function VisualConfigPage() {
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
      if (!record) throw new Error('未选择工作流')
      return patchCase(record.id, { routing })
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.cases.all })
      toast.success('路由配置已保存')
    },
  })

  return (
    <div className='flex h-full min-h-0 w-full flex-col'>
      <div className='flex shrink-0 flex-wrap items-center justify-between gap-3 border-b border-border bg-background px-6 py-4'>
        <div className='flex min-w-0 items-center gap-3'>
          <Workflow className='size-5 text-muted-foreground' />
          <h1 className='truncate text-lg font-semibold tracking-tight'>
            可视化配置
          </h1>
          <p className='hidden text-sm text-muted-foreground md:inline'>
            选择工作流，用画布可视化编辑路由规则
          </p>
        </div>
        <Select
          value={caseId != null ? String(caseId) : undefined}
          onValueChange={handleSelect}
        >
          <SelectTrigger size='sm' className='h-9 w-64' aria-label='选择工作流'>
            <SelectValue placeholder={loading ? '加载中…' : '选择工作流'} />
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
          <TaskFlowEditor
            routing={routing}
            topics={topics}
            attributes={attributes}
            edges={edges}
            presence={presence}
            caseName={record.name}
            onChange={setRouting}
            title={record.name}
            className='h-full border-0'
            headerActions={
              <Button
                type='button'
                size='sm'
                disabled={save.isPending}
                onClick={() => save.mutate()}
                data-visual-config-save
              >
                {save.isPending ? '保存中…' : '保存配置'}
              </Button>
            }
          />
          {save.error ? (
            <p className='px-4 py-3 text-sm text-destructive' role='alert'>
              {errorMessage(save.error)}
            </p>
          ) : null}
        </div>
      ) : (
        <div className='flex flex-1 items-center justify-center px-6'>
          <div className='flex max-w-md flex-col items-center gap-2 text-center'>
            <Workflow className='size-10 text-muted-foreground/60' />
            <p className='text-sm font-medium'>还没有选择工作流</p>
            <p className='text-sm text-muted-foreground'>
              从右上角选择一个工作流，即可在画布中可视化编辑它的路由规则。
            </p>
          </div>
        </div>
      )}
    </div>
  )
}
