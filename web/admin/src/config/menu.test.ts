import { describe, it, expect } from 'vitest'
import { MENU_ITEMS } from './menu'

describe('MENU_ITEMS', () => {
  it('keeps required order', () => {
    expect(MENU_ITEMS.map((m) => m.id)).toEqual([
      'dashboard',
      'instances',
      'cases',
      'tg-menu',
      'tasks',
      'users',
      'sessions',
    ])
  })

  it('maps paths', () => {
    expect(MENU_ITEMS.map((m) => m.path)).toEqual([
      '/',
      '/instances',
      '/cases',
      '/tg-menu',
      '/tasks',
      '/users',
      '/sessions',
    ])
  })
})
