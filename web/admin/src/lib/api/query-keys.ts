export const queryKeys = {
  edges: {
    all: ['edges'] as const,
    presence: ['edges', 'presence'] as const,
    detail: (id: string) => ['edges', id] as const,
    tasks: (id: string, offset: number) =>
      ['edges', id, 'tasks', offset] as const,
    stats: (id: string) => ['edges', id, 'stats'] as const,
    metrics: (id: string) => ['edges', id, 'metrics'] as const,
  },
  cases: {
    all: ['cases'] as const,
    detail: (id: number) => ['cases', id] as const,
    menuPlacements: (id: number) => ['cases', id, 'menu-placements'] as const,
  },
  tasks: {
    all: ['tasks'] as const,
    detail: (id: string) => ['tasks', id] as const,
  },
  stats: {
    tasksDaily: (from: string, to: string) =>
      ['stats', 'tasks', 'daily', from, to] as const,
    tasksErrors: (from: string, to: string) =>
      ['stats', 'tasks', 'errors', from, to] as const,
    tasksEdges: (from: string, to: string) =>
      ['stats', 'tasks', 'edges', from, to] as const,
  },
  users: {
    all: ['users'] as const,
    detail: (id: string) => ['users', id] as const,
  },
  sessions: {
    all: ['sessions'] as const,
    detail: (id: string) => ['sessions', id] as const,
  },
  channels: {
    all: ['channels'] as const,
    detail: (id: string) => ['channels', id] as const,
    menu: (id: string) => ['channels', id, 'menu'] as const,
    extras: (id: string) => ['channels', id, 'menu', 'extras'] as const,
  },
  settings: {
    all: ['settings'] as const,
  },
}
