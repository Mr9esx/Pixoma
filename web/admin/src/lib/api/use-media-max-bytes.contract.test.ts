import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

describe('useMediaMaxBytes', () => {
  it('reads from the settings query cache and falls back to the default', () => {
    const source = readFileSync(
      new URL('./use-media-max-bytes.ts', import.meta.url),
      'utf8'
    )
    expect(source).toContain('queryKeys.settings.all')
    expect(source).toContain('fetchPlatformSettings')
    expect(source).toContain('resolveMediaMaxBytes')
    expect(source).toContain('DEFAULT_MEDIA_MAX_BYTES')
  })
})
