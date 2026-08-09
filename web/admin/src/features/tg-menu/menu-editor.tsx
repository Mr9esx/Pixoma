import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { EmptyState } from '@/components/feedback/empty-state'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { MasterDetailShell } from '@/components/master-detail/master-detail-shell'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { listCases } from '@/lib/api/cases'
import { queryKeys } from '@/lib/api/query-keys'
import {
  getTgMenu,
  putTgMenu,
  type MenuKind,
  type MenuNode,
} from '@/lib/api/tg-menu'
import { cn } from '@/lib/utils'

type FlatNode = { node: MenuNode; depth: number }

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function emptyNode(): MenuNode {
  return {
    id: `btn-${Date.now()}`,
    label: '',
    row: 0,
    col: 0,
    enabled: true,
    kind: 'placeholder',
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
  patch: Partial<MenuNode>,
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

function setNodeKind(nodes: MenuNode[], id: string, kind: MenuKind): MenuNode[] {
  return nodes.map((node) => {
    if (node.id !== id) {
      if (node.children?.length) {
        return { ...node, children: setNodeKind(node.children, id, kind) }
      }
      return node
    }

    const next: MenuNode = {
      id: node.id,
      label: node.label,
      row: node.row,
      col: node.col,
      enabled: node.enabled,
      kind,
    }

    if (kind === 'folder') {
      next.children = node.children ?? []
      next.case_ids = node.case_ids ?? []
    } else if (kind === 'open_case') {
      next.case_ids = node.case_ids?.length ? [node.case_ids[0]] : []
    } else if (kind === 'reply_media') {
      next.reply = { text: '', images: [] }
    } else if (kind === 'list_cases_by_tag') {
      next.tag = node.tag ?? ''
    } else if (kind === 'placeholder') {
      next.placeholder_text = node.placeholder_text ?? ''
    }

    return next
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
  child: MenuNode,
): MenuNode[] {
  return nodes.map((node) => {
    if (node.id === parentId) {
      return { ...node, children: [...(node.children ?? []), child] }
    }
    if (node.children?.length) {
      return { ...node, children: addChildToTree(node.children, parentId, child) }
    }
    return node
  })
}

function normalizeNode(node: MenuNode): MenuNode {
  const next: MenuNode = {
    id: node.id,
    label: node.label,
    row: Number(node.row) || 0,
    col: Number(node.col) || 0,
    enabled: Boolean(node.enabled),
    kind: node.kind,
  }

  switch (node.kind) {
    case 'folder': {
      const caseIds = (node.case_ids ?? []).map((id) => id.trim()).filter(Boolean)
      if (caseIds.length) next.case_ids = caseIds
      if (node.children?.length) {
        next.children = node.children.map(normalizeNode)
      }
      break
    }
    case 'open_case': {
      const caseId = node.case_ids?.[0]?.trim()
      if (caseId) next.case_ids = [caseId]
      break
    }
    case 'list_cases_by_tag':
      next.tag = node.tag?.trim() || undefined
      break
    case 'placeholder':
      next.placeholder_text = node.placeholder_text?.trim() || undefined
      break
    case 'reply_media': {
      const text = node.reply?.text?.trim() || undefined
      const images = (node.reply?.images ?? [])
        .map((url) => url.trim())
        .filter(Boolean)
      next.reply = { text, images: images.length ? images : undefined }
      break
    }
  }

  return next
}

function kindLabelKey(kind: MenuKind): string {
  switch (kind) {
    case 'folder':
      return 'tgMenu.kindFolder'
    case 'open_case':
      return 'tgMenu.kindOpenCase'
    case 'list_cases_by_tag':
      return 'tgMenu.kindListByTag'
    case 'placeholder':
      return 'tgMenu.kindPlaceholder'
    case 'reply_media':
      return 'tgMenu.kindReplyMedia'
  }
}

export function TgMenuEditor() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [items, setItems] = useState<MenuNode[]>([])
  const [updatedAt, setUpdatedAt] = useState('')
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const menuQuery = useQuery({
    queryKey: queryKeys.tgMenu.all,
    queryFn: getTgMenu,
  })

  const casesQuery = useQuery({
    queryKey: [...queryKeys.cases.all, { limit: 200 }] as const,
    queryFn: () => listCases({ limit: 200 }),
  })

  const flatNodes = useMemo(() => flattenTree(items), [items])
  const selected = selectedId ? findNode(items, selectedId) : null

  useEffect(() => {
    if (!menuQuery.data) return
    setItems(structuredClone(menuQuery.data.items))
    setUpdatedAt(menuQuery.data.updated_at)
    setSelectedId((prev) => {
      if (menuQuery.data.items.length === 0) return null
      if (prev && findNode(menuQuery.data.items, prev)) return prev
      return menuQuery.data.items[0]?.id ?? null
    })
  }, [menuQuery.data])

  const saveMutation = useMutation({
    mutationFn: () => putTgMenu(items.map(normalizeNode)),
    onSuccess: (doc) => {
      setItems(structuredClone(doc.items))
      setUpdatedAt(doc.updated_at)
      void queryClient.invalidateQueries({ queryKey: queryKeys.tgMenu.all })
      toast.success(t('common.successSaved'))
    },
  })

  function updateNode(id: string, patch: Partial<MenuNode>) {
    setItems((prev) => updateNodeInTree(prev, id, patch))
  }

  function changeKind(id: string, kind: MenuKind) {
    setItems((prev) => setNodeKind(prev, id, kind))
  }

  function addRootItem() {
    const child = emptyNode()
    setItems((prev) => [...prev, child])
    setSelectedId(child.id)
  }

  function addChildItem(parentId: string) {
    const child = emptyNode()
    setItems((prev) => addChildToTree(prev, parentId, child))
    setSelectedId(child.id)
  }

  function removeSelected() {
    if (!selectedId || countNodes(items) <= 1) return
    const id = selectedId
    setItems((prev) => {
      const next = removeNodeFromTree(prev, id)
      const remaining = flattenTree(next)
      setSelectedId(remaining[0]?.node.id ?? null)
      return next
    })
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
      data-testid='tg-menu-editor'
    >
      {saveMutation.isError ? (
        <ErrorBanner
          message={errorMessage(saveMutation.error) ?? t('common.errorGeneric')}
        />
      ) : null}

      <MasterDetailShell
        hasSelection={selected != null}
        onBackToList={() => setSelectedId(null)}
        list={
          <div className='flex h-full min-h-0 flex-col'>
            <div className='flex items-center justify-between gap-2 border-b px-4 py-3'>
              <div className='min-w-0'>
                <h2 className='text-sm font-semibold'>{t('tgMenu.listTitle')}</h2>
                {updatedAt ? (
                  <p className='truncate text-xs text-muted-foreground'>
                    {t('tgMenu.updatedAt')}: {updatedAt}
                  </p>
                ) : null}
              </div>
              <div className='flex shrink-0 items-center gap-2'>
                <Button type='button' size='sm' variant='outline' onClick={addRootItem}>
                  {t('tgMenu.addItem')}
                </Button>
                <Button
                  type='button'
                  size='sm'
                  onClick={() => saveMutation.mutate()}
                  disabled={saveMutation.isPending}
                >
                  {t('common.save')}
                </Button>
              </div>
            </div>

            {flatNodes.length === 0 ? (
              <EmptyState message={t('common.empty')} />
            ) : (
              <ul className='min-h-0 flex-1 overflow-auto'>
                {flatNodes.map(({ node, depth }) => {
                  const active = selectedId === node.id
                  return (
                    <li key={node.id}>
                      <button
                        type='button'
                        onClick={() => setSelectedId(node.id)}
                        className={cn(
                          'block w-full border-b px-4 py-3 text-left transition-colors',
                          active ? 'bg-muted' : 'hover:bg-muted/50',
                        )}
                        style={{ paddingLeft: `${16 + depth * 16}px` }}
                      >
                        <div className='flex items-center justify-between gap-2'>
                          <span className='truncate text-sm font-medium'>
                            {node.label.trim() || t('tgMenu.untitled')}
                          </span>
                          <span
                            className={cn(
                              'shrink-0 rounded-sm px-1.5 py-0.5 text-[10px]',
                              node.enabled
                                ? 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-400'
                                : 'bg-muted text-muted-foreground',
                            )}
                          >
                            {node.enabled
                              ? t('tgMenu.enabledShort')
                              : t('tgMenu.disabledShort')}
                          </span>
                        </div>
                        <p className='mt-1 truncate text-xs text-muted-foreground'>
                          {t(kindLabelKey(node.kind))} · r{node.row}/c{node.col}
                        </p>
                      </button>
                    </li>
                  )
                })}
              </ul>
            )}
          </div>
        }
        detail={
          selected && selectedId ? (
            <NodeEditor
              node={selected}
              canRemove={countNodes(items) > 1}
              caseOptions={casesQuery.data ?? []}
              onUpdate={updateNode}
              onKind={changeKind}
              onAddChild={addChildItem}
              onRemove={removeSelected}
            />
          ) : (
            <EmptyState message={t('common.selectItem')} />
          )
        }
      />
    </div>
  )
}

function NodeEditor({
  node,
  canRemove,
  caseOptions,
  onUpdate,
  onKind,
  onAddChild,
  onRemove,
}: {
  node: MenuNode
  canRemove: boolean
  caseOptions: { id: string; name: string }[]
  onUpdate: (id: string, patch: Partial<MenuNode>) => void
  onKind: (id: string, kind: MenuKind) => void
  onAddChild: (parentId: string) => void
  onRemove: () => void
}) {
  const { t } = useTranslation()
  const selectedCaseIds = node.case_ids ?? []

  function toggleCaseId(caseId: string, checked: boolean) {
    if (node.kind === 'open_case') {
      onUpdate(node.id, { case_ids: checked ? [caseId] : [] })
      return
    }
    const next = new Set(selectedCaseIds)
    if (checked) next.add(caseId)
    else next.delete(caseId)
    onUpdate(node.id, { case_ids: [...next] })
  }

  return (
    <div className='space-y-4' data-testid='tg-menu-item-editor'>
      <div className='flex items-start justify-between gap-3'>
        <div>
          <h2 className='text-lg font-semibold'>
            {node.label.trim() || t('tgMenu.untitled')}
          </h2>
          <p className='text-sm text-muted-foreground'>{t('tgMenu.detailHeading')}</p>
        </div>
        <div className='flex shrink-0 items-center gap-2'>
          {node.kind === 'folder' ? (
            <Button
              type='button'
              variant='outline'
              size='sm'
              onClick={() => onAddChild(node.id)}
            >
              {t('tgMenu.addChild')}
            </Button>
          ) : null}
          <Button
            type='button'
            variant='ghost'
            size='sm'
            disabled={!canRemove}
            onClick={onRemove}
          >
            {t('tgMenu.removeItem')}
          </Button>
        </div>
      </div>

      <div className='grid gap-3 sm:grid-cols-2'>
        <div className='space-y-1.5'>
          <Label htmlFor={`tg-menu-id-${node.id}`}>{t('tgMenu.fieldId')}</Label>
          <Input
            id={`tg-menu-id-${node.id}`}
            value={node.id}
            onChange={(e) => onUpdate(node.id, { id: e.target.value })}
            autoComplete='off'
          />
        </div>
        <div className='space-y-1.5'>
          <Label htmlFor={`tg-menu-label-${node.id}`}>
            {t('tgMenu.fieldLabel')}
          </Label>
          <Input
            id={`tg-menu-label-${node.id}`}
            value={node.label}
            onChange={(e) => onUpdate(node.id, { label: e.target.value })}
            autoComplete='off'
          />
        </div>
        <div className='space-y-1.5'>
          <Label htmlFor={`tg-menu-row-${node.id}`}>{t('tgMenu.fieldRow')}</Label>
          <Input
            id={`tg-menu-row-${node.id}`}
            type='number'
            value={node.row}
            onChange={(e) => onUpdate(node.id, { row: Number(e.target.value) })}
          />
        </div>
        <div className='space-y-1.5'>
          <Label htmlFor={`tg-menu-col-${node.id}`}>{t('tgMenu.fieldCol')}</Label>
          <Input
            id={`tg-menu-col-${node.id}`}
            type='number'
            value={node.col}
            onChange={(e) => onUpdate(node.id, { col: Number(e.target.value) })}
          />
        </div>
      </div>

      <div className='flex flex-wrap items-end gap-4'>
        <div className='flex items-center justify-between gap-3 rounded-md border px-3 py-2'>
          <Label htmlFor={`tg-menu-enabled-${node.id}`}>
            {t('tgMenu.fieldEnabled')}
          </Label>
          <Switch
            id={`tg-menu-enabled-${node.id}`}
            checked={node.enabled}
            onCheckedChange={(checked) => onUpdate(node.id, { enabled: checked })}
          />
        </div>
        <div className='min-w-56 flex-1 space-y-1.5'>
          <Label>{t('tgMenu.fieldKind')}</Label>
          <Select
            value={node.kind}
            onValueChange={(value) => onKind(node.id, value as MenuKind)}
          >
            <SelectTrigger className='w-full'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='folder'>{t('tgMenu.kindFolder')}</SelectItem>
              <SelectItem value='open_case'>{t('tgMenu.kindOpenCase')}</SelectItem>
              <SelectItem value='list_cases_by_tag'>
                {t('tgMenu.kindListByTag')}
              </SelectItem>
              <SelectItem value='placeholder'>
                {t('tgMenu.kindPlaceholder')}
              </SelectItem>
              <SelectItem value='reply_media'>
                {t('tgMenu.kindReplyMedia')}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {node.kind === 'folder' || node.kind === 'open_case' ? (
        <div className='space-y-2'>
          <Label>
            {node.kind === 'folder'
              ? t('tgMenu.fieldCaseIds')
              : t('tgMenu.fieldCaseId')}
          </Label>
          <div className='max-h-48 space-y-2 overflow-auto rounded-md border p-3'>
            {caseOptions.length === 0 ? (
              <p className='text-sm text-muted-foreground'>{t('common.empty')}</p>
            ) : (
              caseOptions.map((c) => (
                <label
                  key={c.id}
                  className='flex items-center gap-2 text-sm'
                >
                  <Checkbox
                    checked={selectedCaseIds.includes(c.id)}
                    onCheckedChange={(v) => toggleCaseId(c.id, v === true)}
                  />
                  <span className='truncate'>
                    {c.name} ({c.id})
                  </span>
                </label>
              ))
            )}
          </div>
        </div>
      ) : null}

      {node.kind === 'list_cases_by_tag' ? (
        <div className='max-w-md space-y-1.5'>
          <Label htmlFor={`tg-menu-tag-${node.id}`}>{t('tgMenu.fieldTag')}</Label>
          <Input
            id={`tg-menu-tag-${node.id}`}
            value={node.tag ?? ''}
            onChange={(e) => onUpdate(node.id, { tag: e.target.value })}
            autoComplete='off'
          />
        </div>
      ) : null}

      {node.kind === 'placeholder' ? (
        <div className='max-w-xl space-y-1.5'>
          <Label htmlFor={`tg-menu-placeholder-${node.id}`}>
            {t('tgMenu.fieldPlaceholderText')}
          </Label>
          <Input
            id={`tg-menu-placeholder-${node.id}`}
            value={node.placeholder_text ?? ''}
            onChange={(e) =>
              onUpdate(node.id, { placeholder_text: e.target.value })
            }
            autoComplete='off'
          />
        </div>
      ) : null}

      {node.kind === 'reply_media' ? (
        <div className='grid gap-3 md:grid-cols-2'>
          <div className='space-y-1.5'>
            <Label htmlFor={`tg-menu-reply-text-${node.id}`}>
              {t('tgMenu.fieldReplyText')}
            </Label>
            <Textarea
              id={`tg-menu-reply-text-${node.id}`}
              value={node.reply?.text ?? ''}
              onChange={(e) =>
                onUpdate(node.id, {
                  reply: { ...node.reply, text: e.target.value },
                })
              }
            />
          </div>
          <div className='space-y-1.5'>
            <Label htmlFor={`tg-menu-reply-images-${node.id}`}>
              {t('tgMenu.fieldReplyImages')}
            </Label>
            <Textarea
              id={`tg-menu-reply-images-${node.id}`}
              value={(node.reply?.images ?? []).join('\n')}
              onChange={(e) =>
                onUpdate(node.id, {
                  reply: {
                    ...node.reply,
                    images: e.target.value.split('\n'),
                  },
                })
              }
            />
          </div>
        </div>
      ) : null}
    </div>
  )
}
