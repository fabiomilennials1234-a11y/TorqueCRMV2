import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AgentMetricsPage } from '@/features/copilot/AgentMetricsPage'

const fetchMock = vi.fn()
beforeEach(() => vi.stubGlobal('fetch', fetchMock))
afterEach(() => {
  vi.unstubAllGlobals()
  fetchMock.mockReset()
})

function wrap(agentId: string, ui: React.ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[`/copilot/${agentId}/metrics`]}>
        <Routes>
          <Route path="/copilot/:id/metrics" element={ui} />
        </Routes>
      </MemoryRouter>
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

describe('AgentMetricsPage', () => {
  it('renders KPI cards with totals', async () => {
    fetchMock
      .mockResolvedValueOnce(
        mockJSON({
          id: 'a1',
          name: 'Copilot 1',
          system_prompt: 'You are helpful',
          model: 'gpt',
          temperature: 0.3,
          max_output_tokens: 1024,
          tools_allowlist: [],
          kill_switch: false,
          status: 'active',
        })
      )
      .mockResolvedValueOnce(
        mockJSON({
          since: '2026-03-21T00:00:00Z',
          until: '2026-04-20T00:00:00Z',
          total_sessions: 12,
          total_messages: 48,
          tokens_input: 1200,
          tokens_output: 2400,
          avg_latency_ms: 820,
          sessions_by_state: { open: 2, ended: 10 },
        })
      )
    render(wrap('a1', <AgentMetricsPage />))
    expect(await screen.findByText('Copilot 1')).toBeInTheDocument()
    expect(screen.getByText('Sessões')).toBeInTheDocument()
    expect(screen.getByText('48')).toBeInTheDocument()
  })
})
