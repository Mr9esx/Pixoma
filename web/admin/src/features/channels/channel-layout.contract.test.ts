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

  it('route uses 280px master detail with create in a dialog', () => {
    const source = readFileSync(ROUTE, 'utf8')
    expect(source).toContain('MasterDetailShell')
    expect(source).toContain('md:grid-cols-[280px_1fr]')
    expect(source).toContain('ChannelDetailPanel')
    expect(source).toContain('CreateChannelForm')
    expect(source).toContain('<Dialog open={createOpen}')
    expect(source).toContain('onOpenChange={setCreateOpen}')
    expect(source).toMatch(/hasSelection=\{Boolean\(selectedId\)\}/)
    expect(source).not.toContain("id === 'new'")
  })

  it('auto-selects the first channel and navigates the url', () => {
    const source = readFileSync(ROUTE, 'utf8')
    expect(source).toContain('items[0]?.id')
    expect(source).toContain('replace: true')
    expect(source).toContain('backToList')
  })

  it('create form keeps actions in a sticky bottom footer', () => {
    const form = readFileSync(join(here, 'create-channel-form.tsx'), 'utf8')
    expect(form).toMatch(/DialogFooter/)
    expect(form).toMatch(/flex flex-1 flex-col gap-4/)
    expect(form).not.toMatch(/max-w-xl/)
  })

  it('detail panel runs a one-time reachability check and guides to proxy settings', () => {
    const detail = readFileSync(join(here, 'channel-detail-panel.tsx'), 'utf8')
    const api = readFileSync(join(here, '../../lib/api/channels.ts'), 'utf8')
    expect(api).toMatch(/checkChannelReachability/)
    expect(api).toMatch(/\/check`/)
    expect(api).toMatch(/adapter_state/)
    expect(detail).toMatch(/checkChannelReachability\(id\)/)
    expect(detail).toMatch(/channelReferences\(/)
    expect(detail).toMatch(/<LinkHealthAlert/)
    expect(detail).toMatch(/<LinkHealthSection/)
    expect(detail).toMatch(/linkHealth\.title/)
    expect(detail).toMatch(
      /<section id='channel-menu-section'[\s\S]*?<LinkHealthSection/
    )
    expect(detail).not.toMatch(/upstream=\{\{/)
    expect(detail).toMatch(/ChannelReachabilityTag/)
    expect(detail).toMatch(/from '@\/components\/kibo-ui\/pill'/)
    expect(detail).not.toMatch(/kit\.tag/)
    expect(detail).toMatch(/channels\.reachabilityNetwork/)
    expect(detail).not.toMatch(/to='\/settings'/)
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
