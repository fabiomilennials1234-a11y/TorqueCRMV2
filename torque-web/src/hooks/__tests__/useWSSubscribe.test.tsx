import { act, render } from '@testing-library/react'
import { useEffect } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { TorqueWS } from '@/lib/ws'
import { useWSSubscribe } from '@/hooks/useWSSubscribe'

describe('useWSSubscribe', () => {
  let unsub: () => void
  let dispatch: ((raw: unknown) => void) | undefined

  beforeEach(() => {
    unsub = vi.fn()
    dispatch = undefined
    vi.spyOn(TorqueWS.prototype, 'onMessage').mockImplementation(function (
      handler: (data: unknown) => void
    ) {
      dispatch = handler
      return unsub
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  function Harness({
    type,
    onMatch,
  }: {
    type: string | string[]
    onMatch: (evt: unknown) => void
  }) {
    useWSSubscribe(type, onMatch)
    useEffect(() => () => {}, [])
    return null
  }

  it('invokes the handler only for matching event types', () => {
    const handler = vi.fn()
    render(<Harness type="operation.succeeded" onMatch={handler} />)

    act(() => {
      dispatch?.({ type: 'operation.updated', tenant_id: 't1', occurred_at: 'now' })
      dispatch?.({
        type: 'operation.succeeded',
        tenant_id: 't1',
        occurred_at: 'now',
        patch: { id: 'op' },
      })
      dispatch?.({ type: 'lead.updated', tenant_id: 't1', occurred_at: 'now' })
    })

    expect(handler).toHaveBeenCalledTimes(1)
    const evt = handler.mock.calls[0]?.[0] as { type: string }
    expect(evt.type).toBe('operation.succeeded')
  })

  it('accepts an array of types (union match)', () => {
    const handler = vi.fn()
    render(<Harness type={['operation.updated', 'operation.succeeded']} onMatch={handler} />)

    act(() => {
      dispatch?.({ type: 'operation.updated', tenant_id: 't1', occurred_at: 'now' })
      dispatch?.({ type: 'operation.succeeded', tenant_id: 't1', occurred_at: 'now' })
      dispatch?.({ type: 'lead.updated', tenant_id: 't1', occurred_at: 'now' })
    })

    expect(handler).toHaveBeenCalledTimes(2)
  })

  it('ignores payloads that are not a well-formed TorqueEvent', () => {
    const handler = vi.fn()
    render(<Harness type="x" onMatch={handler} />)

    act(() => {
      dispatch?.(null)
      dispatch?.({ type: 'x' }) // missing tenant_id
      dispatch?.('bare string')
    })

    expect(handler).not.toHaveBeenCalled()
  })

  it('returns the TorqueWS unsubscribe on unmount', () => {
    const handler = vi.fn()
    const { unmount } = render(<Harness type="x" onMatch={handler} />)
    unmount()
    expect(unsub).toHaveBeenCalledOnce()
  })
})
