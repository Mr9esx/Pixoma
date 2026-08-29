import { useEffect, useState } from 'react'
import { fetchMediaBlob, resolveMediaKey } from '@/lib/api/media'

/** 拉取鉴权媒体 key 为 Blob objectURL，随 value 变化自动重建并在卸载时释放。 */
export function useMediaObjectUrl(value?: string): string | undefined {
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
