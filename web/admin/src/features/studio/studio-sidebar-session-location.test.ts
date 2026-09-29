import { expect, it } from 'vitest'
import { isActiveStudioSessionItem } from './studio-sidebar-session-location'

it('marks only the list entry from which a project conversation was selected', () => {
  const common = {
    activeSessionId: 'session-1',
    sessionId: 'session-1',
    sessionProjectId: 'project-1',
  }

  expect(
    isActiveStudioSessionItem({
      ...common,
      selection: { sessionId: 'session-1', source: 'project' },
      source: 'project',
    })
  ).toBe(true)
  expect(
    isActiveStudioSessionItem({
      ...common,
      selection: { sessionId: 'session-1', source: 'project' },
      source: 'recent',
    })
  ).toBe(false)
})

it('marks only the recent entry when a project conversation is selected there', () => {
  const common = {
    activeSessionId: 'session-1',
    sessionId: 'session-1',
    sessionProjectId: 'project-1',
    selection: { sessionId: 'session-1', source: 'recent' as const },
  }

  expect(isActiveStudioSessionItem({ ...common, source: 'recent' })).toBe(true)
  expect(isActiveStudioSessionItem({ ...common, source: 'project' })).toBe(
    false
  )
})

it('uses project membership for a session opened directly', () => {
  expect(
    isActiveStudioSessionItem({
      activeSessionId: 'project-session',
      sessionId: 'project-session',
      sessionProjectId: 'project-1',
      source: 'project',
    })
  ).toBe(true)
  expect(
    isActiveStudioSessionItem({
      activeSessionId: 'project-session',
      sessionId: 'project-session',
      sessionProjectId: 'project-1',
      source: 'recent',
    })
  ).toBe(false)
})

it('uses the recent entry after a selected project session leaves its project', () => {
  const common = {
    activeSessionId: 'session-1',
    selection: { sessionId: 'session-1', source: 'project' as const },
    sessionId: 'session-1',
  }

  expect(isActiveStudioSessionItem({ ...common, source: 'recent' })).toBe(true)
  expect(isActiveStudioSessionItem({ ...common, source: 'project' })).toBe(
    false
  )
})
