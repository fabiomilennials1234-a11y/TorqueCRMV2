/**
 * `useAppMutation` is the project-wide wrapper around TanStack's useMutation.
 *
 * It standardizes three things every CRUD mutation needs:
 *
 * 1. **Error surface** — any AppError bubbles up through `notifyAppError`
 *    unless the caller passes `silent: true`. The raw error is still
 *    returned from `mutation.error` for forms that want inline fields.
 * 2. **Optimistic update** — `optimistic` lets callers snapshot, patch, and
 *    roll-back the query cache without re-implementing the full onMutate /
 *    onError / onSettled dance.
 * 3. **Invalidation** — `invalidate` is a tuple of query keys refreshed on
 *    success, so callers rarely need to call `queryClient.invalidateQueries`
 *    manually.
 */

import {
  useMutation,
  useQueryClient,
  type MutationFunction,
  type QueryKey,
  type UseMutationOptions,
} from '@tanstack/react-query'

import { notifyAppError } from '@/api/errors'

export interface OptimisticConfig<TData, TVariables> {
  /** Query key whose cached value should be patched while the mutation runs. */
  queryKey: QueryKey
  /** Pure function: apply the variables to the current cached value. */
  updater: (current: TData | undefined, variables: TVariables) => TData | undefined
}

export interface UseAppMutationOptions<TData, TVariables, TContext>
  extends Omit<UseMutationOptions<TData, unknown, TVariables, TContext>, 'mutationFn'> {
  /** Keys to invalidate on success. */
  invalidate?: QueryKey[]
  /** Optional optimistic-update descriptor. */
  optimistic?: OptimisticConfig<TData, TVariables>
  /** Silence the global toast on error. Error still flows via mutation.error. */
  silent?: boolean
  /** Context string sent with the error toast so the user sees where it failed. */
  errorContext?: string
}

interface OptimisticContext<TData> {
  previous: TData | undefined
}

export function useAppMutation<TData, TVariables = void, TContext = unknown>(
  mutationFn: MutationFunction<TData, TVariables>,
  options: UseAppMutationOptions<TData, TVariables, TContext> = {}
) {
  const client = useQueryClient()
  const { invalidate, optimistic, silent, errorContext, ...rest } = options

  return useMutation<TData, unknown, TVariables, TContext>({
    mutationFn,
    onMutate: async (variables) => {
      // Run the caller's onMutate first so their side-effects win.
      const callerContext = rest.onMutate ? await rest.onMutate(variables) : undefined

      if (!optimistic) return callerContext

      await client.cancelQueries({ queryKey: optimistic.queryKey })
      const previous = client.getQueryData<TData>(optimistic.queryKey)
      client.setQueryData<TData>(optimistic.queryKey, (current) =>
        optimistic.updater(current, variables)
      )
      // Merge caller context with the rollback snapshot.
      const merged = {
        ...(callerContext as object | undefined),
        __optimistic: { previous } satisfies OptimisticContext<TData>,
      } as unknown as TContext
      return merged
    },
    onError: (error, variables, context) => {
      // Rollback.
      if (optimistic && context && typeof context === 'object' && '__optimistic' in context) {
        const snap = (context as { __optimistic: OptimisticContext<TData> }).__optimistic
        client.setQueryData<TData>(optimistic.queryKey, snap.previous)
      }
      if (!silent) {
        notifyAppError(error, errorContext)
      }
      rest.onError?.(error, variables, context)
    },
    onSettled: (data, error, variables, context) => {
      if (invalidate) {
        for (const key of invalidate) {
          void client.invalidateQueries({ queryKey: key })
        }
      }
      rest.onSettled?.(data, error, variables, context)
    },
    ...(rest.onSuccess ? { onSuccess: rest.onSuccess } : {}),
  })
}
