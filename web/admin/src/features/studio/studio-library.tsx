import { useQuery } from '@tanstack/react-query'
import { FolderPlus, Grid2X2, Library, List, Search, Upload } from 'lucide-react'
import { listStudioLibraryAssets } from '@/lib/api/studio'
import { AssetCard } from './studio-assets'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'

export function StudioLibrary() {
  const assets = useQuery({
    queryKey: ['studio', 'library'],
    queryFn: () => listStudioLibraryAssets(),
  })

  return (
    <main id='main-content' className='min-h-0 min-w-0 flex-1 bg-muted/30 p-3 sm:p-4'>
      <section className='flex h-full min-h-0 flex-col overflow-hidden rounded-2xl border bg-card'>
      <header className='flex min-h-16 flex-wrap items-center justify-between gap-3 border-b px-5'>
        <div>
          <h1 className='text-sm font-semibold'>资产库</h1>
          <p className='text-xs text-muted-foreground'>管理跨 Session 复用的创作素材</p>
        </div>
        <div className='flex items-center gap-2'>
          <Button variant='outline' size='sm'>
            <FolderPlus />
            新建文件夹
          </Button>
          <Button size='sm'>
            <Upload />
            上传资产
          </Button>
        </div>
      </header>
      <div className='flex items-center gap-3 border-b px-5 py-3'>
        <div className='relative max-w-md flex-1'>
          <Search className='absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground' />
          <Input className='pl-9' placeholder='搜索资产名称、类型或来源…' />
        </div>
        <Button variant='secondary' size='icon-sm' aria-label='网格视图'>
          <Grid2X2 />
        </Button>
        <Button variant='ghost' size='icon-sm' aria-label='列表视图'>
          <List />
        </Button>
      </div>
      <ScrollArea className='min-h-0 flex-1'>
        {assets.isLoading ? (
          <div className='grid gap-4 p-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4'>
            {Array.from({ length: 8 }).map((_, index) => (
              <Skeleton key={index} className='aspect-[4/3] rounded-xl' />
            ))}
          </div>
        ) : assets.isError ? (
          <LibraryState title='资产读取失败' description='请稍后重试，已有资产不会丢失。' />
        ) : assets.data?.length ? (
          <div className='grid gap-4 p-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4'>
            {assets.data.map((asset) => (
              <AssetCard key={asset.id} asset={asset} onSaveToLibrary={() => undefined} />
            ))}
          </div>
        ) : (
          <LibraryState title='资产库还是空的' description='你可以直接上传资产，也可以在对话中把 Session 资产存到这里。' />
        )}
      </ScrollArea>
      </section>
    </main>
  )
}

function LibraryState({ title, description }: { title: string; description: string }) {
  return (
    <div className='flex min-h-[420px] flex-col items-center justify-center px-6 text-center'>
      <span className='mb-4 flex size-12 items-center justify-center rounded-2xl bg-muted'>
        <Library className='size-5 text-muted-foreground' />
      </span>
      <h2 className='text-sm font-medium'>{title}</h2>
      <p className='mt-1 max-w-sm text-sm leading-6 text-muted-foreground'>{description}</p>
    </div>
  )
}
