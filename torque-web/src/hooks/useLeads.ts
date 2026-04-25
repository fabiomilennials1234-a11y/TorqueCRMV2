/**
 * Lead CRUD hooks (F01).
 *
 * Keys live in `queryKeys.leads`. Mutations invalidate the list by default;
 * callers that have a specific detail key can override via `invalidate`.
 */

import { useQuery, useQueryClient } from '@tanstack/react-query'

import { del, get, patch, post } from '@/api/client'
import { queryKeys } from '@/api/queryKeys'
import { useAppMutation } from '@/hooks/useAppMutation'
import { useInfiniteList } from '@/hooks/useInfiniteList'
import { useWSSubscribe } from '@/hooks/useWSSubscribe'

export interface Lead {
  id: string
  name: string
  company?: string | null
  phone?: string | null
  email?: string | null
  position?: string | null
  responsible_id?: string | null
  rating?: number | null
  qualification_score?: number | null
  segment?: string | null
  origin?: string | null
  custom_fields?: unknown
  created_at: string
  updated_at: string
}

export interface LeadsListFilters {
  search?: string
  responsible_id?: string
  page_size?: number
}

/** GET /api/v1/leads — cursor paginated. */
export function useLeads(filters: LeadsListFilters = {}) {
  const client = useQueryClient()

  // Live updates: patch the existing pages in place when the server tells us a
  // row changed. We do not prepend on `lead.created` — the next refetch picks
  // it up, and splicing into paginated caches is fragile.
  useWSSubscribe<Lead>(['lead.updated', 'lead.deleted'], (evt) => {
    const id = evt.entity_id
    if (!id) return
    client.setQueriesData<{ pages?: Array<{ data?: Lead[] }> }>(
      { queryKey: queryKeys.leads.list() },
      (prev) => {
        if (!prev?.pages) return prev
        return {
          ...prev,
          pages: prev.pages.map((p) => ({
            ...p,
            data: (p.data ?? []).reduce<Lead[]>((acc, row) => {
              if (row.id !== id) {
                acc.push(row)
                return acc
              }
              if (evt.type === 'lead.deleted') return acc
              acc.push({ ...row, ...(evt.patch ?? {}) })
              return acc
            }, []),
          })),
        }
      }
    )
  })

  return useInfiniteList<Lead>({
    path: '/api/v1/leads',
    queryKey: queryKeys.leads.list(filters as Record<string, unknown>),
    params: {
      ...(filters.search ? { search: filters.search } : {}),
      ...(filters.responsible_id ? { responsible_id: filters.responsible_id } : {}),
    },
    pageSize: filters.page_size ?? 25,
  })
}

/** GET /api/v1/leads/:id */
export function useLead(id: string | undefined) {
  return useQuery<Lead>({
    queryKey: id ? queryKeys.leads.detail(id) : ['leads', 'detail', 'disabled'],
    enabled: Boolean(id),
    queryFn: () => get<Lead>(`/api/v1/leads/${id}`),
  })
}

/** POST /api/v1/leads */
export function useCreateLead() {
  return useAppMutation<Lead, Partial<Lead>>((body) => post<Lead>('/api/v1/leads', body), {
    invalidate: [queryKeys.leads.all()],
    errorContext: 'lead.create',
  })
}

/** PATCH /api/v1/leads/:id with optimistic update on the detail cache. */
export function useUpdateLead(id: string) {
  return useAppMutation<Lead, Partial<Lead>>((body) => patch<Lead>(`/api/v1/leads/${id}`, body), {
    optimistic: {
      queryKey: queryKeys.leads.detail(id),
      updater: (prev, patchBody) => (prev ? { ...prev, ...patchBody } : prev),
    },
    invalidate: [queryKeys.leads.detail(id), queryKeys.leads.list()],
    errorContext: 'lead.update',
  })
}

/** DELETE /api/v1/leads/:id */
export function useDeleteLead() {
  return useAppMutation<void, string>((id) => del<void>(`/api/v1/leads/${id}`), {
    invalidate: [queryKeys.leads.all()],
    errorContext: 'lead.delete',
  })
}
