import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { previewTTS } from '@/hooks/useAgents'

const fetchMock = vi.fn()
beforeEach(() => {
  vi.stubGlobal('fetch', fetchMock)
  // JSDOM implementation of URL.createObjectURL is a stub; ensure it
  // returns something deterministic so assertions don't depend on
  // env specifics.
  vi.stubGlobal(
    'URL',
    Object.assign(globalThis.URL, {
      createObjectURL: vi.fn(() => 'blob:mock-url'),
      revokeObjectURL: vi.fn(),
    })
  )
})
afterEach(() => {
  vi.unstubAllGlobals()
  fetchMock.mockReset()
})

describe('S41 — previewTTS helper', () => {
  it('POSTs to /agents/:id/tts/preview and returns an object URL', async () => {
    const mp3 = new Blob([new Uint8Array([0xff, 0xfb])], { type: 'audio/mpeg' })
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      blob: async () => mp3,
      text: async () => '',
    } as Response)

    const url = await previewTTS('a1', { voice_id: 'v1' })
    expect(url).toBe('blob:mock-url')

    const call = fetchMock.mock.calls[0]
    expect(call?.[0]).toBe('/api/v1/agents/a1/tts/preview')
    const init = call?.[1] as RequestInit
    expect(init.method).toBe('POST')
    const body = JSON.parse(String(init.body))
    expect(body.voice_id).toBe('v1')
    const headers = init.headers as Record<string, string>
    expect(headers['Content-Type']).toBe('application/json')
    // CSRF header present (may be empty string in test env — still present).
    expect(Object.keys(headers)).toContain('X-CSRF-Token')
  })

  it('throws when server returns non-2xx', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: false,
      status: 503,
      blob: async () => new Blob(),
      text: async () => 'unavailable',
    } as Response)

    await expect(previewTTS('a1')).rejects.toThrow(/unavailable|503/)
  })
})
