import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  googleConnectURL,
  useConnectMeta,
  useConnectSZChat,
  useConnectTinyERP,
  useDisconnectGoogle,
  useDisconnectMeta,
  useDisconnectSZChat,
  useDisconnectTinyERP,
  useIntegrations,
  useMetaAdsInsights,
  useSyncTinyERPProducts,
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

  it('useSyncTinyERPProducts POSTs /integrations/tinyerp/sync-products and returns the result', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({ fetched: 10, inserted: 3, updated: 6, skipped: 1 }))
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useSyncTinyERPProducts(), { wrapper: wrap(client) })
    const res = await result.current.mutateAsync()
    expect(res).toEqual({ fetched: 10, inserted: 3, updated: 6, skipped: 1 })
    const url = String(fetchMock.mock.calls[0]?.[0])
    expect(url).toContain('/integrations/tinyerp/sync-products')
  })

  it('useConnectMeta POSTs access_token + optional account_id', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({}))
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useConnectMeta(), { wrapper: wrap(client) })
    await result.current.mutateAsync({ access_token: 'EAAB_fake_token_abc', account_id: 'act_99' })
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit | undefined
    expect(init?.method).toBe('POST')
    expect(JSON.parse(String(init?.body))).toEqual({
      access_token: 'EAAB_fake_token_abc',
      account_id: 'act_99',
    })
  })

  it('useDisconnectMeta POSTs /integrations/meta/disconnect', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({}))
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useDisconnectMeta(), { wrapper: wrap(client) })
    await result.current.mutateAsync()
    const url = String(fetchMock.mock.calls[0]?.[0])
    expect(url).toContain('/integrations/meta/disconnect')
  })

  it('useConnectSZChat POSTs the api_key + optional channel_id', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({}))
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useConnectSZChat(), { wrapper: wrap(client) })
    await result.current.mutateAsync({ api_key: 'szchat-key-abcdef012345', channel_id: 'wa_main' })
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit | undefined
    expect(JSON.parse(String(init?.body))).toEqual({
      api_key: 'szchat-key-abcdef012345',
      channel_id: 'wa_main',
    })
  })

  it('useDisconnectSZChat POSTs /integrations/szchat/disconnect', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({}))
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useDisconnectSZChat(), { wrapper: wrap(client) })
    await result.current.mutateAsync()
    const url = String(fetchMock.mock.calls[0]?.[0])
    expect(url).toContain('/integrations/szchat/disconnect')
  })

  it('useMetaAdsInsights builds the querystring + passes account_id when provided', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({
        account_id: '123',
        date_range: 'last_7d',
        currency: 'BRL',
        spend_cents: 10000,
        impressions: 1000,
        clicks: 50,
        leads: 5,
        cpl_cents: 2000,
        campaigns: [],
        fetched_at: '2026-04-22T00:00:00Z',
      })
    )
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(
      () => useMetaAdsInsights({ accountId: 'act_99', dateRange: 'last_7d' }),
      { wrapper: wrap(client) }
    )
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const url = String(fetchMock.mock.calls[0]?.[0])
    expect(url).toContain('/integrations/meta/ads-insights')
    expect(url).toContain('account_id=act_99')
    expect(url).toContain('date_range=last_7d')
    expect(result.current.data?.leads).toBe(5)
  })

  it('useMetaAdsInsights stays suspended when enabled=false', () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(
      () => useMetaAdsInsights({ accountId: null, dateRange: 'last_7d', enabled: false }),
      { wrapper: wrap(client) }
    )
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
