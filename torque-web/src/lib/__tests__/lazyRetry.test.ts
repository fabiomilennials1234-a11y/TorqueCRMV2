import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { lazyRetry } from '@/lib/lazyRetry'

/**
 * Three behaviors that the wrapper must preserve:
 *  1. First call succeeds → no retry, resolves with the module.
 *  2. First attempt fails, second succeeds → retry kicks in with backoff.
 *  3. All 3 attempts fail → reject with AppError('CHUNK_LOAD_FAILED').
 *
 * Tests use vi.useFakeTimers so the exponential backoff does not consume
 * real wall-clock time.
 */

describe('lazyRetry', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    // Silence the notifyAppError toast dispatch — the test for the
    // rejection path checks the error shape, not the toast.
    vi.stubGlobal('dispatchEvent', vi.fn())
  })
  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('resolves without retry when importFn succeeds on first attempt', async () => {
    const importFn = vi.fn().mockResolvedValue({ default: 'PageA' })

    const result = await lazyRetry(importFn)

    expect(result).toEqual({ default: 'PageA' })
    expect(importFn).toHaveBeenCalledTimes(1)
  })

  it('retries with exponential backoff and resolves when a later attempt succeeds', async () => {
    const importFn = vi
      .fn()
      .mockRejectedValueOnce(new Error('ChunkLoadError'))
      .mockResolvedValueOnce({ default: 'PageB' })

    const promise = lazyRetry(importFn)
    // First failure schedules a 200ms backoff.
    await vi.advanceTimersByTimeAsync(200)
    const result = await promise

    expect(result).toEqual({ default: 'PageB' })
    expect(importFn).toHaveBeenCalledTimes(2)
  })

  it('caps at 3 total attempts and rejects with an AppError on final failure', async () => {
    const err = new Error('ChunkLoadError')
    const importFn = vi.fn().mockRejectedValue(err)

    const promise = lazyRetry(importFn)
    // Attach rejection handler immediately so node doesn't flag an
    // unhandled rejection while we advance the timers.
    const rejection = promise.catch((e) => e)
    await vi.advanceTimersByTimeAsync(200) // backoff #1
    await vi.advanceTimersByTimeAsync(400) // backoff #2
    const rejected = await rejection

    expect(importFn).toHaveBeenCalledTimes(3)
    expect(rejected).toMatchObject({
      name: 'AppError',
      code: 'CHUNK_LOAD_FAILED',
      status: 0,
    })
  })
})
