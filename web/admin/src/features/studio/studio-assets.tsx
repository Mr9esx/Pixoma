import { useEffect, useRef, useState, type ReactNode } from 'react'
import { EditorView } from '@codemirror/view'
import { useQuery } from '@tanstack/react-query'
import {
  AudioLines,
  Download,
  Ellipsis,
  Eye,
  File,
  FileJson,
  FilePlus2,
  FileText,
  Grid2X2,
  ImageIcon,
  Library,
  List,
  Maximize2,
  MessageSquareText,
  Minimize2,
  Pencil,
  Rows3,
  Upload,
  Video,
  type LucideProps,
} from 'lucide-react'
import { baseURL } from '@/lib/api/client'
import {
  getStudioTextAssetContent,
  type StudioAsset,
  type StudioAssetPalette,
  type StudioMessage,
} from '@/lib/api/studio'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { IconButtonTooltip } from '@/components/ui/icon-button-tooltip'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import { Textarea } from '@/components/ui/textarea'
import { MessageResponse } from '@/components/ai-elements/message'
import { CodeEditor } from '@/components/code-editor'
import { formatSize } from './studio-asset-format'
import { originLabel } from './studio-asset-origin'
import styles from './studio-assets.module.css'

export type AssetLayout = 'adaptive' | 'single' | 'list'

const assetKindIcons = {
  image: ImageIcon,
  document: FileText,
  video: Video,
  audio: AudioLines,
  data: FileJson,
  file: File,
}

export function AssetKindIcon({ kind, ...props }: { kind: StudioAsset['kind'] } & LucideProps) {
  const Icon = assetKindIcons[kind]
  return <Icon {...props} />
}

type Props = {
  assets: StudioAsset[]
  layout?: AssetLayout
  focusAsset?: { id: string; request: number }
  onFocusAssetClose?: (request: number) => void
  messages?: StudioMessage[]
  onLocateMessage?: (messageId: string) => void
  onUpdateTextAsset?: (input: {
    assetId: string
    content: string
    requestId: string
  }) => Promise<unknown>
}

export function StudioAssets({
  assets,
  layout = 'adaptive',
  focusAsset,
  onFocusAssetClose,
  messages,
  onLocateMessage,
  onUpdateTextAsset,
}: Props) {
  const [assetToView, setAssetToView] = useState<StudioAsset>()
  const focusedAsset = focusAsset && assets.find((asset) => asset.id === focusAsset.id)
  return (
    <div className='flex h-full min-h-0 flex-col'>
      {assets.length === 0 ? (
        <div className='flex min-h-0 flex-1 flex-col items-center justify-center px-8 text-center'>
          <p className='text-sm text-muted-foreground'>暂无资产</p>
        </div>
      ) : (
        <ScrollArea className='min-h-0 flex-1'>
          <div
            className={cn(
              'grid min-w-0 gap-3 p-4 pt-3',
              layout === 'adaptive' ? styles.adaptiveGrid : 'grid-cols-1'
            )}
          >
            {assets.map((asset) => {
              const sourceRunId = asset.usages?.find(
                (usage) => usage.run_id && usage.usage_kind !== 'referenced'
              )?.run_id
              const sourceMessageId =
                asset.origin !== 'user' && sourceRunId
                  ? messages?.find(
                      (message) =>
                        message.role === 'user' &&
                        message.run_id === sourceRunId
                    )?.id
                  : undefined
              return (
                <AssetCard
                  key={asset.id}
                  asset={asset}
                  preview
                  layout={layout === 'list' ? 'list' : 'grid'}
                  onOpenDetails={setAssetToView}
                  onUpdateTextAsset={onUpdateTextAsset}
                  onLocateSource={
                    sourceMessageId && onLocateMessage
                      ? () => onLocateMessage(sourceMessageId)
                      : undefined
                  }
                />
              )
            })}
          </div>
        </ScrollArea>
      )}
      <div className='shrink-0 px-4 py-2'>
        <p className='text-xs text-muted-foreground'>
          共 {assets.length} 项资产
        </p>
      </div>
      <SessionAssetDetailsDialog
        asset={focusedAsset || assetToView}
        onOpenChange={(open) => {
          if (!open) {
            setAssetToView(undefined)
            if (focusAsset) onFocusAssetClose?.(focusAsset.request)
          }
        }}
      />
    </div>
  )
}

