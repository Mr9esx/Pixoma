import { describe, expect, it } from 'vitest'
import { connectorStatus } from './studio-connector-status'

describe('connectorStatus', () => {
  it('marks a usable connector as enabled', () => {
    expect(
      connectorStatus({
        enabled: true,
        policy: 'approval',
        tools: [{ name: 'search', description: 'Search' }],
      })
    ).toEqual({ state: 'ok', text: '已启用', reason: '配置正常' })
  })

  it('marks an enabled connector without tools as unavailable', () => {
    expect(
      connectorStatus({ enabled: true, policy: 'approval', tools: null })
    ).toEqual({ state: 'warn', text: '当前不可用', reason: '尚未发现工具' })
  })

  it('marks forbidden and disabled connectors as unavailable', () => {
    expect(
      connectorStatus({
        enabled: true,
        policy: 'forbidden',
        tools: [{ name: 'search', description: 'Search' }],
      })
    ).toEqual({ state: 'warn', text: '当前不可用', reason: '调用策略禁止调用' })
    expect(
      connectorStatus({
        enabled: false,
        policy: 'approval',
        tools: [{ name: 'search', description: 'Search' }],
      })
    ).toEqual({ state: 'warn', text: '已停用', reason: '已停用' })
  })
})
