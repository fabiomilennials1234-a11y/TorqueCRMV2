/**
 * Canonical query-key factory.
 *
 * Every TanStack Query consumer reads keys from here — free-form arrays
 * in call sites become desync bugs at the first refactor. The factory is
 * organized by domain, and each domain exposes:
 *
 *   - all(): the root key for invalidating the whole domain
 *   - list(filters?): the list key (cursor pagination lives here)
 *   - detail(id): the one-record key
 *
 * Keeping key shapes here also makes WS-driven invalidation predictable —
 * `useWSSubscribe` maps server event types to factory calls.
 */

export const queryKeys = {
  session: {
    all: () => ['session'] as const,
    me: () => ['session', 'me'] as const,
    bootstrap: () => ['session', 'bootstrap'] as const,
  },

  operations: {
    all: () => ['operations'] as const,
    list: (filters?: Record<string, unknown>) =>
      filters ? (['operations', 'list', filters] as const) : (['operations', 'list'] as const),
    detail: (id: string) => ['operations', 'detail', id] as const,
  },

  leads: {
    all: () => ['leads'] as const,
    list: (filters?: Record<string, unknown>) =>
      filters ? (['leads', 'list', filters] as const) : (['leads', 'list'] as const),
    detail: (id: string) => ['leads', 'detail', id] as const,
  },

  pipes: {
    all: () => ['pipes'] as const,
    list: () => ['pipes', 'list'] as const,
    detail: (id: string) => ['pipes', 'detail', id] as const,
    stages: (pipeId: string) => ['pipes', pipeId, 'stages'] as const,
    entries: (pipeId: string, filters?: Record<string, unknown>) =>
      filters
        ? (['pipes', pipeId, 'entries', filters] as const)
        : (['pipes', pipeId, 'entries'] as const),
  },

  tasks: {
    all: () => ['tasks'] as const,
    list: (filters?: Record<string, unknown>) =>
      filters ? (['tasks', 'list', filters] as const) : (['tasks', 'list'] as const),
    detail: (id: string) => ['tasks', 'detail', id] as const,
  },

  inbox: {
    all: () => ['inbox'] as const,
    conversations: (filters?: Record<string, unknown>) =>
      filters ? (['inbox', 'conversations', filters] as const) : (['inbox', 'conversations'] as const),
    conversation: (id: string) => ['inbox', 'conversations', 'detail', id] as const,
    messages: (id: string) => ['inbox', 'conversations', id, 'messages'] as const,
  },
} as const

export type QueryKeyFactory = typeof queryKeys
