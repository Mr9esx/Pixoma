import { useEffect, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Download, FilePlus2, FileText, ImageIcon, Library, Pencil, Upload } from 'lucide-react'
import {
  getStudioTextAssetContent,
  listStudioLibraryFolders,
  type StudioAsset,
  type StudioLibraryFolder,
} from '@/lib/api/studio'
import { baseURL } from '@/lib/api/client'
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
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

type Props = {
  assets: StudioAsset[]
  onSaveToLibrary: (input: { assetId: string; folderId?: string }) => Promise<void>
  onCreateTextAsset?: (input: { name: string; content: string }) => void
  onUpdateTextAsset?: (input: { assetId: string; content: string }) => Promise<unknown>
  onUploadAsset?: (file: File) => void
  uploading?: boolean
}

export function StudioAssets({ assets, onSaveToLibrary, onCreateTextAsset, onUpdateTextAsset, onUploadAsset, uploading }: Props) {
  const [assetToSave, setAssetToSave] = useState<StudioAsset>()
  const folders = useQuery({
    queryKey: ['studio', 'library', 'folders'],
    queryFn: listStudioLibraryFolders,
  })
  if (assets.length === 0) {
    return (
      <div className='flex h-full flex-col items-center justify-center px-8 text-center'>
        <span className='mb-4 flex size-11 items-center justify-center rounded-xl bg-muted'>
          <FileText className='size-5 text-muted-foreground' />
        </span>
        <p className='text-sm font-medium'>当前 Session 还没有资产</p>
        <p className='mt-1 max-w-xs text-xs leading-5 text-muted-foreground'>
          上传文件、让模型生成内容，或执行工作流后，资产会自动汇总到这里。
        </p>
        <AssetActions onCreateTextAsset={onCreateTextAsset} onUploadAsset={onUploadAsset} uploading={uploading} empty />
      </div>
    )
  }
  return (
    <ScrollArea className='h-full'>
      <div className='flex items-center justify-between gap-3 px-4 pb-1 pt-4'>
        <p className='text-xs text-muted-foreground'>{assets.length} 项资产</p>
        <AssetActions onCreateTextAsset={onCreateTextAsset} onUploadAsset={onUploadAsset} uploading={uploading} />
      </div>
      <div className='grid gap-3 p-4 pt-3 sm:grid-cols-2 xl:grid-cols-1 2xl:grid-cols-2'>
        {assets.map((asset) => (
          <AssetCard
            key={asset.id}
            asset={asset}
            onSaveToLibrary={() => setAssetToSave(asset)}
            onUpdateTextAsset={onUpdateTextAsset}
          />
        ))}
      </div>
      <SaveAssetToLibraryDialog
        asset={assetToSave}
        folders={folders.data ?? []}
        foldersLoading={folders.isLoading}
        onOpenChange={(open) => {
          if (!open) setAssetToSave(undefined)
        }}
        onSave={onSaveToLibrary}
      />
    </ScrollArea>
  )
}

