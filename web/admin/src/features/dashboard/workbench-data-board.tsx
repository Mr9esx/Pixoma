import { WorkbenchContribution } from './workbench-contribution'
import { WorkbenchOverviewCards } from './workbench-overview-cards'
import { WorkbenchChartPairs } from './workbench-chart-pairs'
import { TaskRangePicker, type StatsRange } from './task-range-picker'

export function WorkbenchDataBoard({
  range,
  onChangeRange,
}: {
  range: StatsRange
  onChangeRange: (r: StatsRange) => void
}) {
  return (
    <div data-testid='workbench-data-board' className='min-w-0 space-y-4'>
      <WorkbenchContribution />
      <WorkbenchOverviewCards />
      <WorkbenchChartPairs range={range} />
      <div className='flex items-center justify-end'>
        <TaskRangePicker range={range} onChange={onChangeRange} />
      </div>
    </div>
  )
}
