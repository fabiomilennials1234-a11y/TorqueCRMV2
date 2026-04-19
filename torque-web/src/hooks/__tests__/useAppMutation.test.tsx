import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppError } from '@/api/client'
import { useAppMutation } from '@/hooks/useAppMutation'

function makeWrapper(client: QueryClient) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }
}

describe('useAppMutation', () => {
  const toastHandler = vi.fn()

  beforeEach(() => {
    window.addEventListener('torque:toast', toastHandler as EventListener)
  })

  afterEach(() => {
    window.removeEventListener('torque:toast', toastHandler as EventListener)
    toastHandler.mockReset()
  })

  it('emits a toast on AppError by default', async () => {
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(
      () =>
        useAppMutation<string, { bad: true }>(async () => {
          throw new AppError('PERMISSION_DENIED', 'nope', 403)
        }),
      { wrapper: makeWrapper(client) }
    )

    await act(async () => {
      result.current.mutate({ bad: true })
    })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(toastHandler).toHaveBeenCalledOnce()
  })

  it('suppresses the toast when silent: true', async () => {
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(
      () =>
        useAppMutation<string, void>(
          async () => {
            throw new AppError('PERMISSION_DENIED', 'nope', 403)
          },
          { silent: true }
        ),
      { wrapper: makeWrapper(client) }
    )

    await act(async () => {
      result.current.mutate()
    })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(toastHandler).not.toHaveBeenCalled()
  })

  it('invalidates queries on settle', async () => {
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const spy = vi.spyOn(client, 'invalidateQueries')
    const { result } = renderHook(
      () =>
        useAppMutation<number, void>(async () => 7, {
          invalidate: [['leads', 'list']],
        }),
      { wrapper: makeWrapper(client) }
    )

    await act(async () => {
      result.current.mutate()
    })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(spy).toHaveBeenCalledWith({ queryKey: ['leads', 'list'] })
  })

  it('applies an optimistic update and rolls back on error', async () => {
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const key = ['counter']
    client.setQueryData<number>(key, 10)

    const { result } = renderHook(
      () =>
        useAppMutation<number, number>(
          async () => {
            throw new AppError('INTERNAL', 'boom', 500)
          },
          {
            silent: true,
            optimistic: {
              queryKey: key,
              updater: (prev, delta) => (prev ?? 0) + delta,
            },
          }
        ),
      { wrapper: makeWrapper(client) }
    )

    await act(async () => {
      result.current.mutate(5)
    })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(client.getQueryData<number>(key)).toBe(10)
  })
})
