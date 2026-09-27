import { ReactFlowProvider } from '@xyflow/react'
import { CanvasBackground } from '@/components/ai-elements/canvas'

export function StudioWorkbenchBackground() {
  return (
    <div
      aria-hidden='true'
      className='pointer-events-none absolute inset-0 isolate bg-card'
    >
      <ReactFlowProvider>
        <CanvasBackground />
      </ReactFlowProvider>
    </div>
  )
}
