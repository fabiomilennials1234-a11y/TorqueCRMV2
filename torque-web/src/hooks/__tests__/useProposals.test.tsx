import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  useAcceptProposal,
  useSendProposal,
  useUpsertProposal,
} from '@/hooks/useProposals'

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

describe('useProposals', () => {
  it('upsert PUTs /api/v1/proposals/:entryId with body', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true, status: 200,
      json: async () => ({
        pipe_entry_id: 'e1', lead_id: 'l1', title: 't', amount_cents: 1000,
        currency: 'BRL', status: 'draft',
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useUpsertProposal('e1'), { wrapper: wrap(client) })
    await result.current.mutateAsync({
      lead_id: 'l1', title: 't', amount_cents: 1000,
    })

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/proposals/e1')
    expect((fetchMock.mock.calls[0]?.[1] as RequestInit | undefined)?.method).toBe('PUT')
  })

  it('send POSTs /api/v1/proposals/:entryId/send', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true, status: 204, json: async () => null, headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useSendProposal('e1'), { wrapper: wrap(client) })
    await result.current.mutateAsync()

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/proposals/e1/send')
  })

  it('accept POSTs /api/v1/proposals/:entryId/accept', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true, status: 204, json: async () => null, headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useAcceptProposal('e1'), { wrapper: wrap(client) })
    await result.current.mutateAsync()

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/proposals/e1/accept')
  })
})
