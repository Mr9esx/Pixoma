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
      <div className='flex items-center justify-start'>
        <TaskRangePicker range={range} onChange={onChangeRange} />
      </div>
      <WorkbenchChartPairs range={range} />
    </div>
  )
}
