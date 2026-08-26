import { describe, expect, it } from 'vitest'
import {
  isAllowedMedia,
  mediaApiPath,
  MEDIA_MAX_BYTES,
  resolveMediaKey,
} from './media'

describe('resolveMediaKey', () => {
  it('recognizes stored preview key', () => {
    expect(resolveMediaKey('previews/abc.png')).toBe('previews/abc.png')
  })
  it('recognizes full media path', () => {
    expect(resolveMediaKey('/api/v1/media/previews/abc.png')).toBe(
      'previews/abc.png'
    )
  })
  it('returns null for text/unknown values', () => {
    expect(resolveMediaKey('随便一段文本')).toBeNull()
    expect(resolveMediaKey('')).toBeNull()
    expect(resolveMediaKey(undefined)).toBeNull()
  })
})

describe('mediaApiPath', () => {
  it('normalizes leading slash', () => {
    expect(mediaApiPath('previews/a.png')).toBe('/api/v1/media/previews/a.png')
    expect(mediaApiPath('/previews/a.png')).toBe('/api/v1/media/previews/a.png')
  })
})

describe('isAllowedMedia', () => {
  it('accepts whitelisted image/video mime', () => {
    expect(isAllowedMedia('image/png', 'a.png')).toBe(true)
    expect(isAllowedMedia('video/mp4', 'a.mp4')).toBe(true)
  })
  it('rejects unknown types', () => {
    expect(isAllowedMedia('text/plain', 'a.txt')).toBe(false)
  })
  it('defaults size limit to 25MiB', () => {
    expect(MEDIA_MAX_BYTES).toBe(25 * 1024 * 1024)
  })
})
