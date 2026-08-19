import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { listCases } from '@/lib/api/cases'
import {
  getChannelMenu,
  putChannelMenu,
  type MenuNode,
} from '@/lib/api/channel-menu'
import { listCapabilities } from '@/lib/api/channels'
import { queryKeys } from '@/lib/api/query-keys'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/feedback/empty-state'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { countUnsaved, validateMenu } from './lib/menu-validate'
import { NodeConfigPanel } from './node-config-panel'
import { PhoneSimulation } from './phone-simulation'

type FlatNode = { node: MenuNode; depth: number }

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function emptyNode(): MenuNode {
  return {
    id: `btn-${Date.now()}`,
    label: '',
    order: 0,
    enabled: true,
  }
}

function flattenTree(nodes: MenuNode[], depth = 0): FlatNode[] {
  const out: FlatNode[] = []
  for (const node of nodes) {
    out.push({ node, depth })
    if (node.children?.length) {
      out.push(...flattenTree(node.children, depth + 1))
    }
  }
  return out
}

function countNodes(nodes: MenuNode[]): number {
  let count = 0
  for (const node of nodes) {
    count += 1
    if (node.children?.length) count += countNodes(node.children)
  }
  return count
}

function findNode(nodes: MenuNode[], id: string): MenuNode | null {
  for (const node of nodes) {
    if (node.id === id) return node
    if (node.children?.length) {
      const found = findNode(node.children, id)
      if (found) return found
    }
  }
  return null
}

function updateNodeInTree(
  nodes: MenuNode[],
  id: string,
  patch: Partial<MenuNode>
): MenuNode[] {
  return nodes.map((node) => {
    if (node.id === id) {
      return { ...node, ...patch }
    }
    if (node.children?.length) {
      return { ...node, children: updateNodeInTree(node.children, id, patch) }
    }
    return node
  })
}

function removeNodeFromTree(nodes: MenuNode[], id: string): MenuNode[] {
  return nodes
    .filter((node) => node.id !== id)
    .map((node) => ({
      ...node,
      children: node.children?.length
        ? removeNodeFromTree(node.children, id)
        : undefined,
    }))
}

function addChildToTree(
  nodes: MenuNode[],
  parentId: string,
  child: MenuNode
): MenuNode[] {
  return nodes.map((node) => {
    if (node.id === parentId) {
      return { ...node, children: [...(node.children ?? []), child] }
    }
    if (node.children?.length) {
      return {
        ...node,
        children: addChildToTree(node.children, parentId, child),
      }
    }
    return node
  })
}

function normalizeNode(node: MenuNode): MenuNode {
  const next: MenuNode = {
    id: node.id,
    label: node.label,
    order: Number(node.order) || 0,
    enabled: Boolean(node.enabled),
  }
  if (node.capability_id) next.capability_id = node.capability_id
  if (node.params && Object.keys(node.params).length > 0)
    next.params = node.params
  if (node.render_override && Object.keys(node.render_override).length > 0) {
    next.render_override = node.render_override
  }
  const intro = node.intro_text?.trim()
  if (intro) next.intro_text = intro
  const placeholder = node.placeholder_text?.trim()
  if (placeholder) next.placeholder_text = placeholder
  if (node.reply && (node.reply.text?.trim() || node.reply.images?.length)) {
    next.reply = node.reply
  }
  if (node.children?.length) {
    next.children = node.children.map(normalizeNode)
  }
  return next
}

