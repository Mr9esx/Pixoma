import { createFileRoute } from '@tanstack/react-router'
import { QuickConfigPage } from '@/features/quick-config/quick-config-page'

export const Route = createFileRoute('/_app/quick-config')({
  component: QuickConfigPage,
})
