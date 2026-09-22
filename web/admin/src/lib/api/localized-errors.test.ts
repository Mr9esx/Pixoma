import { describe, expect, it } from 'vitest'
import { ApiError } from './client'
import {
  caseDeleteErrorMessage,
  topicDeleteErrorMessage,
} from './localized-errors'

const t = (key: string) => `[${key}]`

describe('caseDeleteErrorMessage', () => {
  it('maps the case delete-conflict code to an i18n key', () => {
    const err = new ApiError(409, 'case is referenced', 4090604)
    expect(caseDeleteErrorMessage(err, t)).toBe('[cases.deleteNeedsAck]')
  })

  it('falls back to undefined for unrecognised or non-ApiError input', () => {
    expect(caseDeleteErrorMessage(new ApiError(409, 'boom'), t)).toBeUndefined()
    expect(caseDeleteErrorMessage(new Error('boom'), t)).toBeUndefined()
  })
})

describe('topicDeleteErrorMessage', () => {
  it('maps the topic delete-conflict code to an i18n key', () => {
    const err = new ApiError(409, 'referenced', 4090904)
    expect(topicDeleteErrorMessage(err, t)).toBe('[topics.deleteNeedsAck]')
  })

  it('maps the protected default topic code to an i18n key', () => {
    const err = new ApiError(409, 'default', 4090913)
    expect(topicDeleteErrorMessage(err, t)).toBe(
      '[topics.deleteDefaultProtected]'
    )
  })
})
