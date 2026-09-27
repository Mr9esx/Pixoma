import type { StudioMCPConnector } from '@/lib/api/studio'

export function connectorStatus(
  connector: Pick<StudioMCPConnector, 'enabled' | 'policy' | 'tools'>
) {
  if (!connector.enabled) {
    return { state: 'warn', text: '已停用', reason: '已停用' } as const
  }
  if (connector.policy === 'forbidden') {
    return {
      state: 'warn',
      text: '当前不可用',
      reason: '调用策略禁止调用',
    } as const
  }
  if (!connector.tools?.length) {
    return {
      state: 'warn',
      text: '当前不可用',
      reason: '尚未发现工具',
    } as const
  }
  return { state: 'ok', text: '已启用', reason: '配置正常' } as const
}
