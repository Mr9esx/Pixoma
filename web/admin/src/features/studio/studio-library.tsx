import { useEffect, useRef, useState, type FormEvent, type KeyboardEvent } from 'react'
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useVirtualizer } from '@tanstack/react-virtual'
import {
  ArrowDownWideNarrow,
  ChevronDown,
  ChevronRight,
  Download,
  Folder,
  FolderOpen,
  Grid2X2,
  Library,
  List,
  MessageCircle,
  Pencil,
  Search,
  Settings2,
  SlidersHorizontal,
  Star,
  Tag,
  Upload,
  X,
} from 'lucide-react'
import { baseURL } from '@/lib/api/client'
import {
  addStudioAssetToProject,
  createStudioAssetTag,
  createStudioLibraryCategory,
  deleteStudioLibraryCategory,
  exportStudioProjectAssets,
  getStudioLibraryPreferences,
  getStudioProjectAsset,
  getStudioProjectAssetDuplicates,
  listStudioAssetTags,
  listStudioLibraryAssets,
  listStudioLibraryCategories,
  listStudioLibraryFormats,
  listStudioLibraryProjects,
  listStudioLibraryTree,
  listStudioSessions,
  referenceStudioAsset,
  retryStudioAssetPalette,
  studioLibraryDateBoundary,
  updateStudioLibraryCategory,
  updateStudioLibraryPreferences,
  updateStudioProjectAsset,
  updateStudioProjectAssetTags,
  updateStudioProjectAssetVersion,
  uploadStudioAsset,
  updateStudioProjectAssetsBatch,
  type StudioAsset,
  type StudioLibraryProject,
  type StudioLibraryCategory,
  type StudioLibraryTreeMode,
  type StudioLibraryTreeNode,
  type StudioProjectAsset,
} from '@/lib/api/studio'
import { cn } from '@/lib/utils'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Empty, EmptyContent, EmptyHeader, EmptyTitle } from '@/components/ui/empty'
import { IconButtonTooltip } from '@/components/ui/icon-button-tooltip'
import { Input } from '@/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from '@/components/ui/resizable'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { AssetCard, AssetDetailsDialog, AssetDetailsInfo, AssetInfoRow, AssetKindIcon, AssetPaletteRow, AssetSizeRow } from './studio-assets'

const treeWidthStorageKey = 'studio.library.treeWidth'
const treeMinWidth = 200
const treeMaxWidth = 420
const treeContentClassName = 'overflow-hidden duration-150 ease-[cubic-bezier(0.22,1,0.36,1)] data-[state=open]:animate-collapsible-down data-[state=closed]:animate-collapsible-up motion-reduce:animate-none'

const treeModes: Array<{ value: StudioLibraryTreeMode; label: string }> = [
  { value: 'asset', label: '资产' },
  { value: 'session', label: '对话' },
  { value: 'category', label: '分类' },
  { value: 'format', label: '格式' },
  { value: 'rating', label: '评分' },
  { value: 'tag', label: '标签' },
]

type BatchChoice = 'category' | 'tags' | 'tags_remove' | 'rating' | 'project'

const batchLabels: Record<BatchChoice, string> = {
  category: '移动到分类',
  tags: '添加标签',
  tags_remove: '移除标签',
  rating: '设置评分',
  project: '添加到项目',
}

type AdvancedFilters = {
  widthMin: string
  widthMax: string
  heightMin: string
  heightMax: string
  sizeMin: string
  sizeMax: string
  addedFrom: string
  addedTo: string
  duplicates: boolean
}

const emptyAdvancedFilters: AdvancedFilters = {
  widthMin: '', widthMax: '', heightMin: '', heightMax: '',
  sizeMin: '', sizeMax: '', addedFrom: '', addedTo: '', duplicates: false,
}

function categoryPath(category: StudioLibraryCategory, categories: StudioLibraryCategory[]) {
  const names = [category.name]
  const visited = new Set([category.id])
  let parentId = category.parent_id
  while (parentId) {
    if (visited.has(parentId)) throw new Error('分类层级存在循环')
    visited.add(parentId)
    const parent = categories.find((item) => item.id === parentId)
    if (!parent) throw new Error('分类父级不存在')
    names.unshift(parent.name)
    parentId = parent.parent_id
  }
  return names.join(' / ')
}

function previewAsset(item: StudioProjectAsset): StudioAsset {
  return {
    ...item.asset,
    name: item.display_name,
    current_version: item.version.version,
    versions: [item.version],
  }
}

