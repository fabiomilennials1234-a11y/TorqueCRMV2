import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { CampaignsPage } from '@/features/campaigns/CampaignsPage'

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

describe('S46 — CampaignsPage', () => {
  it('partitions running vs archived by status', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({
        data: [
          {
            id: 'c1',
            name: 'Ativa',
            template_body: 't',
            status: 'running',
            stats_queued: 1,
            stats_sent: 0,
            stats_failed: 0,
            stats_skipped: 0,
          },
          {
            id: 'c2',
            name: 'Cancelada',
            template_body: 't',
            status: 'cancelled',
            stats_queued: 0,
            stats_sent: 0,
            stats_failed: 0,
            stats_skipped: 0,
          },
          {
            id: 'c3',
            name: 'Concluída',
            template_body: 't',
            status: 'completed',
            stats_queued: 10,
            stats_sent: 10,
            stats_failed: 0,
            stats_skipped: 0,
          },
        ],
      })
    )
    render(wrap(<CampaignsPage />))
    expect(await screen.findByText('Ativa')).toBeInTheDocument()
    // Archived tab hidden by default.
    expect(screen.queryByText('Cancelada')).not.toBeInTheDocument()
    // Switch tab.
    fireEvent.click(screen.getByRole('button', { name: /Arquivadas \(2\)/ }))
    expect(await screen.findByText('Cancelada')).toBeInTheDocument()
    expect(screen.getByText('Concluída')).toBeInTheDocument()
  })

  it('shows empty state when no campaigns match current tab', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({ data: [] }))
    render(wrap(<CampaignsPage />))
    expect(await screen.findByText('Nenhuma campanha em andamento')).toBeInTheDocument()
  })
})
