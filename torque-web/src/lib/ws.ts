/**
 * TorqueWS — WebSocket singleton with exponential reconnect and heartbeat.
 *
 * Does NOT connect automatically. Call `connect(url)` explicitly.
 */

export type WSStatus = 'connecting' | 'connected' | 'disconnected' | 'reconnecting'

type StatusListener = (status: WSStatus) => void
type MessageHandler = (data: unknown) => void

const MIN_RECONNECT_MS = 1_000
const MAX_RECONNECT_MS = 30_000
const HEARTBEAT_INTERVAL_MS = 30_000

export class TorqueWS {
  private static instance: TorqueWS | null = null

  private ws: WebSocket | null = null
  private url: string | null = null
  private currentStatus: WSStatus = 'disconnected'
  private reconnectAttempts = 0
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null
  private intentionallyClosed = false

  private statusListeners = new Set<StatusListener>()
  private messageHandlers = new Set<MessageHandler>()

  private constructor() {}

  static getInstance(): TorqueWS {
    if (!TorqueWS.instance) {
      TorqueWS.instance = new TorqueWS()
    }
    return TorqueWS.instance
  }

  get status(): WSStatus {
    return this.currentStatus
  }

  onStatusChange(listener: StatusListener): () => void {
    this.statusListeners.add(listener)
    return () => {
      this.statusListeners.delete(listener)
    }
  }

  onMessage(handler: MessageHandler): () => void {
    this.messageHandlers.add(handler)
    return () => {
      this.messageHandlers.delete(handler)
    }
  }

  connect(url: string): void {
    this.url = url
    this.intentionallyClosed = false
    this.reconnectAttempts = 0
    this.createConnection()
  }

  close(): void {
    this.intentionallyClosed = true
    this.clearTimers()
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
    this.setStatus('disconnected')
  }

  // ---------------------------------------------------------------------------
  // Private
  // ---------------------------------------------------------------------------

  private createConnection(): void {
    if (!this.url) return

    this.clearTimers()
    this.setStatus(this.reconnectAttempts > 0 ? 'reconnecting' : 'connecting')

    try {
      this.ws = new WebSocket(this.url)
    } catch {
      this.scheduleReconnect()
      return
    }

    this.ws.onopen = () => {
      this.reconnectAttempts = 0
      this.setStatus('connected')
      this.startHeartbeat()
    }

    this.ws.onmessage = (event: MessageEvent) => {
      let parsed: unknown
      try {
        parsed = JSON.parse(String(event.data))
      } catch {
        parsed = event.data
      }
      this.messageHandlers.forEach((h) => h(parsed))
    }

    this.ws.onclose = () => {
      this.stopHeartbeat()
      if (!this.intentionallyClosed) {
        this.scheduleReconnect()
      } else {
        this.setStatus('disconnected')
      }
    }

    this.ws.onerror = () => {
      // onclose will fire after onerror — reconnect handled there
    }
  }

  private scheduleReconnect(): void {
    if (this.intentionallyClosed) return

    this.setStatus('reconnecting')
    const base = Math.min(MIN_RECONNECT_MS * 2 ** this.reconnectAttempts, MAX_RECONNECT_MS)
    // Add jitter: 0.5x–1.5x
    const jitter = base * (0.5 + Math.random())
    this.reconnectAttempts++

    this.reconnectTimer = setTimeout(() => {
      this.createConnection()
    }, jitter)
  }

  private startHeartbeat(): void {
    this.heartbeatTimer = setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ type: 'ping' }))
      }
    }, HEARTBEAT_INTERVAL_MS)
  }

  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer)
      this.heartbeatTimer = null
    }
  }

  private clearTimers(): void {
    this.stopHeartbeat()
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
  }

  private setStatus(status: WSStatus): void {
    if (this.currentStatus === status) return
    this.currentStatus = status
    this.statusListeners.forEach((l) => l(status))
  }
}
