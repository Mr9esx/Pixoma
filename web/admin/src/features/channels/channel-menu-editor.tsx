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
  getChannelMenu,
  putChannelMenu,
  type MenuKind,
  type MenuNode,
} from '@/lib/api/channel-menu'
import { cn } from '@/lib/utils'

type FlatNode = { node: MenuNode; depth: number }

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function findParentNode(nodes: MenuNode[], id: string): MenuNode | null {
  for (const node of nodes) {
    if (node.children?.some((child) => child.id === id)) {
      return node
    }
    if (node.children?.length) {
      const found = findParentNode(node.children, id)
      if (found) return found
    }
  }
  return null
}

function emptyNode(kind: MenuKind = 'placeholder'): MenuNode {
  return {
    id: `btn-${Date.now()}`,
    label: '',
    order: 0,
    enabled: true,
    kind,
    ...(kind === 'folder' ? { children: [], case_ids: [] } : {}),
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
      order: node.order,
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
    order: Number(node.order) || 0,
    enabled: Boolean(node.enabled),
    kind: node.kind,
  }

  switch (node.kind) {
    case 'folder': {
      const intro = node.intro_text?.trim()
      if (intro) next.intro_text = intro
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

function ancestorIds(nodes: MenuNode[], targetId: string, trail: string[] = []): string[] | null {
  for (const node of nodes) {
    if (node.id === targetId) return trail
    if (node.children?.length) {
      const found = ancestorIds(node.children, targetId, [...trail, node.id])
      if (found) return found
    }
  }
  return null
}

function visibleTreeRows(
  nodes: MenuNode[],
  expanded: Set<string>,
  depth = 0,
): FlatNode[] {
  const out: FlatNode[] = []
  for (const node of nodes) {
    out.push({ node, depth })
    const kids = node.children ?? []
    if (node.kind === 'folder' && kids.length > 0 && expanded.has(node.id)) {
      out.push(...visibleTreeRows(kids, expanded, depth + 1))
    }
  }
  return out
}

function kindLabelKey(kind: MenuKind): string {
  switch (kind) {
    case 'folder':
      return 'channelMenu.kindFolder'
    case 'open_case':
      return 'channelMenu.kindOpenCase'
    case 'placeholder':
      return 'channelMenu.kindPlaceholder'
    case 'reply_media':
      return 'channelMenu.kindReplyMedia'
  }
}

export function ChannelMenuEditor({ channelId }: { channelId: string }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [items, setItems] = useState<MenuNode[]>([])
  const [updatedAt, setUpdatedAt] = useState('')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [expandedIds, setExpandedIds] = useState<Set<string>>(() => new Set())

  const menuQuery = useQuery({
    queryKey: queryKeys.channels.menu(channelId),
    queryFn: () => getChannelMenu(channelId),
  })

  const casesQuery = useQuery({
    queryKey: [...queryKeys.cases.all, { limit: 200 }] as const,
    queryFn: () => listCases({ limit: 200 }),
  })

  const visibleNodes = useMemo(
    () => visibleTreeRows(items, expandedIds),
    [items, expandedIds],
  )
  const selected = selectedId ? findNode(items, selectedId) : null
  const parentOfSelected =
    selectedId != null ? findParentNode(items, selectedId) : null
  const parentIsFolder = parentOfSelected?.kind === 'folder'
  const isRootItem = parentOfSelected == null

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

  useEffect(() => {
    if (!selectedId) return
    const ancestors = ancestorIds(items, selectedId)
    if (!ancestors) return
    setExpandedIds((prev) => {
      const next = new Set(prev)
      for (const id of ancestors) next.add(id)
      return next
    })
  }, [selectedId, items])

  const saveMutation = useMutation({
    mutationFn: () => putChannelMenu(channelId, items.map(normalizeNode)),
    onSuccess: (doc) => {
      setItems(structuredClone(doc.items))
      setUpdatedAt(doc.updated_at)
      void queryClient.invalidateQueries({ queryKey: queryKeys.channels.menu(channelId) })
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
    const parent = findNode(items, parentId)
    const child = emptyNode(parent?.kind === 'folder' ? 'folder' : 'placeholder')
    setItems((prev) => addChildToTree(prev, parentId, child))
    setExpandedIds((prev) => new Set(prev).add(parentId))
    setSelectedId(child.id)
  }

  function toggleExpanded(id: string) {
    setExpandedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
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
      data-testid='channel-menu-editor'
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
                <h2 className='text-sm font-semibold'>{t('channelMenu.listTitle')}</h2>
                {updatedAt ? (
                  <p className='truncate text-xs text-muted-foreground'>
                    {t('channelMenu.updatedAt')}: {updatedAt}
                  </p>
                ) : null}
              </div>
              <div className='flex shrink-0 items-center gap-2'>
                <Button type='button' size='sm' variant='outline' onClick={addRootItem}>
                  {t('channelMenu.addItem')}
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

            {visibleNodes.length === 0 ? (
              <EmptyState message={t('common.empty')} />
            ) : (
              <ul className='min-h-0 flex-1 overflow-auto'>
                {visibleNodes.map(({ node, depth }) => {
                  const active = selectedId === node.id
                  const hasKids =
                    node.kind === 'folder' && (node.children?.length ?? 0) > 0
                  const expanded = expandedIds.has(node.id)
                  return (
                    <li key={node.id}>
                      <div
                        className={cn(
                          'flex w-full items-stretch border-b transition-colors',
                          active ? 'bg-muted' : 'hover:bg-muted/50',
                        )}
                        style={{ paddingLeft: `${8 + depth * 16}px` }}
                      >
                        {hasKids ? (
                          <button
                            type='button'
                            className='w-8 shrink-0 text-xs text-muted-foreground'
                            aria-label={expanded ? 'collapse' : 'expand'}
                            onClick={() => toggleExpanded(node.id)}
                          >
                            {expanded ? '▾' : '▸'}
                          </button>
                        ) : (
                          <span className='w-8 shrink-0' />
                        )}
                        <button
                          type='button'
                          onClick={() => setSelectedId(node.id)}
                          className='min-w-0 flex-1 py-3 pr-4 text-left'
                        >
                          <div className='flex items-center justify-between gap-2'>
                            <span className='truncate text-sm font-medium'>
                              {node.kind === 'folder' ? '📁 ' : ''}
                              {node.label.trim() || t('channelMenu.untitled')}
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
                                ? t('channelMenu.enabledShort')
                                : t('channelMenu.disabledShort')}
                            </span>
                          </div>
                          <p className='mt-1 truncate text-xs text-muted-foreground'>
                            {t(kindLabelKey(node.kind))}
                            {depth === 0 ? ` · #${node.order}` : ''}
                          </p>
                        </button>
                      </div>
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
              parentIsFolder={parentIsFolder}
              isRoot={isRootItem}
              onUpdate={updateNode}
              onKind={changeKind}
              onAddChild={addChildItem}
              onSelectChild={setSelectedId}
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
  parentIsFolder,
  isRoot,
  onUpdate,
  onKind,
  onAddChild,
  onSelectChild,
  onRemove,
}: {
  node: MenuNode
  canRemove: boolean
  caseOptions: { id: string; name: string }[]
  parentIsFolder: boolean
  isRoot: boolean
  onUpdate: (id: string, patch: Partial<MenuNode>) => void
  onKind: (id: string, kind: MenuKind) => void
  onAddChild: (parentId: string) => void
  onSelectChild: (id: string) => void
  onRemove: () => void
}) {
  const { t } = useTranslation()
  const selectedCaseIds = node.case_ids ?? []
  const children = node.children ?? []

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
    <div className='space-y-4' data-testid='channel-menu-item-editor'>
      <div className='flex items-start justify-between gap-3'>
        <div>
          <h2 className='text-lg font-semibold'>
            {node.label.trim() || t('channelMenu.untitled')}
          </h2>
          <p className='text-sm text-muted-foreground'>{t('channelMenu.detailHeading')}</p>
        </div>
        <div className='flex shrink-0 items-center gap-2'>
          {node.kind === 'folder' ? (
            <Button
              type='button'
              variant='outline'
              size='sm'
              onClick={() => onAddChild(node.id)}
            >
              {t('channelMenu.addChild')}
            </Button>
          ) : null}
          <Button
            type='button'
            variant='ghost'
            size='sm'
            disabled={!canRemove}
            onClick={onRemove}
          >
            {t('channelMenu.removeItem')}
          </Button>
        </div>
      </div>

      <div className='grid gap-3 sm:grid-cols-2'>
        <div className='space-y-1.5'>
          <Label htmlFor={`channel-menu-id-${node.id}`}>{t('channelMenu.fieldId')}</Label>
          <Input
            id={`channel-menu-id-${node.id}`}
            value={node.id}
            onChange={(e) => onUpdate(node.id, { id: e.target.value })}
            autoComplete='off'
          />
        </div>
        <div className='space-y-1.5'>
          <Label htmlFor={`channel-menu-label-${node.id}`}>
            {t('channelMenu.fieldLabel')}
          </Label>
          <Input
            id={`channel-menu-label-${node.id}`}
            value={node.label}
            onChange={(e) => onUpdate(node.id, { label: e.target.value })}
            autoComplete='off'
          />
        </div>
        {isRoot ? (
          <>
            <div className='space-y-1.5'>
              <Label htmlFor={`channel-menu-order-${node.id}`}>{t('channelMenu.fieldOrder')}</Label>
              <Input
                id={`channel-menu-order-${node.id}`}
                type='number'
                value={node.order}
                onChange={(e) => onUpdate(node.id, { order: Number(e.target.value) })}
              />
            </div>
          </>
        ) : null}
      </div>

      <div className='flex flex-wrap items-end gap-4'>
        <div className='flex items-center justify-between gap-3 rounded-md border px-3 py-2'>
          <Label htmlFor={`channel-menu-enabled-${node.id}`}>
            {t('channelMenu.fieldEnabled')}
          </Label>
          <Switch
            id={`channel-menu-enabled-${node.id}`}
            checked={node.enabled}
            onCheckedChange={(checked) => onUpdate(node.id, { enabled: checked })}
          />
        </div>
        <div className='min-w-56 flex-1 space-y-1.5'>
          <Label>{t('channelMenu.fieldKind')}</Label>
          {parentIsFolder ? (
            <p className='text-sm text-muted-foreground'>{t('channelMenu.kindFolder')}</p>
          ) : (
            <Select
              value={node.kind}
              onValueChange={(value) => onKind(node.id, value as MenuKind)}
            >
              <SelectTrigger className='w-full'>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='folder'>{t('channelMenu.kindFolder')}</SelectItem>
                <SelectItem value='open_case'>{t('channelMenu.kindOpenCase')}</SelectItem>
                <SelectItem value='placeholder'>
                  {t('channelMenu.kindPlaceholder')}
                </SelectItem>
                <SelectItem value='reply_media'>
                  {t('channelMenu.kindReplyMedia')}
                </SelectItem>
              </SelectContent>
            </Select>
          )}
        </div>
      </div>

      {node.kind === 'folder' ? (
        <div className='space-y-1.5'>
          <Label htmlFor={`channel-menu-intro-${node.id}`}>{t('channelMenu.fieldIntro')}</Label>
          <p className='text-xs text-muted-foreground'>{t('channelMenu.fieldIntroHint')}</p>
          <Textarea
            id={`channel-menu-intro-${node.id}`}
            value={node.intro_text ?? ''}
            onChange={(e) => onUpdate(node.id, { intro_text: e.target.value })}
            rows={4}
          />
        </div>
      ) : null}

      {node.kind === 'folder' || node.kind === 'open_case' ? (
        <div className='space-y-2'>
          <Label>
            {node.kind === 'folder'
              ? t('channelMenu.fieldCaseIds')
              : t('channelMenu.fieldCaseId')}
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

      {node.kind === 'folder' ? (
        <div className='space-y-2'>
          <Label>{t('channelMenu.fieldChildren')}</Label>
          {children.length === 0 ? (
            <p className='text-sm text-muted-foreground'>{t('channelMenu.noChildren')}</p>
          ) : (
            <ul className='space-y-1 rounded-md border p-2'>
              {children.map((child) => (
                <li key={child.id}>
                  <button
                    type='button'
                    className='w-full rounded-sm px-2 py-1.5 text-left text-sm hover:bg-muted'
                    onClick={() => onSelectChild(child.id)}
                  >
                    📁 {child.label.trim() || t('channelMenu.untitled')}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      ) : null}

      {node.kind === 'placeholder' ? (
        <div className='max-w-xl space-y-1.5'>
          <Label htmlFor={`channel-menu-placeholder-${node.id}`}>
            {t('channelMenu.fieldPlaceholderText')}
          </Label>
          <Input
            id={`channel-menu-placeholder-${node.id}`}
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
            <Label htmlFor={`channel-menu-reply-text-${node.id}`}>
              {t('channelMenu.fieldReplyText')}
            </Label>
            <Textarea
              id={`channel-menu-reply-text-${node.id}`}
              value={node.reply?.text ?? ''}
              onChange={(e) =>
                onUpdate(node.id, {
                  reply: { ...node.reply, text: e.target.value },
                })
              }
            />
          </div>
          <div className='space-y-1.5'>
            <Label htmlFor={`channel-menu-reply-images-${node.id}`}>
              {t('channelMenu.fieldReplyImages')}
            </Label>
            <Textarea
              id={`channel-menu-reply-images-${node.id}`}
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
