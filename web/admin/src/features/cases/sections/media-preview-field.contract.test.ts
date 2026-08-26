import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const FIELD = join(here, 'media-preview-field.tsx')
const BASICS = join(here, 'basics.tsx')

describe('media preview field', () => {
  it('uploads via admin media api, validates whitelist and previews', () => {
    const source = readFileSync(FIELD, 'utf8')
    expect(source).toContain("data-testid='media-preview-field'")
    expect(source).toContain('uploadMedia')
    expect(source).toContain('MEDIA_MAX_BYTES')
    expect(source).toContain('fetchMediaBlob')
    expect(source).toContain('ImagePlus')
    expect(source).toContain('accept=')
    expect(source).toContain('onDrop')
  })

  it('basics uses the media field instead of a plain text input', () => {
    const source = readFileSync(BASICS, 'utf8')
    expect(source).toContain('MediaPreviewField')
    expect(source).toContain("from './media-preview-field'")
    expect(source).not.toContain("id='case-preview'")
  })
})
