import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useAgents, useCreateAgent, useSetKillSwitch } from '@/hooks/useAgents'

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

describe('useAgents', () => {
  it('fetches /api/v1/agents and returns data[]', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({
        data: [
          {
            id: 'a1',
            name: 'A',
            model: 'gpt',
            temperature: 0.3,
            max_output_tokens: 1024,
            tools_allowlist: [],
            kill_switch: false,
            status: 'active',
          },
        ],
      }),
      headers: new Headers(),
    } as Response)
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useAgents(), { wrapper: wrap(client) })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toHaveLength(1)
    expect(result.current.data?.[0]?.id).toBe('a1')
  })
})

describe('useCreateAgent + useSetKillSwitch', () => {
  it('POSTs the expected endpoints', async () => {
    fetchMock
      .mockResolvedValueOnce({
        ok: true,
        status: 201,
        json: async () => ({
          id: 'new',
          name: 'N',
          model: 'gpt',
          temperature: 0.3,
          max_output_tokens: 1024,
          tools_allowlist: [],
          kill_switch: false,
          status: 'draft',
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
    const { result: create } = renderHook(() => useCreateAgent(), { wrapper: wrap(client) })
    await create.current.mutateAsync({
      name: 'N',
      system_prompt: 'you are helpful and kind',
      model: 'gpt',
    })

    const { result: kill } = renderHook(() => useSetKillSwitch('new'), { wrapper: wrap(client) })
    await kill.current.mutateAsync({ enabled: true })

    const urls = fetchMock.mock.calls.map(([u]) => u)
    expect(urls[0]).toBe('/api/v1/agents')
    expect(urls[1]).toBe('/api/v1/agents/new/kill-switch')
  })
})
