import { describe, expect, it } from 'vitest'
import { PHONE_HEIGHT, PHONE_WIDTH, fitPhoneSize } from './fit-phone'

describe('fitPhoneSize', () => {
  it('does not scale the phone above its native size', () => {
    expect(fitPhoneSize(800, 800)).toEqual({
      width: PHONE_WIDTH,
      height: PHONE_HEIGHT,
    })
  })

  it('fits a short pane by height without using container query units', () => {
    expect(fitPhoneSize(800, 400)).toEqual({
      width: Math.round((PHONE_WIDTH / PHONE_HEIGHT) * 400),
      height: 400,
    })
  })

  it('fits a narrow pane by width', () => {
    expect(fitPhoneSize(200, 800)).toEqual({
      width: 200,
      height: Math.round((PHONE_HEIGHT / PHONE_WIDTH) * 200),
    })
  })

  it('leaves room for pane padding so the phone is not clipped', () => {
    const padded = fitPhoneSize(800 - 24, 400 - 24)
    expect(padded.height).toBe(376)
    expect(padded.height).toBeLessThan(400)
  })

  it('ignores collapsed panes', () => {
    expect(fitPhoneSize(0, 400)).toEqual({ width: 0, height: 0 })
  })
})
