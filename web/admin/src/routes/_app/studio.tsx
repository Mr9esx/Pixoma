import { createFileRoute } from '@tanstack/react-router'
import { StudioWorkspace } from '@/features/studio/studio-workspace'

export const Route = createFileRoute('/_app/studio')({
  component: StudioWorkspace,
})
