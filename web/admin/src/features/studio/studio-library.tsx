import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useVirtualizer } from '@tanstack/react-virtual'
import { Grid2X2, Library, List, Search, Tags, Upload } from 'lucide-react'
import { baseURL } from '@/lib/api/client'
import {
  createStudioLibraryCategory,
  getStudioSession,
  listStudioLibraryAssets,
  listStudioLibraryCategories,
  moveStudioLibraryAsset,
  uploadStudioAsset,
  type StudioAsset,
  type StudioLibraryCategory,
} from '@/lib/api/studio'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
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
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
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
import { originLabel } from './studio-asset-origin'
import { AssetCard, AssetPreview } from './studio-assets'

export function StudioLibrary({
  onOpenSession,
}: {
  onOpenSession: (sessionId: string, sourceRunId?: string) => void
}) {
  const queryClient = useQueryClient()
  const [categoryId, setCategoryId] = useState<string>()
  const [categoryDialogOpen, setCategoryDialogOpen] = useState(false)
  const [categoryName, setCategoryName] = useState('')
  const [selectedAsset, setSelectedAsset] = useState<StudioAsset>()
  const [query, setQuery] = useState('')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [view, setView] = useState<'grid' | 'list'>('grid')
  const [columns, setColumns] = useState(1)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const scrollRootRef = useRef<HTMLDivElement>(null)
  useEffect(() => {
    const timer = window.setTimeout(() => {
      setSearch(query.trim())
      setPage(1)
    }, 250)
    return () => window.clearTimeout(timer)
  }, [query])
  useEffect(() => {
    const viewport = scrollRootRef.current?.querySelector<HTMLElement>(
      '[data-slot="scroll-area-viewport"]'
    )
    if (!viewport) return
    const observer = new ResizeObserver(() => {
      const width = viewport.clientWidth
      setColumns(width >= 1100 ? 4 : width >= 800 ? 3 : width >= 540 ? 2 : 1)
    })
    observer.observe(viewport)
    return () => observer.disconnect()
  }, [])
  const categories = useQuery({
    queryKey: ['studio', 'library', 'categories'],
    queryFn: listStudioLibraryCategories,
  })
  const assets = useQuery({
    queryKey: ['studio', 'library', 'assets', categoryId, search, page],
    queryFn: () =>
      listStudioLibraryAssets({ categoryId, search, page, limit: 50 }),
  })
  const createCategory = useMutation({
    mutationFn: () =>
      createStudioLibraryCategory({ name: categoryName.trim() }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ['studio', 'library', 'categories'],
      })
      setCategoryName('')
      setCategoryDialogOpen(false)
    },
  })
  const upload = useMutation({
    mutationFn: (file: File) => uploadStudioAsset(file),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ['studio', 'library'] }),
  })
  const visibleAssets = assets.data?.assets ?? []
  const total = assets.data?.total ?? 0
  const pageCount = Math.max(1, Math.ceil(total / 50))
  useEffect(() => {
    if (assets.isSuccess && page > pageCount) setPage(pageCount)
  }, [assets.isSuccess, page, pageCount])
  const rowCount =
    view === 'grid'
      ? Math.ceil(visibleAssets.length / columns)
      : visibleAssets.length
  const virtualizer = useVirtualizer({
    count: rowCount,
    getScrollElement: () =>
      scrollRootRef.current?.querySelector<HTMLElement>(
        '[data-slot="scroll-area-viewport"]'
      ) ?? null,
    estimateSize: () => (view === 'grid' ? 350 : 190),
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
  }, [categoryId, search, page, view])

  return (
    <main id='main-content' className='min-h-0 min-w-0 flex-1 p-3 sm:p-4'>
      <section className='flex h-full min-h-0 flex-col overflow-hidden rounded-2xl border bg-card'>
        <header className='flex min-h-16 flex-wrap items-center justify-between gap-3 border-b px-5 py-2'>
          <div>
            <h1 className='text-sm font-semibold'>资产库</h1>
            <p className='text-xs text-muted-foreground'>
              管理跨会话复用的创作素材
            </p>
          </div>
          <div className='flex min-w-0 flex-wrap items-center justify-end gap-2'>
            <div className='relative w-56 max-w-full'>
              <Search className='absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground' />
              <Input
                className='h-8 pl-9'
                aria-label='搜索资产'
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder='搜索资产'
              />
            </div>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant='outline' size='sm'>
                  {view === 'grid' ? <Grid2X2 /> : <List />}
                  {view === 'grid' ? '网格视图' : '列表视图'}
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align='end'>
                <DropdownMenuRadioGroup
                  value={view}
                  onValueChange={(value) => setView(value as 'grid' | 'list')}
                >
                  <DropdownMenuRadioItem value='grid'>
                    <Grid2X2 />
                    网格视图
                  </DropdownMenuRadioItem>
                  <DropdownMenuRadioItem value='list'>
                    <List />
                    列表视图
                  </DropdownMenuRadioItem>
                </DropdownMenuRadioGroup>
              </DropdownMenuContent>
            </DropdownMenu>
            <Button
              variant='outline'
              size='sm'
              disabled={upload.isPending}
              onClick={() => fileInputRef.current?.click()}
            >
              <Upload />
              {upload.isPending ? '正在上传…' : '上传资产'}
            </Button>
            <input
              ref={fileInputRef}
              type='file'
              aria-label='选择上传资产'
              className='sr-only'
              onChange={(event) => {
                const file = event.target.files?.[0]
                if (file) upload.mutate(file)
                event.currentTarget.value = ''
              }}
            />
          </div>
        </header>
        <div className='flex items-center justify-between gap-3 px-5 pt-3 pb-1'>
          <div className='flex min-w-0 items-center gap-2 overflow-x-auto'>
            <Button
              variant={categoryId ? 'ghost' : 'secondary'}
              size='sm'
              onClick={() => {
                setCategoryId(undefined)
                setPage(1)
              }}
            >
              全部资产
            </Button>
            {categories.data?.map((category) => (
              <Button
                key={category.id}
                variant={categoryId === category.id ? 'secondary' : 'ghost'}
                size='sm'
                onClick={() => {
                  setCategoryId(category.id)
                  setPage(1)
                }}
              >
                {category.name}
              </Button>
            ))}
          </div>
          <Button
            variant='outline'
            size='sm'
            className='shrink-0'
            onClick={() => setCategoryDialogOpen(true)}
          >
            <Tags />
            新建分类
          </Button>
        </div>
        {categories.isError ? (
          <Alert variant='destructive' className='mx-5 mt-4 w-auto'>
            <AlertTitle>分类读取失败</AlertTitle>
            <AlertDescription>
              <p>分类列表暂不可用，资产仍可查看。</p>
              <Button
                variant='outline'
                size='sm'
                onClick={() => void categories.refetch()}
              >
                重试读取分类
              </Button>
            </AlertDescription>
          </Alert>
        ) : null}
        {upload.isError ? (
          <Alert variant='destructive' className='mx-5 mt-4 w-auto'>
            <AlertTitle>资产上传失败</AlertTitle>
            <AlertDescription>
              <p>
                {upload.error instanceof Error
                  ? upload.error.message
                  : '上传未成功，重试上传。'}
              </p>
              <Button
                variant='outline'
                size='sm'
                disabled={upload.isPending || !upload.variables}
                onClick={() => {
                  if (upload.variables) upload.mutate(upload.variables)
                }}
              >
                重试上传
              </Button>
            </AlertDescription>
          </Alert>
        ) : null}
        <ScrollArea ref={scrollRootRef} className='min-h-0 flex-1'>
          {assets.isLoading ? (
            <div className='grid gap-4 p-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4'>
              {Array.from({ length: 8 }).map((_, index) => (
                <Skeleton key={index} className='aspect-[4/3] rounded-xl' />
              ))}
            </div>
          ) : assets.isError ? (
            <LibraryState
              title='资产读取失败'
              description='已有资产未变化。'
              actionLabel='重试读取'
              onAction={() => void assets.refetch()}
            />
          ) : visibleAssets.length ? (
            <div
              className='relative w-full'
              style={{ height: virtualizer.getTotalSize() }}
            >
              {virtualizer.getVirtualItems().map((row) => (
                <div
                  key={row.key}
                  ref={virtualizer.measureElement}
                  data-index={row.index}
                  className='absolute top-0 left-0 grid w-full gap-4 px-5 py-2'
                  style={{
                    transform: `translateY(${row.start}px)`,
                    gridTemplateColumns: `repeat(${view === 'grid' ? columns : 1}, minmax(0, 1fr))`,
                  }}
                >
                  {visibleAssets
                    .slice(
                      row.index * (view === 'grid' ? columns : 1),
                      (row.index + 1) * (view === 'grid' ? columns : 1)
                    )
                    .map((asset) => (
                      <AssetCard
                        key={asset.id}
                        asset={asset}
                        preview
                        lazyText
                        layout={view === 'list' ? 'list' : 'grid'}
                        onOpenDetails={setSelectedAsset}
                        onLocateSource={
                          asset.origin !== 'user' &&
                          asset.session_id &&
                          asset.source_run_id
                            ? () =>
                                onOpenSession(
                                  asset.session_id,
                                  asset.source_run_id
                                )
                            : undefined
                        }
                      />
                    ))}
                </div>
              ))}
            </div>
          ) : search ? (
            <LibraryState
              title='没有匹配的资产'
              description='换个关键词试试，或清空搜索条件查看全部资产。'
            />
          ) : (
            <LibraryState
              title='资产库还是空的'
              description='你可以直接上传资产，也可以在对话中把会话资产存到这里。'
            />
          )}
        </ScrollArea>
        {assets.isSuccess ? (
          <div className='flex flex-wrap items-center justify-between gap-3 border-t px-5 py-2 text-xs text-muted-foreground'>
            <span className='tabular-nums'>
              共 {total} 项 · 第 {page} / {pageCount} 页
            </span>
            <div className='flex items-center gap-2'>
              <Button
                variant='outline'
                size='sm'
                disabled={page <= 1}
                onClick={() => setPage((value) => value - 1)}
              >
                上一页
              </Button>
              <Button
                variant='outline'
                size='sm'
                disabled={page >= pageCount}
                onClick={() => setPage((value) => value + 1)}
              >
                下一页
              </Button>
            </div>
          </div>
        ) : null}
        {selectedAsset ? (
          <LibraryAssetDetails
            asset={selectedAsset}
            categories={categories.data ?? []}
            onClose={() => setSelectedAsset(undefined)}
            onOpenSession={(sessionId, sourceRunId) => {
              setSelectedAsset(undefined)
              if (sourceRunId) onOpenSession(sessionId, sourceRunId)
              else onOpenSession(sessionId)
            }}
          />
        ) : null}
        <Dialog open={categoryDialogOpen} onOpenChange={setCategoryDialogOpen}>
          <DialogContent className='sm:max-w-md'>
            <DialogHeader>
              <DialogTitle>新建分类</DialogTitle>
              <DialogDescription>
                分类用于整理可跨会话复用的资产。
              </DialogDescription>
            </DialogHeader>
            <div className='grid gap-2 py-2'>
              <Label htmlFor='studio-category-name'>分类名称</Label>
              <Input
                id='studio-category-name'
                value={categoryName}
                onChange={(event) => setCategoryName(event.target.value)}
                placeholder='例如：角色设定'
              />
            </div>
            {createCategory.isError ? (
              <p role='alert' className='text-sm text-destructive'>
                分类创建失败，请重试。
              </p>
            ) : null}
            <DialogFooter>
              <Button
                variant='outline'
                onClick={() => setCategoryDialogOpen(false)}
              >
                取消
              </Button>
              <Button
                disabled={!categoryName.trim() || createCategory.isPending}
                onClick={() => createCategory.mutate()}
              >
                {createCategory.isPending ? '正在保存…' : '创建分类'}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </section>
    </main>
  )
}

