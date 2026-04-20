/**
 * RouteSkeleton — fallback usado pelo `<Suspense>` ao redor das rotas
 * lazy-loaded. Minimal por design: um header + 3 blocos de skeleton.
 * Evita CLS (Cumulative Layout Shift) ocupando a área do viewport que a
 * rota real vai ocupar no próximo frame.
 *
 * Dark-first como o resto do design system; não usa cor accent para não
 * competir visualmente com o conteúdo que está carregando.
 */

import { Skeleton } from '@/ui/skeleton'

export function RouteSkeleton() {
  return (
    <div
      aria-busy="true"
      aria-label="Carregando página"
      role="status"
      className="mx-auto w-full max-w-[1400px] px-8 py-10"
    >
      <Skeleton className="h-3 w-20" />
      <Skeleton className="mt-3 h-8 w-72" />
      <Skeleton className="mt-2 h-4 w-96" />

      <div className="mt-10 space-y-3">
        <Skeleton className="h-28 w-full" />
        <Skeleton className="h-28 w-full" />
        <Skeleton className="h-28 w-full" />
      </div>
    </div>
  )
}
