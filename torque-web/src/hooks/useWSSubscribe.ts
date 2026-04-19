/**
 * `useWSSubscribe` — subscribe the component to server events filtered by type.
 *
 * The WS wire envelope (backend §ws/event.go) looks like:
 *
 *     { type, tenant_id, entity_type, entity_id, version, patch, occurred_at }
 *
 * This hook attaches a handler via the TorqueWS singleton. The handler is
 * invoked only when the event's `type` matches `eventType` (single string or
 * array). Use this for:
 *
 *   - React to `operation.succeeded` and invalidate the poll query.
 *   - React to `lead.updated` and patch the list cache directly.
 *
 * The hook deliberately stays tiny — it does not own query invalidation.
 * Callers compose `useQueryClient` alongside.
 */

import { useEffect } from 'react'

import { TorqueWS } from '@/lib/ws'

export interface TorqueEvent<TPatch = unknown> {
  type: string
  tenant_id: string
  entity_type?: string
  entity_id?: string
  version?: number
  patch?: TPatch
  occurred_at: string
}

export type WSEventHandler<TPatch = unknown> = (event: TorqueEvent<TPatch>) => void

function isTorqueEvent(x: unknown): x is TorqueEvent {
  if (typeof x !== 'object' || x === null) return false
  const o = x as Record<string, unknown>
  return typeof o.type === 'string' && typeof o.tenant_id === 'string'
}

export function useWSSubscribe<TPatch = unknown>(
  eventType: string | readonly string[],
  handler: WSEventHandler<TPatch>,
  deps: readonly unknown[] = []
): void {
  useEffect(() => {
    const ws = TorqueWS.getInstance()
    const matchSet = new Set(Array.isArray(eventType) ? eventType : [eventType])

    const unsub = ws.onMessage((raw) => {
      if (!isTorqueEvent(raw)) return
      if (!matchSet.has(raw.type)) return
      handler(raw as TorqueEvent<TPatch>)
    })

    return unsub
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [typeof eventType === 'string' ? eventType : eventType.join('|'), ...deps])
}
