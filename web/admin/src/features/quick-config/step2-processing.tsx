import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { createEdge, listEdges, listPresence, patchEdge } from '@/lib/api/edges'
import { queryKeys } from '@/lib/api/query-keys'
import { listRoutingAttributes } from '@/lib/api/routing'
import { createTopic, listTopics } from '@/lib/api/topics'
import type { ComfyEdge, RoutingConfig } from '@/lib/api/types'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { DeployCredentials } from '@/features/edges/deploy-credentials'
import { NodeTopicPicker } from '@/features/edges/node-topic-picker'
import { PresenceTags } from '@/features/edges/presence-tags'
import { topicBindings } from '@/features/task-flow/lib/topic-binding'
import { validateRouting as validateEditorRouting } from '@/features/task-flow/lib/validate'
import { TaskFlowEditor } from '@/features/task-flow/task-flow-editor'
import {
  DEFAULT_TOPIC_KEY,
  type AttributeDescriptor,
  type EdgePresence,
  type EdgeRecord,
  type TopicRecord,
} from '@/features/task-flow/types'
import type { StepActions, WizardShared } from './types'
import { WizardChrome } from './wizard-chrome'

type Props = StepActions & { shared: WizardShared }

const TOPIC_KEY_RE = /^[a-z0-9]+(-[a-z0-9]+)*$/

function isValidTopicKey(key: string): boolean {
  return key.length > 0 && key.length <= 64 && TOPIC_KEY_RE.test(key)
}

