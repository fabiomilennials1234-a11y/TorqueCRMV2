import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { TorqueWS } from '@/lib/ws'

// JSDOM doesn't ship WebSocket; we fake it with a controllable stub so we
// can drive onopen/onmessage/onclose deterministically from the test.

type FakeSocket = {
  url: string
  readyState: number
  sent: string[]
  onopen?: () => void
  onmessage?: (e: MessageEvent) => void
  onclose?: () => void
  onerror?: () => void
  send: (data: string) => void
  close: () => void
}

let instances: FakeSocket[] = []

beforeEach(() => {
  instances = []
  vi.useFakeTimers()
  const FakeWS = vi.fn().mockImplementation(function (this: FakeSocket, url: string) {
    this.url = url
    this.readyState = 0 // CONNECTING
    this.sent = []
    this.send = (data: string) => {
      this.sent.push(data)
    }
    this.close = () => {
      this.readyState = 3
      this.onclose?.()
    }
    instances.push(this)
  }) as unknown as typeof WebSocket
  // Constants the implementation reads.
  ;(FakeWS as unknown as { OPEN: number }).OPEN = 1
  vi.stubGlobal('WebSocket', FakeWS)
  // Force a fresh singleton for each test — the class keeps a module-level
  // ref; we reach in and reset it to isolate cases.
  ;(TorqueWS as unknown as { instance: null }).instance = null
})

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('lib/ws TorqueWS', () => {
  it('getInstance returns a stable singleton', () => {
    const a = TorqueWS.getInstance()
    const b = TorqueWS.getInstance()
    expect(a).toBe(b)
  })

  it('connect instantiates a socket and flips status to connecting/connected', () => {
    const ws = TorqueWS.getInstance()
    const statuses: string[] = []
    ws.onStatusChange((s) => statuses.push(s))
    ws.connect('ws://fake')
    expect(instances).toHaveLength(1)
    expect(instances[0]!.url).toBe('ws://fake')
    expect(statuses).toContain('connecting')
    instances[0]!.readyState = 1
    instances[0]!.onopen?.()
    expect(ws.status).toBe('connected')
    expect(statuses).toContain('connected')
  })

  it('onMessage receives parsed JSON payloads', () => {
    const ws = TorqueWS.getInstance()
    const received: unknown[] = []
    ws.onMessage((d) => received.push(d))
    ws.connect('ws://fake')
    instances[0]!.onopen?.()
    instances[0]!.onmessage?.({ data: JSON.stringify({ hello: 'world' }) } as MessageEvent)
    expect(received).toEqual([{ hello: 'world' }])
  })

  it('falls back to raw data when payload is not JSON', () => {
    const ws = TorqueWS.getInstance()
    const received: unknown[] = []
    ws.onMessage((d) => received.push(d))
    ws.connect('ws://fake')
    instances[0]!.onopen?.()
    instances[0]!.onmessage?.({ data: 'plain-text' } as MessageEvent)
    expect(received).toEqual(['plain-text'])
  })

  it('onclose schedules reconnect with exponential backoff', () => {
    const ws = TorqueWS.getInstance()
    ws.connect('ws://fake')
    instances[0]!.onopen?.()
    // Provider drops the connection unexpectedly.
    instances[0]!.onclose?.()
    expect(ws.status).toBe('reconnecting')
    // Flush the backoff timer → new socket instance.
    vi.runOnlyPendingTimers()
    expect(instances.length).toBeGreaterThanOrEqual(2)
  })

  it('close() marks as intentional and does not reconnect', () => {
    const ws = TorqueWS.getInstance()
    ws.connect('ws://fake')
    instances[0]!.onopen?.()
    ws.close()
    expect(ws.status).toBe('disconnected')
    vi.advanceTimersByTime(60_000)
    expect(instances).toHaveLength(1)
  })

  it('listeners can unsubscribe via returned dispose', () => {
    const ws = TorqueWS.getInstance()
    const statuses: string[] = []
    const off = ws.onStatusChange((s) => statuses.push(s))
    off()
    ws.connect('ws://fake')
    instances[0]!.onopen?.()
    expect(statuses).toHaveLength(0)
  })

  it('startHeartbeat pings every 30s while connection is OPEN', () => {
    const ws = TorqueWS.getInstance()
    ws.connect('ws://fake')
    instances[0]!.readyState = 1
    instances[0]!.onopen?.()
    vi.advanceTimersByTime(30_000)
    // Heartbeat fires once at 30s.
    expect(instances[0]!.sent.length).toBeGreaterThanOrEqual(1)
    expect(instances[0]!.sent[0]).toContain('ping')
  })
})
