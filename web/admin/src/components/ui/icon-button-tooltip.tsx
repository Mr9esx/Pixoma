import type { ReactElement } from 'react'
import { Tooltip, TooltipContent, TooltipTrigger } from './tooltip'

export function IconButtonTooltip({
  label,
  children,
}: {
  label: string
  children: ReactElement
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>{children}</TooltipTrigger>
      <TooltipContent
        side='top'
        data-icon-button-tooltip=''
        style={{ pointerEvents: 'none' }}
      >
        {label}
      </TooltipContent>
    </Tooltip>
  )
}
