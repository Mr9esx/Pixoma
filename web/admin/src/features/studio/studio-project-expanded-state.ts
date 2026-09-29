const storageKey = (projectId: string) => `studio.projectExpanded.${projectId}`

export function readProjectExpanded(projectId: string): boolean {
  return window.localStorage.getItem(storageKey(projectId)) === 'true'
}

export function writeProjectExpanded(
  projectId: string,
  expanded: boolean
): void {
  window.localStorage.setItem(storageKey(projectId), String(expanded))
}
