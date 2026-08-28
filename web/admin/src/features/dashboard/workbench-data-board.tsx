import { WorkbenchContribution } from './workbench-contribution'
import { WorkbenchOverviewCards } from './workbench-overview-cards'
import { WorkbenchChartPairs } from './workbench-chart-pairs'
import type { StatsRange } from './task-range-picker'

export function WorkbenchDataBoard({ range }: { range: StatsRange }) {
  return (
    <div data-testid='workbench-data-board' className='min-w-0 space-y-4'>
      <WorkbenchContribution range={range} />
      <WorkbenchOverviewCards />
      <WorkbenchChartPairs range={range} />
    </div>
  )
}
