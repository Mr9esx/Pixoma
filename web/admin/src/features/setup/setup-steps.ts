export const SETUP_STEPS = [
  'password',
  'database',
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
  storage: {
    title: '文件存储配置',
    desc: '决定了生成的图和视频存放的位置。',
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
