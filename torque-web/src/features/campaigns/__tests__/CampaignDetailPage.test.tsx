import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { CampaignDetailPage } from '@/features/campaigns/CampaignDetailPage'

const fetchMock = vi.fn()
beforeEach(() => vi.stubGlobal('fetch', fetchMock))
afterEach(() => {
  vi.unstubAllGlobals()
  fetchMock.mockReset()
})

function wrap(campaignId: string, ui: React.ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[`/campaigns/${campaignId}`]}>
        <Routes>
          <Route path="/campaigns/:id" element={ui} />
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

describe('CampaignDetailPage', () => {
  it('shows stats + running action bar', async () => {
    fetchMock
      .mockResolvedValueOnce(
        mockJSON({
          id: 'c1',
          name: 'Black Friday',
          template_body: 't',
          status: 'running',
          stats_queued: 5,
          stats_sent: 3,
          stats_failed: 1,
          stats_skipped: 0,
        })
      )
      .mockResolvedValueOnce(mockJSON({ data: [] }))
    render(wrap('c1', <CampaignDetailPage />))
    expect(await screen.findByText('Black Friday')).toBeInTheDocument()
    expect(screen.getByText('Pausar')).toBeInTheDocument()
    expect(screen.getByText('Cancelar')).toBeInTheDocument()
  })

  it('lists recipients with status badges', async () => {
    fetchMock
      .mockResolvedValueOnce(
        mockJSON({
          id: 'c1',
          name: 'B',
          template_body: 't',
          status: 'running',
          stats_queued: 0,
          stats_sent: 1,
          stats_failed: 0,
          stats_skipped: 0,
        })
      )
      .mockResolvedValueOnce(
        mockJSON({
          data: [
            {
              id: 'r1',
              lead_id: '11111111-1111-1111-1111-111111111111',
              status: 'sent',
              sent_at: '2026-04-20T10:00:00Z',
            },
          ],
        })
      )
    render(wrap('c1', <CampaignDetailPage />))
    expect(await screen.findByText('sent')).toBeInTheDocument()
    // lead_id prefix shown as mono
    expect(screen.getByText('11111111')).toBeInTheDocument()
  })
})
