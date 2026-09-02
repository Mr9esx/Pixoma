import { createFileRoute, Outlet } from '@tanstack/react-router'

export const Route = createFileRoute('/_app/channels/$id')({
  component: ChannelIdOutlet,
})

function ChannelIdOutlet() {
  return (
    <div className='flex h-full min-h-0 flex-1 flex-col'>
      <Outlet />
    </div>
  )
}
