import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  renderTemplate,
  useCreateTemplate,
  useMessageTemplates,
} from '@/hooks/useMessageTemplates'

const fetchMock = vi.fn()
beforeEach(() => vi.stubGlobal('fetch', fetchMock))
afterEach(() => {
  vi.unstubAllGlobals()
  fetchMock.mockReset()
})

function wrap(client: QueryClient) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }
}

describe('useMessageTemplates', () => {
  it('fetches /api/v1/message-templates?active_only=1 by default', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ data: [] }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useMessageTemplates(), { wrapper: wrap(client) })
    await waitFor(() => expect(fetchMock).toHaveBeenCalledOnce())
    expect(fetchMock.mock.calls[0]?.[0]).toContain('active_only=1')
  })

  it('omits active_only query param when activeOnly=false', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ data: [] }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useMessageTemplates(false), { wrapper: wrap(client) })
    await waitFor(() => expect(fetchMock).toHaveBeenCalledOnce())
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/message-templates')
  })

  it('useCreateTemplate POSTs /api/v1/message-templates', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 201,
      json: async () => ({
        id: 't1', name: 'x', body: 'y', variables: [], is_active: true,
        created_at: '2026-04-20T00:00:00Z', updated_at: '2026-04-20T00:00:00Z',
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useCreateTemplate(), { wrapper: wrap(client) })
    await result.current.mutateAsync({ name: 'x', body: 'y' })

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/message-templates')
    expect((fetchMock.mock.calls[0]?.[1] as RequestInit | undefined)?.method).toBe('POST')
  })
})

describe('renderTemplate', () => {
  it('substitutes known variables', () => {
    expect(renderTemplate('Olá {{nome}}', { nome: 'Ana' })).toBe('Olá Ana')
  })

  it('leaves unknown variables intact for the user to see', () => {
    expect(renderTemplate('Oi {{nome}}, {{empresa}}', { nome: 'A' })).toBe(
      'Oi A, {{empresa}}',
    )
  })

  it('handles multiple occurrences + whitespace', () => {
    expect(renderTemplate('{{ nome }} {{nome}}', { nome: 'B' })).toBe('B B')
  })

  it('empty value is treated as missing (so user is nudged to fill)', () => {
    expect(renderTemplate('{{nome}}', { nome: '' })).toBe('{{nome}}')
  })
})
