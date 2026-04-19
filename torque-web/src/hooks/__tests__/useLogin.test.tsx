import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor, act } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppError } from '@/api/client'
import { queryKeys } from '@/api/queryKeys'
import { useLogin } from '@/hooks/useLogin'

const fetchMock = vi.fn()

beforeEach(() => {
  vi.stubGlobal('fetch', fetchMock)
})
afterEach(() => {
  vi.unstubAllGlobals()
  fetchMock.mockReset()
})

function wrap(client: QueryClient) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }
}

describe('useLogin', () => {
  it('POSTs /api/v1/auth/login and invalidates session key on success', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({
        user: { id: 'u', email: 'a@b', display_name: 'A' },
        organization: { id: 'o', slug: 'x', name: 'Org' },
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const spy = vi.spyOn(client, 'invalidateQueries')

    const { result } = renderHook(() => useLogin(), { wrapper: wrap(client) })

    await act(async () => {
      await result.current.mutateAsync({ email: 'a@b', password: 'hunter2' })
    })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    const [url, init] = fetchMock.mock.calls[0] ?? []
    expect(url).toBe('/api/v1/auth/login')
    expect((init as RequestInit).method).toBe('POST')
    expect(spy).toHaveBeenCalledWith({ queryKey: queryKeys.session.me() })
  })

  it('exposes AppError without firing a toast (silent mode)', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: false,
      status: 401,
      json: async () => ({ code: 'INVALID_CREDENTIALS', message: 'x', status: 401 }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const toastSpy = vi.fn()
    window.addEventListener('torque:toast', toastSpy as EventListener)

    const { result } = renderHook(() => useLogin(), { wrapper: wrap(client) })

    await act(async () => {
      try {
        await result.current.mutateAsync({ email: 'nope', password: 'bad' })
      } catch {
        // silent — consumed by caller
      }
    })
    await waitFor(() => expect(result.current.isError).toBe(true))

    expect(result.current.error).toBeInstanceOf(AppError)
    expect((result.current.error as AppError).code).toBe('INVALID_CREDENTIALS')
    expect(toastSpy).not.toHaveBeenCalled()

    window.removeEventListener('torque:toast', toastSpy as EventListener)
  })
})
