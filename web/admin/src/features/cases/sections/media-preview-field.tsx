import { useEffect, useRef, useState } from 'react'
import { Film, ImagePlus, Loader2, Trash2, ZoomIn } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import {
  fetchMediaBlob,
  isAllowedMedia,
  MEDIA_MAX_BYTES,
  resolveMediaKey,
  uploadMedia,
} from '@/lib/api/media'
import {
  Attachment,
  AttachmentAction,
  AttachmentActions,
  AttachmentContent,
  AttachmentMedia,
  AttachmentTitle,
  AttachmentTrigger,
} from '@/components/ui/attachment'
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
  const mediaName = key ? key.split('/').pop() : undefined

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
    <div
      data-testid='media-preview-field'
      className='flex flex-col items-start gap-2'
      onDragOver={(e) => {
        if (!disabled) e.preventDefault()
      }}
      onDrop={(e) => {
        e.preventDefault()
        if (!disabled && !uploading) void onFile(e.dataTransfer.files?.[0])
      }}
    >
      <Attachment
        state={error ? 'error' : uploading ? 'uploading' : 'done'}
        className='w-full'
      >
        <AttachmentMedia variant={key && !isVideo ? 'image' : 'icon'}>
          {uploading ? (
            <Loader2 className='animate-spin' />
          ) : isVideo ? (
            <Film />
          ) : key && mediaUrl ? (
            <img src={mediaUrl} alt='preview' />
          ) : (
            <ImagePlus />
          )}
        </AttachmentMedia>
        <AttachmentContent>
          <AttachmentTitle>
            {uploading
              ? t('media.uploading')
              : key
                ? (mediaName ?? t('media.pick'))
                : t('media.pick')}
          </AttachmentTitle>
        </AttachmentContent>
        <AttachmentActions>
          {key ? (
            <>
              <AttachmentAction
                aria-label={t('media.zoom')}
                title={t('media.zoom')}
                onClick={() => setLightboxOpen(true)}
              >
                <ZoomIn />
              </AttachmentAction>
              {!disabled ? (
                <AttachmentAction
                  aria-label={t('media.clear')}
                  title={t('media.clear')}
                  onClick={() => onChange(undefined)}
                >
                  <Trash2 />
                </AttachmentAction>
              ) : null}
            </>
          ) : null}
        </AttachmentActions>
        {!disabled ? (
          <AttachmentTrigger onClick={() => fileRef.current?.click()} />
        ) : null}
      </Attachment>

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
