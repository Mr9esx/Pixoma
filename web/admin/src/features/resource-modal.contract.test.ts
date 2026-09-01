import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))

function read(rel: string) {
  return readFileSync(join(here, rel), 'utf8')
}

describe('resource detail modal', () => {
  it('aligns width with the edge creation modal', () => {
    const constants = read('./resource-modal.ts')
    const edgeModal = read('../routes/_app/edges/route.tsx')
    expect(constants).toMatch(/sm:max-w-\[504px\]/)
    expect(edgeModal).toMatch(/sm:max-w-\[504px\]/)
  })

  it('uses one dialog body rule for task, session, and user details', () => {
    const routes = [
      '../routes/_app/tasks/route.tsx',
      '../routes/_app/sessions/route.tsx',
      '../routes/_app/users/route.tsx',
    ]
    for (const route of routes) {
      const source = read(route)
      expect(source).toMatch(/resourceDetailDialogClassName/)
      expect(source).toMatch(/resourceDetailBodyClassName/)
      expect(source).not.toMatch(/sm:max-w-3xl|px-5 py-4/)
    }
  })

  it('hides cancel unless the task is cancellable', () => {
    const source = read('./tasks/detail-panel.tsx')
    expect(source).toMatch(
      /cancellableStatuses = new Set\(\['pending', 'queued'\]\)/
    )
    expect(source).toMatch(/cancellableStatuses\.has\(task\.status\)/)
  })
})
