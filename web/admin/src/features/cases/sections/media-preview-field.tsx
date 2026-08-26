import { useEffect, useRef, useState } from 'react'
import { ImagePlus, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import {
  fetchMediaBlob,
  isAllowedMedia,
  MEDIA_MAX_BYTES,
  resolveMediaKey,
  uploadMedia,
} from '@/lib/api/media'
import { Dialog, DialogContent } from '@/components/ui/dialog'

type Props = {
  value?: string
  onChange: (next?: string) => void
  disabled?: boolean
}

function useMediaObjectUrl(value?: string): string | undefined {
  const key = resolveMediaKey(value)
  const [loaded, setLoaded] = useState<
    { key: string; url: string } | undefined
  >(undefined)
  useEffect(() => {
    if (!key) return
    let cancelled = false
    let objectUrl: string | undefined
    fetchMediaBlob(key)
      .then((blob) => {
        if (cancelled) return
        objectUrl = URL.createObjectURL(blob)
        setLoaded({ key, url: objectUrl })
      })
      .catch(() => {
        // ignore: render without a preview
      })
    return () => {
      cancelled = true
      if (objectUrl) URL.revokeObjectURL(objectUrl)
    }
  }, [key])
  return loaded && loaded.key === key ? loaded.url : undefined
}

/** 基础信息「预览效果图」字段：上传图片/视频并内嵌预览（管理端鉴权）。 */
export function MediaPreviewField({ value, onChange, disabled }: Props) {
  const { t } = useTranslation()
  const fileRef = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState<string | undefined>()
  const [lightboxOpen, setLightboxOpen] = useState(false)
  const mediaUrl = useMediaObjectUrl(value)
  const key = resolveMediaKey(value)
  const isVideo = /\.(mp4|webm)$/i.test(key ?? '')

  async function onFile(file: File | undefined) {
    if (!file) return
    setError(undefined)
    if (!isAllowedMedia(file.type, file.name)) {
      setError(t('media.errorType'))
      return
    }
    if (file.size > MEDIA_MAX_BYTES) {
      setError(t('media.errorSize'))
      return
    }
    setUploading(true)
    try {
      const uploaded = await uploadMedia(file)
      onChange(uploaded.key)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('media.uploadFailed'))
    } finally {
      setUploading(false)
    }
  }

  return (
    <div data-testid='media-preview-field' className='space-y-2'>
      {mediaUrl && key ? (
        <div className='relative overflow-hidden rounded-md border border-border bg-muted/20'>
          <button
            type='button'
            aria-label={t('media.zoom')}
            className='block w-full cursor-zoom-in'
            onClick={() => setLightboxOpen(true)}
          >
            {isVideo ? (
              <video
                src={mediaUrl}
                muted
                playsInline
                className='pointer-events-none max-h-64 w-full object-contain'
              />
            ) : (
              <img
                src={mediaUrl}
                alt='preview'
                className='max-h-64 w-full object-contain'
              />
            )}
          </button>
          {!disabled ? (
            <div className='absolute inset-x-0 bottom-0 flex items-center justify-end gap-1.5 bg-gradient-to-t from-black/55 to-transparent p-2'>
              <button
                type='button'
                disabled={uploading}
                onClick={() => {
                  if (!uploading) fileRef.current?.click()
                }}
                className='rounded-md bg-white/20 px-2 py-1 text-xs font-medium text-white backdrop-blur-sm hover:bg-white/35 disabled:opacity-50'
              >
                {t('media.replace')}
              </button>
              <button
                type='button'
                onClick={() => onChange(undefined)}
                className='rounded-md bg-white/20 px-2 py-1 text-xs font-medium text-white backdrop-blur-sm hover:bg-white/35'
              >
                {t('media.clear')}
              </button>
            </div>
          ) : null}
        </div>
      ) : (
        <div
          role='button'
          tabIndex={0}
          aria-label={t('media.pick')}
          className='cursor-pointer rounded-md border border-dashed p-3 text-center text-sm text-muted-foreground hover:bg-accent/50 disabled:cursor-not-allowed disabled:opacity-50'
          onClick={() => {
            if (!disabled && !uploading) fileRef.current?.click()
          }}
          onKeyDown={(e) => {
            if (disabled || uploading) return
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault()
              fileRef.current?.click()
            }
          }}
          onDragOver={(e) => {
            e.preventDefault()
            e.stopPropagation()
          }}
          onDrop={(e) => {
            e.preventDefault()
            e.stopPropagation()
            if (!disabled && !uploading) void onFile(e.dataTransfer.files?.[0])
          }}
        >
          <span className='inline-flex items-center gap-1.5'>
            {uploading ? (
              <Loader2 className='size-4 animate-spin' />
            ) : (
              <ImagePlus className='size-4' />
            )}
            {uploading ? t('media.uploading') : t('media.pick')}
          </span>
        </div>
      )}

      <input
        ref={fileRef}
        type='file'
        accept='image/png,image/jpeg,image/webp,image/gif,video/mp4,video/webm'
        className='hidden'
        disabled={disabled || uploading}
        onChange={(e) => {
          void onFile(e.target.files?.[0])
          e.target.value = ''
        }}
      />

      {error ? (
        <p className='text-xs text-destructive' role='alert'>
          {error}
        </p>
      ) : null}

      <Dialog open={lightboxOpen} onOpenChange={setLightboxOpen}>
        <DialogContent className="max-w-[min(92vw,1100px)] border-0 bg-foreground/95 p-2 sm:max-w-[min(92vw,1100px)] [&_[data-slot='dialog-close']]:size-8 [&_[data-slot='dialog-close']]:bg-black/40 [&_[data-slot='dialog-close']]:text-white [&_[data-slot='dialog-close']]:opacity-100 [&_[data-slot='dialog-close']]:hover:bg-black/60">
          {isVideo ? (
            <video
              src={mediaUrl}
              controls
              autoPlay
              className='max-h-[85vh] w-full object-contain'
            />
          ) : (
            <img
              src={mediaUrl}
              alt='preview'
              className='max-h-[85vh] w-full object-contain'
            />
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
