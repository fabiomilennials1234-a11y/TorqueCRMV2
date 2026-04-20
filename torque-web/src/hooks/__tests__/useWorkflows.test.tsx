import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useWorkflows, useCreateStep, useEnqueueRun } from '@/hooks/useWorkflows'

const fetchMock = vi.fn()
beforeEach(() => vi.stubGlobal('fetch', fetchMock))
afterEach(() => {
  vi.unstubAllGlobals()
  fetchMock.mockReset()
})

function wrap(client: QueryClient) {
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
}

describe('useWorkflows suite', () => {
  it('list hits /api/v1/workflows', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true, status: 200,
      json: async () => ({ data: [] }),
      headers: new Headers(),
    } as Response)
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useWorkflows(), { wrapper: wrap(client) })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/workflows')
  })

  it('createStep POSTs under workflow id', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true, status: 201,
      json: async () => ({ id: 's1', kind: 'send_message', name: 'Greet', config: {}, next_step_ids: [] }),
      headers: new Headers(),
    } as Response)
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useCreateStep('wf-1'), { wrapper: wrap(client) })
    await result.current.mutateAsync({ kind: 'send_message', name: 'Greet' })
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/workflows/wf-1/steps')
  })

  it('enqueueRun POSTs to /runs subresource', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true, status: 202,
      json: async () => ({ id: 'r1', workflow_id: 'wf-1', trigger_source: 'manual', status: 'pending', created_at: '2026-04-19T00:00:00Z' }),
      headers: new Headers(),
    } as Response)
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useEnqueueRun('wf-1'), { wrapper: wrap(client) })
    await result.current.mutateAsync({ trigger_source: 'manual' })
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/workflows/wf-1/runs')
  })
})