export function StudioAssetActions({
  onCreateTextAsset,
  onUploadAsset,
  onOpenLibrary,
  uploading,
  layout,
  onLayoutChange,
}: {
  onCreateTextAsset?: (input: {
    name: string
    content: string
    requestId: string
  }) => Promise<unknown>
  onUploadAsset?: (file: File) => void
  onOpenLibrary?: () => void
  uploading?: boolean
  layout: AssetLayout
  onLayoutChange: (layout: AssetLayout) => void
}) {
  const fileInputRef = useRef<HTMLInputElement>(null)
  return (
    <div data-slot='studio-asset-actions' className='flex items-center gap-1'>
      {onUploadAsset ? (
        <>
          <IconButtonTooltip label={uploading ? '正在上传…' : '上传资产'}>
            <Button variant='ghost' size='icon' aria-label='上传资产' disabled={uploading} onClick={() => fileInputRef.current?.click()}><Upload /></Button>
          </IconButtonTooltip>
          <input
            ref={fileInputRef}
            type='file'
            className='sr-only'
            onChange={(event) => {
              const file = event.target.files?.[0]
              if (file) onUploadAsset(file)
              event.currentTarget.value = ''
            }}
          />
        </>
      ) : null}
      {onCreateTextAsset ? (
        <TextAssetDialog onCreate={onCreateTextAsset} />
      ) : null}
      {onOpenLibrary ? <IconButtonTooltip label='查看项目资产'>
        <Button variant='ghost' size='icon' aria-label='查看项目资产' onClick={onOpenLibrary}><Library /></Button>
      </IconButtonTooltip> : null}
      <DropdownMenu>
        <IconButtonTooltip label={layout === 'adaptive' ? '网格视图' : layout === 'single' ? '单列视图' : '列表视图'}>
          <DropdownMenuTrigger asChild>
            <Button variant='ghost' size='icon' aria-label={layout === 'adaptive' ? '网格视图' : layout === 'single' ? '单列视图' : '列表视图'}>
              {layout === 'adaptive' ? <Grid2X2 /> : layout === 'single' ? <Rows3 /> : <List />}
            </Button>
          </DropdownMenuTrigger>
        </IconButtonTooltip>
        <DropdownMenuContent align='end'>
          <DropdownMenuRadioGroup value={layout} onValueChange={(value) => onLayoutChange(value as AssetLayout)}>
            <DropdownMenuRadioItem value='adaptive'><Grid2X2 />网格视图</DropdownMenuRadioItem>
            <DropdownMenuRadioItem value='list'><List />列表视图</DropdownMenuRadioItem>
            <DropdownMenuRadioItem value='single'><Rows3 />单列视图</DropdownMenuRadioItem>
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}

function TextAssetDialog({
  onCreate,
}: {
  onCreate: (input: { name: string; content: string; requestId: string }) => Promise<unknown>
}) {
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('创作笔记.md')
  const [content, setContent] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const requestId = useRef(crypto.randomUUID())
  const save = async () => {
    setSaving(true)
    setError('')
    try {
      await onCreate({ name, content, requestId: requestId.current })
      setOpen(false)
      setContent('')
      requestId.current = crypto.randomUUID()
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : '创建文档失败，重试保存。'
      )
    } finally {
      setSaving(false)
    }
  }
  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!saving) setOpen(nextOpen)
      }}
    >
      <IconButtonTooltip label='新建文档'>
        <DialogTrigger asChild>
          <Button variant='ghost' size='icon' aria-label='新建文档'><FilePlus2 /></Button>
        </DialogTrigger>
      </IconButtonTooltip>
      <DialogContent className='sm:max-w-xl'>
        <DialogHeader>
          <DialogTitle>新建会话文档</DialogTitle>
          <DialogDescription>
            文档属于当前会话，可立即作为下一次对话或工作流的输入。
          </DialogDescription>
        </DialogHeader>
        <div className='space-y-4 py-2'>
          <Input
            aria-label='文档名称'
            value={name}
            onChange={(event) => { setName(event.target.value); requestId.current = crypto.randomUUID() }}
            placeholder='例如：角色设定.md'
            disabled={saving}
          />
          <Textarea
            aria-label='文档内容'
            value={content}
            onChange={(event) => { setContent(event.target.value); requestId.current = crypto.randomUUID() }}
            placeholder='写下故事、角色、提示词或其他创作素材…'
            className='min-h-56 font-mono text-sm leading-6'
            disabled={saving}
          />
        </div>
        {error ? (
          <p role='alert' className='text-sm text-destructive'>
            {error}
          </p>
        ) : null}
        <DialogFooter>
          <Button
            variant='outline'
            disabled={saving}
            onClick={() => setOpen(false)}
          >
            取消
          </Button>
          <Button
            disabled={!name.trim() || !content.trim() || saving}
            onClick={() => void save()}
          >
            {saving ? '保存中…' : '创建文档'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export function AssetCard({
  asset,
  preview = false,
  lazyText = false,
  layout = 'grid',
  onOpenDetails,
  onSelect,
  selected = false,
  onLocateSource,
  onUpdateTextAsset,
}: {
  asset: StudioAsset
  preview?: boolean
  lazyText?: boolean
  layout?: 'grid' | 'list'
  onOpenDetails?: (asset: StudioAsset) => void
  onSelect?: () => void
  selected?: boolean
  onLocateSource?: () => void
  onUpdateTextAsset?: (input: {
    assetId: string
    content: string
    requestId: string
  }) => Promise<unknown>
}) {
  const version = asset.versions[asset.versions.length - 1]
  const contentURL = version ? `${baseURL()}${version.content_url}` : undefined
  const [editing, setEditing] = useState(false)
  const canEdit = asset.kind === 'document' && Boolean(onUpdateTextAsset)
  const secondaryActionCount =
    Number(Boolean(contentURL)) +
    Number(Boolean(onLocateSource)) +
    Number(canEdit)
  return (
    <article
      className={cn(
        'group relative min-w-0 overflow-hidden rounded-lg border bg-card',
        styles.card,
        layout === 'list' && 'flex',
        onSelect && 'hover:border-primary/60',
        selected && 'border-primary bg-accent/30'
      )}
    >
      {onSelect ? <Button type='button' variant='ghost' size='icon' className='absolute inset-0 z-10 h-full w-full rounded-lg opacity-0 focus-visible:opacity-100 focus-visible:ring-2 focus-visible:ring-ring' aria-label={selected ? `取消选择 ${asset.name}` : `选择 ${asset.name}`} aria-pressed={selected} onClick={onSelect} onDoubleClick={() => onOpenDetails?.(asset)} onKeyDown={(event) => { if (event.key === 'Enter' && onOpenDetails) { event.preventDefault(); onOpenDetails(asset) } }} /> : null}
      <div
        className={cn(
          'aspect-[16/10] px-3 pt-3',
          layout === 'list' && 'aspect-square w-28 shrink-0 p-2'
        )}
      >
        <div className='flex size-full items-center justify-center overflow-hidden rounded-md border bg-muted/50'>
          <AssetPreview
            asset={asset}
            preview={preview}
            lazyText={lazyText}
            compact={layout === 'list'}
          />
        </div>
      </div>
      <div
        className={cn(
          'min-w-0 space-y-3 p-3',
          layout === 'list' &&
            'flex flex-1 flex-col justify-between gap-3 space-y-0'
        )}
      >
        <div className='flex items-start gap-2'>
          <div className='min-w-0 flex-1'>
            <p className='truncate text-sm font-medium'>{asset.name}</p>
            <p className='mt-1 text-xs text-muted-foreground'>
              版本 {asset.current_version} · {originLabel(asset.origin)}
            </p>
          </div>
          <Badge variant='secondary' className='shrink-0'>
            <AssetKindIcon kind={asset.kind} />
            {kindLabel(asset.kind)}
          </Badge>
        </div>
        <div
          className={cn('relative z-20 flex items-center gap-1', styles.actions)}
          data-secondary-count={secondaryActionCount}
        >
          {onOpenDetails ? (
            <Button
              variant='outline'
              size='sm'
              className='min-w-0 flex-1'
              onClick={() => onOpenDetails(asset)}
            >
              <Eye />
              查看
            </Button>
          ) : (
            <Button
              variant='outline'
              size='sm'
              className='min-w-0 flex-1'
              asChild
              disabled={!contentURL}
            >
              <a href={contentURL} target='_blank' rel='noreferrer'>
                <Eye />
                查看
              </a>
            </Button>
          )}
          <div
            className={cn(
              'flex shrink-0 items-center gap-1',
              styles.secondaryActions
            )}
          >
            {contentURL ? (
              <IconButtonTooltip label='下载资产'>
                <Button variant='ghost' size='icon-sm' asChild>
                  <a
                    href={contentURL}
                    download={asset.name}
                    aria-label='下载资产'
                  >
                    <Download />
                  </a>
                </Button>
              </IconButtonTooltip>
            ) : null}
            {onLocateSource ? (
              <IconButtonTooltip label='定位生成对话'>
                <Button
                  variant='ghost'
                  size='icon-sm'
                  onClick={onLocateSource}
                  aria-label='定位生成对话'
                >
                  <MessageSquareText />
                </Button>
              </IconButtonTooltip>
            ) : null}
            {canEdit ? (
              <IconButtonTooltip label='编辑文档'>
                <Button
                  variant='ghost'
                  size='icon-sm'
                  onClick={() => setEditing(true)}
                  aria-label={`编辑文档 ${asset.name}`}
                >
                  <Pencil />
                </Button>
              </IconButtonTooltip>
            ) : null}
          </div>
          <DropdownMenu>
            <IconButtonTooltip label='更多操作'>
              <DropdownMenuTrigger asChild>
                <Button
                  variant='ghost'
                  size='icon-sm'
                  className={styles.moreActions}
                  aria-label='更多操作'
                >
                  <Ellipsis />
                </Button>
              </DropdownMenuTrigger>
            </IconButtonTooltip>
            <DropdownMenuContent align='end'>
              {contentURL ? (
                <DropdownMenuItem asChild>
                  <a href={contentURL} download={asset.name}>
                    <Download />
                    下载资产
                  </a>
                </DropdownMenuItem>
              ) : null}
              {onLocateSource ? (
                <DropdownMenuItem onSelect={onLocateSource}>
                  <MessageSquareText />
                  定位生成对话
                </DropdownMenuItem>
              ) : null}
              {canEdit ? (
                <DropdownMenuItem onSelect={() => setEditing(true)}>
                  <Pencil />
                  编辑文档
                </DropdownMenuItem>
              ) : null}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
      {canEdit && onUpdateTextAsset ? (
        <TextAssetEditDialog
          asset={asset}
          onSave={onUpdateTextAsset}
          open={editing}
          onOpenChange={setEditing}
        />
      ) : null}
    </article>
  )
}

export function AssetPreview({
  asset,
  preview,
  lazyText = false,
  compact = false,
  expanded = false,
}: {
  asset: StudioAsset
  preview: boolean
  lazyText?: boolean
  compact?: boolean
  expanded?: boolean
}) {
  const version = asset.versions[asset.versions.length - 1]
  const contentURL = version ? `${baseURL()}${version.content_url}` : undefined
  if (version?.mime_type.startsWith('image/') && contentURL) {
    return (
      <img
        src={contentURL}
        alt={asset.name}
        className={cn(
          'size-full transition-transform duration-300',
          expanded ? 'object-contain' : 'object-cover group-hover:scale-[1.02]'
        )}
      />
    )
  }
  if (preview && version?.mime_type.startsWith('video/') && contentURL) {
    return (
      <video
        src={contentURL}
        controls={!compact}
        playsInline
        preload='metadata'
        aria-label={asset.name}
        className={cn('size-full', expanded ? 'object-contain' : 'object-cover')}
      />
    )
  }
  if (compact) return <AssetKindIcon kind={asset.kind} className='size-7 text-muted-foreground' />
  if (preview && version?.mime_type.startsWith('audio/') && contentURL) {
    return (
      <audio
        src={contentURL}
        controls
        preload='metadata'
        aria-label={asset.name}
        className='w-full px-3'
      />
    )
  }
  if (preview && version?.mime_type === 'application/pdf' && contentURL) {
    return (
      <iframe
        src={contentURL}
        title={asset.name}
        loading='lazy'
        className='size-full border-0'
      />
    )
  }
  if (
    preview &&
    version &&
    (version.mime_type.startsWith('text/') ||
      version.mime_type === 'application/json')
  ) {
    return (
      <TextAssetPreview
        asset={asset}
        contentURL={version.content_url}
        mimeType={version.mime_type}
        expanded={expanded}
        lazy={lazyText}
      />
    )
  }
  return <AssetKindIcon kind={asset.kind} className='size-9 text-muted-foreground' />
}

export function AssetDetailPreview({ asset, expanded, onExpandedChange }: {
  asset: StudioAsset
  expanded: boolean
  onExpandedChange?: (expanded: boolean) => void
}) {
  return (
    <div className={cn('relative flex min-h-72 min-w-0 items-center justify-center overflow-hidden bg-background lg:w-2/3', expanded && 'lg:w-full')}>
      <AssetPreview asset={asset} preview expanded />
      {onExpandedChange ? <IconButtonTooltip label={expanded ? '恢复详情' : '放大资产'}>
        <Button variant='outline' size='icon' className='absolute top-4 right-4' aria-label={expanded ? '恢复详情' : '放大资产'} onClick={() => onExpandedChange(!expanded)}>
          {expanded ? <Minimize2 /> : <Maximize2 />}
        </Button>
      </IconButtonTooltip> : null}
    </div>
  )
}

export function AssetDetailsDialog({ open, onOpenChange, asset, expanded, onExpandedChange, description, title = '资产详情', previewContent, nameAction, nameEditor, footer, fallback, children }: {
  open: boolean
  onOpenChange: (open: boolean) => void
  asset?: StudioAsset
  expanded?: boolean
  onExpandedChange?: (expanded: boolean) => void
  description: string
  title?: string
  previewContent?: ReactNode
  nameAction?: ReactNode
  nameEditor?: ReactNode
  footer?: ReactNode
  fallback?: ReactNode
  children?: ReactNode
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='flex h-[min(94svh,1100px)] w-[calc(100vw-2rem)] flex-col gap-0 overflow-hidden p-0 sm:max-w-[1500px]'>
        <DialogHeader className={cn('shrink-0 border-b px-5 py-4 pr-12', expanded && 'sr-only')}>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        {asset ? <div className='flex min-h-0 flex-1 flex-col lg:flex-row'>
          {previewContent ?? <AssetDetailPreview asset={asset} expanded={Boolean(expanded)} onExpandedChange={onExpandedChange} />}
          {!expanded ? <div className='flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden border-t lg:w-1/3 lg:flex-none lg:border-t-0 lg:border-l'>
            <ScrollArea className='min-h-0 flex-1'>
              <div className='flex flex-col gap-5 p-5'>
                <section>{nameEditor ?? <div className='flex items-start justify-between gap-2'>
                  <h2 className='min-w-0 break-all text-base font-semibold'>{asset.name}</h2>
                  {nameAction}
                </div>}</section>
                {children}
              </div>
            </ScrollArea>
            {footer ? <footer className='shrink-0 border-t px-5 py-3'>{footer}</footer> : null}
          </div> : null}
        </div> : fallback}
      </DialogContent>
    </Dialog>
  )
}

export function AssetDetailsInfo({ children }: { children: ReactNode }) {
  return <section className='flex flex-col gap-2'>
    <h2 className='text-sm font-semibold'>基本信息</h2>
    <div className='grid grid-cols-[5rem_minmax(0,1fr)] items-center gap-x-3 gap-y-3 text-sm tabular-nums'>{children}</div>
  </section>
}

export function AssetInfoRow({ label, value }: { label: string; value: ReactNode }) {
  return <><span className='text-muted-foreground'>{label}</span><div className='min-w-0 break-all text-sm tabular-nums'>{value}</div></>
}

export function AssetSizeRow({ bytes }: { bytes?: number }) {
  return <AssetInfoRow label='文件大小' value={bytes === undefined ? '—' : <span className='flex flex-col'>
    <span>{formatSize(bytes)}</span><span className='text-xs text-muted-foreground'>{bytes.toLocaleString()} 字节</span>
  </span>} />
}

export function AssetPaletteRow({ palette, onRetry, retrying = false }: {
  palette?: StudioAssetPalette | null
  onRetry?: () => void
  retrying?: boolean
}) {
  return <AssetInfoRow label='调色盘' value={palette?.status === 'ready' ? <div className='flex flex-wrap items-center gap-1.5' aria-label='调色盘'>
    {palette.colors.map((color) => <span key={color.hex} role='img' aria-label={`${color.hex}，${Math.round(color.ratio * 100)}%`} title={`${color.hex} · ${Math.round(color.ratio * 100)}%`} className='size-5 rounded-full border' style={{ backgroundColor: color.hex }} />)}
  </div> : palette?.status === 'failed' ? <div className='flex items-center gap-2'>
    分析失败{onRetry ? <Button variant='outline' size='sm' disabled={retrying} onClick={onRetry}>重新分析</Button> : null}
  </div> : <span className='text-muted-foreground'>{palette?.status === 'pending' || palette?.status === 'running' ? '分析中…' : '暂无调色盘'}</span>} />
}

function SessionAssetDetailsDialog({ asset, onOpenChange }: { asset?: StudioAsset; onOpenChange: (open: boolean) => void }) {
  const [expanded, setExpanded] = useState(false)
  const version = asset?.versions[asset.versions.length - 1]
  return <AssetDetailsDialog
    open={Boolean(asset)}
    onOpenChange={(open) => { if (!open) setExpanded(false); onOpenChange(open) }}
    asset={asset}
    expanded={expanded}
    onExpandedChange={setExpanded}
    description={asset ? `版本 ${asset.current_version} · ${originLabel(asset.origin)}` : ''}
    footer={version && asset ? <div className='flex flex-wrap gap-2'>
      <Button variant='outline' size='sm' asChild><a href={`${baseURL()}${version.content_url}`} download={asset.name}><Download />下载资产</a></Button>
      <Button variant='outline' size='sm' asChild><a href={`${baseURL()}${version.content_url}`} target='_blank' rel='noreferrer'>打开原文件</a></Button>
    </div> : undefined}
  >
    <AssetDetailsInfo>
      <AssetInfoRow label='版本' value={asset?.current_version ?? '—'} />
      <AssetInfoRow label='来源' value={asset ? originLabel(asset.origin) : '—'} />
      <AssetSizeRow bytes={version?.size_bytes} />
      <AssetInfoRow label='格式' value={version?.format?.toUpperCase() ?? version?.mime_type ?? '—'} />
      {asset && (asset.kind === 'image' || asset.kind === 'video') ? <AssetPaletteRow palette={version?.palette} /> : null}
    </AssetDetailsInfo>
  </AssetDetailsDialog>
}

function TextAssetPreview({
  asset,
  contentURL,
  mimeType,
  expanded,
  lazy,
}: {
  asset: StudioAsset
  contentURL: string
  mimeType: string
  expanded: boolean
  lazy: boolean
}) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [visible, setVisible] = useState(!lazy || expanded)
  useEffect(() => {
    if (!lazy || visible || expanded || !containerRef.current) return
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) setVisible(true)
      },
      { rootMargin: '120px' }
    )
    observer.observe(containerRef.current)
    return () => observer.disconnect()
  }, [expanded, lazy, visible])
  const text = useQuery({
    queryKey: ['studio', 'asset', asset.id, contentURL, 'content'],
    queryFn: () => getStudioTextAssetContent(contentURL),
    enabled: !lazy || expanded || visible,
    staleTime: Infinity,
  })

  if (lazy && !expanded && !visible) {
    return (
      <div
        ref={containerRef}
        className='flex size-full items-center justify-center'
      >
        <AssetKindIcon kind={asset.kind} className='size-9 text-muted-foreground' />
      </div>
    )
  }
  if (text.isPending) return <Skeleton className='size-full rounded-none' />
  if (text.isError) {
    return (
      <Button variant='ghost' size='sm' onClick={() => void text.refetch()}>
        重新读取预览
      </Button>
    )
  }
  const content: unknown = text.data
  const value =
    typeof content === 'string' ? content : JSON.stringify(content, null, 2)
  if (mimeType.startsWith('text/')) {
    return (
      <div className='size-full overflow-y-auto p-3 text-left text-xs leading-5 break-words'>
        <MessageResponse className='h-auto min-h-full w-full [&_h1]:mt-0 [&_h1]:mb-1 [&_h1]:text-sm [&_h2]:mt-1 [&_h2]:mb-1 [&_h2]:text-xs [&_h3]:text-xs [&_p]:my-1'>
          {value}
        </MessageResponse>
      </div>
    )
  }
  return (
    <pre className='size-full overflow-y-auto p-3 text-left font-mono text-xs leading-5 break-words whitespace-pre-wrap'>
      {value}
    </pre>
  )
}

