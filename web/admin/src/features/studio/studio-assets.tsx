import { useState } from 'react'
import { Download, FilePlus2, FileText, ImageIcon, Library, MoreHorizontal } from 'lucide-react'
import type { StudioAsset } from '@/lib/api/studio'
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
import { ScrollArea } from '@/components/ui/scroll-area'
import { Textarea } from '@/components/ui/textarea'

type Props = {
  assets: StudioAsset[]
  onSaveToLibrary: (assetId: string) => void
  onCreateTextAsset?: (input: { name: string; content: string }) => void
}

export function StudioAssets({ assets, onSaveToLibrary, onCreateTextAsset }: Props) {
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
        {onCreateTextAsset ? <TextAssetDialog onCreate={onCreateTextAsset} /> : null}
      </div>
    )
  }
  return (
    <ScrollArea className='h-full'>
      <div className='flex items-center justify-between gap-3 px-4 pb-1 pt-4'>
        <p className='text-xs text-muted-foreground'>{assets.length} 项资产</p>
        {onCreateTextAsset ? <TextAssetDialog onCreate={onCreateTextAsset} /> : null}
      </div>
      <div className='grid gap-3 p-4 pt-3 sm:grid-cols-2 xl:grid-cols-1 2xl:grid-cols-2'>
        {assets.map((asset) => (
          <AssetCard
            key={asset.id}
            asset={asset}
            onSaveToLibrary={onSaveToLibrary}
          />
        ))}
      </div>
    </ScrollArea>
  )
}

function TextAssetDialog({ onCreate }: { onCreate: (input: { name: string; content: string }) => void }) {
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('创作笔记.md')
  const [content, setContent] = useState('')
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant='outline' size='sm' className='mt-4'><FilePlus2 />新建文档</Button>
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
}: {
  asset: StudioAsset
  onSaveToLibrary: (assetId: string) => void
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
          <Button variant='ghost' size='icon-sm' aria-label='更多操作'>
            <MoreHorizontal />
          </Button>
        </div>
      </div>
    </article>
  )
}

function kindLabel(kind: StudioAsset['kind']) {
  return { document: '文档', image: '图片', video: '视频', audio: '音频', data: '数据', file: '文件' }[kind]
}

function originLabel(origin: StudioAsset['origin']) {
  return { user: '用户创建', agent: 'Agent 生成', model: '模型生成', workflow: '工作流产出', library: '资产库引用' }[origin]
}
