export const LIST_TASKS_SAMPLE_ZH = [
  '我的任务',
  '',
  '当前任务',
  '✅ 当前没有排队中的任务',
  '',
  '最近任务',
  '✅ #1120186 · 工作流名 · 已完成 · 08-30 07:50',
].join('\n')

export function ListTasksPreview() {
  return (
    <p
      data-testid='list-tasks-preview'
      className='m-0 rounded-md border border-border bg-background px-3.5 py-3 text-sm whitespace-pre-wrap'
    >
      {LIST_TASKS_SAMPLE_ZH}
    </p>
  )
}
