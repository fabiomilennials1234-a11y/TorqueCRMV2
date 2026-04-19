import { describe, it, expect, vi } from 'vitest'

import { AppError } from '../client'
import { friendlyMessage, isRetryable, notifyAppError } from '../errors'

describe('friendlyMessage', () => {
  it('returns the curated message for known AppError codes', () => {
    const err = new AppError('AUTH_EXPIRED', 'raw server text', 401)
    expect(friendlyMessage(err)).toContain('sessão expirou')
  })

  it('falls back to the server message when the code is unknown', () => {
    const err = new AppError('SOMETHING_WEIRD', 'quota reached', 400)
    expect(friendlyMessage(err)).toBe('quota reached')
  })

  it('handles plain Errors', () => {
    expect(friendlyMessage(new Error('boom'))).toBe('boom')
  })

  it('handles non-Error values', () => {
    expect(friendlyMessage(null)).toMatch(/inesperado/i)
    expect(friendlyMessage('nope')).toMatch(/inesperado/i)
  })
})

describe('isRetryable', () => {
  it('is true for 429 and 5xx', () => {
    expect(isRetryable(new AppError('RATE_LIMITED', '', 429))).toBe(true)
    expect(isRetryable(new AppError('INTERNAL', '', 500))).toBe(true)
    expect(isRetryable(new AppError('BAD_GATEWAY', '', 502))).toBe(true)
  })

  it('is false for 4xx (except 429) and non-AppError', () => {
    expect(isRetryable(new AppError('INVALID_BODY', '', 400))).toBe(false)
    expect(isRetryable(new AppError('UNAUTHENTICATED', '', 401))).toBe(false)
    expect(isRetryable(new Error('boom'))).toBe(false)
    expect(isRetryable(null)).toBe(false)
  })
})

describe('notifyAppError', () => {
  it('dispatches a torque:toast event with AppError details', () => {
    const handler = vi.fn()
    window.addEventListener('torque:toast', handler as EventListener)
    notifyAppError(new AppError('PERMISSION_DENIED', 'raw', 403), 'leads.create')
    const evt = handler.mock.calls[0]?.[0] as CustomEvent<{
      level: string
      code: string
      status: number
      message: string
      context: string
    }>
    expect(evt.detail.level).toBe('error')
    expect(evt.detail.code).toBe('PERMISSION_DENIED')
    expect(evt.detail.status).toBe(403)
    expect(evt.detail.context).toBe('leads.create')
    expect(evt.detail.message).toMatch(/permissão/i)
    window.removeEventListener('torque:toast', handler as EventListener)
  })

  it('still dispatches for non-AppError values', () => {
    const handler = vi.fn()
    window.addEventListener('torque:toast', handler as EventListener)
    notifyAppError(new Error('boom'))
    expect(handler).toHaveBeenCalledOnce()
    window.removeEventListener('torque:toast', handler as EventListener)
  })
})
