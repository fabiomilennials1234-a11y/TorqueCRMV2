/**
 * `useBootstrap` — fetches `/api/bootstrap` once on cold start.
 *
 * The payload tells the client:
 *   - which Sentry DSN to use (public, client-side);
 *   - which WS URL to connect to;
 *   - which feature flags to honor;
 *   - the server's current time for clock-drift detection.
 *
 * Cached with `staleTime: Infinity` and never refetched on focus — the shape
 * is deploy-time-stable. A deploy invalidates by breaking the cache-bust
 * query string on the SPA bundle (handled in `index.html`).
 */

import { useQuery } from '@tanstack/react-query'

import { get } from '@/api/client'
import { queryKeys } from '@/api/queryKeys'

export interface BootstrapConfig {
  app_version: string
  env: 'dev' | 'staging' | 'prod'
  ws_url: string
  sentry_dsn: string
  feature_flags: Record<string, boolean>
  server_time: string
}

export function useBootstrap() {
  return useQuery<BootstrapConfig, unknown>({
    queryKey: queryKeys.session.bootstrap(),
    queryFn: () => get<BootstrapConfig>('/api/bootstrap'),
    staleTime: Number.POSITIVE_INFINITY,
    gcTime: Number.POSITIVE_INFINITY,
    refetchOnWindowFocus: false,
    retry: 2,
  })
}
