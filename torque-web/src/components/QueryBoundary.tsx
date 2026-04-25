/**
 * `QueryBoundary` — one-stop loading/error/empty renderer for queries.
 *
 * Callers pass a TanStack Query result plus render props for each state. The
 * component keeps surface areas consistent (skeleton chrome, error copy, empty
 * illustrations) without forcing every page to re-invent the shell.
 *
 * Example:
 *
 *     <QueryBoundary query={leadsQuery} isEmpty={(d) => d.items.length === 0}>
 *       {(leads) => <LeadList items={leads.items} />}
 *     </QueryBoundary>
 */

import { type ReactNode } from 'react'

import { friendlyMessage } from '@/api/errors'
import { Button } from '@/ui/button'
import { EmptyState } from '@/ui/empty-state'
import { Skeleton } from '@/ui/skeleton'

type MinimalQuery<TData> = {
  data: TData | undefined
  error: unknown
  isLoading: boolean
  isError: boolean
  isSuccess: boolean
  refetch: () => unknown
}

export interface QueryBoundaryProps<TData> {
  query: MinimalQuery<TData>
  children: (data: TData) => ReactNode

  /** Skeleton element; defaults to three vertical rows. */
  loadingFallback?: ReactNode
  /** Empty check; when true, render `emptyFallback` instead of children. */
  isEmpty?: (data: TData) => boolean
  /** Empty-state element; defaults to a terse primitive. */
  emptyFallback?: ReactNode
  /** Error-state element; defaults to a recover button + error message. */
  errorFallback?: (error: unknown, retry: () => void) => ReactNode
}

function DefaultLoading() {
  return (
    <div aria-busy="true" role="status" className="flex flex-col gap-3">
      <Skeleton className="h-6 w-1/3" />
      <Skeleton className="h-4 w-full" />
      <Skeleton className="h-4 w-5/6" />
      <Skeleton className="h-4 w-4/6" />
    </div>
  )
}

function DefaultError({ error, retry }: { error: unknown; retry: () => void }) {
  return (
    <EmptyState
      title="Não foi possível carregar."
      description={friendlyMessage(error)}
      action={
        <Button variant="ghost" size="sm" onClick={retry}>
          Tentar de novo
        </Button>
      }
    />
  )
}

export function QueryBoundary<TData>(props: QueryBoundaryProps<TData>) {
  const { query, children, loadingFallback, isEmpty, emptyFallback, errorFallback } = props

  if (query.isLoading) {
    return <>{loadingFallback ?? <DefaultLoading />}</>
  }
  if (query.isError || !query.isSuccess || query.data === undefined) {
    const retry = () => void query.refetch()
    return (
      <>
        {errorFallback ? (
          errorFallback(query.error, retry)
        ) : (
          <DefaultError error={query.error} retry={retry} />
        )}
      </>
    )
  }
  if (isEmpty && isEmpty(query.data)) {
    return (
      <>
        {emptyFallback ?? (
          <EmptyState
            title="Nada aqui ainda."
            description="Quando houver dados, eles aparecem aqui."
          />
        )}
      </>
    )
  }
  return <>{children(query.data)}</>
}
