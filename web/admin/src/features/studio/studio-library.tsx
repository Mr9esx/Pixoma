import { useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  FolderPlus,
  Grid2X2,
  Library,
  List,
  Search,
  Upload,
} from 'lucide-react'
import { baseURL } from '@/lib/api/client'
import {
  createStudioLibraryFolder,
  getStudioSession,
  getStudioTextAssetContent,
  listStudioLibraryAssets,
  listStudioLibraryFolders,
  moveStudioLibraryAsset,
  uploadStudioAsset,
  type StudioAsset,
  type StudioLibraryFolder,
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
import { AssetCard } from './studio-assets'

export function StudioLibrary({
  onOpenSession,
}: {
  onOpenSession: (sessionId: string) => void
}) {
  const queryClient = useQueryClient()
  const [folderId, setFolderId] = useState<string>()
  const [folderDialogOpen, setFolderDialogOpen] = useState(false)
  const [folderName, setFolderName] = useState('')
  const [selectedAsset, setSelectedAsset] = useState<StudioAsset>()
  const [query, setQuery] = useState('')
  const [view, setView] = useState<'grid' | 'list'>('grid')
  const fileInputRef = useRef<HTMLInputElement>(null)
  const folders = useQuery({
    queryKey: ['studio', 'library', 'folders'],
    queryFn: listStudioLibraryFolders,
  })
  const assets = useQuery({
    queryKey: ['studio', 'library', folderId],
    queryFn: () => listStudioLibraryAssets(folderId),
  })
  const createFolder = useMutation({
    mutationFn: () => createStudioLibraryFolder({ name: folderName.trim() }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ['studio', 'library', 'folders'],
      })
      setFolderName('')
      setFolderDialogOpen(false)
    },
  })
  const upload = useMutation({
    mutationFn: (file: File) => uploadStudioAsset(file),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ['studio', 'library'] }),
  })
  const visibleAssets = (assets.data ?? []).filter((asset) => {
    const value = query.trim().toLocaleLowerCase()
    return (
      !value ||
      [asset.name, asset.kind, asset.origin].some((part) =>
        part.toLocaleLowerCase().includes(value)
      )
    )
  })

  return (
    <main id='main-content' className='min-h-0 min-w-0 flex-1 p-3 sm:p-4'>
      <section className='flex h-full min-h-0 flex-col overflow-hidden rounded-2xl border bg-card'>
        <header className='flex min-h-16 flex-wrap items-center justify-between gap-3 border-b px-5'>
          <div>
            <h1 className='text-sm font-semibold'>资产库</h1>
            <p className='text-xs text-muted-foreground'>
              管理跨 Session 复用的创作素材
            </p>
          </div>
          <div className='flex items-center gap-2'>
            <Button
              variant='outline'
              size='sm'
              onClick={() => setFolderDialogOpen(true)}
            >
              <FolderPlus />
              新建文件夹
            </Button>
            <Button
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
        <div className='flex items-center gap-3 border-b px-5 py-3'>
          <div className='relative max-w-md flex-1'>
            <Search className='absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground' />
            <Input
              className='pl-9'
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder='搜索资产名称、类型或来源…'
            />
          </div>
          <Button
            variant={view === 'grid' ? 'secondary' : 'ghost'}
            size='icon-sm'
            aria-label='网格视图'
            aria-pressed={view === 'grid'}
            onClick={() => setView('grid')}
          >
            <Grid2X2 />
          </Button>
          <Button
            variant={view === 'list' ? 'secondary' : 'ghost'}
            size='icon-sm'
            aria-label='列表视图'
            aria-pressed={view === 'list'}
            onClick={() => setView('list')}
          >
            <List />
          </Button>
        </div>
        <div className='flex items-center gap-2 overflow-x-auto border-b px-5 py-2'>
          <Button
            variant={folderId ? 'ghost' : 'secondary'}
            size='sm'
            onClick={() => setFolderId(undefined)}
          >
            全部资产
          </Button>
          {folders.data?.map((folder) => (
            <Button
              key={folder.id}
              variant={folderId === folder.id ? 'secondary' : 'ghost'}
              size='sm'
              onClick={() => setFolderId(folder.id)}
            >
              {folder.name}
            </Button>
          ))}
        </div>
        {folders.isError ? (
          <Alert variant='destructive' className='mx-5 mt-4 w-auto'>
            <AlertTitle>文件夹读取失败</AlertTitle>
            <AlertDescription>
              <p>文件夹列表暂不可用，资产仍可查看。</p>
              <Button
                variant='outline'
                size='sm'
                onClick={() => void folders.refetch()}
              >
                重试读取文件夹
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
        <ScrollArea className='min-h-0 flex-1'>
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
              className={
                view === 'grid'
                  ? 'grid gap-4 p-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4'
                  : 'grid gap-3 p-5'
              }
            >
              {visibleAssets.map((asset) => (
                <AssetCard
                  key={asset.id}
                  asset={asset}
                  onSaveToLibrary={() => undefined}
                  onOpenDetails={setSelectedAsset}
                />
              ))}
            </div>
          ) : assets.data?.length ? (
            <LibraryState
              title='没有匹配的资产'
              description='换个关键词试试，或清空搜索条件查看全部资产。'
            />
          ) : (
            <LibraryState
              title='资产库还是空的'
              description='你可以直接上传资产，也可以在对话中把 Session 资产存到这里。'
            />
          )}
        </ScrollArea>
        {selectedAsset ? (
          <LibraryAssetDetails
            asset={selectedAsset}
            folders={folders.data ?? []}
            onClose={() => setSelectedAsset(undefined)}
            onOpenSession={(sessionId) => {
              setSelectedAsset(undefined)
              onOpenSession(sessionId)
            }}
          />
        ) : null}
        <Dialog open={folderDialogOpen} onOpenChange={setFolderDialogOpen}>
          <DialogContent className='sm:max-w-md'>
            <DialogHeader>
              <DialogTitle>新建文件夹</DialogTitle>
              <DialogDescription>
                文件夹用于整理可跨 Session 复用的资产。
              </DialogDescription>
            </DialogHeader>
            <div className='grid gap-2 py-2'>
              <Label htmlFor='studio-folder-name'>文件夹名称</Label>
              <Input
                id='studio-folder-name'
                value={folderName}
                onChange={(event) => setFolderName(event.target.value)}
                placeholder='例如：角色设定'
              />
            </div>
            {createFolder.isError ? (
              <p role='alert' className='text-sm text-destructive'>
                文件夹创建失败，请重试。
              </p>
            ) : null}
            <DialogFooter>
              <Button
                variant='outline'
                onClick={() => setFolderDialogOpen(false)}
              >
                取消
              </Button>
              <Button
                disabled={!folderName.trim() || createFolder.isPending}
                onClick={() => createFolder.mutate()}
              >
                {createFolder.isPending ? '正在保存…' : '保存文件夹'}
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
  folders,
  onClose,
  onOpenSession,
}: {
  asset: StudioAsset
  folders: StudioLibraryFolder[]
  onClose: () => void
  onOpenSession: (sessionId: string) => void
}) {
  const queryClient = useQueryClient()
  const [moving, setMoving] = useState(false)
  const [targetFolder, setTargetFolder] = useState('root')
  const move = useMutation({
    mutationFn: () =>
      moveStudioLibraryAsset(
        asset.id,
        targetFolder === 'root' ? undefined : targetFolder
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
  const text = useQuery({
    queryKey: [
      'studio',
      'asset',
      asset.id,
      version?.id,
      version?.content_url,
      'content',
    ],
    queryFn: () => getStudioTextAssetContent(version!.content_url),
    enabled: Boolean(version && version.mime_type === 'text/markdown'),
  })

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onClose()
      }}
    >
      <DialogContent className='max-h-[calc(100vh-2rem)] overflow-y-auto sm:max-w-2xl'>
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
        <div className='flex min-h-48 items-center justify-center overflow-hidden rounded-lg border bg-muted/30'>
          {asset.kind === 'image' && contentURL ? (
            <img
              src={contentURL}
              alt={asset.name}
              className='max-h-[55vh] max-w-full object-contain'
            />
          ) : asset.kind === 'video' && contentURL ? (
            <video
              src={contentURL}
              controls
              className='max-h-[55vh] max-w-full'
              aria-label={asset.name}
            />
          ) : asset.kind === 'audio' && contentURL ? (
            <audio
              src={contentURL}
              controls
              className='w-full px-4'
              aria-label={asset.name}
            />
          ) : version?.mime_type === 'text/markdown' ? (
            text.isLoading ? (
              <p className='text-sm text-muted-foreground'>读取文档中…</p>
            ) : text.isError ? (
              <Button variant='outline' onClick={() => void text.refetch()}>
                读取失败，重试
              </Button>
            ) : (
              <pre className='max-h-[55vh] w-full overflow-auto p-4 text-sm leading-6 break-words whitespace-pre-wrap'>
                {text.data}
              </pre>
            )
          ) : (
            <p className='text-sm text-muted-foreground'>
              此格式可打开原文件查看。
            </p>
          )}
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
              <p className='font-medium'>用户上传</p>
            )}
          </div>
          <div className='flex flex-wrap gap-2'>
            <Button
              variant='outline'
              size='sm'
              onClick={() => setMoving((value) => !value)}
            >
              移动到文件夹
            </Button>
            {asset.session_id ? (
              <Button
                variant='outline'
                size='sm'
                onClick={() => onOpenSession(asset.session_id)}
              >
                打开来源对话
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
              <Label htmlFor='studio-library-target-folder'>目标文件夹</Label>
              <Select value={targetFolder} onValueChange={setTargetFolder}>
                <SelectTrigger
                  id='studio-library-target-folder'
                  className='w-full'
                  aria-label='目标文件夹'
                >
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value='root'>未分类</SelectItem>
                    {folders.map((folder) => (
                      <SelectItem key={folder.id} value={folder.id}>
                        {folder.name}
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
