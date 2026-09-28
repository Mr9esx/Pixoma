import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { StudioContext } from './studio-context'
import { StudioTrace } from './studio-trace'

export function StudioTraceTabs({ sessionId }: { sessionId: string }) {
  return (
    <Tabs defaultValue='trace' className='min-h-0 flex-1 gap-0'>
      <div className='flex h-10 shrink-0 items-center border-b bg-card px-3'>
        <TabsList aria-label='会话轨迹页面' className='h-8'>
          <TabsTrigger
            value='trace'
            className='data-[state=active]:shadow-none'
          >
            Trace
          </TabsTrigger>
          <TabsTrigger
            value='context'
            className='data-[state=active]:shadow-none'
          >
            上下文
          </TabsTrigger>
        </TabsList>
      </div>
      <TabsContent value='trace' className='min-h-0 overflow-hidden'>
        <StudioTrace sessionId={sessionId} />
      </TabsContent>
      <TabsContent value='context' className='min-h-0 overflow-hidden'>
        <StudioContext sessionId={sessionId} />
      </TabsContent>
    </Tabs>
  )
}
