import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { UpsellPage } from '@/features/upsell/UpsellPage'

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

describe('UpsellPage', () => {
  it('shows empty state when leads list is empty', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({ data: [], meta: { next_cursor: null } }))
    render(wrap(<UpsellPage />))
    expect(await screen.findByText('Nenhuma oportunidade ainda')).toBeInTheDocument()
  })

  it('renders cards with segment/origin badges', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({
        data: [
          {
            id: 'l1',
            name: 'Alice',
            company: 'Acme',
            segment: 'enterprise',
            origin: 'meta-ads',
            created_at: '2026-04-18T00:00:00Z',
            updated_at: '2026-04-19T00:00:00Z',
          },
        ],
        meta: { next_cursor: null },
      })
    )
    render(wrap(<UpsellPage />))
    expect(await screen.findByText('Alice')).toBeInTheDocument()
    expect(screen.getByText('Acme')).toBeInTheDocument()
    expect(screen.getByText('enterprise')).toBeInTheDocument()
    expect(screen.getByText('meta-ads')).toBeInTheDocument()
  })
})
