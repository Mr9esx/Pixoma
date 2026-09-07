import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))

function read(rel: string) {
  return readFileSync(join(here, rel), 'utf8')
}

describe('resource detail modal', () => {
  it('uses a 640px operations dialog, not the create-form width', () => {
    const constants = read('./resource-modal.ts')
    expect(constants).toMatch(/sm:max-w-\[640px\]/)
    expect(constants).not.toMatch(/sm:max-w-\[504px\]/)
  })

  it('shares one operations dialog for task, session, and user pages', () => {
    const routes = [
      '../routes/_app/tasks/route.tsx',
      '../routes/_app/sessions/route.tsx',
      '../routes/_app/users/route.tsx',
    ]
    for (const route of routes) {
      const source = read(route)
      expect(source).toMatch(/OperationsDetailDialog/)
      expect(source).not.toMatch(/sm:max-w-3xl|px-5 py-4/)
    }
  })

  it('hides cancel unless the task is cancellable, and puts it in the footer', () => {
    const source = read('./tasks/detail-panel.tsx')
    expect(source).toMatch(
      /cancellableStatuses = new Set\(\['pending', 'queued'\]\)/
    )
    expect(source).toMatch(/cancellableStatuses\.has\(task\.status\)/)
    expect(source).toMatch(/DialogFooter/)
  })

  it('leads with identity, related chips, and identifiers—not a second heading', () => {
    const task = read('./tasks/detail-panel.tsx')
    const session = read('./sessions/detail-panel.tsx')
    const user = read('./users/detail-panel.tsx')
    for (const source of [task, session, user]) {
      expect(source).toMatch(/ResourceDetailLayout/)
    }
    expect(task).toMatch(/fieldStartedAt/)
    expect(task).toMatch(/dispatch_topic/)
    expect(session).not.toMatch(/JSON\.stringify/)
    expect(user).not.toMatch(/fieldFirstName/)
  })

  it('keeps the title and status on the left, aligned with the close button', () => {
    const layout = read('./operations/detail-layout.tsx')
    const dialog = read('./operations/detail-dialog.tsx')
    expect(layout).toMatch(/DialogTitle/)
    expect(layout).toMatch(/pr-8/)
    expect(layout).toMatch(/flex min-w-0 items-center gap-2/)
    expect(layout).not.toMatch(/justify-between/)
    expect(dialog).not.toMatch(/sr-only/)
  })
})
