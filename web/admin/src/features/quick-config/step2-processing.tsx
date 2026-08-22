import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Plus } from 'lucide-react'
import { createEdge, listEdges, listPresence } from '@/lib/api/edges'
import { listRoutingAttributes } from '@/lib/api/routing'
import { createTopic, listTopics } from '@/lib/api/topics'
import { queryKeys } from '@/lib/api/query-keys'
import type { RoutingConfig } from '@/lib/api/types'
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
import { TaskFlowCanvas } from '@/features/task-flow/task-flow-canvas'
import { topicBindings } from '@/features/task-flow/lib/topic-binding'
import type {
  AttributeDescriptor,
  EdgePresence,
  EdgeRecord,
  TopicRecord,
} from '@/features/task-flow/types'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { WizardChrome } from './wizard-chrome'
import type { StepActions, WizardShared } from './types'

type Props = StepActions & { shared: WizardShared }

function validateRouting(routing: RoutingConfig | undefined): string | null {
  if (!routing || routing.rules.length === 0) {
    return 'quickConfig.requireRule'
  }
  if (routing.rules.some((rule) => !rule.topic)) {
    return 'quickConfig.unboundRule'
  }
  return null
}

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
  const [topicKey, setTopicKey] = useState('')
  const [topicName, setTopicName] = useState('')
  const [nodeName, setNodeName] = useState('')
  const [nodeCaps, setNodeCaps] = useState('')

  const createTopicMutation = useMutation({
    mutationFn: () => createTopic({ key: topicKey.trim(), name: topicName.trim() || topicKey.trim() }),
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
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.edges.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.edges.presence })
      setNodeOpen(false)
      setNodeName('')
      setNodeCaps('')
    },
  })

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
  const attributes: AttributeDescriptor[] = attributesQuery.data?.attributes ?? []
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
    const usedTopics = [
      ...new Set(
        (routing?.rules ?? [])
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
      bound: bindings
        .filter((b) => b.status !== 'unbound')
        .map((b) => b.topic),
      online: bindings
        .filter((b) => b.status === 'ready')
        .map((b) => b.topic),
    }
  }, [edgesQuery.data, presenceQuery.data, routing])

  const loading =
    topicsQuery.isLoading ||
    attributesQuery.isLoading ||
    edgesQuery.isLoading ||
    presenceQuery.isLoading

  function handleNext() {
    const invalid = validateRouting(routing)
    if (invalid) {
      setError(invalid)
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
          <div className='mb-2 flex flex-wrap gap-2'>
            <Button type='button' variant='outline' size='sm' onClick={() => setTopicOpen(true)}>
              <Plus className='size-4' />
              {t('quickConfig.newTopic')}
            </Button>
            <Button type='button' variant='outline' size='sm' onClick={() => setNodeOpen(true)}>
              <Plus className='size-4' />
              {t('quickConfig.newNode')}
            </Button>
          </div>
          <div className='h-[520px] overflow-hidden rounded-lg border border-border bg-muted/20'>
            <TaskFlowCanvas
              routing={routing}
              topics={topics}
              attributes={attributes}
              edges={edges}
              presence={presence}
              defaultTopicKey='default'
              readOnly={false}
              caseName={shared.caseRecord?.name ?? 'Case 任务'}
              onChange={setRouting}
            />
          </div>
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
            <DialogDescription>{t('quickConfig.topicKeyHint')}</DialogDescription>
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
              disabled={!isValidTopicKey(topicKey.trim()) || createTopicMutation.isPending}
              onClick={() => createTopicMutation.mutate()}
            >
              {t('quickConfig.create')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={nodeOpen} onOpenChange={setNodeOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('quickConfig.newNode')}</DialogTitle>
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
              <Label htmlFor='node-caps'>{t('quickConfig.capabilities')}</Label>
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
      </Dialog>
    </WizardChrome>
  )
}
