import type * as React from 'react'

type DitherProps = {
  waveColor?: [number, number, number]
  backgroundColor?: [number, number, number]
  waveAmplitude?: number
  waveFrequency?: number
  waveSpeed?: number
  colorNum?: number
  pixelSize?: number
  disableAnimation?: boolean
  enableMouseInteraction?: boolean
  mouseRadius?: number
}

declare const Dither: React.FC<DitherProps>

export default Dither
