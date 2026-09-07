import { useQuery } from '@tanstack/react-query'
import { getLinkHealth } from '@/lib/api/link-health'
import { queryKeys } from '@/lib/api/query-keys'

export function useLinkHealthQuery(opts?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.linkHealth,
    queryFn: getLinkHealth,
    enabled: opts?.enabled ?? true,
    staleTime: 30_000,
  })
}