export function formatDate(value?: string | null) {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return `${date.getFullYear()}/${String(date.getMonth() + 1).padStart(2, '0')}/${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

export { formatSize } from './studio-asset-format'

export function StudioLibrary({
  onOpenSession,
  initialProjectId,
  initialSessionId,
  initialSessionTitle,
}: {
  onOpenSession: (sessionId: string, sourceRunId?: string) => void
  initialProjectId?: string
  initialSessionId?: string
  initialSessionTitle?: string
}) {
  const queryClient = useQueryClient()
  const projects = useQuery({
    queryKey: ['studio', 'library', 'projects'],
    queryFn: listStudioLibraryProjects,
  })
  const preferences = useQuery({
    queryKey: ['studio', 'library', 'preferences'],
    queryFn: getStudioLibraryPreferences,
  })
  const [projectId, setProjectId] = useState<string>()
  const [treeMode, setTreeMode] = useState<StudioLibraryTreeMode>('asset')
  const [treeFocusId, setTreeFocusId] = useState<string>()
  const [sessionFilterId, setSessionFilterId] = useState(initialSessionId)
  const [treeOpen, setTreeOpen] = useState(false)
  const [categoryId, setCategoryId] = useState('all')
  const [kind, setKind] = useState('all')
  const [format, setFormat] = useState('all')
  const [draftKind, setDraftKind] = useState('all')
  const [draftFormat, setDraftFormat] = useState('all')
  const [treeSettingsOpen, setTreeSettingsOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [search, setSearch] = useState('')
  const [sort, setSort] = useState('recent')
  const [view, setView] = useState<'grid' | 'list'>('grid')
  const [showArchived, setShowArchived] = useState(false)
  const [draftShowArchived, setDraftShowArchived] = useState(false)
  const [advancedOpen, setAdvancedOpen] = useState(false)
  const [draftFilters, setDraftFilters] = useState<AdvancedFilters>(emptyAdvancedFilters)
  const [appliedFilters, setAppliedFilters] = useState<AdvancedFilters>(emptyAdvancedFilters)
  const [filterError, setFilterError] = useState('')
  const [selectedIds, setSelectedIds] = useState<string[]>([])
  const [selectedAssetId, setSelectedAssetId] = useState<string>()
  const [categoryDialog, setCategoryDialog] = useState<'create' | 'rename'>()
  const [categoryName, setCategoryName] = useState('')
  const [categoryParentId, setCategoryParentId] = useState('')
  const [deleteCategoryOpen, setDeleteCategoryOpen] = useState(false)
  const [archiveOpen, setArchiveOpen] = useState(false)
  const [batchChoice, setBatchChoice] = useState<BatchChoice>()
  const [batchValue, setBatchValue] = useState('')
  const [columns, setColumns] = useState(1)
  const [treeDocked, setTreeDocked] = useState(() => window.matchMedia('(min-width: 1280px)').matches)
  const [treeWidth] = useState(() => {
    const stored = Number(window.localStorage.getItem(treeWidthStorageKey))
    return stored >= treeMinWidth && stored <= treeMaxWidth ? stored : 256
  })
  const fileInputRef = useRef<HTMLInputElement>(null)
  const scrollRootRef = useRef<HTMLDivElement>(null)
  const clearFilters = () => {
    setCategoryId('all')
    setKind('all')
    setFormat('all')
    setDraftKind('all')
    setDraftFormat('all')
    setQuery('')
    setSearch('')
    setTreeFocusId(undefined)
    setSessionFilterId(undefined)
    setShowArchived(false)
    setDraftShowArchived(false)
    setDraftFilters(emptyAdvancedFilters)
    setAppliedFilters(emptyAdvancedFilters)
    setFilterError('')
    setSelectedIds([])
  }
  const savePreferences = useMutation({
    mutationFn: updateStudioLibraryPreferences,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['studio', 'library', 'preferences'] }),
  })

  useEffect(() => {
    if (!preferences.data || !projects.data || projectId !== undefined) return
    const last = initialProjectId ?? preferences.data.last_project_id
    setProjectId(projects.data.some((project) => project.id === last) ? last : '')
    setTreeMode(preferences.data.tree_mode)
  }, [preferences.data, projects.data, projectId, initialProjectId])
  useEffect(() => {
    const timer = window.setTimeout(() => setSearch(query.trim()), 250)
    return () => window.clearTimeout(timer)
  }, [query])
  useEffect(() => {
    const media = window.matchMedia('(min-width: 1280px)')
    const update = () => setTreeDocked(media.matches)
    media.addEventListener('change', update)
    return () => media.removeEventListener('change', update)
  }, [])
  useEffect(() => {
    const viewport = scrollRootRef.current?.querySelector<HTMLElement>(
      '[data-slot="scroll-area-viewport"]'
    )
    if (!viewport) return
    const updateColumns = () => {
      const width = viewport.clientWidth
      setColumns(width >= 1100 ? 4 : width >= 800 ? 3 : width >= 520 ? 2 : 1)
    }
    const observer = new ResizeObserver(updateColumns)
    observer.observe(viewport)
    updateColumns()
    return () => observer.disconnect()
  }, [projectId, treeDocked])

  const categories = useQuery({
    queryKey: ['studio', 'library', 'categories', projectId],
    queryFn: () => listStudioLibraryCategories(projectId!),
    enabled: projectId !== undefined,
  })
  const formats = useQuery({
    queryKey: ['studio', 'library', 'formats', projectId],
    queryFn: () => listStudioLibraryFormats(projectId!),
    enabled: projectId !== undefined,
  })
  const tags = useQuery({
    queryKey: ['studio', 'library', 'tags'],
    queryFn: listStudioAssetTags,
    enabled: selectedIds.length > 0,
  })
  const advancedFilterCount = Object.values(appliedFilters).filter((value) => value !== '' && value !== false).length + Number(showArchived) + Number(kind !== 'all') + Number(format !== 'all')
  const assets = useInfiniteQuery({
    queryKey: [
      'studio', 'library', 'assets', projectId, categoryId, kind, format,
      search, sort, showArchived,
      sessionFilterId,
      appliedFilters,
    ],
    queryFn: ({ pageParam }) =>
      listStudioLibraryAssets({
        projectId: projectId!,
        categoryId: categoryId === 'all' ? undefined : categoryId === 'uncategorized' ? 'none' : categoryId,
        kind: kind === 'all' ? undefined : kind,
        format: format === 'all' ? undefined : format,
        search,
        sort,
        archived: showArchived,
        sessionId: sessionFilterId,
        widthMin: appliedFilters.widthMin ? Number(appliedFilters.widthMin) : undefined,
        widthMax: appliedFilters.widthMax ? Number(appliedFilters.widthMax) : undefined,
        heightMin: appliedFilters.heightMin ? Number(appliedFilters.heightMin) : undefined,
        heightMax: appliedFilters.heightMax ? Number(appliedFilters.heightMax) : undefined,
        sizeMin: appliedFilters.sizeMin ? Math.round(Number(appliedFilters.sizeMin) * 1024 ** 2) : undefined,
        sizeMax: appliedFilters.sizeMax ? Math.round(Number(appliedFilters.sizeMax) * 1024 ** 2) : undefined,
        addedFrom: appliedFilters.addedFrom ? studioLibraryDateBoundary(appliedFilters.addedFrom) : undefined,
        addedTo: appliedFilters.addedTo ? studioLibraryDateBoundary(appliedFilters.addedTo, true) : undefined,
        duplicates: appliedFilters.duplicates || undefined,
        cursor: pageParam,
        limit: 50,
      }),
    enabled: projectId !== undefined,
    initialPageParam: '',
    getNextPageParam: (page) => page.next_cursor || undefined,
  })
  const upload = useMutation({
    mutationFn: ({ file, requestId }: { file: File; requestId: string }) =>
      uploadStudioAsset(file, undefined, projectId!, requestId),
    onSuccess: (result) => {
      clearFilters()
      setSelectedAssetId(result.project_asset_id)
      void queryClient.invalidateQueries({ queryKey: ['studio', 'library'] })
    },
  })
  const createCategory = useMutation({
    mutationFn: () =>
      createStudioLibraryCategory({ projectId: projectId!, name: categoryName.trim(), parentId: categoryParentId }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['studio', 'library'] })
      setCategoryDialog(undefined)
      setCategoryName('')
      setCategoryParentId('')
    },
  })
  const renameCategory = useMutation({
    mutationFn: () =>
      updateStudioLibraryCategory(categoryId, { name: categoryName.trim(), parentId: categoryParentId }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['studio', 'library'] })
      setCategoryDialog(undefined)
      setCategoryName('')
      setCategoryParentId('')
    },
  })
  const removeCategory = useMutation({
    mutationFn: () => deleteStudioLibraryCategory(categoryId),
    onSuccess: async () => {
      setCategoryId('all')
      setDeleteCategoryOpen(false)
      await queryClient.invalidateQueries({ queryKey: ['studio', 'library'] })
    },
  })
  const batch = useMutation({
    mutationFn: (input: Omit<Parameters<typeof updateStudioProjectAssetsBatch>[0], 'projectAssetIds'>) =>
      updateStudioProjectAssetsBatch({ projectAssetIds: selectedIds, ...input }),
    onSuccess: async ({ results }) => {
      setSelectedIds(results.filter((result) => !result.success).map((result) => result.project_asset_id))
      await queryClient.invalidateQueries({ queryKey: ['studio', 'library'] })
    },
  })
  const batchExport = useMutation({
    mutationFn: () => exportStudioProjectAssets(selectedIds),
    onSuccess: ({ blob, filename }) => {
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = filename ? decodeURIComponent(filename) : 'assets.zip'
      document.body.append(link)
      link.click()
      link.remove()
      window.setTimeout(() => URL.revokeObjectURL(url), 60_000)
    },
  })

  const applyBatchChoice = () => {
    if (!batchChoice || !batchValue) return
    if (batchChoice === 'category') batch.mutate({ action: 'category', categoryId: batchValue === '__none__' ? '' : batchValue }, { onSuccess: () => setBatchChoice(undefined) })
    else if (batchChoice === 'tags' || batchChoice === 'tags_remove') batch.mutate({ action: batchChoice, tagIds: [batchValue] }, { onSuccess: () => setBatchChoice(undefined) })
    else if (batchChoice === 'rating') batch.mutate({ action: 'rating', rating: Number(batchValue) }, { onSuccess: () => setBatchChoice(undefined) })
    else batch.mutate({ action: 'project', projectId: batchValue === '__unassigned__' ? '' : batchValue }, { onSuccess: () => setBatchChoice(undefined) })
  }

  const visibleAssets = assets.data?.pages.flatMap((page) => page.items) ?? []
  const total = assets.data?.pages[0]?.total ?? 0
  const rowCount = view === 'grid'
    ? Math.ceil(visibleAssets.length / columns)
    : visibleAssets.length
  const virtualizer = useVirtualizer({
    count: rowCount,
    getScrollElement: () =>
      scrollRootRef.current?.querySelector<HTMLElement>(
        '[data-slot="scroll-area-viewport"]'
      ) ?? null,
    estimateSize: () => view === 'grid' ? 350 : 190,
    measureElement: (element) => element.getBoundingClientRect().height,
    overscan: 2,
    initialRect: { width: 0, height: 600 },
  })
  useEffect(() => virtualizer.measure(), [columns, view, virtualizer])
  useEffect(() => {
    const viewport = scrollRootRef.current?.querySelector<HTMLElement>(
      '[data-slot="scroll-area-viewport"]'
    )
    if (viewport) viewport.scrollTop = 0
  }, [projectId, categoryId, kind, format, search, sort, view, appliedFilters])

  const selectedCategory = categories.data?.find((item) => item.id === categoryId)
  const hasFilters = Boolean(search || categoryId !== 'all' || kind !== 'all' || format !== 'all' || sessionFilterId || advancedFilterCount || showArchived)
  const setProject = (id: string) => {
    const changed = id !== projectId
    setProjectId(id)
    setSessionFilterId(undefined)
    if (changed) {
      setTreeFocusId(`project:${id}`)
      setCategoryId('all')
      setFormat('all')
      setQuery('')
      setSearch('')
    }
    setSelectedAssetId(undefined)
    setSelectedIds([])
    savePreferences.mutate({ tree_mode: treeMode, last_project_id: id })
  }
  const setMode = (mode: StudioLibraryTreeMode) => {
    setTreeMode(mode)
    setTreeFocusId(undefined)
    setSelectedIds([])
    savePreferences.mutate({ tree_mode: mode, last_project_id: projectId ?? '' })
  }
  const applyAdvancedFilters = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const ranges: Array<[string, string]> = [
      [draftFilters.widthMin, draftFilters.widthMax],
      [draftFilters.heightMin, draftFilters.heightMax],
      [draftFilters.sizeMin, draftFilters.sizeMax],
      [draftFilters.addedFrom, draftFilters.addedTo],
    ]
    if (ranges.some(([min, max]) => min && max && Number.isFinite(Number(min)) && Number.isFinite(Number(max)) && Number(min) > Number(max)) ||
      (draftFilters.addedFrom && draftFilters.addedTo && draftFilters.addedFrom > draftFilters.addedTo)) {
      setFilterError('范围起点不能晚于终点')
      return
    }
    setFilterError('')
    setAppliedFilters({ ...draftFilters })
    setKind(draftKind)
    setFormat(draftFormat)
    setShowArchived(draftShowArchived)
    setSelectedIds([])
    setAdvancedOpen(false)
  }
  const projectTree = (
    <div className='flex w-full min-w-0 flex-col gap-1 p-3' role='tree' aria-label='项目资产树' onKeyDown={(event) => {
      if (!['ArrowUp', 'ArrowDown', 'Home', 'End'].includes(event.key)) return
      const targets = Array.from(event.currentTarget.querySelectorAll<HTMLButtonElement>('[data-tree-focus-target]')).filter((target) => !target.closest('[data-slot="collapsible-content"][data-state="closed"]'))
      const current = (event.target as HTMLElement).closest('[role="treeitem"]')?.querySelector<HTMLButtonElement>('[data-tree-focus-target]')
      if (!current || !targets.length) return
      const index = targets.indexOf(current)
      const nextIndex = event.key === 'Home' ? 0 : event.key === 'End' ? targets.length - 1 : Math.max(0, Math.min(targets.length - 1, index + (event.key === 'ArrowDown' ? 1 : -1)))
      event.preventDefault()
      targets[nextIndex]?.focus()
    }}>
      {(projects.data ?? []).map((project) => (
        <ProjectTreeBranch
          key={project.id || 'unassigned'}
          project={project}
          mode={treeMode}
          selectedProjectId={projectId}
          treeFocusId={treeFocusId}
          onFocusTreeItem={setTreeFocusId}
          isFirstProject={project.id === projects.data?.[0]?.id}
          onProject={setProject}
          onAsset={(id, node) => {
            setProject(id)
            if (!node.asset_id) throw new Error('资产节点缺少 asset_id')
            setSelectedAssetId(node.asset_id)
            setTreeFocusId(`node:${id}:${node.id}`)
            setTreeOpen(false)
          }}
        />
      ))}
      {projects.isError ? <Button variant='ghost' onClick={() => void projects.refetch()}>重试读取项目</Button> : null}
    </div>
  )
  const treeSettings = (
    <Popover open={treeSettingsOpen} onOpenChange={setTreeSettingsOpen}>
      <IconButtonTooltip label='设置文件树'>
        <PopoverTrigger asChild>
          <Button variant='ghost' size='icon-sm' aria-label='设置文件树'><Settings2 /></Button>
        </PopoverTrigger>
      </IconButtonTooltip>
      <PopoverContent align='end' className='w-52 p-1.5'>
        <h2 className='px-2 py-1.5 text-sm font-medium'>项目下显示</h2>
        <RadioGroup aria-label='项目下显示' value={treeMode} onValueChange={(value) => { setMode(value as StudioLibraryTreeMode); setTreeSettingsOpen(false) }} className='gap-0'>
          {treeModes.map((mode) => <div key={mode.value} className='flex h-8 items-center gap-2 rounded-sm px-2 hover:bg-accent focus-within:bg-accent'>
            <RadioGroupItem id={`library-tree-mode-${mode.value}`} value={mode.value} className='size-3.5' />
            <label htmlFor={`library-tree-mode-${mode.value}`} className='min-w-0 flex-1 cursor-pointer truncate text-sm'>{mode.label}</label>
          </div>)}
        </RadioGroup>
      </PopoverContent>
    </Popover>
  )

  return (
    <main id='main-content' className='min-h-0 min-w-0 flex-1 py-3 pr-3 sm:py-4 sm:pr-4'>
      <section className='flex h-full min-h-0 flex-col overflow-hidden rounded-2xl border bg-card'>
          <input ref={fileInputRef} type='file' aria-label='选择上传资产' className='sr-only' onChange={(event) => {
            const file = event.target.files?.[0]
            if (file && projectId !== undefined) upload.mutate({ file, requestId: crypto.randomUUID() })
            event.currentTarget.value = ''
          }} />
        <ResizablePanelGroup orientation='horizontal' className='min-h-0 flex-1'>
          {treeDocked ? <>
            <ResizablePanel
              id='library-tree'
              defaultSize={treeWidth}
              minSize={treeMinWidth}
              maxSize={treeMaxWidth}
              groupResizeBehavior='preserve-pixel-size'
              className='min-w-0'
              onResize={({ inPixels }) => window.localStorage.setItem(treeWidthStorageKey, String(Math.round(inPixels)))}
            >
              <aside className='flex h-full min-w-0 flex-col bg-muted/20'>
                <div className='flex h-14 shrink-0 items-center justify-between border-b px-4'>
                  <h1 className='text-sm font-semibold'>资产库</h1>{treeSettings}
                </div>
                <div className='studio-scrollbar min-h-0 w-full min-w-0 flex-1 overflow-x-hidden overflow-y-auto'>{projectTree}</div>
              </aside>
            </ResizablePanel>
            <ResizableHandle withHandle aria-label='调整文件树宽度' className='after:w-3' />
          </> : null}
          <ResizablePanel id='library-content' minSize={treeDocked ? 400 : 0} className='min-w-0'>
          <div className='relative flex h-full min-w-0 flex-col'>
            <div data-slot='studio-library-toolbar' className='flex min-h-14 flex-wrap items-center gap-2 border-b px-4 py-2 lg:px-5'>
              <Button variant='outline' size='sm' className='xl:hidden' aria-label='打开项目资产树' onClick={() => setTreeOpen(true)}><Folder />项目</Button>
              <div className='relative min-w-44 flex-1 sm:max-w-sm'>
                <Search className='absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground' />
                <Input value={query} onChange={(event) => { setQuery(event.target.value); setSelectedIds([]) }} placeholder='搜索名称或标签' aria-label='搜索名称或标签' className='pl-9' />
              </div>
              <div className='flex shrink-0 items-center gap-1'>
                <Popover open={advancedOpen} onOpenChange={(open) => {
                  if (open) {
                    setDraftFilters(appliedFilters)
                    setDraftKind(kind)
                    setDraftFormat(format)
                    setDraftShowArchived(showArchived)
                    setFilterError('')
                  }
                  setAdvancedOpen(open)
                }}>
                  <IconButtonTooltip label={`筛选${advancedFilterCount ? ` · ${advancedFilterCount}` : ''}`}><PopoverTrigger asChild><Button variant='ghost' size='icon' aria-label={`筛选资产${advancedFilterCount ? `，已应用 ${advancedFilterCount} 项` : ''}`}><SlidersHorizontal /></Button></PopoverTrigger></IconButtonTooltip>
                  <PopoverContent align='end' className='max-h-[min(75svh,640px)] w-80 overflow-y-auto p-4'>
                    <form className='flex flex-col gap-4' onSubmit={applyAdvancedFilters}>
                      <h2 className='text-sm font-medium'>筛选资产</h2>
                      <div className='grid grid-cols-2 gap-2'>
                        <div className='flex flex-col gap-1.5'><span className='text-xs text-muted-foreground'>类型</span><Select value={draftKind} onValueChange={setDraftKind}>
                          <SelectTrigger aria-label='按资产类型筛选' className='w-full'><SelectValue /></SelectTrigger>
                          <SelectContent><SelectGroup>{[['all', '全部类型'], ['image', '图片'], ['video', '视频'], ['audio', '音频'], ['document', '文档'], ['file', '其他']].map(([value, label]) => <SelectItem key={value} value={value}>{label}</SelectItem>)}</SelectGroup></SelectContent>
                        </Select></div>
                        <div className='flex flex-col gap-1.5'><span className='text-xs text-muted-foreground'>格式</span><Select value={draftFormat} onValueChange={setDraftFormat}>
                          <SelectTrigger aria-label='按格式筛选' className='w-full'><SelectValue /></SelectTrigger>
                          <SelectContent><SelectGroup><SelectItem value='all'>全部格式</SelectItem>{(formats.data ?? []).map((value) => <SelectItem key={value} value={value}>{value.toUpperCase()}</SelectItem>)}</SelectGroup></SelectContent>
                        </Select></div>
                      </div>
                      <div className='flex flex-col gap-2'><span className='text-sm font-medium'>宽度（px）</span><div className='flex items-center gap-2'><Input aria-label='最小宽度' type='number' min='0' step='1' placeholder='最小' value={draftFilters.widthMin} onChange={(event) => setDraftFilters((current) => ({ ...current, widthMin: event.target.value }))} /><span>—</span><Input aria-label='最大宽度' type='number' min='0' step='1' placeholder='最大' value={draftFilters.widthMax} onChange={(event) => setDraftFilters((current) => ({ ...current, widthMax: event.target.value }))} /></div></div>
                      <div className='flex flex-col gap-2'><span className='text-sm font-medium'>高度（px）</span><div className='flex items-center gap-2'><Input aria-label='最小高度' type='number' min='0' step='1' placeholder='最小' value={draftFilters.heightMin} onChange={(event) => setDraftFilters((current) => ({ ...current, heightMin: event.target.value }))} /><span>—</span><Input aria-label='最大高度' type='number' min='0' step='1' placeholder='最大' value={draftFilters.heightMax} onChange={(event) => setDraftFilters((current) => ({ ...current, heightMax: event.target.value }))} /></div></div>
                      <div className='flex flex-col gap-2'><span className='text-sm font-medium'>文件大小（MB）</span><div className='flex items-center gap-2'><Input aria-label='最小文件大小' type='number' min='0' step='any' placeholder='最小' value={draftFilters.sizeMin} onChange={(event) => setDraftFilters((current) => ({ ...current, sizeMin: event.target.value }))} /><span>—</span><Input aria-label='最大文件大小' type='number' min='0' step='any' placeholder='最大' value={draftFilters.sizeMax} onChange={(event) => setDraftFilters((current) => ({ ...current, sizeMax: event.target.value }))} /></div></div>
                      <div className='flex flex-col gap-2'><span className='text-sm font-medium'>添加日期</span><div className='flex items-center gap-2'><Input aria-label='添加日期起点' type='date' value={draftFilters.addedFrom} onChange={(event) => setDraftFilters((current) => ({ ...current, addedFrom: event.target.value }))} /><span>—</span><Input aria-label='添加日期终点' type='date' value={draftFilters.addedTo} onChange={(event) => setDraftFilters((current) => ({ ...current, addedTo: event.target.value }))} /></div></div>
                      <label className='flex items-center gap-2 text-sm'><Checkbox checked={draftFilters.duplicates} onCheckedChange={(checked) => setDraftFilters((current) => ({ ...current, duplicates: checked === true }))} />仅显示重复文件</label>
                      <label className='flex items-center gap-2 text-sm'><Checkbox checked={draftShowArchived} onCheckedChange={(checked) => setDraftShowArchived(checked === true)} />仅看已归档</label>
                      {filterError ? <p role='alert' className='text-sm text-destructive'>{filterError}</p> : null}
                      <div className='flex justify-end gap-2'><Button type='button' variant='ghost' onClick={() => { setDraftFilters(emptyAdvancedFilters); setAppliedFilters(emptyAdvancedFilters); setDraftKind('all'); setKind('all'); setDraftFormat('all'); setFormat('all'); setDraftShowArchived(false); setShowArchived(false); setFilterError(''); setSelectedIds([]); setAdvancedOpen(false) }}>清除</Button><Button type='submit'>应用筛选</Button></div>
                    </form>
                  </PopoverContent>
                </Popover>
              </div>
              <div aria-label='排列资产' className='flex shrink-0 items-center gap-1'>
                <DropdownMenu>
                  <IconButtonTooltip label={sort === 'recent' ? '最近添加' : sort === 'name' ? '名称排序' : '评分排序'}><DropdownMenuTrigger asChild><Button variant='ghost' size='icon' aria-label='排序资产'><ArrowDownWideNarrow /></Button></DropdownMenuTrigger></IconButtonTooltip>
                  <DropdownMenuContent align='end'>
                    <DropdownMenuLabel>排序方式</DropdownMenuLabel>
                    <DropdownMenuRadioGroup value={sort} onValueChange={(value) => { setSort(value); setSelectedIds([]) }}>
                      <DropdownMenuRadioItem value='recent' className='data-[state=checked]:bg-secondary data-[state=checked]:text-secondary-foreground'>最近添加</DropdownMenuRadioItem>
                      <DropdownMenuRadioItem value='name' className='data-[state=checked]:bg-secondary data-[state=checked]:text-secondary-foreground'>名称排序</DropdownMenuRadioItem>
                      <DropdownMenuRadioItem value='rating' className='data-[state=checked]:bg-secondary data-[state=checked]:text-secondary-foreground'>评分排序</DropdownMenuRadioItem>
                    </DropdownMenuRadioGroup>
                  </DropdownMenuContent>
                </DropdownMenu>
                <IconButtonTooltip label={view === 'grid' ? '切换列表视图' : '切换网格视图'}><Button variant='ghost' size='icon' aria-label={view === 'grid' ? '切换列表视图' : '切换网格视图'} onClick={() => setView(view === 'grid' ? 'list' : 'grid')}>
                  {view === 'grid' ? <List /> : <Grid2X2 />}
                </Button></IconButtonTooltip>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild><Button variant='outline' size='sm' className='shrink-0'>分类操作<ChevronDown /></Button></DropdownMenuTrigger>
                  <DropdownMenuContent align='end'>
                    <DropdownMenuGroup>
                      <DropdownMenuItem onSelect={() => { setCategoryName(''); setCategoryParentId(selectedCategory?.id ?? ''); setCategoryDialog('create') }}>新建分类</DropdownMenuItem>
                      <DropdownMenuItem disabled={!selectedCategory} onSelect={() => { if (!selectedCategory) return; setCategoryName(selectedCategory.name); setCategoryParentId(selectedCategory.parent_id ?? ''); setCategoryDialog('rename') }}>编辑分类</DropdownMenuItem>
                    </DropdownMenuGroup>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem disabled={!selectedCategory} onSelect={() => setDeleteCategoryOpen(true)}>删除分类</DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
                <Button size='sm' className='shrink-0' disabled={projectId === undefined || upload.isPending} onClick={() => fileInputRef.current?.click()}><Upload />{upload.isPending ? '正在上传…' : '上传资产'}</Button>
              </div>
            </div>
            <div className='flex min-w-0 items-center gap-2 px-4 py-2 lg:px-5'>
              <div className='min-w-0 flex-1 overflow-x-auto overscroll-x-contain [scrollbar-width:none] [&::-webkit-scrollbar]:hidden' role='group' aria-label='按分类筛选'>
                <div className='flex w-max items-center gap-1'>
                  {[{ value: 'all', label: '全部资产' }, { value: 'uncategorized', label: '未分类' }, ...(categories.data ?? []).map((item) => ({ value: item.id, label: categoryPath(item, categories.data ?? []) }))].map((item) => <Button key={item.value} variant={categoryId === item.value ? 'secondary' : 'ghost'} size='sm' className='max-w-40 shrink-0' aria-pressed={categoryId === item.value} title={item.label} onClick={() => { setCategoryId(item.value); setSelectedIds([]) }}><span className='truncate'>{item.label}</span></Button>)}
                </div>
              </div>
            </div>
          {sessionFilterId ? <div className='flex items-center gap-2 border-b px-4 py-2 text-xs lg:px-5'><span className='text-muted-foreground'>来源对话</span><Badge variant='secondary'>{initialSessionTitle ?? sessionFilterId}<button type='button' aria-label='清除来源对话筛选' onClick={() => { setSessionFilterId(undefined); setSelectedIds([]) }}><X className='size-3' /></button></Badge></div> : null}
          {selectedIds.length > 0 ? <div className='absolute bottom-12 left-1/2 z-20 flex max-w-[calc(100%-2rem)] -translate-x-1/2 items-center gap-1 rounded-lg border bg-popover p-1 text-popover-foreground shadow-md'>
            <span className='shrink-0 px-2 text-sm font-medium tabular-nums'>已选 {selectedIds.length} 项</span>
            <Button variant='ghost' size='sm' disabled={batchExport.isPending} onClick={() => batchExport.mutate()}><Download />导出</Button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild><Button variant='ghost' size='sm' disabled={batch.isPending}>批量操作<ChevronDown /></Button></DropdownMenuTrigger>
              <DropdownMenuContent align='end'>
                {(Object.keys(batchLabels) as BatchChoice[]).map((choice) => <DropdownMenuItem key={choice} onSelect={() => { setBatchChoice(choice); setBatchValue(''); batch.reset() }}>{batchLabels[choice]}</DropdownMenuItem>)}
                <DropdownMenuItem onSelect={() => showArchived ? batch.mutate({ action: 'archive', archived: false }) : setArchiveOpen(true)}>{showArchived ? '恢复资产' : '归档资产'}</DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
            <IconButtonTooltip label='取消选择'><Button variant='ghost' size='icon-sm' aria-label='取消选择' onClick={() => setSelectedIds([])}><X /></Button></IconButtonTooltip>
          </div> : null}
          {selectedIds.length > 0 && (batch.isError || batchExport.isError || batch.isSuccess && batch.data.results.some((result) => !result.success)) ? <p role='alert' className='absolute right-4 bottom-27 left-4 z-20 mx-auto max-w-xl rounded-md border bg-popover px-3 py-2 text-xs text-destructive shadow-md'>{batch.error?.message ?? batchExport.error?.message ?? `${batch.data?.results.filter((result) => !result.success).length} 项处理失败，可重试`}</p> : null}
          {upload.isError ? <Alert variant='destructive' className='m-4 w-auto'><AlertTitle>资产上传失败</AlertTitle><AlertDescription>{upload.error.message}<Button variant='outline' size='sm' disabled={!upload.variables || upload.isPending} onClick={() => upload.variables && upload.mutate(upload.variables)}>重试上传</Button></AlertDescription></Alert> : null}
          {savePreferences.isError ? <Alert variant='destructive' className='m-4 w-auto'><AlertTitle>文件树设置保存失败</AlertTitle><AlertDescription>{savePreferences.error.message}<Button variant='outline' size='sm' onClick={() => savePreferences.variables && savePreferences.mutate(savePreferences.variables)}>重试保存</Button></AlertDescription></Alert> : null}
          <ScrollArea ref={scrollRootRef} className='min-h-0 flex-1'>
            {projectId === undefined && (projects.isError || preferences.isError) ? (
              <LibraryState title='项目设置读取失败' onRetry={() => { void projects.refetch(); void preferences.refetch() }} />
            ) : assets.isPending || projectId === undefined ? (
              <div className='grid gap-3 p-4 sm:grid-cols-2 xl:grid-cols-3'>{Array.from({ length: 6 }, (_, index) => <Skeleton key={index} className='aspect-[4/3]' />)}</div>
            ) : assets.isError ? (
              <LibraryState title='资产读取失败' onRetry={() => void assets.refetch()} />
            ) : visibleAssets.length ? (
              <>
                <div className='relative w-full' style={{ height: virtualizer.getTotalSize() }}>
                  {virtualizer.getVirtualItems().map((row) => (
                    <div key={row.key} ref={virtualizer.measureElement} data-index={row.index} className='absolute top-0 left-0 grid w-full gap-3 px-4 py-2 lg:px-5' style={{ transform: `translateY(${row.start}px)`, gridTemplateColumns: `repeat(${view === 'grid' ? columns : 1}, minmax(0, 1fr))` }}>
                      {visibleAssets.slice(row.index * (view === 'grid' ? columns : 1), (row.index + 1) * (view === 'grid' ? columns : 1)).map((item) => (
                        <AssetCard key={item.id} asset={previewAsset(item)} preview lazyText layout={view} selected={selectedIds.includes(item.id)} onSelect={() => setSelectedIds((current) => current.includes(item.id) ? current.filter((id) => id !== item.id) : [...current, item.id])} onOpenDetails={() => setSelectedAssetId(item.id)} />
                      ))}
                    </div>
                  ))}
                </div>
                {assets.hasNextPage ? <Button variant='ghost' className='mx-auto mb-4 flex' disabled={assets.isFetchingNextPage} onClick={() => void assets.fetchNextPage()}>{assets.isFetchingNextPage ? '读取中…' : '加载更多资产'}</Button> : null}
              </>
            ) : hasFilters ? <LibraryState title='没有匹配的资产' actionLabel='清除筛选' onAction={clearFilters} /> : <LibraryState title='当前项目还没有资产' actionLabel='上传资产' onAction={() => fileInputRef.current?.click()} />}
          </ScrollArea>
          <footer className='border-t px-4 py-2 text-xs text-muted-foreground tabular-nums lg:px-5'>共 {total} 项资产</footer>
          </div>
          </ResizablePanel>
        </ResizablePanelGroup>
      </section>

      <Dialog open={treeOpen} onOpenChange={setTreeOpen}>
        <DialogContent className='flex max-h-[85svh] flex-col overflow-hidden sm:max-w-md'>
          <DialogHeader className='pr-10'><div className='flex items-center justify-between'><DialogTitle>项目资产</DialogTitle>{treeSettings}</div></DialogHeader>
          <ScrollArea className='min-h-0 flex-1'>{projectTree}</ScrollArea>
        </DialogContent>
      </Dialog>
      <LibraryAssetDetails key={selectedAssetId} projectAssetId={selectedAssetId} projects={projects.data ?? []} onSelectProjectAsset={setSelectedAssetId} onClose={() => setSelectedAssetId(undefined)} onOpenSession={onOpenSession} />
      <Dialog open={Boolean(batchChoice)} onOpenChange={(open) => { if (!open) setBatchChoice(undefined) }}>
        <DialogContent className='sm:max-w-sm'>
          <DialogHeader><DialogTitle>{batchChoice ? batchLabels[batchChoice] : '批量操作'}</DialogTitle><DialogDescription>已选 {selectedIds.length} 项资产</DialogDescription></DialogHeader>
          <Select value={batchValue} onValueChange={setBatchValue}>
            <SelectTrigger aria-label={batchChoice ? batchLabels[batchChoice] : '批量操作'} className='w-full'><SelectValue placeholder='请选择' /></SelectTrigger>
            <SelectContent><SelectGroup>
              {batchChoice === 'category' ? <><SelectItem value='__none__'>未分类</SelectItem>{(categories.data ?? []).map((item) => <SelectItem key={item.id} value={item.id}>{categoryPath(item, categories.data ?? [])}</SelectItem>)}</> : null}
              {batchChoice === 'tags' || batchChoice === 'tags_remove' ? (tags.data ?? []).map((item) => <SelectItem key={item.id} value={item.id}>{item.name}</SelectItem>) : null}
              {batchChoice === 'rating' ? [0, 1, 2, 3, 4, 5].map((value) => <SelectItem key={value} value={String(value)}>{value === 0 ? '未评分' : `${value} 星`}</SelectItem>) : null}
              {batchChoice === 'project' ? (projects.data ?? []).filter((item) => item.id !== projectId).map((item) => <SelectItem key={item.id || 'unassigned'} value={item.id || '__unassigned__'}>{item.name}</SelectItem>) : null}
            </SelectGroup></SelectContent>
          </Select>
          {batch.isError ? <p role='alert' className='text-sm text-destructive'>{batch.error.message}</p> : null}
          <DialogFooter><Button variant='outline' onClick={() => setBatchChoice(undefined)}>取消</Button><Button disabled={!batchValue || batch.isPending} onClick={applyBatchChoice}>确认</Button></DialogFooter>
        </DialogContent>
      </Dialog>
      <Dialog open={Boolean(categoryDialog)} onOpenChange={(open) => { if (!open) setCategoryDialog(undefined) }}>
        <DialogContent className='sm:max-w-md'>
          <DialogHeader><DialogTitle>{categoryDialog === 'rename' ? '编辑分类' : '新建分类'}</DialogTitle></DialogHeader>
          <form className='flex flex-col gap-4' onSubmit={(event) => { event.preventDefault(); if (categoryDialog === 'rename') renameCategory.mutate(); else createCategory.mutate() }}>
            <Input autoFocus aria-label='分类名称' value={categoryName} onChange={(event) => setCategoryName(event.target.value)} placeholder='分类名称' />
            <Select value={categoryParentId || '__root__'} onValueChange={(value) => setCategoryParentId(value === '__root__' ? '' : value)}>
              <SelectTrigger aria-label='父分类' className='w-full'><SelectValue /></SelectTrigger>
              <SelectContent><SelectGroup><SelectItem value='__root__'>项目根目录</SelectItem>{(categories.data ?? []).filter((item) => categoryDialog !== 'rename' || item.id !== categoryId).map((item) => <SelectItem key={item.id} value={item.id}>{categoryPath(item, categories.data ?? [])}</SelectItem>)}</SelectGroup></SelectContent>
            </Select>
            {createCategory.isError || renameCategory.isError ? <p role='alert' className='text-sm text-destructive'>{(createCategory.error ?? renameCategory.error)?.message}</p> : null}
            <DialogFooter><Button type='button' variant='outline' onClick={() => setCategoryDialog(undefined)}>取消</Button><Button type='submit' disabled={!categoryName.trim() || createCategory.isPending || renameCategory.isPending}>保存分类</Button></DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
      <AlertDialog open={deleteCategoryOpen} onOpenChange={setDeleteCategoryOpen}>
        <AlertDialogContent>
          <AlertDialogHeader><AlertDialogTitle>删除分类「{selectedCategory?.name}」？</AlertDialogTitle><AlertDialogDescription>分类中的资产保留。</AlertDialogDescription></AlertDialogHeader>
          {removeCategory.isError ? <p role='alert' className='text-sm text-destructive'>{removeCategory.error.message}</p> : null}
          <AlertDialogFooter><AlertDialogCancel>取消</AlertDialogCancel><AlertDialogAction disabled={removeCategory.isPending} onClick={(event) => { event.preventDefault(); removeCategory.mutate() }}>删除分类</AlertDialogAction></AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      <AlertDialog open={archiveOpen} onOpenChange={setArchiveOpen}>
        <AlertDialogContent>
          <AlertDialogHeader><AlertDialogTitle>归档 {selectedIds.length} 项项目资产？</AlertDialogTitle><AlertDialogDescription>这些项目资产将进入已归档列表。</AlertDialogDescription></AlertDialogHeader>
          <AlertDialogFooter><AlertDialogCancel>取消</AlertDialogCancel><AlertDialogAction disabled={batch.isPending} onClick={(event) => { event.preventDefault(); batch.mutate({ action: 'archive', archived: true }, { onSuccess: () => setArchiveOpen(false) }) }}>归档</AlertDialogAction></AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </main>
  )
}

function TreeName({ name }: { name: string }) {
  const ref = useRef<HTMLSpanElement>(null)
  const [overflowing, setOverflowing] = useState(false)
  useEffect(() => {
    const element = ref.current
    if (!element) return
    const measure = () => setOverflowing(element.scrollWidth > element.clientWidth)
    const observer = new ResizeObserver(measure)
    observer.observe(element)
    measure()
    return () => observer.disconnect()
  }, [name])
  return <Tooltip><TooltipTrigger asChild><span ref={ref} className='min-w-0 flex-1 truncate text-left'>{name}</span></TooltipTrigger>{overflowing ? <TooltipContent side='right' sideOffset={8} className='max-w-80 break-all'>{name}</TooltipContent> : null}</Tooltip>
}

export function ProjectTreeBranch({
  project, mode, selectedProjectId, treeFocusId, onFocusTreeItem, isFirstProject, onProject, onAsset,
}: {
  project: StudioLibraryProject
  mode: StudioLibraryTreeMode
  selectedProjectId?: string
  treeFocusId?: string
  onFocusTreeItem: (id: string) => void
  isFirstProject: boolean
  onProject: (id: string) => void
  onAsset: (id: string, node: StudioLibraryTreeNode) => void
}) {
  const [manualExpanded, setManualExpanded] = useState<boolean>()
  const expanded = manualExpanded ?? project.id === selectedProjectId
  const selected = selectedProjectId === project.id
  const focusId = `project:${project.id}`
  const tree = useInfiniteQuery({
    queryKey: ['studio', 'library', 'tree', project.id, mode],
    queryFn: ({ pageParam }) => listStudioLibraryTree({ projectId: project.id, mode, cursor: pageParam }),
    enabled: expanded,
    initialPageParam: '',
    getNextPageParam: (page) => page.next_cursor || undefined,
  })
  const onArrowKey = (event: KeyboardEvent<HTMLButtonElement>) => {
    if (event.key === 'ArrowRight') setManualExpanded(true)
    else if (event.key === 'ArrowLeft') setManualExpanded(false)
    else return
    event.preventDefault()
    event.stopPropagation()
  }
  return <Collapsible open={expanded} onOpenChange={setManualExpanded} role='treeitem' aria-expanded={expanded} aria-selected={selected} className='min-w-0'>
    <div className={cn('flex h-8 min-w-0 items-center gap-1 rounded-md hover:bg-accent', selected && 'bg-secondary')}>
      <CollapsibleTrigger asChild>
        <Button variant='ghost' size='sm' data-tree-focus-target='' tabIndex={treeFocusId === focusId || (!treeFocusId && isFirstProject) ? 0 : -1} onFocus={() => onFocusTreeItem(focusId)} className='min-w-0 flex-1 justify-start px-2 font-normal hover:bg-transparent [&_svg]:size-4' onKeyDown={onArrowKey} onClick={() => onProject(project.id)}>{expanded ? <FolderOpen /> : <Folder />}<TreeName name={project.name} /></Button>
      </CollapsibleTrigger>
      <span className='shrink-0 text-xs tabular-nums text-muted-foreground'>{project.asset_count}</span>
      <CollapsibleTrigger asChild>
        <Button variant='ghost' size='icon-sm' tabIndex={-1} className='shrink-0' aria-label={expanded ? `收起${project.name}` : `展开${project.name}`}><ChevronRight className={cn('transition-transform duration-150 motion-reduce:transition-none', expanded && 'rotate-90')} /></Button>
      </CollapsibleTrigger>
    </div>
    <CollapsibleContent className={treeContentClassName} inert={!expanded} aria-hidden={!expanded}>
      <div role='group' className='ml-4 flex min-w-0 flex-col gap-0.5 border-l border-border/60 pl-[calc(var(--spacing)*1.5_-_1px)]'>
        {tree.isPending ? <Skeleton className='h-8 w-full' /> : tree.isError ? <Button variant='ghost' size='sm' onClick={() => void tree.refetch()}>重试读取</Button> : tree.data.pages.flatMap((page) => page.nodes).map((node) => (
          <TreeNode key={node.id} node={node} projectId={project.id} mode={mode} treeFocusId={treeFocusId} onFocusTreeItem={onFocusTreeItem} onAsset={onAsset} />
        ))}
        {tree.hasNextPage ? <Button variant='ghost' size='sm' disabled={tree.isFetchingNextPage} onClick={() => void tree.fetchNextPage()}>{tree.isFetchingNextPage ? '读取中…' : '加载更多'}</Button> : null}
      </div>
    </CollapsibleContent>
  </Collapsible>
}

export function TreeNode({ node, projectId, mode, treeFocusId, onFocusTreeItem, onAsset }: {
  node: StudioLibraryTreeNode
  projectId: string
  mode: StudioLibraryTreeMode
  treeFocusId?: string
  onFocusTreeItem: (id: string) => void
  onAsset: (id: string, node: StudioLibraryTreeNode) => void
}) {
  const [expanded, setExpanded] = useState(false)
  const children = useInfiniteQuery({
    queryKey: ['studio', 'library', 'tree', projectId, mode, node.id],
    queryFn: ({ pageParam }) => listStudioLibraryTree({
      projectId, mode, parentId: node.id, cursor: pageParam,
    }),
    enabled: expanded && node.kind === 'group',
    initialPageParam: '',
    getNextPageParam: (page) => page.next_cursor || undefined,
  })
  const focusId = `node:${projectId}:${node.id}`
  const onArrowKey = (event: KeyboardEvent<HTMLButtonElement>) => {
    if (node.kind !== 'group') return
    if (event.key === 'ArrowRight') setExpanded(true)
    else if (event.key === 'ArrowLeft') setExpanded(false)
    else return
    event.preventDefault()
    event.stopPropagation()
  }
  return <Collapsible open={expanded} role='treeitem' aria-expanded={node.kind === 'group' ? expanded : undefined} className='min-w-0'>
    <div className='flex h-8 min-w-0 items-center'>
          <Button
            variant='ghost'
            size='sm'
            data-tree-focus-target=''
            tabIndex={treeFocusId === focusId ? 0 : -1}
            onFocus={() => onFocusTreeItem(focusId)}
            className='min-w-0 flex-1 justify-start px-2 font-normal [&_svg]:size-4'
            onKeyDown={onArrowKey}
            onClick={() => {
              if (node.kind === 'asset') {
                onAsset(projectId, node)
                return
              }
              setExpanded((value) => !value)
            }}
          >
            {node.kind === 'asset' ? <AssetKindIcon kind={node.asset_kind} data-icon='inline-start' /> : mode === 'session' ? <MessageCircle /> : mode === 'category' ? expanded ? <FolderOpen /> : <Folder /> : mode === 'rating' ? <Star /> : mode === 'tag' ? <Tag /> : <Library />}
            <TreeName name={node.label} />
            {node.kind === 'group' ? <span className='shrink-0 tabular-nums text-muted-foreground'>{node.count}</span> : null}
          </Button>
    </div>
    {node.kind === 'group' ? <CollapsibleContent className={treeContentClassName} inert={!expanded} aria-hidden={!expanded}>
      <div role='group' className='ml-4 flex min-w-0 flex-col pl-1.5'>
        {children.isPending ? <Skeleton className='h-8 w-full' /> : children.isError ? <Button variant='ghost' size='sm' onClick={() => void children.refetch()}>重试读取</Button> : children.data.pages.flatMap((page) => page.nodes).map((child) => <TreeNode key={child.id} node={child} projectId={projectId} mode={mode} treeFocusId={treeFocusId} onFocusTreeItem={onFocusTreeItem} onAsset={onAsset} />)}
        {children.hasNextPage ? <Button variant='ghost' size='sm' disabled={children.isFetchingNextPage} onClick={() => void children.fetchNextPage()}>加载更多</Button> : null}
      </div>
    </CollapsibleContent> : null}
  </Collapsible>
}

function LibraryState({ title, onRetry, actionLabel, onAction }: { title: string; onRetry?: () => void; actionLabel?: string; onAction?: () => void }) {
  return <Empty className='min-h-80'><EmptyHeader><Library className='size-8 text-muted-foreground' /><EmptyTitle>{title}</EmptyTitle></EmptyHeader>{onRetry || onAction ? <EmptyContent>{onRetry ? <Button variant='outline' onClick={onRetry}>重试读取</Button> : null}{onAction ? <Button variant='outline' onClick={onAction}>{actionLabel}</Button> : null}</EmptyContent> : null}</Empty>
}

function LibraryAssetDetails({ projectAssetId, projects, onSelectProjectAsset, onClose, onOpenSession }: {
  projectAssetId?: string
  projects: StudioLibraryProject[]
  onSelectProjectAsset: (id: string) => void
  onClose: () => void
  onOpenSession: (sessionId: string, sourceRunId?: string) => void
}) {
  const queryClient = useQueryClient()
  const [expanded, setExpanded] = useState(false)
  const [editingName, setEditingName] = useState(false)
  const [name, setName] = useState('')
  const [targetProjectId, setTargetProjectId] = useState<string>()
  const [tagName, setTagName] = useState('')
  const [versionId, setVersionId] = useState('')
  const [downloadError, setDownloadError] = useState('')
  const [archiveOpen, setArchiveOpen] = useState(false)
  const [useOpen, setUseOpen] = useState(false)
  const [targetSessionId, setTargetSessionId] = useState('')
  const referenceRequestId = useRef<string | undefined>(undefined)
  const detail = useQuery({
    queryKey: ['studio', 'library', 'asset', projectAssetId],
    queryFn: () => getStudioProjectAsset(projectAssetId!),
    enabled: Boolean(projectAssetId),
  })
  const detailCategories = useQuery({
    queryKey: ['studio', 'library', 'categories', detail.data?.project_id],
    queryFn: () => listStudioLibraryCategories(detail.data!.project_id),
    enabled: Boolean(detail.data),
  })
  const duplicates = useQuery({
    queryKey: ['studio', 'library', 'duplicates', projectAssetId],
    queryFn: () => getStudioProjectAssetDuplicates(projectAssetId!),
    enabled: Boolean(projectAssetId),
  })
  const sessions = useInfiniteQuery({
    queryKey: ['studio', 'sessions', 'asset-use'],
    queryFn: ({ pageParam }) => listStudioSessions({ limit: 50, offset: pageParam }),
    enabled: useOpen,
    initialPageParam: 0,
    getNextPageParam: (page, pages) => page.length === 50 ? pages.length * 50 : undefined,
  })
  const tags = useQuery({ queryKey: ['studio', 'library', 'tags'], queryFn: listStudioAssetTags, enabled: Boolean(projectAssetId) })
  const refresh = async () => {
    await queryClient.invalidateQueries({ queryKey: ['studio', 'library'] })
  }
  const patch = useMutation({
    mutationFn: (input: Parameters<typeof updateStudioProjectAsset>[1]) => updateStudioProjectAsset(projectAssetId!, input),
    onSuccess: refresh,
  })
  const saveTags = useMutation({
    mutationFn: (ids: string[]) => updateStudioProjectAssetTags(projectAssetId!, ids),
    onSuccess: refresh,
  })
  const createTag = useMutation({
    mutationFn: async () => {
      const created = await createStudioAssetTag(tagName.trim())
      const ids = detail.data?.tags.map((tag) => tag.id) ?? []
      await updateStudioProjectAssetTags(projectAssetId!, [...ids, created.id])
    },
    onSuccess: async () => { setTagName(''); await refresh() },
  })
  const copy = useMutation({
    mutationFn: () => addStudioAssetToProject(projectAssetId!, targetProjectId!, detail.data!.asset_version_id),
    onSuccess: refresh,
  })
  const changeVersion = useMutation({
    mutationFn: () => updateStudioProjectAssetVersion(projectAssetId!, versionId),
    onSuccess: refresh,
  })
  const retryPalette = useMutation({
    mutationFn: () => retryStudioAssetPalette(detail.data!.asset_id, detail.data!.asset_version_id),
    onSuccess: refresh,
  })
  const reference = useMutation({
    mutationFn: () => {
      if (!targetSessionId || !detail.data) throw new Error('请选择对话')
      referenceRequestId.current ??= crypto.randomUUID()
      return referenceStudioAsset(targetSessionId, detail.data.asset_id, detail.data.asset_version_id, referenceRequestId.current, detail.data.id)
    },
    onSuccess: async () => {
      referenceRequestId.current = undefined
      await queryClient.invalidateQueries({ queryKey: ['studio', 'session', targetSessionId] })
      await queryClient.invalidateQueries({ queryKey: ['studio', 'library'] })
      setUseOpen(false)
      onClose()
      onOpenSession(targetSessionId)
    },
  })
  const exportAsset = useMutation({
    mutationFn: () => exportStudioProjectAssets([projectAssetId!]),
    onSuccess: ({ blob, filename }) => {
      setDownloadError('')
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = filename ? decodeURIComponent(filename) : detail.data?.display_name ?? 'asset'
      document.body.append(link)
      link.click()
      link.remove()
      window.setTimeout(() => URL.revokeObjectURL(url), 60_000)
    },
    onError: (error) => setDownloadError(error.message),
  })
  const item = detail.data
  const matchingAssets = duplicates.data?.items.filter((candidate) => candidate.id !== item?.id) ?? []
  const preview = item ? previewAsset(item) : undefined
  const source = item?.usages.find((usage) => usage.usage_kind !== 'referenced') ?? item?.usages[0]
  const createdAt = item?.version.content_origin === 'upload'
    ? item.version.source_created_at
    : item?.asset.created_at
  const modifiedAt = item?.version.content_origin === 'upload'
    ? item.version.source_modified_at
    : item?.version.created_at
  const close = () => { setExpanded(false); setEditingName(false); setDownloadError(''); onClose() }

  const footer = item ? <>
    {downloadError || patch.isError || saveTags.isError || createTag.isError || copy.isError || changeVersion.isError || retryPalette.isError ? <p role='alert' className='mb-2 text-sm text-destructive'>{downloadError || patch.error?.message || saveTags.error?.message || createTag.error?.message || copy.error?.message || changeVersion.error?.message || retryPalette.error?.message}</p> : null}
    <div className='flex flex-wrap gap-2'>
      <Button variant='default' size='sm' onClick={() => setUseOpen(true)}>用于创作</Button>
      <Button variant='outline' size='sm' disabled={exportAsset.isPending} onClick={() => exportAsset.mutate()}><Download />导出</Button>
      <Button variant='outline' size='sm' asChild><a href={`${baseURL()}${item.version.content_url}`} target='_blank' rel='noreferrer'>打开原文件</a></Button>
      <Button variant='outline' size='sm' disabled={patch.isPending} onClick={() => item.archived_at ? patch.mutate({ archived: false }) : setArchiveOpen(true)}>{item.archived_at ? '恢复资产' : '归档资产'}</Button>
    </div>
  </> : undefined
  return <><AssetDetailsDialog
    open={Boolean(projectAssetId)} onOpenChange={(open) => { if (!open) close() }} asset={preview} expanded={expanded} onExpandedChange={setExpanded}
    description={item ? `固定版本 v${item.version.version}` : '读取资产中…'}
    nameEditor={editingName ? <form className='flex gap-2' onSubmit={(event) => { event.preventDefault(); if (name.trim()) patch.mutate({ display_name: name.trim() }, { onSuccess: () => setEditingName(false) }) }}><Input autoFocus aria-label='资产名称' value={name} onChange={(event) => setName(event.target.value)} /><Button type='submit' disabled={!name.trim() || patch.isPending}>保存</Button></form> : undefined}
    nameAction={item ? <Button variant='ghost' size='icon-sm' className='shrink-0' aria-label='重命名资产' onClick={() => { setName(item.display_name); setEditingName(true) }}><Pencil /></Button> : undefined}
    footer={footer}
    fallback={detail.isPending ? <Skeleton className='m-5 min-h-0 flex-1' /> : detail.isError ? <LibraryState title='资产详情读取失败' onRetry={() => void detail.refetch()} /> : undefined}
  >
    {item ? <>
      <AssetDetailsInfo>
        <AssetInfoRow label='评分' value={<div className='flex items-center gap-0' role='group' aria-label='资产评分'>{[1, 2, 3, 4, 5].map((rating) => <Button key={rating} variant='ghost' size='icon-sm' className='p-0 first:-ml-2' disabled={patch.isPending} aria-label={`评分 ${rating} 星`} aria-pressed={item.rating === rating} onClick={() => patch.mutate({ rating: item.rating === rating ? 0 : rating })}><Star className={cn('size-4', rating <= item.rating && 'fill-current')} /></Button>)}</div>} />
        <AssetInfoRow label='尺寸' value={item.version.width_px && item.version.height_px ? `${item.version.width_px} × ${item.version.height_px}` : '—'} />
        <AssetSizeRow bytes={item.version.size_bytes} />
        <AssetInfoRow label='格式' value={item.version.format?.toUpperCase() ?? item.version.mime_type} />
        <AssetInfoRow label='添加日期' value={formatDate(item.added_at)} />
        <AssetInfoRow label='创建日期' value={formatDate(createdAt)} />
        <AssetInfoRow label='修改日期' value={formatDate(modifiedAt)} />
        {(item.asset.kind === 'image' || item.asset.kind === 'video') ? <AssetPaletteRow palette={item.version.palette} onRetry={() => retryPalette.mutate()} retrying={retryPalette.isPending} /> : null}
      </AssetDetailsInfo>
            <section className='flex flex-col gap-2'>
              <h2 className='text-sm font-semibold'>分类</h2>
              <Select value={item.category_id || '__none__'} onValueChange={(value) => patch.mutate({ category_id: value === '__none__' ? '' : value })} disabled={patch.isPending}>
                <SelectTrigger aria-label='资产分类' className='w-full'><SelectValue /></SelectTrigger><SelectContent><SelectGroup><SelectItem value='__none__'>未分类</SelectItem>{(detailCategories.data ?? []).map((category) => <SelectItem key={category.id} value={category.id}>{categoryPath(category, detailCategories.data ?? [])}</SelectItem>)}</SelectGroup></SelectContent>
              </Select>
            </section>
            <section className='flex flex-col gap-2'><h2 className='text-sm font-semibold'>标签</h2>
              <div className='flex flex-wrap gap-1'>{item.tags.map((tag) => <Button key={tag.id} variant='secondary' size='sm' disabled={saveTags.isPending} onClick={() => saveTags.mutate(item.tags.filter((value) => value.id !== tag.id).map((value) => value.id))}>{tag.name}<X /></Button>)}</div>
              <Select value='__choose__' onValueChange={(value) => saveTags.mutate([...item.tags.map((tag) => tag.id), value])} disabled={saveTags.isPending}>
                <SelectTrigger aria-label='添加标签' className='w-full'><SelectValue placeholder='添加标签' /></SelectTrigger><SelectContent><SelectGroup><SelectItem value='__choose__' disabled>添加标签</SelectItem>{(tags.data ?? []).filter((tag) => !item.tags.some((entry) => entry.id === tag.id)).map((tag) => <SelectItem key={tag.id} value={tag.id}>{tag.name}</SelectItem>)}</SelectGroup></SelectContent>
              </Select>
              <form className='flex gap-2' onSubmit={(event) => { event.preventDefault(); if (tagName.trim()) createTag.mutate() }}><Input aria-label='新标签名称' placeholder='新标签名称' value={tagName} onChange={(event) => setTagName(event.target.value)} /><Button type='submit' variant='outline' disabled={!tagName.trim() || createTag.isPending}>添加</Button></form>
            </section>
            <section className='flex flex-col gap-2'><h2 className='text-sm font-semibold'>来源</h2>
              {source ? <div className='flex items-center justify-between gap-2 text-sm'><span className='min-w-0 truncate'>{source.session_title_snapshot}</span>{source.session_available ? <Button variant='outline' size='sm' onClick={() => { close(); onOpenSession(source.session_id, source.run_id) }}>打开对话</Button> : <span className='text-muted-foreground'>对话已删除</span>}</div> : <p className='text-sm text-muted-foreground'>无来源对话</p>}
              {item.copy_source ? <p className='text-xs text-muted-foreground'>复制来源：{item.copy_source.project_name_snapshot || '未归属项目'} / {item.copy_source.display_name}{item.copy_source.deleted ? '（来源条目已删除）' : ''}</p> : null}
            </section>
            {matchingAssets.length > 0 ? <section className='flex flex-col gap-2'><h2 className='text-sm font-semibold'>发现相同文件</h2>{matchingAssets.map((candidate) => <Button key={candidate.id} variant='outline' size='sm' className='h-auto justify-start text-left' onClick={() => onSelectProjectAsset(candidate.id)}><span className='min-w-0 truncate'>{candidate.display_name}</span><span className='ml-auto shrink-0 text-xs text-muted-foreground'>{projects.find((project) => project.id === candidate.project_id)?.name ?? '项目已删除'}</span></Button>)}</section> : null}
            <section className='flex flex-col gap-2'><h2 className='text-sm font-semibold'>版本</h2>
              <Select value={versionId || item.asset_version_id} onValueChange={setVersionId}><SelectTrigger aria-label='选择资产版本' className='w-full'><SelectValue /></SelectTrigger><SelectContent><SelectGroup>{item.versions.map((version) => <SelectItem key={version.id} value={version.id}>v{version.version} · {formatDate(version.created_at)}</SelectItem>)}</SelectGroup></SelectContent></Select>
              {versionId && versionId !== item.asset_version_id ? <Button variant='outline' size='sm' disabled={changeVersion.isPending} onClick={() => changeVersion.mutate()}>更新项目资产版本</Button> : null}
            </section>
            <section className='flex flex-col gap-2'><h2 className='text-sm font-semibold'>添加到其他项目</h2>
              <Select value={targetProjectId === undefined ? '__choose__' : targetProjectId || '__unassigned__'} onValueChange={(value) => setTargetProjectId(value === '__unassigned__' ? '' : value)}><SelectTrigger aria-label='目标项目' className='w-full'><SelectValue placeholder='选择目标项目' /></SelectTrigger><SelectContent><SelectGroup><SelectItem value='__choose__' disabled>选择目标项目</SelectItem>{projects.filter((project) => project.id !== item.project_id).map((project) => <SelectItem key={project.id || 'unassigned'} value={project.id || '__unassigned__'}>{project.name}</SelectItem>)}</SelectGroup></SelectContent></Select>
              {targetProjectId !== undefined ? <Button variant='outline' size='sm' disabled={copy.isPending} onClick={() => copy.mutate()}>添加到项目</Button> : null}
            </section>
    </> : null}
  </AssetDetailsDialog>
  <AlertDialog open={archiveOpen} onOpenChange={setArchiveOpen}>
    <AlertDialogContent><AlertDialogHeader><AlertDialogTitle>归档 1 项项目资产？</AlertDialogTitle><AlertDialogDescription>此项目资产将进入已归档列表。</AlertDialogDescription></AlertDialogHeader>{patch.isError ? <p role='alert' className='text-sm text-destructive'>{patch.error.message}</p> : null}<AlertDialogFooter><AlertDialogCancel>取消</AlertDialogCancel><AlertDialogAction disabled={patch.isPending} onClick={(event) => { event.preventDefault(); patch.mutate({ archived: true }, { onSuccess: () => setArchiveOpen(false) }) }}>归档</AlertDialogAction></AlertDialogFooter></AlertDialogContent>
  </AlertDialog>
  <Dialog open={useOpen} onOpenChange={setUseOpen}>
    <DialogContent className='sm:max-w-md'><DialogHeader><DialogTitle>用于创作</DialogTitle><DialogDescription>选择目标对话</DialogDescription></DialogHeader>
      <Select value={targetSessionId || '__choose__'} onValueChange={(value) => { setTargetSessionId(value); referenceRequestId.current = undefined }}>
        <SelectTrigger aria-label='目标对话' className='w-full'><SelectValue placeholder='选择对话' /></SelectTrigger>
        <SelectContent><SelectGroup><SelectItem value='__choose__' disabled>选择对话</SelectItem>{sessions.data?.pages.flatMap((page) => page).map((session) => <SelectItem key={session.id} value={session.id}>{session.title} · {projects.find((project) => project.id === (session.project_id ?? ''))?.name ?? '未归属项目'}</SelectItem>)}</SelectGroup></SelectContent>
      </Select>
      {sessions.isPending ? <Skeleton className='h-9 w-full' /> : sessions.isError ? <Button variant='outline' onClick={() => void sessions.refetch()}>重试读取对话</Button> : sessions.data.pages.every((page) => page.length === 0) ? <p className='text-sm text-muted-foreground'>暂无对话</p> : null}
      {sessions.hasNextPage ? <Button variant='ghost' disabled={sessions.isFetchingNextPage} onClick={() => void sessions.fetchNextPage()}>加载更多对话</Button> : null}
      {reference.isError ? <p role='alert' className='text-sm text-destructive'>{reference.error.message}</p> : null}
      <DialogFooter><Button variant='outline' onClick={() => setUseOpen(false)}>取消</Button><Button disabled={!targetSessionId || reference.isPending} onClick={() => reference.mutate()}>添加到对话</Button></DialogFooter>
    </DialogContent>
  </Dialog></>
}
