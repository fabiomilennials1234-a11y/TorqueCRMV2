import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AgendaPage } from '@/features/agenda/AgendaPage'

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

describe('S48 — AgendaPage', () => {
  it('empty state when no meetings in window', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({ data: [], from: '2026-04-13T00:00:00Z', to: '2026-05-20T00:00:00Z' })
    )
    render(wrap(<AgendaPage />))
    expect(await screen.findByText('Nenhuma reunião no período')).toBeInTheDocument()
  })

  it('renders meetings grouped and shows status badges', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({
        data: [
          {
            id: 'm1',
            title: 'Demo Acme',
            starts_at: '2026-04-20T14:00:00Z',
            ends_at: '2026-04-20T15:00:00Z',
            status: 'scheduled',
            created_at: '2026-04-18T00:00:00Z',
            updated_at: '2026-04-18T00:00:00Z',
          },
        ],
        from: '2026-04-13T00:00:00Z',
        to: '2026-05-20T00:00:00Z',
      })
    )
    render(wrap(<AgendaPage />))
    expect(await screen.findByText('Demo Acme')).toBeInTheDocument()
    expect(screen.getByText('scheduled')).toBeInTheDocument()
  })
})
