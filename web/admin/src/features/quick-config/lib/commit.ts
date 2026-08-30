import type { CaseRecord } from '@/lib/api/types'
import { createCase, enableCase } from '@/lib/api/cases'
import { patchEdge } from '@/lib/api/edges'
import { createTopic, getTopic } from '@/lib/api/topics'
import type { TopicDraft } from './session'

export function alwaysRouting(topicKey: string): CaseRecord['routing'] {
  return { rules: [{ when: { always: true }, topic: topicKey }] }
}

async function ensureTopic(draft: TopicDraft): Promise<void> {
  try {
    await createTopic({ key: draft.key, name: draft.name })
  } catch {
    await getTopic(draft.key)
  }
}

export async function commitQuickCreate(input: {
  draft: CaseRecord
  topicKey: string
  topicDraft: TopicDraft | null
  selectedEdgeId: string | null
  subscribedTopics: string[]
}): Promise<CaseRecord> {
  if (input.topicDraft && input.topicDraft.key === input.topicKey) {
    await ensureTopic(input.topicDraft)
  }
  const saved = await createCase({
    ...input.draft,
    enabled: false,
    routing: alwaysRouting(input.topicKey),
  })
  if (input.selectedEdgeId) {
    const subscribed = input.subscribedTopics.filter(Boolean)
    if (!subscribed.includes(input.topicKey)) {
      await patchEdge(input.selectedEdgeId, {
        subscribe_topics: [...new Set([...subscribed, input.topicKey])],
      })
    }
  }
  return enableCase(saved.id)
}
