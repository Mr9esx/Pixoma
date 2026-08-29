import { Dialog, DialogContent } from '@/components/ui/dialog'

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  src?: string
  isVideo: boolean
  alt?: string
}

/** 放大预览（lightbox）：详情页缩略图与编辑面板共用同一套样式。 */
export function MediaLightbox({ open, onOpenChange, src, isVideo, alt }: Props) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-w-[min(92vw,1100px)] border-0 bg-foreground/95 p-2 sm:max-w-[min(92vw,1100px)] [&_[data-slot=&apos;dialog-close&apos;]]:size-8 [&_[data-slot=&apos;dialog-close&apos;]]:text-white [&_[data-slot=&apos;dialog-close&apos;]]:opacity-80 [&_[data-slot=&apos;dialog-close&apos;]]:hover:bg-black/60 [&_[data-slot=&apos;dialog-close&apos;]]:hover:opacity-100'>
        {isVideo ? (
          <video
            src={src}
            controls
            autoPlay
            className='max-h-[85vh] w-full object-contain'
          />
        ) : (
          <img
            src={src}
            alt={alt}
            className='max-h-[85vh] w-full object-contain'
          />
        )}
      </DialogContent>
    </Dialog>
  )
}