function AssetActions({
  onCreateTextAsset,
  onUploadAsset,
  uploading,
  empty,
}: {
  onCreateTextAsset?: (input: { name: string; content: string }) => void
  onUploadAsset?: (file: File) => void
  uploading?: boolean
  empty?: boolean
}) {
  const fileInputRef = useRef<HTMLInputElement>(null)
  return (
    <div className={empty ? 'mt-4 flex flex-wrap justify-center gap-2' : 'flex items-center gap-2'}>
      {onUploadAsset ? (
        <>
          <Button variant='outline' size='sm' disabled={uploading} onClick={() => fileInputRef.current?.click()}>
            <Upload />
            {uploading ? '正在上传…' : '上传资产'}
          </Button>
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
      {onCreateTextAsset ? <TextAssetDialog onCreate={onCreateTextAsset} /> : null}
    </div>
  )
}

function TextAssetDialog({ onCreate }: { onCreate: (input: { name: string; content: string }) => void }) {
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('创作笔记.md')
  const [content, setContent] = useState('')
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant='outline' size='sm'><FilePlus2 />新建文档</Button>
      </DialogTrigger>
      <DialogContent className='sm:max-w-xl'>
        <DialogHeader>
          <DialogTitle>新建 Session 文档</DialogTitle>
          <DialogDescription>文档属于当前 Session，可立即作为下一次对话或工作流的输入。</DialogDescription>
        </DialogHeader>
        <div className='space-y-4 py-2'>
          <Input aria-label='文档名称' value={name} onChange={(event) => setName(event.target.value)} placeholder='例如：角色设定.md' />
          <Textarea aria-label='文档内容' value={content} onChange={(event) => setContent(event.target.value)} placeholder='写下故事、角色、提示词或其他创作素材…' className='min-h-56 font-mono text-sm leading-6' />
        </div>
        <DialogFooter>
          <Button variant='outline' onClick={() => setOpen(false)}>取消</Button>
          <Button disabled={!content.trim()} onClick={() => { onCreate({ name, content }); setOpen(false); setContent('') }}>创建文档</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export function AssetCard({
  asset,
  onSaveToLibrary,
  onUpdateTextAsset,
}: {
  asset: StudioAsset
  onSaveToLibrary: (assetId: string) => void
  onUpdateTextAsset?: (input: { assetId: string; content: string }) => Promise<unknown>
}) {
  const version = asset.versions[asset.versions.length - 1]
  const contentURL = version ? `${baseURL()}${version.content_url}` : undefined
  return (
    <article className='group overflow-hidden rounded-xl border bg-card'>
      <div className='flex aspect-[16/10] items-center justify-center overflow-hidden bg-muted/50'>
        {asset.kind === 'image' && contentURL ? (
          <img
            src={contentURL}
            alt={asset.name}
            className='size-full object-cover transition-transform duration-300 group-hover:scale-[1.02]'
          />
        ) : (
          <FileText className='size-9 text-muted-foreground' />
        )}
      </div>
      <div className='space-y-3 p-3'>
        <div className='flex items-start gap-2'>
          <div className='min-w-0 flex-1'>
            <p className='truncate text-sm font-medium'>{asset.name}</p>
            <p className='mt-1 text-xs text-muted-foreground'>
              版本 {asset.current_version} · {originLabel(asset.origin)}
            </p>
          </div>
          <Badge variant='secondary' className='shrink-0'>
            {asset.kind === 'image' ? (
              <ImageIcon className='size-3' />
            ) : (
              <FileText className='size-3' />
            )}
            {kindLabel(asset.kind)}
          </Badge>
        </div>
        <div className='flex gap-1'>
          <Button variant='outline' size='sm' className='flex-1' asChild disabled={!contentURL}>
            <a href={contentURL} target='_blank' rel='noreferrer'>
              <Download />
              查看
            </a>
          </Button>
          <Button
            variant={asset.saved_to_library ? 'secondary' : 'outline'}
            size='icon-sm'
            onClick={() => onSaveToLibrary(asset.id)}
            disabled={asset.saved_to_library}
            aria-label={asset.saved_to_library ? '已存入资产库' : '存入资产库'}
          >
            <Library />
          </Button>
          {asset.kind === 'document' && onUpdateTextAsset ? (
            <TextAssetEditDialog asset={asset} onSave={onUpdateTextAsset} />
          ) : null}
        </div>
      </div>
    </article>
  )
}

function TextAssetEditDialog({
  asset,
  onSave,
}: {
  asset: StudioAsset
  onSave: (input: { assetId: string; content: string }) => Promise<unknown>
}) {
  const [open, setOpen] = useState(false)
  const [content, setContent] = useState('')
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const version = asset.versions[asset.versions.length - 1]

  useEffect(() => {
    if (!open || !version) return
    let active = true
    setLoading(true)
    setError('')
    void getStudioTextAssetContent(version.content_url)
      .then((value) => {
        if (active) setContent(value)
      })
      .catch((cause) => {
        if (active) setError(cause instanceof Error ? cause.message : '读取文档失败')
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [open, version])

  const save = async () => {
    setSaving(true)
    setError('')
    try {
      await onSave({ assetId: asset.id, content })
      setOpen(false)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : '保存新版本失败')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <Tooltip>
        <TooltipTrigger asChild>
          <DialogTrigger asChild>
            <Button variant='outline' size='icon-sm' aria-label={`编辑文档 ${asset.name}`}>
              <Pencil />
            </Button>
          </DialogTrigger>
        </TooltipTrigger>
        <TooltipContent side='top' sideOffset={6}>编辑文档</TooltipContent>
      </Tooltip>
      <DialogContent className='sm:max-w-xl'>
        <DialogHeader>
          <DialogTitle>编辑文档</DialogTitle>
          <DialogDescription>保存后追加一个新版本，已引用的旧版本不会变化。</DialogDescription>
        </DialogHeader>
        <Textarea
          aria-label='文档内容'
          value={content}
          onChange={(event) => setContent(event.target.value)}
          disabled={loading || saving}
          placeholder={loading ? '读取文档中…' : '写下新的内容…'}
          className='min-h-64 font-mono text-sm leading-6'
        />
        {error ? <p role='alert' className='text-sm text-destructive'>{error}</p> : null}
        <DialogFooter>
          <Button variant='outline' disabled={saving} onClick={() => setOpen(false)}>取消</Button>
          <Button disabled={loading || saving || !content.trim()} onClick={save}>{saving ? '保存中…' : '保存新版本'}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function SaveAssetToLibraryDialog({
  asset,
  folders,
  foldersLoading,
  onOpenChange,
  onSave,
}: {
  asset?: StudioAsset
  folders: StudioLibraryFolder[]
  foldersLoading: boolean
  onOpenChange: (open: boolean) => void
  onSave: (input: { assetId: string; folderId?: string }) => Promise<void>
}) {
  const [folderId, setFolderId] = useState('root')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const handleOpenChange = (open: boolean) => {
    if (!open && !saving) {
      setFolderId('root')
      setError('')
      onOpenChange(false)
    }
  }

  const save = async () => {
    if (!asset) return
    setSaving(true)
    setError('')
    try {
      await onSave({
        assetId: asset.id,
        folderId: folderId === 'root' ? undefined : folderId,
      })
      setFolderId('root')
      onOpenChange(false)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : '资产保存失败，请稍后重试。')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={Boolean(asset)} onOpenChange={handleOpenChange}>
      <DialogContent className='sm:max-w-md'>
        <DialogHeader>
          <DialogTitle>存入资产库</DialogTitle>
          <DialogDescription>
            选择资产库文件夹。保存后，这个资产可以在其他 Session 中作为输入引用。
          </DialogDescription>
        </DialogHeader>
        <div className='flex flex-col gap-2 py-2'>
          <Label htmlFor='studio-asset-library-folder'>资产库文件夹</Label>
          <Select value={folderId} onValueChange={setFolderId} disabled={saving}>
            <SelectTrigger id='studio-asset-library-folder'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='root'>根目录</SelectItem>
              {foldersLoading ? (
                <SelectItem value='loading' disabled>正在读取文件夹…</SelectItem>
              ) : null}
              {folders.map((folder) => (
                <SelectItem key={folder.id} value={folder.id}>{folder.name}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        {error ? <p role='alert' className='text-sm text-destructive'>{error}</p> : null}
        <DialogFooter>
          <Button variant='outline' disabled={saving} onClick={() => handleOpenChange(false)}>取消</Button>
          <Button disabled={saving} onClick={save}>{saving ? '正在保存…' : '存入资产库'}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function kindLabel(kind: StudioAsset['kind']) {
  return { document: '文档', image: '图片', video: '视频', audio: '音频', data: '数据', file: '文件' }[kind]
}

function originLabel(origin: StudioAsset['origin']) {
  return { user: '用户创建', agent: 'Agent 生成', model: '模型生成', workflow: '工作流产出', library: '资产库引用' }[origin]
}
