import { Download, FileText, ImageIcon, Library, MoreHorizontal } from 'lucide-react'
import type { StudioAsset } from '@/lib/api/studio'
import { baseURL } from '@/lib/api/client'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'

type Props = {
  assets: StudioAsset[]
  onSaveToLibrary: (assetId: string) => void
}

export function StudioAssets({ assets, onSaveToLibrary }: Props) {
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
      </div>
    )
  }
  return (
    <ScrollArea className='h-full'>
      <div className='grid gap-3 p-4 sm:grid-cols-2 xl:grid-cols-1 2xl:grid-cols-2'>
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
