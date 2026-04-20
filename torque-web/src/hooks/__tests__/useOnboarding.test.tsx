import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  shouldGate,
  useCompleteStep,
  useOnboarding,
  type OnboardingStatus,
} from '@/hooks/useOnboarding'

const fetchMock = vi.fn()
beforeEach(() => vi.stubGlobal('fetch', fetchMock))
afterEach(() => {
  vi.unstubAllGlobals()
  fetchMock.mockReset()
})

function wrap(client: QueryClient) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }
}

describe('useOnboarding', () => {
  it('GETs /api/v1/onboarding', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true, status: 200,
      json: async () => ({
        current_step: 'welcome',
        steps_completed: [],
        all_steps: ['welcome', 'finish'],
        dismissed: false,
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useOnboarding(), { wrapper: wrap(client) })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/onboarding')
  })

  it('useCompleteStep POSTs /api/v1/onboarding/steps', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true, status: 200,
      json: async () => ({
        current_step: 'finish',
        steps_completed: ['welcome'],
        all_steps: ['welcome', 'finish'],
        dismissed: false,
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useCompleteStep(), { wrapper: wrap(client) })
    await result.current.mutateAsync({ step: 'welcome' })

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/onboarding/steps')
  })

  it('shouldGate true when not dismissed and not completed', () => {
    const status: OnboardingStatus = {
      current_step: 'welcome',
      steps_completed: [],
      all_steps: ['welcome'],
      dismissed: false,
    }
    expect(shouldGate(status)).toBe(true)
    expect(shouldGate({ ...status, dismissed: true })).toBe(false)
    expect(shouldGate({ ...status, completed_at: '2026-04-20T00:00:00Z' })).toBe(false)
    expect(shouldGate(undefined)).toBe(false)
  })
})
