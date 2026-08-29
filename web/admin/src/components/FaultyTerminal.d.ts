import type * as React from 'react'

declare interface FaultyTerminalProps {
  scale?: number
  gridMul?: [number, number]
  digitSize?: number
  timeScale?: number
  pause?: boolean
  scanlineIntensity?: number
  glitchAmount?: number
  flickerAmount?: number
  noiseAmp?: number
  chromaticAberration?: number
  dither?: number | boolean
  curvature?: number
  tint?: string
  mouseReact?: boolean
  mouseStrength?: number
  pageLoadAnimation?: boolean
  brightness?: number
  dpr?: number
  className?: string
  style?: React.CSSProperties
}

declare const FaultyTerminal: React.FC<FaultyTerminalProps>

export default FaultyTerminal
