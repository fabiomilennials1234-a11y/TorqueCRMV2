import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { COCKPIT_QUERY_KEY } from '@/hooks/useCockpit'
import { useTaskActions } from '@/hooks/useTaskActions'
import type { Task, TaskCockpitBundle } from '@/contracts/manual'

/**
 * useTaskActions mutates the cockpit cache locally (optimistic-only —
 * backend wiring is a follow-up per TODO in the hook). Tests seed a
 * deterministic bundle + exercise each action + assert the resulting
 * cache shape.
 */

function makeTask(id: string, overrides: Partial<Task> = {}): Task {
  return {
    id,
    organizationId: 'org-1',
    leadId: 'lead-' + id,
    assignedTo: 'me',
    createdBy: 'me',
    kind: 'call',
    title: 'Task ' + id,
    description: null,
    priority: 'normal',
    status: 'pending',
    inQueue: true,
    queuePosition: 1,
    dueAt: null,
    startedAt: null,
    completedAt: null,
    completedBy: null,
    cancelledAt: null,
    cancelledReason: null,
    missedReason: null,
    origin: 'manual',
    context: null,
    resultNote: null,
    createdAt: '2026-04-20T00:00:00Z',
    updatedAt: '2026-04-20T00:00:00Z',
    ...overrides,
  }
}

function makeBundle(overrides: Partial<TaskCockpitBundle> = {}): TaskCockpitBundle {
  return {
    inProgress: null,
    queue: [],
    backlog: [],
    missed: [],
    activeLead: null,
    pipeStages: [],
    counts: { queue: 0, backlog: 0, missed: 0, completedToday: 0 },
    ...overrides,
  }
}

function wrap(client: QueryClient) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }
}

describe('useTaskActions', () => {
  it('startTask promotes a queued task to in_progress and shrinks queue counts', () => {
    const client = new QueryClient()
    client.setQueryData<TaskCockpitBundle>(
      COCKPIT_QUERY_KEY,
      makeBundle({
        queue: [makeTask('a'), makeTask('b', { queuePosition: 2 })],
        counts: { queue: 2, backlog: 0, missed: 0, completedToday: 0 },
      }),
    )

    const { result } = renderHook(() => useTaskActions(), { wrapper: wrap(client) })
    act(() => result.current.startTask('a'))

    const bundle = client.getQueryData<TaskCockpitBundle>(COCKPIT_QUERY_KEY)!
    expect(bundle.inProgress?.id).toBe('a')
    expect(bundle.inProgress?.status).toBe('in_progress')
    expect(bundle.queue.map((t) => t.id)).toEqual(['b'])
    expect(bundle.counts.queue).toBe(1)
  })

  it('completeTask clears inProgress and increments completedToday', () => {
    const client = new QueryClient()
    client.setQueryData<TaskCockpitBundle>(
      COCKPIT_QUERY_KEY,
      makeBundle({
        inProgress: makeTask('a', { status: 'in_progress', inQueue: false, queuePosition: null }),
      }),
    )

    const { result } = renderHook(() => useTaskActions(), { wrapper: wrap(client) })
    act(() => result.current.completeTask('a', 'note'))

    const bundle = client.getQueryData<TaskCockpitBundle>(COCKPIT_QUERY_KEY)!
    expect(bundle.inProgress).toBeNull()
    expect(bundle.counts.completedToday).toBe(1)
  })

  it('reopenMissed moves task from missed to head of queue with resequenced positions', () => {
    const client = new QueryClient()
    client.setQueryData<TaskCockpitBundle>(
      COCKPIT_QUERY_KEY,
      makeBundle({
        missed: [makeTask('m1', { missedReason: 'no_show' })],
        queue: [makeTask('q1', { queuePosition: 1 })],
        counts: { queue: 1, backlog: 0, missed: 1, completedToday: 0 },
      }),
    )

    const { result } = renderHook(() => useTaskActions(), { wrapper: wrap(client) })
    act(() => result.current.reopenMissed('m1'))

    const bundle = client.getQueryData<TaskCockpitBundle>(COCKPIT_QUERY_KEY)!
    expect(bundle.missed).toHaveLength(0)
    expect(bundle.queue.map((t) => t.id)).toEqual(['m1', 'q1'])
    expect(bundle.queue[0]?.queuePosition).toBe(1)
    expect(bundle.queue[1]?.queuePosition).toBe(2)
  })
})
