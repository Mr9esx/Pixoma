export const SETUP_STEPS = [
  'password',
  'database',
  'placement',
  'storage',
] as const

export type SetupStep = (typeof SETUP_STEPS)[number]

export const SETUP_STEP_COPY: Record<
  SetupStep,
  { title: string; desc: string; submit: string }
> = {
  password: {
    title: '设管理员密码',
    desc: '设个新的管理员密码。',
    submit: '保存密码',
  },
  database: {
    title: '数据库配置',
    desc: '',
    submit: '继续',
  },
  placement: {
    title: '出图机器在哪',
    desc: 'Comfy 和后台是否在同一台电脑。远程 GPU 必须用对象存储。',
    submit: '下一步',
  },
  storage: {
    title: '文件放哪',
    desc: '生成的图和视频存在本机目录，或对象存储。',
    submit: '完成',
  },
}

export function setupStepsFor(mustChangePassword: boolean): SetupStep[] {
  if (mustChangePassword) return [...SETUP_STEPS]
  return SETUP_STEPS.filter((step) => step !== 'password')
}

export function initialSetupStep(status: {
  must_change_password: boolean
  wizard_step?: string
}): SetupStep {
  if (status.must_change_password) return 'password'
  const steps = setupStepsFor(false)
  if (status.wizard_step === 'edge') return 'storage'
  const saved = status.wizard_step as SetupStep | undefined
  if (saved && steps.includes(saved)) return saved
  return 'database'
}

export function previousSetupStep(
  steps: readonly SetupStep[],
  current: SetupStep
): SetupStep | null {
  const index = steps.indexOf(current)
  if (index <= 0) return null
  return steps[index - 1] ?? null
}

export function setupStepIndex(
  steps: readonly SetupStep[],
  current: SetupStep
): number {
  const index = steps.indexOf(current)
  return index < 0 ? 0 : index
}
