export const PHONE_WIDTH = 375
export const PHONE_HEIGHT = 667

export function fitPhoneSize(
  boxW: number,
  boxH: number,
  aspectW = PHONE_WIDTH,
  aspectH = PHONE_HEIGHT
): { width: number; height: number } {
  if (boxW < 8 || boxH < 8) return { width: 0, height: 0 }
  const s = Math.min(boxW / aspectW, boxH / aspectH, 1)
  return {
    width: Math.round(aspectW * s),
    height: Math.round(aspectH * s),
  }
}
