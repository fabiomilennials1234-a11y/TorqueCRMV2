/**
 * `useInfiniteList` — cursor-paginated list wrapper (ADR-004).
 *
 * Every list endpoint in Torque returns:
 *
 *     { data: T[], meta: { next_cursor: string | null } }
 *
 * This hook:
 *   - sends the cursor as `?cursor=...` to the server;
 *   - flattens all pages into `items` so callers render a single array;
 *   - exposes `fetchNextPage`, `hasNextPage`, `isFetchingNextPage` directly.
 *
 * No offset pagination, ever. The backend will 400 on `?page=` on purpose.
 */

import { useInfiniteQuery, type QueryKey, type UseInfiniteQueryResult } from '@tanstack/react-query'
import { useMemo } from 'react'

import { get } from '@/api/client'

export interface CursorPage<T> {
  data: T[]
  meta?: {
    next_cursor: string | null
  }
}

export interface UseInfiniteListOptions {
  /** Absolute API path, e.g. `/api/v1/leads`. */
  path: string
  /** Stable base key; the hook appends filters. */
  queryKey: QueryKey
  /** Initial query-string params appended on every page fetch. */
  params?: Record<string, string | number | boolean | undefined>
  /** Page size hint for the server. */
  pageSize?: number
  /** Keep the query enabled only when the caller has the prerequisites. */
  enabled?: boolean
  /** Stale-time override for this specific list. */
  staleTime?: number
}

export interface UseInfiniteListResult<T>
  extends Omit<UseInfiniteQueryResult<CursorPage<T>, unknown>, 'data'> {
  items: T[]
  totalPagesLoaded: number
}

export function useInfiniteList<T>(opts: UseInfiniteListOptions): UseInfiniteListResult<T> {
  const {
    path,
    queryKey,
    params,
    pageSize,
    enabled = true,
    staleTime,
  } = opts

  const query = useInfiniteQuery<CursorPage<T>, unknown>({
    queryKey,
    enabled,
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) => lastPage.meta?.next_cursor ?? null,
    staleTime,
    queryFn: ({ pageParam, signal }) => {
      const search = new URLSearchParams()
      if (pageSize !== undefined) search.set('page_size', String(pageSize))
      if (typeof pageParam === 'string' && pageParam.length > 0) {
        search.set('cursor', pageParam)
      }
      if (params) {
        for (const [k, v] of Object.entries(params)) {
          if (v === undefined) continue
          search.set(k, String(v))
        }
      }
      const suffix = search.toString()
      const url = suffix ? `${path}?${suffix}` : path
      // `signal` is forwarded as part of future refactors — current client
      // does not expose abort. Kept to document intent.
      void signal
      return get<CursorPage<T>>(url)
    },
  })

  const items = useMemo(() => {
    return query.data?.pages.flatMap((p) => p.data) ?? []
  }, [query.data])

  const { data: _dropped, ...rest } = query
  return {
    ...rest,
    items,
    totalPagesLoaded: query.data?.pages.length ?? 0,
  }
}
