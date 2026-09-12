export const LIST_TASKS_BUILTIN =
  '我的任务\n\n当前任务\n{{ current }}\n\n最近任务\n{{ recent }}'

const LIST_TASKS_SAMPLE = {
  current: '当前没有排队中的任务',
  recent: '#1120186 · 工作流名 · 成功 · 08-30 07:50',
}

export function interpolate(
  template: string,
  vars: Record<string, string>
): string {
  return template.replace(/\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}\}/g, (_, key) =>
    key in vars ? vars[key] : `{{ ${key} }}`
  )
}

export function composeListTasksPreview(
  overrides: Record<string, string> = {}
): string {
  const template = overrides.list_tasks || LIST_TASKS_BUILTIN
  return interpolate(template, LIST_TASKS_SAMPLE)
}
