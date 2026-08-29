import { createFileRoute } from '@tanstack/react-router'
import { VisualConfigPage } from '@/features/visual-config/visual-config-page'

export const Route = createFileRoute('/_app/visual-config')({
  component: VisualConfigPage,
})
