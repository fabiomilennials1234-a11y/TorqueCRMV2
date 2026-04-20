import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { MessageComposer } from '@/features/inbox/MessageComposer'

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

describe('MessageComposer', () => {
  it('send button is disabled while body is empty', () => {
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    render(<MessageComposer conversationId="c1" />, { wrapper: wrap(client) })
    const btn = screen.getByText('Enviar').closest('button')!
    expect(btn.disabled).toBe(true)
  })

  it('POSTs /conversations/:id/messages with the trimmed body on send', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 202,
      json: async () => ({
        id: 'm1', conversation_id: 'c1', direction: 'outbound', kind: 'text',
        body: 'oi', status: 'queued', occurred_at: '2026-04-20T00:00:00Z',
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    render(<MessageComposer conversationId="c1" />, { wrapper: wrap(client) })

    const textarea = screen.getByPlaceholderText(/Digite uma mensagem/) as HTMLTextAreaElement
    fireEvent.change(textarea, { target: { value: '  oi  ' } })

    const btn = screen.getByText('Enviar').closest('button')!
    fireEvent.click(btn)

    await waitFor(() => expect(fetchMock).toHaveBeenCalledOnce())
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit | undefined
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/conversations/c1/messages')
    expect(init?.method).toBe('POST')
    expect(String(init?.body)).toContain('"body":"oi"')
  })

  it('disables textarea + send when the conversation is resolved', () => {
    const client = new QueryClient()
    render(<MessageComposer conversationId="c1" disabled />, { wrapper: wrap(client) })
    const textarea = screen.getByPlaceholderText(/Conversa encerrada/) as HTMLTextAreaElement
    expect(textarea.disabled).toBe(true)
  })
})
