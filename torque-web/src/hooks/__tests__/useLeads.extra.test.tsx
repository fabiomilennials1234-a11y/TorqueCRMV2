import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useCreateLead, useDeleteLead, useLead, useUpdateLead } from '@/hooks/useLeads'

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
  return { ok: true, status, json: async () => body, headers: new Headers() } as Response
}

describe('useLeads CRUD', () => {
  it('useLead GETs /api/v1/leads/:id', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({ id: 'l1', name: 'Alice' }))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useLead('l1'), { wrapper: wrap(client) })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/leads/l1')
  })

  it('useCreateLead POSTs /api/v1/leads', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({ id: 'new', name: 'New' }, 201))
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useCreateLead(), { wrapper: wrap(client) })
    await result.current.mutateAsync({ name: 'New', phone: '+5511999999999' })
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/leads')
    expect((fetchMock.mock.calls[0]?.[1] as RequestInit).method).toBe('POST')
  })

  it('useUpdateLead PATCHes /api/v1/leads/:id', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({ id: 'l1', name: 'Renamed' }))
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useUpdateLead('l1'), { wrapper: wrap(client) })
    await result.current.mutateAsync({ name: 'Renamed' })
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/leads/l1')
    expect((fetchMock.mock.calls[0]?.[1] as RequestInit).method).toBe('PATCH')
  })

  it('useDeleteLead DELETEs /api/v1/leads/:id', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON(null, 204))
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useDeleteLead(), { wrapper: wrap(client) })
    await result.current.mutateAsync('l1')
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/leads/l1')
    expect((fetchMock.mock.calls[0]?.[1] as RequestInit).method).toBe('DELETE')
  })
})