/** Step 2 处理流程：内嵌 TaskFlowCanvas，下一步保存 routing 与节点订阅。 */
export function Step2Processing({ shared, next, back }: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [routing, setRouting] = useState<RoutingConfig | undefined>(
    shared.routing ?? { rules: [] }
  )
  const [error, setError] = useState<string | null>(null)
  const [topicOpen, setTopicOpen] = useState(false)
  const [nodeOpen, setNodeOpen] = useState(false)
  const [nodeStep, setNodeStep] = useState<'form' | 'deploy'>('form')
  const [createdEdge, setCreatedEdge] = useState<ComfyEdge | null>(null)
  const [topicKey, setTopicKey] = useState('')
  const [topicName, setTopicName] = useState('')
  const [nodeName, setNodeName] = useState('')
  const [nodeCaps, setNodeCaps] = useState('')

  function openNodeDialog() {
    setNodeStep('form')
    setCreatedEdge(null)
    setNodeName('')
    setNodeCaps('')
    setNodeOpen(true)
  }
  function closeNodeDialog() {
    setNodeOpen(false)
    setNodeStep('form')
    setCreatedEdge(null)
    setNodeName('')
    setNodeCaps('')
  }

  const createTopicMutation = useMutation({
    mutationFn: () =>
      createTopic({
        key: topicKey.trim(),
        name: topicName.trim() || topicKey.trim(),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.topics.all })
      setTopicOpen(false)
      setTopicKey('')
      setTopicName('')
    },
  })
  const createNodeMutation = useMutation({
    mutationFn: () =>
      createEdge({
        name: nodeName.trim(),
        description: '',
        capabilities: nodeCaps
          .split(',')
          .map((s) => s.trim())
          .filter(Boolean),
      }),
    onSuccess: async (edge) => {
      // 自动把新建节点绑定到当前路由用到的已启用任务队列，让 Topic→节点 绑定立刻成立。
      if (routingTopics.length > 0) {
        try {
          await patchEdge(edge.id, { subscribe_topics: routingTopics })
        } catch {
          // 绑定失败不阻塞创建，用户可在部署弹窗里手动勾选重试。
        }
      }
      void queryClient.invalidateQueries({ queryKey: queryKeys.edges.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.edges.presence })
      setCreatedEdge(edge)
      setNodeStep('deploy')
    },
  })

  // 调整任意节点的订阅通道 → 立即落库绑定，校验随之清除（部署弹窗与画布连线共用）。
  const bindEdgeTopicsMutation = useMutation({
    mutationFn: ({
      edgeId,
      topicKeys,
    }: {
      edgeId: string
      topicKeys: string[]
    }) => patchEdge(edgeId, { subscribe_topics: topicKeys }),
    onSuccess: (updated) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.edges.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.edges.presence })
      if (createdEdge && updated.id === createdEdge.id) {
        setCreatedEdge(updated)
      }
    },
  })

  /** 画布 Topic→节点 连线/删线：结算该 edge 新的订阅集合并落库。 */
  function handleEdgeSubscription(
    edgeId: string,
    topicKey: string,
    add: boolean
  ) {
    const rec = edgesQuery.data?.find((e) => e.id === edgeId)
    const current = new Set(rec?.subscribe_topics ?? [])
    if (add) {
      current.add(topicKey)
    } else {
      current.delete(topicKey)
    }
    bindEdgeTopicsMutation.mutate({ edgeId, topicKeys: [...current] })
  }

  const topicsQuery = useQuery({
    queryKey: queryKeys.topics.all,
    queryFn: () => listTopics(),
  })
  const attributesQuery = useQuery({
    queryKey: ['routing', 'attributes'],
    queryFn: () => listRoutingAttributes(),
  })
  const edgesQuery = useQuery({
    queryKey: queryKeys.edges.all,
    queryFn: listEdges,
  })
  const presenceQuery = useQuery({
    queryKey: queryKeys.edges.presence,
    queryFn: listPresence,
  })

  const topics: TopicRecord[] = topicsQuery.data ?? []
  const enabledTopicKeys = new Set(
    topics.filter((t) => t.enabled).map((t) => t.key)
  )
  /** 当前路由用到的任务队列（无规则时回退默认 Topic），仅保留已启用项，用于新建节点的自动绑定。 */
  const routingTopics = (() => {
    const keys = (routing?.rules ?? [])
      .map((rule) => rule.topic)
      .filter((topic): topic is string => Boolean(topic))
    const used = keys.length > 0 ? [...new Set(keys)] : [DEFAULT_TOPIC_KEY]
    return used.filter((key) => enabledTopicKeys.has(key))
  })()

  const attributes: AttributeDescriptor[] =
    attributesQuery.data?.attributes ?? []
  const edges: EdgeRecord[] = (edgesQuery.data ?? []).map(
    ({ id, name, enabled, subscribe_topics, effective_topics }) => ({
      id,
      name,
      enabled,
      subscribe_topics: subscribe_topics ?? [],
      effective_topics: effective_topics ?? [],
    })
  )
  const presence: EdgePresence[] = presenceQuery.data ?? []

  const bindingSummary = useMemo(() => {
    const rules = routing?.rules ?? []
    const usedTopics =
      rules.length === 0
        ? [DEFAULT_TOPIC_KEY]
        : [
            ...new Set(
              rules
                .map((rule) => rule.topic)
                .filter((topic): topic is string => Boolean(topic))
            ),
          ]
    const mappedEdges: EdgeRecord[] = (edgesQuery.data ?? []).map(
      ({ id, name, enabled, subscribe_topics, effective_topics }) => ({
        id,
        name,
        enabled,
        subscribe_topics: subscribe_topics ?? [],
        effective_topics: effective_topics ?? [],
      })
    )
    const mappedPresence: EdgePresence[] = presenceQuery.data ?? []
    const bindings = topicBindings(mappedEdges, mappedPresence, usedTopics)
    return {
      bound: bindings.filter((b) => b.status !== 'unbound').map((b) => b.topic),
      online: bindings.filter((b) => b.status === 'ready').map((b) => b.topic),
    }
  }, [edgesQuery.data, presenceQuery.data, routing])

  const validation = useMemo(
    () =>
      validateEditorRouting(
        routing,
        topics,
        attributes,
        new Set(bindingSummary.bound)
      ),
    [routing, topics, attributes, bindingSummary]
  )

  const loading =
    topicsQuery.isLoading ||
    attributesQuery.isLoading ||
    edgesQuery.isLoading ||
    presenceQuery.isLoading

  function handleNext() {
    if (!validation.valid) {
      setError(validation.issues[0]?.message ?? 'quickConfig.requireRule')
      return
    }
    setError(null)
    shared.updateRouting(routing ?? { rules: [] })
    next({})
  }

  return (
    <WizardChrome
      step={2}
      onBack={() => back({})}
      onNext={handleNext}
      nextLabel={t('quickConfig.next')}
    >
      {error ? (
        <p className='mb-2 text-sm text-destructive' role='alert'>
          {t(error)}
        </p>
      ) : null}
      {loading ? (
        <LoadingSkeleton rows={4} />
      ) : (
        <>
          <TaskFlowEditor
            routing={routing}
            topics={topics}
            attributes={attributes}
            edges={edges}
            presence={presence}
            caseName={shared.caseRecord?.name ?? 'Case 任务'}
            onChange={setRouting}
            onChangeEdgeSubscription={handleEdgeSubscription}
            headerActions={
              <>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  onClick={() => setTopicOpen(true)}
                >
                  <Plus className='size-4' />
                  {t('quickConfig.newTopic')}
                </Button>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  onClick={openNodeDialog}
                >
                  <Plus className='size-4' />
                  {t('quickConfig.newNode')}
                </Button>
              </>
            }
          />
        </>
      )}
      {bindingSummary.bound.length > 0 ? (
        <p className='mt-2 text-xs text-muted-foreground'>
          {t('quickConfig.boundNodes', {
            nodes: bindingSummary.bound.join('、'),
          })}
          {bindingSummary.online.length < bindingSummary.bound.length
            ? t('quickConfig.partialOffline')
            : ''}
        </p>
      ) : null}

      <Dialog open={topicOpen} onOpenChange={setTopicOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('quickConfig.newTopic')}</DialogTitle>
            <DialogDescription>
              {t('quickConfig.topicKeyHint')}
            </DialogDescription>
          </DialogHeader>
          <div className='space-y-3'>
            <div className='space-y-1'>
              <Label htmlFor='topic-key'>Key</Label>
              <Input
                id='topic-key'
                value={topicKey}
                onChange={(e) => setTopicKey(e.target.value)}
                placeholder='fast-gpu'
              />
            </div>
            <div className='space-y-1'>
              <Label htmlFor='topic-name'>{t('quickConfig.name')}</Label>
              <Input
                id='topic-name'
                value={topicName}
                onChange={(e) => setTopicName(e.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              type='button'
              disabled={
                !isValidTopicKey(topicKey.trim()) ||
                createTopicMutation.isPending
              }
              onClick={() => createTopicMutation.mutate()}
            >
              {t('quickConfig.create')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={nodeOpen} onOpenChange={setNodeOpen}>
        {nodeStep === 'deploy' && createdEdge ? (
          <DialogContent className='max-h-[85vh] flex-col'>
            <DialogHeader>
              <DialogTitle>
                {t('quickConfig.nodeDeployTitle', { name: createdEdge.name })}
              </DialogTitle>
            </DialogHeader>
            <div className='min-h-0 flex-1 space-y-3 overflow-y-auto'>
              <PresenceTags
                edgeOnline={
                  presence.find((p) => p.id === createdEdge.id)?.edge_online ===
                  true
                }
                comfyRunning={
                  presence.find((p) => p.id === createdEdge.id)
                    ?.comfy_running === true
                }
              />
              <p className='text-xs text-muted-foreground'>
                {t('quickConfig.nodeDeployHint')}
              </p>
              <p className='text-xs text-muted-foreground'>
                {t('quickConfig.nodeTopicBindHint')}
              </p>
              <NodeTopicPicker
                value={createdEdge.subscribe_topics ?? []}
                onChange={(topicKeys) =>
                  bindEdgeTopicsMutation.mutate({
                    edgeId: createdEdge.id,
                    topicKeys,
                  })
                }
              />
              <DeployCredentials edge={createdEdge} showToken={false} />
            </div>
            <DialogFooter>
              <Button type='button' onClick={closeNodeDialog}>
                {t('quickConfig.nodeDeployDone')}
              </Button>
            </DialogFooter>
          </DialogContent>
        ) : (
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{t('quickConfig.newNode')}</DialogTitle>
              <DialogDescription>
                {t('quickConfig.newNodeHint')}
              </DialogDescription>
            </DialogHeader>
            <div className='space-y-3'>
              <div className='space-y-1'>
                <Label htmlFor='node-name'>{t('quickConfig.name')}</Label>
                <Input
                  id='node-name'
                  value={nodeName}
                  onChange={(e) => setNodeName(e.target.value)}
                />
              </div>
              <div className='space-y-1'>
                <Label htmlFor='node-caps'>
                  {t('quickConfig.capabilities')}
                </Label>
                <Input
                  id='node-caps'
                  value={nodeCaps}
                  onChange={(e) => setNodeCaps(e.target.value)}
                  placeholder='comfy, gpu'
                />
              </div>
            </div>
            <DialogFooter>
              <Button
                type='button'
                disabled={!nodeName.trim() || createNodeMutation.isPending}
                onClick={() => createNodeMutation.mutate()}
              >
                {t('quickConfig.create')}
              </Button>
            </DialogFooter>
          </DialogContent>
        )}
      </Dialog>
    </WizardChrome>
  )
}
