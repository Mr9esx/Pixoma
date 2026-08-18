import { describe, it, expect } from 'vitest'
import { MENU_ITEMS } from './menu'

describe('MENU_ITEMS', () => {
  it('keeps required order', () => {
    expect(MENU_ITEMS.map((m) => m.id)).toEqual([
      'dashboard',
      'edges',
      'cases',
      'channels',
      'tasks',
      'users',
      'sessions',
      'settings',
    ])
  })

  it('maps paths', () => {
    expect(MENU_ITEMS.map((m) => m.path)).toEqual([
      '/',
      '/edges',
      '/cases',
      '/channels',
      '/tasks',
      '/users',
      '/sessions',
      '/settings',
    ])
  })
})
