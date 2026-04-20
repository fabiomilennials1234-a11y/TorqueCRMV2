import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { WorkflowListPage } from '@/features/workflows/WorkflowListPage'

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

describe('S43 — WorkflowListPage', () => {
  it('renders the list of workflows with status + trigger labels', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({
        data: [
          {
            id: 'w1',
            name: 'Boas-vindas',
            description: 'Acolhe leads novos',
            trigger: 'lead_created',
            status: 'active',
          },
          {
            id: 'w2',
            name: 'Nurture',
            trigger: 'manual',
            status: 'draft',
          },
        ],
      })
    )
    render(wrap(<WorkflowListPage />))
    expect(await screen.findByText('Boas-vindas')).toBeInTheDocument()
    expect(screen.getByText('Nurture')).toBeInTheDocument()
    expect(screen.getByText('Lead criado')).toBeInTheDocument()
    expect(screen.getByText('Manual')).toBeInTheDocument()
  })

  it('shows empty state when no workflows exist', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({ data: [] }))
    render(wrap(<WorkflowListPage />))
    expect(await screen.findByText('Sem workflows ainda')).toBeInTheDocument()
  })
})
