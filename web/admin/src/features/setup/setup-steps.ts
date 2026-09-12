export const SETUP_STEPS = [
  'password',
  'profile',
  'database',
  'storage',
] as const

export type SetupStep = (typeof SETUP_STEPS)[number]

export const SETUP_STEP_COPY: Record<
  SetupStep,
  { title: string; desc: string; submit: string }
> = {
  password: {
    title: 'setup.passwordTitle',
    desc: '',
    submit: 'setup.passwordSubmit',
  },
  profile: {
    title: 'setup.nicknameTitle',
    desc: '',
    submit: 'setup.continue',
  },
  database: {
    title: 'setup.databaseTitle',
    desc: '',
    submit: 'setup.continue',
  },
  storage: {
    title: 'setup.storageTitle',
    desc: '',
    submit: 'setup.saveAndRestart',
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
