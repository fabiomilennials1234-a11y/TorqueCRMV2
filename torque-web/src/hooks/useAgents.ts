/**
 * F06 Copilot hooks — agent CRUD, session lifecycle, knowledge base.
 *
 * The backend (S15) gates all of these endpoints under RequireRole(admin).
 * A non-admin member hitting any of these mutations gets 403 PERMISSION_DENIED
 * via the standard error pipeline. The hooks do NOT try to pre-authorize;
 * call sites either come from admin-only surfaces (AgentsPage) or must
 * tolerate the 403 via the toast.
 */

import { useQuery, useQueryClient } from '@tanstack/react-query'

import { del, get, patch, post } from '@/api/client'
import { useAppMutation } from '@/hooks/useAppMutation'
import { useWSSubscribe } from '@/hooks/useWSSubscribe'

export type AgentStatus = 'draft' | 'active' | 'disabled'

export interface Agent {
  id: string
  name: string
  description?: string | null
  /**
   * system_prompt was added to the view in S38 so the Playground editor
   * can render + edit the persisted prompt without a separate roundtrip.
   * Admin-only endpoint already guards access.
   */
  system_prompt: string
  model: string
  temperature: number
  max_output_tokens: number
  tools_allowlist: string[]
  kill_switch: boolean
  status: AgentStatus
}

export interface UpdateAgentPayload {
  name?: string
  description?: string | null
  system_prompt?: string
  model?: string
  temperature?: number
  max_output_tokens?: number
  tools_allowlist?: string[]
}

export interface CreateAgentPayload {
  name: string
  description?: string
  system_prompt: string
  model: string
  temperature?: number
  max_output_tokens?: number
  tools_allowlist?: string[]
}

function keys() {
  return {
    list: () => ['copilot', 'agents'] as const,
    detail: (id: string) => ['copilot', 'agents', id] as const,
    session: (id: string) => ['copilot', 'sessions', id] as const,
    messages: (id: string) => ['copilot', 'sessions', id, 'messages'] as const,
  }
}

export function useAgents() {
  const client = useQueryClient()

  useWSSubscribe<Agent>(
    ['agent.created', 'agent.activated', 'agent.disabled', 'agent.kill_switch'],
    () => void client.invalidateQueries({ queryKey: ['copilot', 'agents'] })
  )

  return useQuery<Agent[]>({
    queryKey: keys().list(),
    queryFn: async () => (await get<{ data: Agent[] }>('/api/v1/agents')).data,
    staleTime: 30 * 1000,
  })
}

export function useAgent(id: string | undefined) {
  return useQuery<Agent>({
    queryKey: id ? keys().detail(id) : ['copilot', 'agents', 'disabled'],
    enabled: Boolean(id),
    queryFn: () => get<Agent>(`/api/v1/agents/${id}`),
  })
}

export function useCreateAgent() {
  return useAppMutation<Agent, CreateAgentPayload>(
    (body) => post<Agent>('/api/v1/agents', body),
    { invalidate: [['copilot', 'agents']], errorContext: 'copilot.agent.create' }
  )
}

export function useActivateAgent(id: string) {
  return useAppMutation<void, void>(
    () => post<void>(`/api/v1/agents/${id}/activate`, {}),
    { invalidate: [['copilot', 'agents']], errorContext: 'copilot.agent.activate' }
  )
}

export function useDisableAgent(id: string) {
  return useAppMutation<void, void>(
    () => post<void>(`/api/v1/agents/${id}/disable`, {}),
    { invalidate: [['copilot', 'agents']], errorContext: 'copilot.agent.disable' }
  )
}

export function useSetKillSwitch(id: string) {
  return useAppMutation<void, { enabled: boolean }>(
    (body) => post<void>(`/api/v1/agents/${id}/kill-switch`, body),
    { invalidate: [['copilot', 'agents']], errorContext: 'copilot.agent.kill_switch' }
  )
}

// -------- sessions --------------------------------------------------

export interface AgentSession {
  id: string
  agent_id: string
  state: 'open' | 'ended' | 'escalated'
  started_at: string
}

export interface AgentMessage {
  id: string
  session_id: string
  role: 'system' | 'user' | 'assistant' | 'tool'
  content: string
  tool_name?: string | null
  occurred_at: string
}

export function useOpenSession(agentId: string) {
  return useAppMutation<AgentSession, { lead_id?: string; conversation_id?: string }>(
    (body) => post<AgentSession>(`/api/v1/agents/${agentId}/sessions`, body ?? {}),
    { errorContext: 'copilot.session.open' }
  )
}

export function useEndSession(sessionId: string) {
  return useAppMutation<void, void>(
    () => post<void>(`/api/v1/sessions/${sessionId}/end`, {}),
    { errorContext: 'copilot.session.end' }
  )
}

export function useSessionMessages(sessionId: string | undefined) {
  const client = useQueryClient()

  useWSSubscribe<AgentMessage>(
    ['agent_session.opened', 'agent_session.ended'],
    () => {
      if (sessionId) void client.invalidateQueries({ queryKey: keys().messages(sessionId) })
    },
    [sessionId]
  )

  return useQuery<AgentMessage[]>({
    queryKey: sessionId ? keys().messages(sessionId) : ['copilot', 'messages', 'disabled'],
    enabled: Boolean(sessionId),
    queryFn: async () =>
      (await get<{ data: AgentMessage[] }>(`/api/v1/sessions/${sessionId}/messages`)).data,
  })
}

// -------- knowledge -------------------------------------------------

export interface KnowledgeCollectionRef {
  id: string
  name: string
}

export function useCreateCollection() {
  return useAppMutation<KnowledgeCollectionRef, { name: string; description?: string }>(
    (body) => post<KnowledgeCollectionRef>('/api/v1/knowledge/collections', body),
    { errorContext: 'knowledge.collection.create' }
  )
}

export function useEnqueueSource(collectionId: string) {
  return useAppMutation<
    { id: string; status: string },
    { kind: string; title: string; uri?: string; metadata?: unknown }
  >(
    (body) =>
      post<{ id: string; status: string }>(
        `/api/v1/knowledge/collections/${collectionId}/sources`,
        body
      ),
    { errorContext: 'knowledge.source.enqueue' }
  )
}

/** S38: PATCH /api/v1/agents/:id — partial update do config do agente. */
export function useUpdateAgent(id: string) {
  return useAppMutation<Agent, UpdateAgentPayload>(
    (body) => patch<Agent>(`/api/v1/agents/${id}`, body),
    {
      invalidate: [['copilot', 'agents']],
      errorContext: 'copilot.agent.update',
    }
  )
}

// Fallback hook kept for generic cleanup paths.
export function useDeleteAgent(id: string) {
  // Backend currently has no delete endpoint — use disable instead.
  // Kept as a convenience alias so the UI can show "Excluir agente" mapped
  // semantically to disable. Rename when DELETE lands.
  return useAppMutation<void, void>(
    () => del<void>(`/api/v1/agents/${id}`),
    { invalidate: [['copilot', 'agents']], errorContext: 'copilot.agent.delete' }
  )
}
