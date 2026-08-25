import { describe, expect, it } from 'vitest'
import {
  initialSetupStep,
  previousSetupStep,
  setupStepIndex,
  setupStepsFor,
} from './setup-steps'

describe('setupStepsFor', () => {
  it('keeps password as the first step when it must be changed', () => {
    expect(setupStepsFor(true)[0]).toBe('password')
    expect(setupStepsFor(true)).toHaveLength(3)
  })

  it('skips password after it has already been changed', () => {
    expect(setupStepsFor(false)[0]).toBe('database')
    expect(setupStepsFor(false)).not.toContain('password')
    expect(setupStepsFor(false)).toEqual(['database', 'storage'])
  })
})

describe('initialSetupStep', () => {
  it('starts at password when the account still uses the bootstrap secret', () => {
    expect(
      initialSetupStep({ must_change_password: true, wizard_step: 'storage' })
    ).toBe('password')
  })

  it('resumes a saved wizard step after password is done', () => {
    expect(
      initialSetupStep({ must_change_password: false, wizard_step: 'storage' })
    ).toBe('storage')
  })

  it('maps a leftover edge step to the final storage step', () => {
    expect(
      initialSetupStep({ must_change_password: false, wizard_step: 'edge' })
    ).toBe('storage')
  })
})

describe('previousSetupStep', () => {
  const withPassword = setupStepsFor(true)
  const withoutPassword = setupStepsFor(false)

  it('has no back action on the first step', () => {
    expect(previousSetupStep(withPassword, 'password')).toBeNull()
    expect(previousSetupStep(withoutPassword, 'database')).toBeNull()
  })

  it('returns the previous step so the wizard can go back', () => {
    expect(previousSetupStep(withPassword, 'database')).toBe('password')
    expect(previousSetupStep(withoutPassword, 'storage')).toBe('database')
  })

  it('tracks progress index from 0', () => {
    expect(setupStepIndex(withPassword, 'password')).toBe(0)
    expect(setupStepIndex(withPassword, 'storage')).toBe(2)
  })
})
