/**
 * useAgentStream — conversa com o Copilot Playground via SSE (S38).
 *
 * A rota `POST /api/v1/agents/:id/playground/message` devolve um
 * `text/event-stream`. EventSource nativo do browser só faz GET, então
 * aqui o streaming é feito via `fetch` + `response.body.getReader()` com
 * parser SSE caseiro. Benefício colateral: suportar `X-CSRF-Token` e
 * corpo com mensagens estruturadas.
 *
 * Shape do stream (contrato backend em handler/agents/playground.go):
 *   event: delta\n
 *   data: {"content":"..."}\n\n
 *   event: done\n
 *   data: {"input_tokens":N,"output_tokens":M}\n\n
 *   event: error\n
 *   data: {"code":"AUTH_FAILED","message":"..."}\n\n
 *
 * O hook acumula os deltas em `pending` e expõe callbacks declarativos
 * (onToken/onDone/onError). Cancelar via `stop()` aborta a request —
 * isso é o path feliz de unmount ou de "regenerar resposta".
 */

import { useCallback, useEffect, useRef, useState } from 'react'

export type StreamState = 'idle' | 'streaming' | 'done' | 'error' | 'cancelled'

export interface StreamError {
  code: string
  message: string
}

export interface StreamResult {
  state: StreamState
  pending: string
  error: StreamError | null
  inputTokens: number
  outputTokens: number
}

export interface SendOptions {
  messages: Array<{ role: 'user' | 'assistant'; content: string }>
  onToken?: (delta: string) => void
  onDone?: (usage: { inputTokens: number; outputTokens: number }) => void
  onError?: (err: StreamError) => void
}

const INITIAL: StreamResult = {
  state: 'idle',
  pending: '',
  error: null,
  inputTokens: 0,
  outputTokens: 0,
}

function readCsrfToken(): string {
  const match = document.cookie.split('; ').find((row) => row.startsWith('__torque_csrf='))
  return match ? decodeURIComponent(match.split('=')[1] ?? '') : ''
}

/**
 * Parse a single SSE frame (already split by the \n\n delimiter).
 * Returns `null` on comment/heartbeat lines that can be ignored.
 */
function parseFrame(chunk: string): { event: string; data: string } | null {
  let event = 'message'
  const dataLines: string[] = []
  for (const line of chunk.split('\n')) {
    if (!line || line.startsWith(':')) continue
    if (line.startsWith('event:')) {
      event = line.slice(6).trim()
      continue
    }
    if (line.startsWith('data:')) {
      dataLines.push(line.slice(5).trimStart())
    }
  }
  if (dataLines.length === 0) return null
  return { event, data: dataLines.join('\n') }
}

/**
 * Hook factory: bind to an agent id and return `{send, stop, state,
 * pending, error, reset}`. Multiple calls to `send` while streaming are
 * no-ops (the UI must wait or `stop` first).
 */
export function useAgentStream(agentId: string | undefined) {
  const [result, setResult] = useState<StreamResult>(INITIAL)
  const abortRef = useRef<AbortController | null>(null)

  const stop = useCallback(() => {
    abortRef.current?.abort()
    abortRef.current = null
    setResult((r) => (r.state === 'streaming' ? { ...r, state: 'cancelled' } : r))
  }, [])

  const reset = useCallback(() => {
    stop()
    setResult(INITIAL)
  }, [stop])

  const send = useCallback(
    async (opts: SendOptions) => {
      if (!agentId) return
      if (abortRef.current) return // already streaming; caller must stop first
      const controller = new AbortController()
      abortRef.current = controller

      setResult({ ...INITIAL, state: 'streaming' })

      let res: Response
      try {
        res = await fetch(`/api/v1/agents/${agentId}/playground/message`, {
          method: 'POST',
          credentials: 'include',
          signal: controller.signal,
          headers: {
            'Content-Type': 'application/json',
            Accept: 'text/event-stream',
            'X-CSRF-Token': readCsrfToken(),
          },
          body: JSON.stringify({ messages: opts.messages }),
        })
      } catch (err) {
        if ((err as Error).name !== 'AbortError') {
          const se: StreamError = { code: 'NETWORK', message: (err as Error).message }
          setResult((r) => ({ ...r, state: 'error', error: se }))
          opts.onError?.(se)
        }
        abortRef.current = null
        return
      }

      if (!res.ok || !res.body) {
        const se: StreamError = {
          code: `HTTP_${res.status}`,
          message: `Provider returned ${res.status}`,
        }
        setResult((r) => ({ ...r, state: 'error', error: se }))
        opts.onError?.(se)
        abortRef.current = null
        return
      }

      const reader = res.body.getReader()
      const decoder = new TextDecoder('utf-8')
      let buffer = ''
      let streamError: StreamError | null = null

      try {
        for (;;) {
          const { value, done } = await reader.read()
          if (done) break
          buffer += decoder.decode(value, { stream: true })

          let delimiterIdx = buffer.indexOf('\n\n')
          while (delimiterIdx !== -1) {
            const rawFrame = buffer.slice(0, delimiterIdx)
            buffer = buffer.slice(delimiterIdx + 2)
            delimiterIdx = buffer.indexOf('\n\n')

            const frame = parseFrame(rawFrame)
            if (!frame) continue
            if (frame.event === 'delta') {
              try {
                const { content } = JSON.parse(frame.data) as { content: string }
                if (content) {
                  setResult((r) => ({ ...r, pending: r.pending + content }))
                  opts.onToken?.(content)
                }
              } catch {
                // malformed delta — ignore rather than kill the stream
              }
            } else if (frame.event === 'done') {
              try {
                const usage = JSON.parse(frame.data) as {
                  input_tokens?: number
                  output_tokens?: number
                }
                const inputTokens = usage.input_tokens ?? 0
                const outputTokens = usage.output_tokens ?? 0
                setResult((r) => ({
                  ...r,
                  state: 'done',
                  inputTokens,
                  outputTokens,
                }))
                opts.onDone?.({ inputTokens, outputTokens })
              } catch {
                setResult((r) => ({ ...r, state: 'done' }))
                opts.onDone?.({ inputTokens: 0, outputTokens: 0 })
              }
            } else if (frame.event === 'error') {
              try {
                const err = JSON.parse(frame.data) as StreamError
                streamError = err
                setResult((r) => ({ ...r, state: 'error', error: err }))
                opts.onError?.(err)
              } catch {
                streamError = { code: 'UNKNOWN', message: 'malformed error frame' }
              }
            }
          }
        }
      } catch (err) {
        if ((err as Error).name !== 'AbortError') {
          const se: StreamError = { code: 'NETWORK', message: (err as Error).message }
          setResult((r) => ({ ...r, state: 'error', error: se }))
          opts.onError?.(se)
        }
      } finally {
        abortRef.current = null
        // If the stream ended without an explicit `done` or `error`
        // event, treat it as done silently so the UI can unlock.
        setResult((r) => {
          if (r.state === 'streaming' && !streamError) {
            return { ...r, state: 'done' }
          }
          return r
        })
      }
    },
    [agentId]
  )

  // Ensure we don't leak an in-flight request on unmount.
  useEffect(
    () => () => {
      abortRef.current?.abort()
      abortRef.current = null
    },
    []
  )

  return {
    ...result,
    send,
    stop,
    reset,
  }
}
