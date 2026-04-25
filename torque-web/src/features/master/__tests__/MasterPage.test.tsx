import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { MasterPage } from '@/features/master/MasterPage'

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

describe('MasterPage', () => {
  it('hits /master/health and /master/organizations in parallel', async () => {
    fetchMock.mockImplementation(async (input: RequestInfo | URL) => {
      const url = typeof input === 'string' ? input : input.toString()
      if (url.endsWith('/master/health')) {
        return {
          ok: true,
          status: 200,
          json: async () => ({
            org_count: 3,
            active_org_count: 3,
            user_count: 10,
            lead_count: 42,
            active_subscriptions: 2,
            pending_subscriptions: 1,
            operations_running: 0,
            operations_failed_24h: 0,
          }),
          headers: new Headers(),
        } as Response
      }
      if (url.endsWith('/master/organizations')) {
        return {
          ok: true,
          status: 200,
          json: async () => ({
            data: [
              {
                id: 'o1',
                slug: 'orgA',
                name: 'Org A',
                payment_status: 'active',
                member_count: 1,
                lead_count: 20,
                created_at: '2026-04-20T00:00:00Z',
              },
            ],
          }),
          headers: new Headers(),
        } as Response
      }
      return { ok: false, status: 404, json: async () => ({}), headers: new Headers() } as Response
    })

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(<MasterPage />, { wrapper: wrap(client) })

    await waitFor(() => expect(screen.getByText('Operações globais')).toBeDefined())
    await waitFor(() => expect(screen.getByText('Org A')).toBeDefined())
  })
})
