import type { StudioAsset } from '@/lib/api/studio'

export function originLabel(origin: StudioAsset['origin']) {
  return {
    user: '用户创建',
    agent: 'Agent 生成',
    model: '模型生成',
    workflow: '工作流产出',
    library: '资产库引用',
  }[origin]
}