function TextAssetEditDialog({
  asset,
  onSave,
  open,
  onOpenChange,
}: {
  asset: StudioAsset
  onSave: (input: { assetId: string; content: string; requestId: string }) => Promise<unknown>
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const [draft, setDraft] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const requestId = useRef(crypto.randomUUID())
  const version = asset.versions[asset.versions.length - 1]

  const text = useQuery({
    queryKey: ['studio', 'asset', asset.id, version?.content_url, 'content'],
    queryFn: () => getStudioTextAssetContent(version.content_url),
    enabled: open && Boolean(version),
    staleTime: Infinity,
  })
  const content = draft ?? text.data ?? ''

  const changeOpen = (nextOpen: boolean) => {
    if (!nextOpen) {
      setDraft(null)
      setError('')
    }
    onOpenChange(nextOpen)
  }

  const save = async () => {
    setSaving(true)
    setError('')
    try {
      await onSave({ assetId: asset.id, content, requestId: requestId.current })
      changeOpen(false)
      requestId.current = crypto.randomUUID()
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : '保存新版本失败')
    } finally {
      setSaving(false)
    }
  }

  return (
    <AssetDetailsDialog open={open} onOpenChange={changeOpen} asset={asset} title='编辑文档' description='保存后创建新版本'
      previewContent={<div className='flex min-h-72 min-w-0 flex-1 flex-col overflow-hidden bg-background lg:w-2/3'>
        {text.isPending ? <Skeleton className='size-full rounded-none' /> : text.isError ? <div className='flex size-full flex-col items-center justify-center gap-3 text-sm'>
          <p role='alert' className='text-destructive'>读取文档失败：{text.error.message}</p>
          <Button variant='outline' size='sm' onClick={() => void text.refetch()}>重新读取</Button>
        </div> : <CodeEditor className='rounded-none border-0' language='markdown' extensions={[EditorView.contentAttributes.of({ 'aria-label': '文档内容' })]} fill value={content} editable={!saving} onChange={(value) => { setDraft(value); requestId.current = crypto.randomUUID() }} />}
      </div>}
      footer={<div className='flex flex-col gap-3'>
        {error ? <p role='alert' className='text-sm text-destructive'>{error}</p> : null}
        <div className='flex justify-end gap-2'>
          <Button variant='outline' size='sm' disabled={saving} onClick={() => changeOpen(false)}>取消</Button>
          <Button size='sm' disabled={!text.isSuccess || saving || !content.trim()} onClick={save}>{saving ? '保存中…' : '保存新版本'}</Button>
        </div>
      </div>}
    >
      <AssetDetailsInfo>
        <AssetInfoRow label='版本' value={asset.current_version} />
        <AssetInfoRow label='来源' value={originLabel(asset.origin)} />
        <AssetSizeRow bytes={version?.size_bytes} />
        <AssetInfoRow label='格式' value={version?.format?.toUpperCase() ?? version?.mime_type ?? '—'} />
      </AssetDetailsInfo>
    </AssetDetailsDialog>
  )
}

function kindLabel(kind: StudioAsset['kind']) {
  return {
    document: '文档',
    image: '图片',
    video: '视频',
    audio: '音频',
    data: '数据',
    file: '文件',
  }[kind]
}
