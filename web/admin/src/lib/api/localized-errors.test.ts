import { describe, expect, it } from 'vitest'
import { ApiError } from './client'
import { caseDeleteErrorMessage } from './localized-errors'

const t = (key: string) => `[${key}]`

describe('caseDeleteErrorMessage', () => {
  it('maps case_delete_needs_ack to i18n key', () => {
    const err = new ApiError(409, 'case is referenced', 'case_delete_needs_ack')
    expect(caseDeleteErrorMessage(err, t)).toBe('[cases.deleteNeedsAck]')
  })

  it('falls back to undefined for unrecognised or non-ApiError input', () => {
    expect(caseDeleteErrorMessage(new ApiError(409, 'boom'), t)).toBeUndefined()
    expect(caseDeleteErrorMessage(new Error('boom'), t)).toBeUndefined()
  })
})
