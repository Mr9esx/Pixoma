import { useQuery, type QueryClient } from '@tanstack/react-query'
import { getLinkHealth } from '@/lib/api/link-health'
import { queryKeys } from '@/lib/api/query-keys'

export function invalidateLinkHealth(qc: QueryClient) {
  return qc.invalidateQueries({ queryKey: queryKeys.linkHealth })
}

export function useLinkHealthQuery(opts?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.linkHealth,
    queryFn: getLinkHealth,
    enabled: opts?.enabled ?? true,
    staleTime: 0,
    refetchInterval: 5000,
  })
}
