import { describe, expect, it } from 'vitest'
import { blobErrorCopy } from './blob-error'

describe('blobErrorCopy', () => {
  it('maps InvalidAccessKeyId to a credentials title', () => {
    const c = blobErrorCopy(new Error('InvalidAccessKeyId: The AWS Access Key Id you provided does not exist'))
    expect(c.title).toBe('对象存储密钥错误，请检查 Access Key / Secret Key')
  })

  it('maps NoSuchBucket to a bucket title', () => {
    const c = blobErrorCopy(new Error('NoSuchBucket: The specified bucket does not exist'))
    expect(c.title).toBe('bucket 或对象不存在，请检查 bucket 名称')
  })

  it('does not treat a plain request 404 as a missing bucket', () => {
    const c = blobErrorCopy(new Error('Request failed (404)'))
    expect(c.title).toBe('对象存储配置失败，请重试')
    expect(c.detail).toBe('Request failed (404)')
  })

  it('maps connection failures to a network title', () => {
    const c = blobErrorCopy(new Error('connect: connection refused'))
    expect(c.title).toBe('连不上对象存储，请检查 endpoint 和网络')
  })

  it('falls back to generic copy', () => {
    const c = blobErrorCopy(new Error('boom'))
    expect(c.title).toBe('对象存储配置失败，请重试')
  })
})
