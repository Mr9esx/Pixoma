import { useQuery } from '@tanstack/react-query'
import { getLinkHealth } from '@/lib/api/link-health'
import { queryKeys } from '@/lib/api/query-keys'
import { buildLinkGraph } from './lib/build-link-graph'

export function useTopologySource(opts?: { enabled?: boolean }) {
  const enabled = opts?.enabled ?? true
  const query = useQuery({
    queryKey: queryKeys.linkHealth,
    queryFn: getLinkHealth,
    enabled,
    staleTime: 30_000,
  })

  const graphAll = query.data
    ? buildLinkGraph(query.data, { type: 'all' })
    : { nodes: [], edges: [] }

  return {
    source: query.data ?? null,
    graphAll,
    isLoading: enabled && query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: () => {
      void query.refetch()
    },
  }
}
