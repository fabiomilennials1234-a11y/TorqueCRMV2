import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  googleConnectURL,
  useConnectTinyERP,
  useDisconnectGoogle,
  useDisconnectTinyERP,
  useIntegrations,
} from '@/hooks/useIntegrations'

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
    ok: status < 400,
    status,
    json: async () => body,
    headers: new Headers(),
  } as Response
}

describe('useIntegrations — hooks', () => {
  it('googleConnectURL is the same-origin OAuth kickoff path', () => {
    expect(googleConnectURL()).toBe('/api/v1/integrations/google/connect')
  })

  it('useIntegrations GETs /integrations and unwraps data', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({
        data: [
          {
            provider: 'google',
            connected: true,
            external_account_id: 'ops@example.test',
            last_success_at: '2026-04-20T12:00:00Z',
          },
        ],
      })
    )
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useIntegrations(), { wrapper: wrap(client) })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.[0]?.provider).toBe('google')
    expect(result.current.data?.[0]?.connected).toBe(true)
    const url = String(fetchMock.mock.calls[0]?.[0])
    expect(url).toContain('/integrations')
  })

  it('useDisconnectGoogle POSTs /integrations/google/disconnect', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({}))
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useDisconnectGoogle(), { wrapper: wrap(client) })
    await result.current.mutateAsync()
    const url = String(fetchMock.mock.calls[0]?.[0])
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit | undefined
    expect(url).toContain('/integrations/google/disconnect')
    expect(init?.method).toBe('POST')
  })

  it('useConnectTinyERP POSTs the api_key body', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({}))
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useConnectTinyERP(), { wrapper: wrap(client) })
    await result.current.mutateAsync('abc123xyz')
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit | undefined
    expect(init?.method).toBe('POST')
    expect(JSON.parse(String(init?.body))).toEqual({ api_key: 'abc123xyz' })
  })

  it('useDisconnectTinyERP POSTs /integrations/tinyerp/disconnect', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({}))
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useDisconnectTinyERP(), { wrapper: wrap(client) })
    await result.current.mutateAsync()
    const url = String(fetchMock.mock.calls[0]?.[0])
    expect(url).toContain('/integrations/tinyerp/disconnect')
  })
})
