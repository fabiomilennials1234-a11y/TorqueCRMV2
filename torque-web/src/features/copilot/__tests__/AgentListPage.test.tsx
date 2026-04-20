import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AgentListPage } from '@/features/copilot/AgentListPage'

const fetchMock = vi.fn()
beforeEach(() => vi.stubGlobal('fetch', fetchMock))
afterEach(() => {
  vi.unstubAllGlobals()
  fetchMock.mockReset()
})

function wrap(client: QueryClient) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={client}>
        <MemoryRouter>{children}</MemoryRouter>
      </QueryClientProvider>
    )
  }
}

describe('AgentListPage', () => {
  it('renders the real agent list from /api/v1/agents', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({
        data: [
          {
            id: 'a1',
            name: 'SDR Primário',
            system_prompt: 'Você é um SDR útil.',
            model: 'anthropic/claude-sonnet-4',
            temperature: 0.3,
            max_output_tokens: 1024,
            tools_allowlist: [],
            kill_switch: false,
            status: 'active',
          },
        ],
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(<AgentListPage />, { wrapper: wrap(client) })

    await waitFor(() => expect(screen.getByText('SDR Primário')).toBeDefined())
    // Status badge reflects active.
    expect(screen.getByText('ativo')).toBeDefined()
    // Model metadata visible in card.
    expect(screen.getByText('anthropic/claude-sonnet-4')).toBeDefined()
  })

  it('shows empty state when no agents exist', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true, status: 200,
      json: async () => ({ data: [] }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(<AgentListPage />, { wrapper: wrap(client) })
    await waitFor(() => expect(screen.getByText('Sem agentes ainda')).toBeDefined())
  })
})
