import { useState } from 'react'
import { daysAgo, type StatsRange } from './date-range'
import { WorkbenchDataBoard } from './workbench-data-board'
import { WorkbenchWelcomeCard } from './workbench-welcome-card'

export function DashboardPage() {
  const [range, setRange] = useState<StatsRange>({
    from: daysAgo(29),
    to: daysAgo(0),
  })

  return (
    <div
      data-testid='dashboard-page'
      className='min-h-0 flex-1 space-y-6 overflow-auto'
    >
      <div
        data-testid='workbench-grid'
        className='grid gap-4 lg:grid-cols-[minmax(0,1fr)_360px]'
      >
        <WorkbenchDataBoard range={range} onChangeRange={setRange} />
        <WorkbenchWelcomeCard />
      </div>
    </div>
  )
}
