import { createFileRoute } from '@tanstack/react-router'
import { MenuPuckEditor } from '@/features/menu/menu-puck-editor'

export const Route = createFileRoute('/_app/channels/$id/menu')({
  component: ChannelMenuPage,
})

function ChannelMenuPage() {
  const { id } = Route.useParams()
  return <MenuPuckEditor channelId={id} />
}
