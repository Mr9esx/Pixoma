import type { ComponentType, SVGProps } from 'react'

export interface Option {
  label: string
  value: string
  count?: number
  icon?: ComponentType<SVGProps<SVGSVGElement>>
}
