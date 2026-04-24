import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useAddMember, useMembers, useSetMemberPermission } from '@/hooks/useMembers'

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

describe('useMembers', () => {
  it('fetches /api/v1/members and returns data[]', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({
        data: [
          {
            id: 'm1',
            user_id: 'u1',
            email: 'a@b.com',
            display_name: 'Alice',
            role: 'admin',
            is_active: true,
            joined_at: '2026-04-19T00:00:00Z',
          },
        ],
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useMembers(), { wrapper: wrap(client) })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toHaveLength(1)
    expect(result.current.data?.[0]?.email).toBe('a@b.com')
  })

  it('appends include_inactive=1 when requested', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ data: [] }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useMembers(true), { wrapper: wrap(client) })
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    const url = fetchMock.mock.calls[0]?.[0] as string
    expect(url).toContain('include_inactive=1')
  })
})

describe('useAddMember + useSetMemberPermission', () => {
  it('POSTs /api/v1/members and PUTs /api/v1/members/:id/permissions/:key', async () => {
    fetchMock
      .mockResolvedValueOnce({
        ok: true,
        status: 201,
        json: async () => ({
          id: 'new',
          user_id: 'u2',
          email: 'x@y.com',
          display_name: 'X',
          role: 'membro',
          is_active: true,
          joined_at: '2026-04-19T00:00:00Z',
        }),
        headers: new Headers(),
      } as Response)
      .mockResolvedValueOnce({
        ok: true,
        status: 204,
        json: async () => null,
        headers: new Headers(),
      } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result: add } = renderHook(() => useAddMember(), { wrapper: wrap(client) })
    await add.current.mutateAsync({ email: 'x@y.com', display_name: 'X', role: 'membro' })

    const { result: perm } = renderHook(() => useSetMemberPermission('new'), {
      wrapper: wrap(client),
    })
    await perm.current.mutateAsync({ feature_key: 'leads.delete', value: false })

    const calls = fetchMock.mock.calls
    expect(calls[0]?.[0]).toBe('/api/v1/members')
    expect((calls[0]?.[1] as RequestInit | undefined)?.method).toBe('POST')
    expect(calls[1]?.[0]).toBe('/api/v1/members/new/permissions/leads.delete')
    expect((calls[1]?.[1] as RequestInit | undefined)?.method).toBe('PUT')
  })
})
