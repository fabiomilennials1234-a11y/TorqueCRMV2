/**
 * `lazyRetry` — retry wrapper for `React.lazy()` import functions.
 *
 * After a deploy, the browser may still hold a stale bundle while the server
 * has rotated chunks; any `import('./PageX')` then throws `ChunkLoadError`.
 * This wrapper retries with an exponential backoff (200ms → 400ms → 800ms,
 * capped at 2s) so the user rarely sees the raw error.
 *
 * Why not port v8's shape verbatim: their version retries with a flat 1s
 * delay and silently swallows the final rejection into `throw err`. We:
 *   - back off exponentially (reduces load when the edge is cold);
 *   - forward the final failure to `notifyAppError` so Sentry + toast see
 *     it (v8 had no structured error pipeline for this);
 *   - cap attempts at 2 (3 total calls) — matches v8 retries=2 default.
 *
 * Usage:
 *   const MyPage = lazy(() => lazyRetry(() => import('@/features/my/MyPage')))
 */

import { AppError } from '@/api/client'
import { notifyAppError } from '@/api/errors'

/** Maximum total attempts, including the initial one. v8 parity. */
const MAX_ATTEMPTS = 3

/** Base delay for backoff. First retry waits this long, then 2×, then 4×. */
const BASE_DELAY_MS = 200

/** Hard cap on any single backoff step — keeps worst-case wait bounded. */
const MAX_DELAY_MS = 2_000

/**
 * `lazyRetry` executes `importFn`, retrying on failure with exponential
 * backoff. The generic `T` is the module shape, which for lazy routes is
 * `{ default: ComponentType }`.
 *
 * The function is re-entrant: on retry it calls itself, so the same
 * `MAX_ATTEMPTS` budget flows through the stack.
 */
export function lazyRetry<T extends { default: unknown }>(
  importFn: () => Promise<T>,
  attempt = 1,
): Promise<T> {
  return importFn().catch((err: unknown) => {
    if (attempt >= MAX_ATTEMPTS) {
      // Final failure: surface via the global error pipeline so Sentry
      // captures the ChunkLoadError with context, and the user sees a
      // toast. Then re-throw so React's `<Suspense>` boundary fires.
      const wrapped =
        err instanceof AppError
          ? err
          : new AppError(
              'CHUNK_LOAD_FAILED',
              'Não foi possível carregar esta página. Atualize a aba.',
              0,
            )
      notifyAppError(wrapped, 'lazy_chunk_load')
      throw wrapped
    }
    const delay = Math.min(BASE_DELAY_MS * 2 ** (attempt - 1), MAX_DELAY_MS)
    return new Promise<T>((resolve, reject) => {
      setTimeout(() => {
        lazyRetry(importFn, attempt + 1).then(resolve, reject)
      }, delay)
    })
  })
}
