import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { TVDashboardPage } from '@/features/tv/TVDashboardPage'

const fetchMock = vi.fn()
beforeEach(() => vi.stubGlobal('fetch', fetchMock))
afterEach(() => {
  vi.unstubAllGlobals()
  fetchMock.mockReset()
})

function wrap(ui: React.ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={client}>{ui}</QueryClientProvider>
}

function mockJSON(body: unknown) {
  return {
    ok: true,
    status: 200,
    json: async () => body,
    headers: new Headers(),
  } as Response
}

describe('TVDashboardPage', () => {
  it('renders proposals widget with BRL value on first render', async () => {
    // All three widget queries fire in parallel because their hooks
    // are mounted simultaneously, even though only one is visible.
    fetchMock.mockResolvedValue(
      mockJSON({
        sent_in_window: 12,
        viewed_in_window: 8,
        accepted_in_window: 4,
        rejected_in_window: 1,
        won_amount_cents: 150000,
      })
    )
    render(wrap(<TVDashboardPage />))
    expect(await screen.findByText('Propostas (30d)')).toBeInTheDocument()
    // Number "12" fills the hero slot once useProposalsSummary resolves —
    // waitFor handles the second render after query success.
    expect(await screen.findByText('12', {}, { timeout: 2000 })).toBeInTheDocument()
  })
})
