import { describe, it, expect } from 'vitest'
import { MENU_GROUPS, MENU_ITEMS } from './menu'

describe('MENU_GROUPS', () => {
  it('keeps required group and item order', () => {
    expect(MENU_GROUPS.map((g) => g.items.map((m) => m.id))).toEqual([
      ['dashboard', 'quick-config'],
      ['edges', 'cases', 'channels', 'topics'],
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
      'edges',
      'cases',
      'channels',
      'topics',
      'tasks',
      'sessions',
      'users',
      'settings',
    ])
    expect(MENU_ITEMS.map((m) => m.path)).toEqual([
      '/',
      '/quick-config',
      '/edges',
      '/cases',
      '/channels',
      '/topics',
      '/tasks',
      '/sessions',
      '/users',
      '/settings',
    ])
  })
})
