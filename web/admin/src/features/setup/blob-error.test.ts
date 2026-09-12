import { describe, expect, it } from 'vitest'
import { blobErrorCopy } from './blob-error'

describe('blobErrorCopy', () => {
  it('maps InvalidAccessKeyId to a credentials title', () => {
    const c = blobErrorCopy(
      new Error(
        'InvalidAccessKeyId: The AWS Access Key Id you provided does not exist'
      )
    )
    expect(c.key).toBe('setup.errBlobKey')
  })

  it('maps NoSuchBucket to a bucket title', () => {
    const c = blobErrorCopy(
      new Error('NoSuchBucket: The specified bucket does not exist')
    )
    expect(c.key).toBe('setup.errBlobBucket')
  })

  it('does not treat a plain request 404 as a missing bucket', () => {
    const c = blobErrorCopy(new Error('Request failed (404)'))
    expect(c.key).toBe('setup.errBlobGeneric')
    expect(c.detail).toBe('Request failed (404)')
  })

  it('maps connection failures to a network title', () => {
    const c = blobErrorCopy(new Error('connect: connection refused'))
    expect(c.key).toBe('setup.errBlobNet')
  })

  it('falls back to generic copy', () => {
    const c = blobErrorCopy(new Error('boom'))
    expect(c.key).toBe('setup.errBlobGeneric')
  })
})
