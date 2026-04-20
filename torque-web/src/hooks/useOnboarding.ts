/**
 * F13 Onboarding wizard — first-run state for each team member.
 *
 * Server stamps `completed_at` when the last canonical step is done; the
 * gate below reads that + `dismissed` to decide whether to force the
 * wizard before rendering the app.
 */

import { useQuery, useQueryClient } from '@tanstack/react-query'

import { get, post } from '@/api/client'
import { useAppMutation } from '@/hooks/useAppMutation'

export interface OnboardingStatus {
  current_step: string
  steps_completed: string[]
  all_steps: string[]
  dismissed: boolean
  completed_at?: string | null
}

export function useOnboarding() {
  return useQuery<OnboardingStatus>({
    queryKey: ['onboarding'],
    queryFn: () => get<OnboardingStatus>('/api/v1/onboarding'),
    staleTime: 30 * 1000,
  })
}

export function useCompleteStep() {
  const client = useQueryClient()
  return useAppMutation<OnboardingStatus, { step: string }>(
    (body) => post<OnboardingStatus>('/api/v1/onboarding/steps', body),
    {
      errorContext: 'onboarding.step',
      onSuccess: (data) => {
        client.setQueryData(['onboarding'], data)
      },
    }
  )
}

export function useDismissOnboarding() {
  return useAppMutation<void, void>(
    () => post<void>('/api/v1/onboarding/dismiss', {}),
    { invalidate: [['onboarding']], errorContext: 'onboarding.dismiss' }
  )
}

export function useResetOnboarding() {
  return useAppMutation<void, void>(
    () => post<void>('/api/v1/onboarding/reset', {}),
    { invalidate: [['onboarding']], errorContext: 'onboarding.reset' }
  )
}

/**
 * Predicate used by the route gate. True when the user should be forced
 * through the wizard before reaching the product.
 */
export function shouldGate(status: OnboardingStatus | undefined): boolean {
  if (!status) return false
  if (status.dismissed) return false
  return !status.completed_at
}
