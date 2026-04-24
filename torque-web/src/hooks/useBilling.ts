/**
 * F14 Billing — checkout flow + active subscription view.
 *
 * Server returns `{ data: null }` when there's no live subscription; the
 * hook surfaces `undefined` for "no subscription" so UI can branch cleanly.
 */

import { useQuery, useQueryClient } from '@tanstack/react-query'

import { get, post } from '@/api/client'
import { useAppMutation } from '@/hooks/useAppMutation'
import { useWSSubscribe } from '@/hooks/useWSSubscribe'

export type SubscriptionStatus = 'pending' | 'active' | 'past_due' | 'cancelled' | 'expired'

export interface Subscription {
  id: string
  plan_id: string
  status: SubscriptionStatus
  provider: 'mock' | 'asaas'
  amount_cents: number
  currency: string
  pix_qr_code?: string | null
  pix_qr_code_image?: string | null
  pix_expires_at?: string | null
  current_period_end?: string | null
  cancelled_at?: string | null
  created_at: string
}

interface SubscriptionResponse {
  data?: Subscription | null
}

export function useSubscription() {
  const client = useQueryClient()

  useWSSubscribe(
    [
      'subscription.created',
      'subscription.activated',
      'subscription.past_due',
      'subscription.cancelled',
    ],
    () => void client.invalidateQueries({ queryKey: ['billing', 'subscription'] })
  )

  return useQuery<Subscription | undefined>({
    queryKey: ['billing', 'subscription'],
    queryFn: async () => {
      const res = await get<SubscriptionResponse | Subscription>('/api/v1/billing/subscription')
      // Server returns either the view directly or `{ data: null }` when
      // no live sub exists. Handle both without widening the contract.
      if (res && typeof res === 'object' && 'data' in (res as Record<string, unknown>)) {
        return (res as SubscriptionResponse).data ?? undefined
      }
      return res as Subscription
    },
    staleTime: 30 * 1000,
  })
}

export function useStartCheckout() {
  return useAppMutation<Subscription, { plan_id: string; amount_cents: number; currency?: string }>(
    (body) => post<Subscription>('/api/v1/billing/checkout', body),
    {
      invalidate: [['billing', 'subscription']],
      errorContext: 'billing.checkout',
    }
  )
}

export function useCancelSubscription() {
  return useAppMutation<void, void>(() => post<void>('/api/v1/billing/subscription/cancel', {}), {
    invalidate: [['billing', 'subscription']],
    errorContext: 'billing.cancel',
  })
}
