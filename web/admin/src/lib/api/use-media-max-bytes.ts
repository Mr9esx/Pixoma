import { useQuery } from '@tanstack/react-query'
import { DEFAULT_MEDIA_MAX_BYTES, resolveMediaMaxBytes } from './media'
import { queryKeys } from './query-keys'
import { fetchPlatformSettings } from './setup'

/**
 * 读取当前生效的单文件上传上限（字节）。
 * 数据源：设置页缓存的 `media_max_bytes`；未配置或加载未就绪时回退到默认值。
 */
export function useMediaMaxBytes(): number {
  const { data } = useQuery({
    queryKey: queryKeys.settings.all,
    queryFn: fetchPlatformSettings,
    staleTime: 30_000,
  })
  return resolveMediaMaxBytes(data?.settings?.media_max_bytes)
}

/** 客户端预检：单文件是否超出当前上限。 */
export function isOverMediaMaxBytes(size: number, max: number): boolean {
  return size > (max > 0 ? max : DEFAULT_MEDIA_MAX_BYTES)
}
