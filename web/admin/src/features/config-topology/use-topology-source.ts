import { useQueries, useQuery, useQueryClient } from '@tanstack/react-query'
import { getMenu } from '@/lib/api/channel-menu'
import {
  listChannels,
  type ChannelReachability,
} from '@/lib/api/channels'
import { listCases } from '@/lib/api/cases'
import { listEdges, listPresence } from '@/lib/api/edges'
import { queryKeys } from '@/lib/api/query-keys'
import { listTopics } from '@/lib/api/topics'
import {
  buildLinkGraph,
  type TopologySource,
} from './lib/build-link-graph'

export function useTopologySource(opts?: { enabled?: boolean }) {
  const enabled = opts?.enabled ?? true
  const queryClient = useQueryClient()
  const channels = useQuery({
    queryKey: queryKeys.channels.all,
    queryFn: listChannels,
    enabled,
  })
  const cases = useQuery({
    queryKey: queryKeys.cases.all,
    queryFn: () => listCases(),
    enabled,
  })
  const topics = useQuery({
    queryKey: queryKeys.topics.all,
    queryFn: () => listTopics(),
    enabled,
  })
  const edges = useQuery({
    queryKey: queryKeys.edges.all,
    queryFn: listEdges,
    enabled,
  })
  const presence = useQuery({
    queryKey: queryKeys.edges.presence,
    queryFn: listPresence,
    enabled,
  })
  const channelRows = enabled ? (channels.data ?? []) : []
  const menus = useQueries({
    queries: channelRows.map((ch) => ({
      queryKey: queryKeys.channels.menu(ch.id),
      queryFn: () => getMenu(ch.id),
      enabled,
    })),
  })

  const menuMap: TopologySource['menus'] = {}
  channelRows.forEach((ch, i) => {
    menuMap[ch.id] = menus[i]?.data
  })

  const reachability: TopologySource['reachability'] = {}
  for (const ch of channelRows) {
    reachability[ch.id] = queryClient.getQueryData<ChannelReachability>([
      'channels',
      ch.id,
      'reachability',
    ])
  }

  const menusLoading = enabled && menus.some((q) => q.isLoading)
  const menusError = menus.find((q) => q.isError)?.error
  const isLoading =
    enabled &&
    (channels.isLoading ||
      cases.isLoading ||
      topics.isLoading ||
      edges.isLoading ||
      presence.isLoading ||
      menusLoading)
  const error =
    channels.error ??
    cases.error ??
    topics.error ??
    edges.error ??
    presence.error ??
    menusError
  const isError = Boolean(error)

  const source: TopologySource | null =
    enabled &&
    channels.data &&
    cases.data &&
    topics.data &&
    edges.data &&
    presence.data
      ? {
          channels: channels.data.map((c) => ({ id: c.id, name: c.name })),
          menus: menuMap,
          cases: cases.data.map((c) => ({
            id: c.id,
            name: c.name,
            routing: c.routing,
          })),
          topics: topics.data.map((t) => ({ key: t.key, name: t.name })),
          edges: edges.data,
          presence: presence.data,
          reachability,
        }
      : null

  const graphAll = source
    ? buildLinkGraph(source, { type: 'all' })
    : { nodes: [], edges: [] }

  const refetch = () => {
    void channels.refetch()
    void cases.refetch()
    void topics.refetch()
    void edges.refetch()
    void presence.refetch()
    for (const q of menus) void q.refetch()
  }

  return { source, graphAll, isLoading, isError, error, refetch }
}
