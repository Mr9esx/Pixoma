import type { CaseRecord } from '@/lib/api/types'

export function emptyCase(): CaseRecord {
  return {
    id: '',
    name: '',
    price: 0,
    inputs: [],
    outputs: [],
    bindings: { workflow: {}, inputs: [], outputs: [] },
    input_schema: {},
    enabled: true,
  }
}
