import { describe, it, expect } from 'vitest'
import { MENU_GROUPS, MENU_ITEMS, filterMenuGroupsForDemo } from './menu'

describe('MENU_GROUPS', () => {
  it('keeps required group and item order', () => {
    expect(MENU_GROUPS.map((g) => g.items.map((m) => m.id))).toEqual([
      ['dashboard', 'quick-config'],
      ['cases', 'channels', 'topics', 'edges'],
      ['tasks', 'sessions', 'users'],
      ['settings'],
    ])
  })

  it('labels only grouped sections', () => {
    expect(MENU_GROUPS.map((g) => g.titleKey)).toEqual([
      undefined,
      'menu.groupConfig',
      'menu.groupOperations',
      'menu.groupSystem',
    ])
  })
})

describe('MENU_ITEMS', () => {
  it('flattens all items in order', () => {
    expect(MENU_ITEMS.map((m) => m.id)).toEqual([
      'dashboard',
      'quick-config',
      'cases',
      'channels',
      'topics',
      'edges',
      'tasks',
      'sessions',
      'users',
      'settings',
    ])
    expect(MENU_ITEMS.map((m) => m.path)).toEqual([
      '/',
      '/quick-config',
      '/cases',
      '/channels',
      '/topics',
      '/edges',
      '/tasks',
      '/sessions',
      '/users',
      '/settings',
    ])
  })
})

describe('filterMenuGroupsForDemo', () => {
  it('hides settings and removes empty system group in demo mode', () => {
    expect(
      filterMenuGroupsForDemo(MENU_GROUPS, true).map((group) => ({
        titleKey: group.titleKey,
        items: group.items.map((item) => item.id),
      })),
    ).toEqual([
      { titleKey: undefined, items: ['dashboard', 'quick-config'] },
      { titleKey: 'menu.groupConfig', items: ['cases', 'channels', 'topics', 'edges'] },
      { titleKey: 'menu.groupOperations', items: ['tasks', 'sessions', 'users'] },
    ])
  })

  it('keeps all groups in normal mode', () => {
    expect(filterMenuGroupsForDemo(MENU_GROUPS, false)).toEqual(MENU_GROUPS)
  })
})
