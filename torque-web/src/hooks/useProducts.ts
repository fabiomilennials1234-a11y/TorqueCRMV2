/**
 * F11 Produtos — catalog used by Propostas (F03/F12) to assemble line items.
 *
 * List uses the cursor-paginated helper (ADR-004). Mutations are admin-only;
 * read is member-accessible (salespeople need the catalog).
 */

import { useQuery, useQueryClient } from '@tanstack/react-query'

import { del, get, patch, post } from '@/api/client'
import { useAppMutation } from '@/hooks/useAppMutation'
import { useInfiniteList } from '@/hooks/useInfiniteList'
import { useWSSubscribe } from '@/hooks/useWSSubscribe'

export interface Product {
  id: string
  name: string
  description?: string | null
  sku?: string | null
  price_cents: number
  currency: string
  is_active: boolean
  metadata?: unknown
  created_by?: string | null
  created_at: string
  updated_at: string
}

export interface ProductsFilters {
  search?: string
  active_only?: boolean
}

function productKeys() {
  return {
    list: (f: ProductsFilters = {}) => ['products', 'list', f] as const,
    detail: (id: string) => ['products', 'detail', id] as const,
  }
}

/**
 * Paginated list. Uses useInfiniteList so the Propostas picker can lazy-load.
 * WS invalidation is coarse (entire list) on product.* events; catalog is
 * small enough that a refetch is cheaper than surgical patching.
 */
export function useProducts(filters: ProductsFilters = {}) {
  const client = useQueryClient()

  useWSSubscribe(
    ['product.created', 'product.updated', 'product.archived'],
    () => void client.invalidateQueries({ queryKey: ['products'] })
  )

  const params: Record<string, string> = {}
  if (filters.search) params.search = filters.search
  if (filters.active_only) params.active_only = '1'

  return useInfiniteList<Product>({
    queryKey: productKeys().list(filters),
    path: '/api/v1/products',
    params,
    pageSize: 50,
    staleTime: 60 * 1000,
  })
}

export function useProduct(id: string | undefined) {
  return useQuery<Product>({
    queryKey: id ? productKeys().detail(id) : ['products', 'detail', 'disabled'],
    enabled: Boolean(id),
    queryFn: () => get<Product>(`/api/v1/products/${id}`),
  })
}

// -------- mutations --------------------------------------------------

export function useCreateProduct() {
  return useAppMutation<
    Product,
    {
      name: string
      description?: string
      sku?: string
      price_cents: number
      currency?: string
      metadata?: unknown
    }
  >((body) => post<Product>('/api/v1/products', body), {
    invalidate: [['products']],
    errorContext: 'product.create',
  })
}

export function useUpdateProduct(id: string) {
  return useAppMutation<
    Product,
    {
      name?: string
      description?: string | null
      sku?: string | null
      price_cents?: number
      currency?: string
      is_active?: boolean
      metadata?: unknown
    }
  >((body) => patch<Product>(`/api/v1/products/${id}`, body), {
    invalidate: [['products']],
    errorContext: 'product.update',
  })
}

export function useArchiveProduct(id: string) {
  return useAppMutation<void, void>(() => del<void>(`/api/v1/products/${id}`), {
    invalidate: [['products']],
    errorContext: 'product.archive',
  })
}

export { productKeys }
