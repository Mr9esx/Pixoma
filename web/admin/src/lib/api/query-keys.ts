export const queryKeys = {
  edges: {
    all: ['edges'] as const,
    presence: ['edges', 'presence'] as const,
    detail: (id: string) => ['edges', id] as const,
    tasks: (id: string, offset: number) =>
      ['edges', id, 'tasks', offset] as const,
    stats: (id: string) => ['edges', id, 'stats'] as const,
    metrics: (id: string, window: string) =>
      ['edges', id, 'metrics', window] as const,
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
    casesTop: (from: string, to: string) =>
      ['stats', 'cases', 'top', from, to] as const,
    fleet: ['stats', 'fleet'] as const,
  },
  users: {
    all: ['users'] as const,
    detail: (id: string) => ['users', id] as const,
  },
  adminUsers: {
    all: ['adminUsers'] as const,
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
  topics: {
    all: ['topics'] as const,
    detail: (key: string) => ['topics', key] as const,
    stats: (key: string, range: { from: string; to: string }) =>
      ['topics', key, 'stats', range] as const,
  },
  settings: {
    all: ['settings'] as const,
  },
  textTemplates: {
    all: ['text-templates'] as const,
    channel: (channelId: string) => ['text-templates', channelId] as const,
  },
}
