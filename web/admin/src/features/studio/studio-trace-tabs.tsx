import { useState } from 'react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { StudioContext } from './studio-context'
import { StudioTrace } from './studio-trace'

export function StudioTraceTabs({ sessionId }: { sessionId: string }) {
  const [activeTab, setActiveTab] = useState<'trace' | 'context'>('trace')
  const [toolbarTarget, setToolbarTarget] = useState<HTMLDivElement | null>(
    null
  )

  return (
    <Tabs
      value={activeTab}
      onValueChange={(value) => setActiveTab(value as 'trace' | 'context')}
      className='min-h-0 flex-1 gap-0'
    >
      <div className='flex min-w-0 shrink-0 items-center gap-2 border-t bg-card px-4 py-2'>
        <TabsList aria-label='会话轨迹页面' className='shrink-0'>
          <TabsTrigger value='trace'>Trace</TabsTrigger>
          <TabsTrigger value='context'>上下文</TabsTrigger>
        </TabsList>
        <div
          ref={setToolbarTarget}
          role='toolbar'
          aria-label='轨迹工具栏'
          hidden={activeTab !== 'trace'}
          className='ml-auto flex min-w-0 items-center justify-end gap-1 overflow-x-auto'
        />
      </div>
      <TabsContent
        value='trace'
        className='flex min-h-0 flex-col overflow-hidden'
      >
        <StudioTrace sessionId={sessionId} toolbarTarget={toolbarTarget} />
      </TabsContent>
      <TabsContent
        value='context'
        className='flex min-h-0 flex-col overflow-hidden'
      >
        <StudioContext sessionId={sessionId} />
      </TabsContent>
    </Tabs>
  )
}
