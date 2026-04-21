import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  useArchiveWorkflow,
  useCancelRun,
  useCreateStep,
  useCreateWorkflow,
  useDeleteStep,
  useEnqueueRun,
  usePauseWorkflow,
  usePublishWorkflow,
  useSetEntryStep,
  useUpdateStep,
  useWorkflow,
  useWorkflowRunSteps,
  useWorkflowRuns,
  useWorkflowSteps,
  useWorkflows,
} from '@/hooks/useWorkflows'

const fetchMock = vi.fn()
beforeEach(() => vi.stubGlobal('fetch', fetchMock))
afterEach(() => {
  vi.unstubAllGlobals()
  fetchMock.mockReset()
})

function wrap(client: QueryClient) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }
}

function mockJSON(body: unknown, status = 200) {
  return {
    ok: true,
    status,
    json: async () => body,
    headers: new Headers(),
  } as Response
}

describe('useWorkflows — extras', () => {
  it('useWorkflows GETs /api/v1/workflows', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({ data: [] }))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useWorkflows(), { wrapper: wrap(client) })
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/workflows')
  })

  it('useWorkflow disabled without id', () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useWorkflow(undefined), { wrapper: wrap(client) })
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('useCreateWorkflow POSTs trigger body', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({ id: 'w1', name: 'N', trigger: 'manual', status: 'draft' })
    )
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useCreateWorkflow(), { wrapper: wrap(client) })
    await result.current.mutateAsync({ name: 'N', trigger: 'manual' })
    expect(JSON.parse(String((fetchMock.mock.calls[0]?.[1] as RequestInit).body)).trigger).toBe(
      'manual'
    )
  })

  it('publish/pause/archive hit the right sub-paths', async () => {
    fetchMock
      .mockResolvedValueOnce(mockJSON(null, 204))
      .mockResolvedValueOnce(mockJSON(null, 204))
      .mockResolvedValueOnce(mockJSON(null, 204))
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const pub = renderHook(() => usePublishWorkflow('w1'), { wrapper: wrap(client) })
    await pub.result.current.mutateAsync()
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/workflows/w1/publish')
    const pause = renderHook(() => usePauseWorkflow('w1'), { wrapper: wrap(client) })
    await pause.result.current.mutateAsync()
    expect(fetchMock.mock.calls[1]?.[0]).toBe('/api/v1/workflows/w1/pause')
    const arc = renderHook(() => useArchiveWorkflow('w1'), { wrapper: wrap(client) })
    await arc.result.current.mutateAsync()
    expect(fetchMock.mock.calls[2]?.[0]).toBe('/api/v1/workflows/w1/archive')
  })

  // step CRUD mutations each invalidate the steps query, triggering a
  // re-fetch. Use permissive default mock + search calls by URL instead
  // of relying on array ordering.
  it('useWorkflowSteps GETs scoped path', async () => {
    fetchMock.mockResolvedValue(mockJSON({ data: [] }))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useWorkflowSteps('w1'), { wrapper: wrap(client) })
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/workflows/w1/steps')
  })

  it('useCreateStep POSTs /steps', async () => {
    fetchMock.mockResolvedValue(
      mockJSON({ id: 's1', kind: 'send_message', name: 'S', config: {}, next_step_ids: [] })
    )
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useCreateStep('w1'), { wrapper: wrap(client) })
    await result.current.mutateAsync({ kind: 'send_message', name: 'S' })
    const call = fetchMock.mock.calls.find((c) => c[0] === '/api/v1/workflows/w1/steps')
    expect((call?.[1] as RequestInit).method).toBe('POST')
  })

  it('useUpdateStep PUTs /steps/:id', async () => {
    fetchMock.mockResolvedValue(
      mockJSON({ id: 's1', kind: 'send_message', name: 'S2', config: {}, next_step_ids: [] })
    )
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useUpdateStep('w1', 's1'), { wrapper: wrap(client) })
    await result.current.mutateAsync({ kind: 'send_message', name: 'S2' })
    const call = fetchMock.mock.calls.find((c) => c[0] === '/api/v1/workflows/w1/steps/s1')
    expect((call?.[1] as RequestInit).method).toBe('PUT')
  })

  it('useDeleteStep DELETEs /steps/:id', async () => {
    fetchMock.mockResolvedValue(mockJSON(null, 204))
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useDeleteStep('w1', 's1'), { wrapper: wrap(client) })
    await result.current.mutateAsync()
    const call = fetchMock.mock.calls.find((c) => c[0] === '/api/v1/workflows/w1/steps/s1')
    expect((call?.[1] as RequestInit).method).toBe('DELETE')
  })

  it('useSetEntryStep POSTs /entry/:id', async () => {
    fetchMock.mockResolvedValue(mockJSON(null, 204))
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useSetEntryStep('w1'), { wrapper: wrap(client) })
    await result.current.mutateAsync({ step_id: 's1' })
    const call = fetchMock.mock.calls.find((c) => c[0] === '/api/v1/workflows/w1/entry/s1')
    expect(call).toBeDefined()
  })

  it('useWorkflowRuns GETs /runs', async () => {
    fetchMock.mockResolvedValue(mockJSON({ data: [] }))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useWorkflowRuns('w1'), { wrapper: wrap(client) })
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/workflows/w1/runs')
  })

  it('useEnqueueRun POSTs /runs', async () => {
    fetchMock.mockResolvedValue(
      mockJSON({
        id: 'r1',
        workflow_id: 'w1',
        trigger_source: 'manual',
        status: 'pending',
        created_at: '2026-04-20T00:00:00Z',
      })
    )
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useEnqueueRun('w1'), { wrapper: wrap(client) })
    await result.current.mutateAsync({ trigger_source: 'manual' })
    const call = fetchMock.mock.calls.find((c) => c[0] === '/api/v1/workflows/w1/runs')
    expect((call?.[1] as RequestInit).method).toBe('POST')
  })

  it('useCancelRun POSTs /runs/:id/cancel', async () => {
    fetchMock.mockResolvedValue(mockJSON(null, 204))
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useCancelRun('r1'), { wrapper: wrap(client) })
    await result.current.mutateAsync()
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/runs/r1/cancel')
  })

  it('useWorkflowRunSteps disabled vs enabled', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useWorkflowRunSteps(undefined), { wrapper: wrap(client) })
    expect(fetchMock).not.toHaveBeenCalled()

    fetchMock.mockResolvedValueOnce(mockJSON({ data: [] }))
    const c2 = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useWorkflowRunSteps('r1'), { wrapper: wrap(c2) })
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/runs/r1/steps')
  })
})
