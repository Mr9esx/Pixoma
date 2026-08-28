import { WorkbenchContribution } from './workbench-contribution'
import { WorkbenchOverviewCards } from './workbench-overview-cards'
import { WorkbenchChartPairs } from './workbench-chart-pairs'
import { TimeRangeControl, type RangePreset } from '@/components/time-range-control'
import { useTranslation } from 'react-i18next'
import type { StatsRange } from './task-range-picker'

export function WorkbenchDataBoard({
  range,
  onChangeRange,
}: {
  range: StatsRange
  onChangeRange: (r: StatsRange) => void
}) {
  const { t } = useTranslation()
  const presets: RangePreset[] = [
    { value: '7d', label: t('dashboard.range7d'), days: 7 },
    { value: '30d', label: t('dashboard.range30d'), days: 30 },
    { value: '90d', label: t('dashboard.range90d'), days: 90 },
  ]

  return (
    <div data-testid='workbench-data-board' className='min-w-0 space-y-4'>
      <WorkbenchContribution />
      <WorkbenchOverviewCards />
      <div className='flex items-center justify-start'>
        <TimeRangeControl
          presets={presets}
          from={range.from}
          to={range.to}
          onChange={onChangeRange}
        />
      </div>
      <WorkbenchChartPairs range={range} />
    </div>
  )
}
