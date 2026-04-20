import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { COCKPIT_QUERY_KEY, useCockpit } from '@/hooks/useCockpit'

function wrap(client: QueryClient) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }
}

describe('useCockpit', () => {
  it('hydrates from the fixture and exposes the canonical query key', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useCockpit(), { wrapper: wrap(client) })
    await waitFor(() => expect(result.current.isSuccess).toBe(true), { timeout: 2000 })
    expect(result.current.data).toBeDefined()
    // Canonical query key is stable between mounts so cache invalidation
    // via COCKPIT_QUERY_KEY works from anywhere (useTaskActions relies on it).
    expect(COCKPIT_QUERY_KEY).toEqual(['cockpit', 'me'])
  })
})
