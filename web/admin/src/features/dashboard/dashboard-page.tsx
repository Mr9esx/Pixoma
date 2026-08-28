import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { daysAgo } from './date-range'
import type { StatsRange } from './task-range-picker'
import { WorkbenchWelcomeCard } from './workbench-welcome-card'
import { WorkbenchDataBoard } from './workbench-data-board'

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
      <WorkbenchWelcomeCard />

      <div
        data-testid='workbench-grid'
        className='grid gap-4 lg:grid-cols-[minmax(0,1fr)_360px]'
      >
        <WorkbenchDataBoard range={range} onChangeRange={setRange} />
        <PlaceholderCard />
      </div>
    </div>
  )
}

function PlaceholderCard() {
  const { t } = useTranslation()
  return (
    <div data-testid='workbench-placeholder' className='rounded-xl border bg-card text-muted-foreground'>
      <div className='flex h-full min-h-[200px] items-center justify-center p-6 text-sm'>
        {t('dashboard.workbench.attentionPlaceholder')}
      </div>
    </div>
  )
}
