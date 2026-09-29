import { afterEach, expect, it } from 'vitest'
import {
  readProjectExpanded,
  writeProjectExpanded,
} from './studio-project-expanded-state'

afterEach(() => window.localStorage.clear())

it('remembers expansion independently for each project', () => {
  expect(readProjectExpanded('project-a')).toBe(false)

  writeProjectExpanded('project-a', true)
  writeProjectExpanded('project-b', true)
  writeProjectExpanded('project-a', false)

  expect(readProjectExpanded('project-a')).toBe(false)
  expect(readProjectExpanded('project-b')).toBe(true)
})
