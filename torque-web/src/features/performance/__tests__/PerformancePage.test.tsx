import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { PerformancePage } from '@/features/performance/PerformancePage'

const fetchMock = vi.fn()
beforeEach(() => vi.stubGlobal('fetch', fetchMock))
afterEach(() => {
  vi.unstubAllGlobals()
  fetchMock.mockReset()
})

function wrap(ui: React.ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter>{ui}</MemoryRouter>
    </QueryClientProvider>
  )
}

function mockJSON(body: unknown) {
  return {
    ok: true,
    status: 200,
    json: async () => body,
    headers: new Headers(),
  } as Response
}

describe('S47 — PerformancePage', () => {
  it('renders ranking bars from leaderboard entries', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({
        data: [
          {
            member_id: '11111111-1111-1111-1111-111111111111',
            member_name: 'Alice',
            deals_won: 3,
            revenue_cents: 150000,
          },
          {
            member_id: '22222222-2222-2222-2222-222222222222',
            member_name: 'Bob',
            deals_won: 2,
            revenue_cents: 80000,
          },
        ],
        since: '2026-03-21T00:00:00Z',
        until: '2026-04-20T00:00:00Z',
      })
    )
    render(wrap(<PerformancePage />))
    expect(await screen.findByText('Alice')).toBeInTheDocument()
    expect(screen.getByText('Bob')).toBeInTheDocument()
    expect(screen.getByText('3 deals')).toBeInTheDocument()
  })

  it('switches to commissions tab and shows empty state', async () => {
    // Ranking call on initial render.
    fetchMock.mockResolvedValueOnce(
      mockJSON({ data: [], since: '2026-03-21T00:00:00Z', until: '2026-04-20T00:00:00Z' })
    )
    // Commissions call after tab click.
    fetchMock.mockResolvedValueOnce(mockJSON({ data: [] }))
    render(wrap(<PerformancePage />))
    fireEvent.click(screen.getByRole('button', { name: 'Comissões' }))
    expect(await screen.findByText('Sem comissões ainda')).toBeInTheDocument()
  })
})
