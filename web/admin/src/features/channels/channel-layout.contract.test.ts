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

  it('route uses 280px master detail with create form in the detail pane', () => {
    const source = readFileSync(ROUTE, 'utf8')
    expect(source).toContain('MasterDetailShell')
    expect(source).toContain('md:grid-cols-[280px_1fr]')
    expect(source).toContain('ChannelDetailPanel')
    expect(source).toContain('CreateChannelForm')
    expect(source).toContain("id === 'new'")
    expect(source).toContain("to='/channels/$id' params={{ id: 'new' }}")
    expect(source).toMatch(
      /hasSelection=\{Boolean\(selectedId\) \|\| id === 'new'\}/
    )
    expect(source).not.toContain('setCreateOpen')
  })

  it('auto-selects the first channel and navigates the url', () => {
    const source = readFileSync(ROUTE, 'utf8')
    expect(source).toContain('items[0]?.id')
    expect(source).toContain('replace: true')
    expect(source).toContain('backToList')
  })

  it('delete is available while enabled and shows impact', () => {
    const source = readFileSync(join(here, 'channel-detail-panel.tsx'), 'utf8')
    expect(source).toContain('disabled={deleteMutation.isPending')
    expect(source).not.toContain('ch.enabled || deleteMutation.isPending')
    expect(source).toContain('channels.deleteWillEndSessions')
    expect(source).toContain('channels.deleteInFlightTasks')
    expect(source).toContain('channels.deleteAckImpact')
  })
})
