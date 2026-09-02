import { useEffect, useRef, useState } from 'react'
import { User, XIcon } from 'lucide-react'
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
  AttachmentTrigger,
} from '@/components/ui/attachment'
import { PixomaLoading } from '@/components/feedback/pixoma-loading'

type Props = {
  id?: string
  value?: string
  onChange: (next?: string) => void
  disabled?: boolean
}

const AVATAR_SIZE = 'h-[140px] w-[140px]'
const AVATAR_ACCEPT = 'image/png,image/jpeg,image/webp,image/gif'
const AVATAR_OUTPUT_SIZE = 512

function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => resolve(img)
    img.onerror = () => reject(new Error('无法读取图片'))
    img.src = src
  })
}

/** 把原图裁成居中正方形（固定 512px PNG），再走媒体上传。 */
async function toSquareFile(file: File): Promise<File> {
  const url = URL.createObjectURL(file)
  try {
    const img = await loadImage(url)
    const size = Math.min(img.naturalWidth, img.naturalHeight)
    const sx = (img.naturalWidth - size) / 2
    const sy = (img.naturalHeight - size) / 2
    const canvas = document.createElement('canvas')
    canvas.width = AVATAR_OUTPUT_SIZE
    canvas.height = AVATAR_OUTPUT_SIZE
    const ctx = canvas.getContext('2d')
    if (!ctx) throw new Error('无法处理图片')
    ctx.drawImage(
      img,
      sx,
      sy,
      size,
      size,
      0,
      0,
      AVATAR_OUTPUT_SIZE,
      AVATAR_OUTPUT_SIZE
    )
    return await new Promise((resolve, reject) => {
      canvas.toBlob((blob) => {
        if (!blob) {
          reject(new Error('无法处理图片'))
          return
        }
        resolve(new File([blob], 'avatar.png', { type: 'image/png' }))
      }, 'image/png')
    })
  } finally {
    URL.revokeObjectURL(url)
  }
}

function useAvatarObjectUrl(value?: string): string | undefined {
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

/** 头像上传：一个 134×134 圆形头像 Attachment，图片裁成正方形后上传。 */
export function AvatarUpload({ id, value, onChange, disabled }: Props) {
  const fileRef = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState<string | undefined>()
  const avatarUrl = useAvatarObjectUrl(value)
  const hasAvatar = Boolean(resolveMediaKey(value))

  async function onFile(file: File | undefined) {
    if (!file) return
    setError(undefined)
    if (!isAllowedMedia(file.type, file.name)) {
      setError('仅支持 png / jpeg / webp / gif 图片')
      return
    }
    if (file.size > MEDIA_MAX_BYTES) {
      setError('图片过大，单张上限 25MB')
      return
    }
    setUploading(true)
    try {
      const square = await toSquareFile(file)
      const uploaded = await uploadMedia(square)
      onChange(uploaded.key)
    } catch (err) {
      setError(err instanceof Error ? err.message : '上传失败')
    } finally {
      setUploading(false)
    }
  }

  return (
    <div
      data-testid='avatar-upload'
      className='flex flex-col items-center gap-2'
      onDragOver={(e) => {
        if (!disabled) e.preventDefault()
      }}
      onDrop={(e) => {
        e.preventDefault()
        if (!disabled && !uploading) void onFile(e.dataTransfer.files?.[0])
      }}
    >
      <Attachment
        orientation='vertical'
        state={error ? 'error' : uploading ? 'uploading' : 'done'}
        className={AVATAR_SIZE}
      >
        <div className='m-auto flex size-24 items-center justify-center overflow-hidden rounded-full bg-muted text-foreground'>
          {uploading ? (
            <PixomaLoading />
          ) : hasAvatar && avatarUrl ? (
            <img
              src={avatarUrl}
              alt='头像'
              className='size-full object-cover'
            />
          ) : (
            <User className='size-10 text-foreground/70' />
          )}
        </div>
        {hasAvatar && !disabled ? (
          <AttachmentActions className='top-2 right-2'>
            <AttachmentAction
              variant='outline'
              size='icon-xs'
              className='size-6 rounded-full border bg-background text-muted-foreground hover:text-destructive'
              aria-label='移除头像'
              title='移除头像'
              onClick={() => onChange(undefined)}
            >
              <XIcon className='size-3.5' />
            </AttachmentAction>
          </AttachmentActions>
        ) : null}
        {!disabled ? (
          <AttachmentTrigger onClick={() => fileRef.current?.click()} />
        ) : null}
      </Attachment>

      <input
        id={id}
        ref={fileRef}
        type='file'
        accept={AVATAR_ACCEPT}
        className='sr-only'
        disabled={disabled || uploading}
        onChange={(e) => {
          void onFile(e.target.files?.[0])
          e.target.value = ''
        }}
      />

      {error ? (
        <p
          className='max-w-44 text-center text-xs text-destructive'
          role='alert'
        >
          {error}
        </p>
      ) : null}
    </div>
  )
}