function LibraryAssetDetails({
  asset,
  categories,
  onClose,
  onOpenSession,
}: {
  asset: StudioAsset
  categories: StudioLibraryCategory[]
  onClose: () => void
  onOpenSession: (sessionId: string, sourceRunId?: string) => void
}) {
  const queryClient = useQueryClient()
  const [moving, setMoving] = useState(false)
  const [targetCategory, setTargetCategory] = useState('root')
  const move = useMutation({
    mutationFn: () =>
      moveStudioLibraryAsset(
        asset.id,
        targetCategory === 'root' ? undefined : targetCategory
      ),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['studio', 'library'] })
      setMoving(false)
    },
  })
  const version =
    asset.versions.find((item) => item.version === asset.current_version) ??
    asset.versions.at(-1)
  const contentURL = version ? `${baseURL()}${version.content_url}` : undefined
  const source = useQuery({
    queryKey: ['studio', 'session', asset.session_id],
    queryFn: () => getStudioSession(asset.session_id),
    enabled: Boolean(asset.session_id),
  })
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onClose()
      }}
    >
      <DialogContent className='max-h-[calc(100vh-2rem)] overflow-y-auto sm:max-w-3xl'>
        <DialogHeader>
          <DialogTitle className='break-all'>{asset.name}</DialogTitle>
          <DialogDescription>
            {asset.kind === 'document'
              ? '文档'
              : asset.kind === 'image'
                ? '图片'
                : asset.kind === 'video'
                  ? '视频'
                  : asset.kind === 'audio'
                    ? '音频'
                    : '文件'}{' '}
            · 版本 {asset.current_version}
          </DialogDescription>
        </DialogHeader>
        <div className='flex h-[60vh] max-h-[40rem] min-h-48 items-center justify-center overflow-hidden rounded-md border bg-background'>
          <AssetPreview asset={asset} preview expanded />
        </div>
        <div className='flex flex-wrap items-center justify-between gap-3 border-t pt-4 text-sm'>
          <div className='min-w-0'>
            <p className='text-xs text-muted-foreground'>来源</p>
            {asset.session_id ? (
              <p className='truncate font-medium'>
                {source.data?.session.title ??
                  (source.isError ? '来源对话暂不可用' : '读取来源中…')}
              </p>
            ) : (
              <p className='font-medium'>{originLabel(asset.origin)}</p>
            )}
          </div>
          <div className='flex flex-wrap gap-2'>
            <Button
              variant='outline'
              size='sm'
              onClick={() => setMoving((value) => !value)}
            >
              移动到分类
            </Button>
            {asset.session_id && asset.origin !== 'user' ? (
              <Button
                variant='outline'
                size='sm'
                onClick={() =>
                  onOpenSession(asset.session_id, asset.source_run_id)
                }
              >
                {asset.source_run_id ? '定位生成对话' : '打开来源对话'}
              </Button>
            ) : null}
            {contentURL ? (
              <Button variant='outline' size='sm' asChild>
                <a href={contentURL} target='_blank' rel='noreferrer'>
                  打开原文件
                </a>
              </Button>
            ) : null}
          </div>
        </div>
        {moving ? (
          <div className='flex flex-wrap items-end gap-3 rounded-lg border bg-muted/30 p-3'>
            <div className='flex min-w-44 flex-1 flex-col gap-2'>
              <Label htmlFor='studio-library-target-category'>目标分类</Label>
              <Select value={targetCategory} onValueChange={setTargetCategory}>
                <SelectTrigger
                  id='studio-library-target-category'
                  className='w-full'
                  aria-label='目标分类'
                >
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value='root'>未分类</SelectItem>
                    {categories.map((category) => (
                      <SelectItem key={category.id} value={category.id}>
                        {category.name}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>
            <Button
              size='sm'
              disabled={move.isPending}
              onClick={() => move.mutate()}
            >
              {move.isPending ? '移动中…' : '移动资产'}
            </Button>
            {move.isError ? (
              <p role='alert' className='w-full text-sm text-destructive'>
                移动资产失败，重试移动。
              </p>
            ) : null}
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  )
}

function LibraryState({
  title,
  description,
  actionLabel,
  onAction,
}: {
  title: string
  description: string
  actionLabel?: string
  onAction?: () => void
}) {
  return (
    <div className='flex min-h-[420px] flex-col items-center justify-center px-6 text-center'>
      <span className='mb-4 flex size-12 items-center justify-center rounded-2xl bg-muted'>
        <Library className='size-5 text-muted-foreground' />
      </span>
      <h2 className='text-sm font-medium'>{title}</h2>
      <p className='mt-1 max-w-sm text-sm leading-6 text-muted-foreground'>
        {description}
      </p>
      {actionLabel && onAction ? (
        <Button variant='outline' size='sm' className='mt-4' onClick={onAction}>
          {actionLabel}
        </Button>
      ) : null}
    </div>
  )
}
