import { renderHook, waitFor, act } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useAgentStream } from '@/hooks/useAgentStream'

/**
 * Helpers to fabricate a streaming Response whose body the hook will
 * read chunk-by-chunk via getReader(). Using ReadableStream directly
 * mirrors what fetch() returns in the browser.
 */
function sseResponse(frames: string[]): Response {
  const encoder = new TextEncoder()
  const stream = new ReadableStream<Uint8Array>({
    start(controller) {
      for (const frame of frames) controller.enqueue(encoder.encode(frame))
      controller.close()
    },
  })
  return new Response(stream, {
    status: 200,
    headers: { 'Content-Type': 'text/event-stream' },
  })
}

const fetchMock = vi.fn()
beforeEach(() => {
  vi.stubGlobal('fetch', fetchMock)
  // useAgentStream reads document.cookie for CSRF; keep it empty.
  Object.defineProperty(document, 'cookie', { value: '', configurable: true })
})
afterEach(() => {
  vi.unstubAllGlobals()
  fetchMock.mockReset()
})

describe('useAgentStream', () => {
  it('parses delta + done frames and concatenates tokens', async () => {
    fetchMock.mockResolvedValueOnce(
      sseResponse([
        'event: delta\ndata: {"content":"Olá"}\n\n',
        'event: delta\ndata: {"content":", "}\n\n',
        'event: delta\ndata: {"content":"mundo!"}\n\n',
        'event: done\ndata: {"input_tokens":4,"output_tokens":3}\n\n',
      ]),
    )
    const onToken = vi.fn()
    const onDone = vi.fn()

    const { result } = renderHook(() => useAgentStream('a1'))
    await act(async () => {
      await result.current.send({
        messages: [{ role: 'user', content: 'oi' }],
        onToken,
        onDone,
      })
    })

    await waitFor(() => expect(result.current.state).toBe('done'))
    expect(result.current.pending).toBe('Olá, mundo!')
    expect(result.current.inputTokens).toBe(4)
    expect(result.current.outputTokens).toBe(3)
    expect(onToken).toHaveBeenCalledTimes(3)
    expect(onDone).toHaveBeenCalledWith({ inputTokens: 4, outputTokens: 3 })
  })

  it('surfaces server-sent error frames with code + message', async () => {
    fetchMock.mockResolvedValueOnce(
      sseResponse([
        'event: error\ndata: {"code":"AUTH_FAILED","message":"bad key"}\n\n',
      ]),
    )
    const onError = vi.fn()
    const { result } = renderHook(() => useAgentStream('a1'))
    await act(async () => {
      await result.current.send({
        messages: [{ role: 'user', content: 'oi' }],
        onError,
      })
    })
    await waitFor(() => expect(result.current.state).toBe('error'))
    expect(result.current.error).toEqual({ code: 'AUTH_FAILED', message: 'bad key' })
    expect(onError).toHaveBeenCalledOnce()
  })

  it('maps HTTP non-2xx to HTTP_<status> before reading the body', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: false, status: 503, body: null, headers: new Headers(),
    } as unknown as Response)
    const { result } = renderHook(() => useAgentStream('a1'))
    await act(async () => {
      await result.current.send({ messages: [{ role: 'user', content: 'oi' }] })
    })
    await waitFor(() => expect(result.current.state).toBe('error'))
    expect(result.current.error?.code).toBe('HTTP_503')
  })

  it('ignores heartbeat/malformed frames without tearing the stream', async () => {
    fetchMock.mockResolvedValueOnce(
      sseResponse([
        ': heartbeat\n\n',
        'event: delta\ndata: not-json\n\n',
        'event: delta\ndata: {"content":"ok"}\n\n',
        'event: done\ndata: {"input_tokens":0,"output_tokens":0}\n\n',
      ]),
    )
    const { result } = renderHook(() => useAgentStream('a1'))
    await act(async () => {
      await result.current.send({ messages: [{ role: 'user', content: 'oi' }] })
    })
    await waitFor(() => expect(result.current.state).toBe('done'))
    expect(result.current.pending).toBe('ok')
  })
})
