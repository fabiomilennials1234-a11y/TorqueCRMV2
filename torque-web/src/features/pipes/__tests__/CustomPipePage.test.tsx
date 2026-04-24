import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { CustomPipePage } from '@/features/pipes/CustomPipePage'

const fetchMock = vi.fn()
beforeEach(() => vi.stubGlobal('fetch', fetchMock))
afterEach(() => {
  vi.unstubAllGlobals()
  fetchMock.mockReset()
})

function wrap(pipeId: string, ui: React.ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[`/pipe/${pipeId}`]}>
        <Routes>
          <Route path="/pipe/:id" element={ui} />
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

describe('CustomPipePage', () => {
  it('renders the stage board with entry count badges', async () => {
    fetchMock
      .mockResolvedValueOnce(
        mockJSON({
          data: [
            {
              id: 'p1',
              kind: 'custom',
              name: 'Churn Rescue',
              is_default: false,
              is_archived: false,
              position: 0,
            },
          ],
        })
      )
      .mockResolvedValueOnce(
        mockJSON({
          data: [
            {
              id: 's1',
              name: 'Novo',
              position: 0,
              is_final_positive: false,
              is_final_negative: false,
            },
            {
              id: 's2',
              name: 'Recuperado',
              position: 1,
              is_final_positive: true,
              is_final_negative: false,
            },
          ],
        })
      )
      .mockResolvedValueOnce(
        mockJSON({
          data: [
            { id: 'e1', stage_id: 's1', lead_id: 'l1', entered_stage_at: '2026-04-20T00:00:00Z' },
            { id: 'e2', stage_id: 's1', lead_id: 'l2', entered_stage_at: '2026-04-20T00:00:00Z' },
          ],
        })
      )
    render(wrap('p1', <CustomPipePage />))
    expect(await screen.findByText('Churn Rescue')).toBeInTheDocument()
    expect(screen.getByText('Novo')).toBeInTheDocument()
    expect(screen.getByText('Recuperado')).toBeInTheDocument()
    expect(screen.getByText('ganho')).toBeInTheDocument()
  })

  it('shows not-found state when pipe id is absent from list', async () => {
    fetchMock
      .mockResolvedValueOnce(mockJSON({ data: [] }))
      .mockResolvedValueOnce(mockJSON({ data: [] }))
      .mockResolvedValueOnce(mockJSON({ data: [] }))
    render(wrap('pX', <CustomPipePage />))
    expect(await screen.findByText(/Pipe não encontrado/)).toBeInTheDocument()
  })
})
