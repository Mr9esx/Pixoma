import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const LIST_PANEL = join(here, 'channel-list-panel.tsx')
const ROUTE = join(here, '../../routes/_app/channels/route.tsx')

describe('channel layout aligned with compute nodes', () => {
  it('list panel has search only and no header', () => {
    const source = readFileSync(LIST_PANEL, 'utf8')
    expect(source).toContain("data-testid='channels-list-panel'")
    expect(source).toContain('channels.listSearch')
    expect(source).not.toContain('<h2')
  })

  it('route uses 280px master detail with top-right create button', () => {
    const source = readFileSync(ROUTE, 'utf8')
    expect(source).toContain('MasterDetailShell')
    expect(source).toContain('md:grid-cols-[280px_1fr]')
    expect(source).toContain('ChannelDetailPanel')
    expect(source).toContain("to='/channels/new'")
  })
})
