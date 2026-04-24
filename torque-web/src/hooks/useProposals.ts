/**
 * F03 Proposal hooks — 1:1 com pipe_entry em pipe do tipo proposal.
 *
 * Status lifecycle: draft → sent → viewed → {accepted | rejected | expired}.
 * Upsert só muta enquanto `status === 'draft'`; o backend responde 409 se o
 * cliente tentar editar um proposal já enviado (intencional: uma proposta
 * enviada é imutável).
 */

import { useQuery } from '@tanstack/react-query'

import { get, post, put } from '@/api/client'
import { useAppMutation } from '@/hooks/useAppMutation'

export type ProposalStatus = 'draft' | 'sent' | 'viewed' | 'accepted' | 'rejected' | 'expired'

export interface Proposal {
  pipe_entry_id: string
  lead_id: string
  title: string
  amount_cents: number
  currency: string
  status: ProposalStatus
  attachment_key?: string | null
  sent_at?: string | null
  first_viewed_at?: string | null
  accepted_at?: string | null
  rejected_at?: string | null
  expires_at?: string | null
}

export interface ProposalUpsertPayload {
  lead_id: string
  title: string
  amount_cents: number
  currency?: string
  attachment_key?: string
  attachment_size?: number
  notes?: string
  expires_at?: string
}

export function useProposal(entryId: string | undefined) {
  return useQuery<Proposal>({
    queryKey: entryId ? ['proposals', 'detail', entryId] : ['proposals', 'disabled'],
    enabled: Boolean(entryId),
    queryFn: () => get<Proposal>(`/api/v1/proposals/${entryId}`),
  })
}

export function useUpsertProposal(entryId: string) {
  return useAppMutation<Proposal, ProposalUpsertPayload>(
    (body) => put<Proposal>(`/api/v1/proposals/${entryId}`, body),
    {
      invalidate: [['proposals', 'detail', entryId]],
      errorContext: 'proposal.upsert',
    }
  )
}

export function useSendProposal(entryId: string) {
  return useAppMutation<void, void>(() => post<void>(`/api/v1/proposals/${entryId}/send`, {}), {
    invalidate: [['proposals', 'detail', entryId]],
    errorContext: 'proposal.send',
  })
}

export function useAcceptProposal(entryId: string) {
  return useAppMutation<void, void>(() => post<void>(`/api/v1/proposals/${entryId}/accept`, {}), {
    invalidate: [['proposals', 'detail', entryId]],
    errorContext: 'proposal.accept',
  })
}

export function useRejectProposal(entryId: string) {
  return useAppMutation<void, { reason?: string }>(
    (body) => post<void>(`/api/v1/proposals/${entryId}/reject`, body),
    { invalidate: [['proposals', 'detail', entryId]], errorContext: 'proposal.reject' }
  )
}
