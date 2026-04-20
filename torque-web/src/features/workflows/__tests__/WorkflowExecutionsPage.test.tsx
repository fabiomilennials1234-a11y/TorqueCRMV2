import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { WorkflowExecutionsPage } from '@/features/workflows/WorkflowExecutionsPage'

const fetchMock = vi.fn()
beforeEach(() => vi.stubGlobal('fetch', fetchMock))
afterEach(() => {
  vi.unstubAllGlobals()
  fetchMock.mockReset()
})

function wrap(workflowId: string, ui: React.ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[`/workflows/${workflowId}/executions`]}>
        <Routes>
          <Route path="/workflows/:id/executions" element={ui} />
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

describe('S45 — WorkflowExecutionsPage', () => {
  it('shows empty state when no runs', async () => {
    // workflow detail
    fetchMock.mockResolvedValueOnce(
      mockJSON({ id: 'w1', name: 'Boas-vindas', trigger: 'manual', status: 'active' })
    )
    // runs list
    fetchMock.mockResolvedValueOnce(mockJSON({ data: [] }))
    render(wrap('w1', <WorkflowExecutionsPage />))
    expect(await screen.findByText('Nenhuma execução ainda')).toBeInTheDocument()
  })

  it('renders runs + allows selection', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({ id: 'w1', name: 'Boas-vindas', trigger: 'manual', status: 'active' })
    )
    fetchMock.mockResolvedValueOnce(
      mockJSON({
        data: [
          {
            id: 'r1',
            workflow_id: 'w1',
            trigger_source: 'debug',
            status: 'succeeded',
            created_at: '2026-04-20T12:00:00Z',
          },
        ],
      })
    )
    render(wrap('w1', <WorkflowExecutionsPage />))
    expect(await screen.findByText('debug')).toBeInTheDocument()
    expect(screen.getByText('succeeded')).toBeInTheDocument()
  })
})
