import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { daysAgo } from './date-range'
import { TaskRangePicker, type StatsRange } from './task-range-picker'
import { WorkbenchWelcomeCard } from './workbench-welcome-card'
import { WorkbenchDataBoard } from './workbench-data-board'
import { WorkbenchAttention } from './workbench-attention'

export function DashboardPage() {
  const { t } = useTranslation()
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

      <div className='flex flex-wrap items-center justify-between gap-3'>
        <h2 className='text-base font-semibold'>
          {t('dashboard.workbench.attentionTitle')}
        </h2>
        <TaskRangePicker range={range} onChange={setRange} />
      </div>

      <div
        data-testid='workbench-grid'
        className='grid gap-4 lg:grid-cols-[minmax(0,1fr)_360px]'
      >
        <WorkbenchDataBoard range={range} />
        <WorkbenchAttention range={range} />
      </div>
    </div>
  )
}
