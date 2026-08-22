import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { CaseAnalysisSection } from './case-analysis-section'
import { daysAgo } from './date-range'
import { RealtimeStatusSection } from './realtime-status-section'
import { TaskRangePicker, type StatsRange } from './task-range-picker'
import { TaskStatsSection } from './task-stats-section'

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
      <div>
        <h1 className='text-2xl font-bold tracking-tight'>
          {t('dashboard.title')}
        </h1>
        <p className='text-sm text-muted-foreground'>
          {t('dashboard.fullAccuracyNote')}
        </p>
      </div>

      <section aria-label={t('dashboard.realtimeStatus')}>
        <div className='mb-3 flex flex-wrap items-baseline gap-2'>
          <h2 className='text-base font-semibold'>{t('dashboard.realtimeStatus')}</h2>
          <span className='text-xs text-muted-foreground'>
            {t('dashboard.realtimeBadge')}
          </span>
        </div>
        <RealtimeStatusSection />
      </section>

      <section aria-label={t('dashboard.taskPerformance')}>
        <div className='mb-3 flex flex-wrap items-center justify-between gap-3'>
          <div>
            <h2 className='text-base font-semibold'>{t('dashboard.taskPerformance')}</h2>
            <p className='text-xs text-muted-foreground'>{t('dashboard.taskPerfDesc')}</p>
          </div>
          <TaskRangePicker range={range} onChange={setRange} />
        </div>
        <TaskStatsSection range={range} />
      </section>

      <section aria-label={t('dashboard.businessAnalysis')}>
        <div className='mb-3'>
          <h2 className='text-base font-semibold'>{t('dashboard.businessAnalysis')}</h2>
          <p className='text-xs text-muted-foreground'>{t('dashboard.businessDesc')}</p>
        </div>
        <CaseAnalysisSection range={range} />
      </section>
    </div>
  )
}
