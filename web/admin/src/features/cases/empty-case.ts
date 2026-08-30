import type { CaseRecord } from '@/lib/api/types'
import { DEFAULT_TOPIC_KEY } from '../task-flow/types'

export function emptyCase(): CaseRecord {
  return {
    id: 0,
    name: '',
    inputs: [],
    outputs: [],
    bindings: { workflow: {}, inputs: [], outputs: [] },
    input_schema: {},
    workflow_filename: '',
    enabled: true,
    routing: {
      rules: [{ when: { always: true }, topic: DEFAULT_TOPIC_KEY }],
    },
  }
}
