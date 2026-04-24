import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useCreateWebhook, useOrganization, useUpdateOrganization } from '@/hooks/useOrgSettings'

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

describe('useOrgSettings', () => {
  it('useOrganization hits /api/v1/organization', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({
        id: 'o1',
        slug: 's',
        name: 'n',
        updated_at: '2026-04-20T00:00:00Z',
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useOrganization(), { wrapper: wrap(client) })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/organization')
  })

  it('useUpdateOrganization PATCHes the org endpoint', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({
        id: 'o1',
        slug: 's',
        name: 'new',
        updated_at: '2026-04-20T00:00:00Z',
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useUpdateOrganization(), { wrapper: wrap(client) })
    await result.current.mutateAsync({ name: 'new' })

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/organization')
    expect((fetchMock.mock.calls[0]?.[1] as RequestInit | undefined)?.method).toBe('PATCH')
  })

  it('useCreateWebhook POSTs /api/v1/webhooks', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 201,
      json: async () => ({
        id: 'w1',
        url: 'https://x',
        event_types: ['lead.*'],
        is_active: true,
        created_at: '2026-04-20T00:00:00Z',
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useCreateWebhook(), { wrapper: wrap(client) })
    await result.current.mutateAsync({
      url: 'https://x',
      event_types: ['lead.*'],
      secret: 'sixteen-chars-minimum!!',
    })

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/webhooks')
    expect((fetchMock.mock.calls[0]?.[1] as RequestInit | undefined)?.method).toBe('POST')
  })
})
