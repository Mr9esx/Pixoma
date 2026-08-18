export const queryKeys = {
  edges: {
    all: ['edges'] as const,
    presence: ['edges', 'presence'] as const,
    detail: (id: string) => ['edges', id] as const,
    tasks: (id: string) => ['edges', id, 'tasks'] as const,
    stats: (id: string) => ['edges', id, 'stats'] as const,
    metrics: (id: string) => ['edges', id, 'metrics'] as const,
  },
  cases: {
    all: ['cases'] as const,
    detail: (id: string) => ['cases', id] as const,
    menuPlacements: (id: string) => ['cases', id, 'menu-placements'] as const,
  },
  tasks: {
    all: ['tasks'] as const,
    detail: (id: string) => ['tasks', id] as const,
  },
  users: {
    all: ['users'] as const,
    detail: (id: string) => ['users', id] as const,
  },
  sessions: {
    all: ['sessions'] as const,
    detail: (id: string) => ['sessions', id] as const,
  },
  tgMenu: {
    all: ['tg-menu'] as const,
  },
  settings: {
    all: ['settings'] as const,
  },
}