export function ChannelMenuEditor({ channelId }: { channelId: string }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [draft, setDraft] = useState<MenuNode[] | null>(null)
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const menuQuery = useQuery({
    queryKey: queryKeys.channels.menu(channelId),
    queryFn: () => getChannelMenu(channelId),
  })

  const casesQuery = useQuery({
    queryKey: [...queryKeys.cases.all, { limit: 200 }] as const,
    queryFn: () => listCases({ limit: 200 }),
  })

  const capsQuery = useQuery({
    queryKey: ['channels', 'capabilities'] as const,
    queryFn: listCapabilities,
  })

  const items = draft ?? menuQuery.data?.items ?? []
  const updatedAt = menuQuery.data?.updated_at ?? ''
  const firstRoot = items.find((it) => it.enabled) ?? items[0]
  const rawColumns = firstRoot?.render_override?.columns
  const rootColumns =
    typeof rawColumns === 'number' && rawColumns >= 1 && rawColumns <= 8
      ? rawColumns
      : 2
  const effectiveSelectedId = useMemo(() => {
    if (selectedId && findNode(items, selectedId)) return selectedId
    return items[0]?.id ?? null
  }, [selectedId, items])
  const selected = effectiveSelectedId
    ? findNode(items, effectiveSelectedId)
    : null
  const baseline = useMemo(() => menuQuery.data?.items ?? [], [menuQuery.data])
  const validation = useMemo(() => validateMenu(items), [items])
  const dirtyCount = useMemo(
    () => countUnsaved(items, baseline),
    [items, baseline]
  )
  const ERROR_KEYS: Record<string, string> = {
    name: 'channelMenu.invalidNameEmpty',
    action: 'channelMenu.invalidMissingAction',
    open_case: 'channelMenu.invalidOpenCaseEmpty',
    root_count: 'channelMenu.invalidMainKeyboard',
    nested: 'channelMenu.invalidNested',
  }
  const firstError = validation.errors[0]
  const firstErrorKey = firstError ? ERROR_KEYS[firstError.message] : undefined

  const saveMutation = useMutation({
    mutationFn: () => putChannelMenu(channelId, items.map(normalizeNode)),
    onSuccess: () => {
      setDraft(null)
      void queryClient.invalidateQueries({
        queryKey: queryKeys.channels.menu(channelId),
      })
      toast.success(t('common.successSaved'))
    },
  })

  function updateNode(id: string, patch: Partial<MenuNode>) {
    setDraft((prev) => updateNodeInTree(prev ?? items, id, patch))
  }

  function addRootItem() {
    const child = emptyNode()
    setDraft((prev) => [...(prev ?? items), child])
    setSelectedId(child.id)
  }

  function addChildItem(parentId: string) {
    const child = emptyNode()
    setDraft((prev) => addChildToTree(prev ?? items, parentId, child))
    setSelectedId(child.id)
  }

  function removeSelected() {
    if (!effectiveSelectedId || countNodes(items) <= 1) return
    const id = effectiveSelectedId
    setDraft((prev) => {
      const base = prev ?? items
      const next = removeNodeFromTree(base, id)
      const remaining = flattenTree(next)
      setSelectedId(remaining[0]?.node.id ?? null)
      return next
    })
  }

  function setRootColumns(columns: number) {
    setDraft((prev) => {
      const base = prev ?? items
      const first = base.find((it) => it.enabled) ?? base[0]
      if (!first) return prev
      return updateNodeInTree(base, first.id, {
        render_override: { ...(first.render_override ?? {}), columns },
      })
    })
  }

  function save() {
    if (!validation.ok) return
    saveMutation.mutate()
  }

  if (menuQuery.isLoading) {
    return (
      <div className='min-h-0 flex-1 overflow-hidden rounded-md border p-4'>
        <LoadingSkeleton rows={8} />
      </div>
    )
  }

  if (menuQuery.isError) {
    return (
      <div className='min-h-0 flex-1 space-y-3 overflow-auto rounded-md border p-4'>
        <ErrorBanner
          message={errorMessage(menuQuery.error) ?? t('common.errorGeneric')}
          onRetry={() => void menuQuery.refetch()}
        />
      </div>
    )
  }

  return (
    <div
      className='flex min-h-0 flex-1 flex-col gap-3'
      data-testid='channel-menu-editor'
    >
      <div className='flex shrink-0 items-center justify-between gap-3'>
        <div className='min-w-0'>
          <h2 className='text-sm font-semibold'>{t('channelMenu.title')}</h2>
          <p className='truncate text-xs text-muted-foreground'>
            {dirtyCount > 0
              ? t('channelMenu.unsavedCount', { count: dirtyCount })
              : updatedAt
                ? `${t('channelMenu.updatedAt')}: ${updatedAt}`
                : null}
          </p>
        </div>
        <div className='flex shrink-0 items-center gap-2'>
          <Button
            type='button'
            size='sm'
            variant='outline'
            onClick={addRootItem}
          >
            {t('channelMenu.addItem')}
          </Button>
          <Button
            type='button'
            size='sm'
            onClick={save}
            disabled={saveMutation.isPending}
          >
            {t('common.save')}
          </Button>
        </div>
      </div>

      {firstErrorKey ? (
        <div
          role='alert'
          className='rounded-md border border-red-600/30 bg-red-50 px-3 py-2 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400'
        >
          {t(firstErrorKey)}
        </div>
      ) : null}

      {saveMutation.isError ? (
        <ErrorBanner
          message={errorMessage(saveMutation.error) ?? t('common.errorGeneric')}
        />
      ) : null}

      <div className='grid min-h-0 flex-1 gap-3 md:grid-cols-[minmax(0,1fr)_400px]'>
        <div className='min-h-0 overflow-auto rounded-md border p-4'>
          <PhoneSimulation
            items={items}
            selectedId={effectiveSelectedId}
            onSelect={setSelectedId}
          />
        </div>
        <div className='min-h-0 overflow-auto rounded-md border p-4'>
          {selected && effectiveSelectedId ? (
            <NodeConfigPanel
              node={selected}
              capabilities={capsQuery.data ?? []}
              caseOptions={casesQuery.data ?? []}
              rootColumns={rootColumns}
              onUpdate={(patch) => updateNode(effectiveSelectedId, patch)}
              onColumnsChange={setRootColumns}
              onAddChild={() => addChildItem(effectiveSelectedId)}
              onRemove={removeSelected}
            />
          ) : (
            <EmptyState message={t('common.selectItem')} />
          )}
        </div>
      </div>
    </div>
  )
}
