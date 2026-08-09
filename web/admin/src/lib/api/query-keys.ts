export const queryKeys = {
  instances: {
    all: ['instances'] as const,
    detail: (id: string) => ['instances', id] as const,
    system: (id: string) => ['instances', id, 'system'] as const,
    queue: (id: string) => ['instances', id, 'queue'] as const,
    tasks: (id: string) => ['instances', id, 'tasks'] as const,
  },
  cases: {
    all: ['cases'] as const,
    detail: (id: string) => ['cases', id] as const,
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
}
