import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { relativeTime } from '@/features/settings/IntegrationsSection'

describe('relativeTime', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-04-22T12:00:00Z'))
  })
  afterEach(() => vi.useRealTimers())

  it('returns "agora" within the same minute', () => {
    expect(relativeTime('2026-04-22T11:59:40Z')).toBe('agora')
  })

  it('formats minutes under an hour', () => {
    expect(relativeTime('2026-04-22T11:55:00Z')).toBe('há 5 min')
  })

  it('formats hours under a day', () => {
    expect(relativeTime('2026-04-22T09:00:00Z')).toBe('há 3 h')
  })

  it('formats days under a month', () => {
    expect(relativeTime('2026-04-20T12:00:00Z')).toBe('há 2 d')
  })

  it('falls back to absolute date for month+ ago', () => {
    const out = relativeTime('2026-01-01T00:00:00Z')
    // Exact locale-format varies; we assert it produced a non-relative string.
    expect(out).not.toContain('há')
    expect(out.length).toBeGreaterThan(0)
  })

  it('returns empty string for invalid input', () => {
    expect(relativeTime('not-a-date')).toBe('')
  })
})
