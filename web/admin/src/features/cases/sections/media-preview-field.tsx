import { useRef, useState } from 'react'
import { Film, ImagePlus, Loader2, Trash2, ZoomIn } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import {
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
import { useMediaObjectUrl } from '../lib/use-media-object-url'
import { MediaLightbox } from '../components/media-lightbox'

type Props = {
  value?: string
  onChange: (next?: string) => void
  disabled?: boolean
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
        <AttachmentMedia variant={key && mediaUrl ? 'image' : 'icon'}>
          {uploading ? (
            <Loader2 className='animate-spin' />
          ) : isVideo ? (
            mediaUrl ? (
              <video
                src={`${mediaUrl}#t=0.1`}
                muted
                playsInline
                preload='metadata'
                disablePictureInPicture
                className='h-full w-full object-contain'
              />
            ) : (
              <Film />
            )
          ) : key && mediaUrl ? (
            <img
              src={mediaUrl}
              alt='preview'
              className='h-full w-full object-contain'
            />
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

      <MediaLightbox
        open={lightboxOpen}
        onOpenChange={setLightboxOpen}
        src={mediaUrl}
        isVideo={isVideo}
      />
    </div>
  )
}
