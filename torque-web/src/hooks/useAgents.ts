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

import { del, get, patch, post, put } from '@/api/client'
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
  /**
   * S39 — optional RAG binding. When non-null, the playground SSE
   * handler runs topK retrieval against this collection before the
   * LLM call. Null / undefined = no retrieval (base system prompt only).
   */
  knowledge_collection_id?: string | null
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

/**
 * S39 — knowledge collection as returned by GET /knowledge/collections.
 * `source_count` comes from a LEFT JOIN aggregate in the repo, so it
 * stays in sync without a second request.
 */
export interface KnowledgeCollection {
  id: string
  name: string
  description?: string | null
  source_count: number
  created_at: string
  updated_at: string
}

export type KnowledgeSourceStatus = 'queued' | 'ingesting' | 'ready' | 'failed'

/**
 * S39 — one knowledge source inside a collection. `status` transitions
 * queued → ingesting → ready (or failed with `error` populated). The
 * ingest pipeline runs detached from the HTTP request, so the UI must
 * poll or subscribe via WS for updates.
 */
export interface KnowledgeSource {
  id: string
  collection_id: string
  kind: string
  title: string
  uri?: string | null
  status: KnowledgeSourceStatus
  error?: string | null
  ingested_at?: string | null
  created_at: string
  updated_at: string
}

function knowledgeKeys() {
  return {
    collections: () => ['copilot', 'knowledge', 'collections'] as const,
    sources: (cid: string) => ['copilot', 'knowledge', 'collections', cid, 'sources'] as const,
  }
}

/**
 * useKnowledgeCollections lists every collection in the tenant and
 * keeps the cache fresh via the knowledge.collection_created WS event.
 * source_count is a live aggregate — no secondary round-trip needed
 * to render the list with per-collection counts.
 */
export function useKnowledgeCollections() {
  const client = useQueryClient()

  useWSSubscribe<KnowledgeCollection>(
    ['knowledge.collection_created'],
    () => void client.invalidateQueries({ queryKey: knowledgeKeys().collections() })
  )

  return useQuery<KnowledgeCollection[]>({
    queryKey: knowledgeKeys().collections(),
    queryFn: async () =>
      (await get<{ data: KnowledgeCollection[] }>('/api/v1/knowledge/collections')).data,
    staleTime: 30 * 1000,
  })
}

/**
 * useKnowledgeSources lists every source inside a collection. The WS
 * event `knowledge.source_enqueued` triggers an invalidation so the
 * row appears immediately after POST, then transitions as the ingest
 * service updates status. A poll interval of 3s covers status flips
 * since the backend currently does not publish ingesting/ready events
 * (follow-up sprint will).
 */
export function useKnowledgeSources(collectionId: string | undefined) {
  const client = useQueryClient()

  useWSSubscribe<KnowledgeSource>(
    ['knowledge.source_enqueued'],
    () => {
      if (collectionId) void client.invalidateQueries({ queryKey: knowledgeKeys().sources(collectionId) })
    },
    [collectionId]
  )

  return useQuery<KnowledgeSource[]>({
    queryKey: collectionId ? knowledgeKeys().sources(collectionId) : ['copilot', 'knowledge', 'sources', 'disabled'],
    enabled: Boolean(collectionId),
    queryFn: async () =>
      (
        await get<{ data: KnowledgeSource[] }>(
          `/api/v1/knowledge/collections/${collectionId}/sources`
        )
      ).data,
    refetchInterval: (query) => {
      // Stop polling once every source is terminal (ready|failed).
      const data = query.state.data
      if (!data || data.some((s) => s.status === 'queued' || s.status === 'ingesting')) {
        return 3_000
      }
      return false
    },
  })
}

export function useCreateCollection() {
  return useAppMutation<KnowledgeCollectionRef, { name: string; description?: string }>(
    (body) => post<KnowledgeCollectionRef>('/api/v1/knowledge/collections', body),
    {
      invalidate: [['copilot', 'knowledge', 'collections']],
      errorContext: 'knowledge.collection.create',
    }
  )
}

/**
 * S39 — inline-text source ingest. The handler gates kind=text|markdown
 * on non-empty `content`; URL kinds require `uri`. The service runs
 * chunk → embed → insert detached from this mutation; the UI should
 * refetch sources (useKnowledgeSources) to see status transitions.
 */
export function useEnqueueSource(collectionId: string) {
  return useAppMutation<
    { id: string; status: string },
    {
      kind: 'text' | 'markdown' | 'url'
      title: string
      content?: string
      uri?: string
      metadata?: unknown
    }
  >(
    (body) =>
      post<{ id: string; status: string }>(
        `/api/v1/knowledge/collections/${collectionId}/sources`,
        body
      ),
    {
      invalidate: [['copilot', 'knowledge', 'collections', collectionId, 'sources']],
      errorContext: 'knowledge.source.enqueue',
    }
  )
}

/**
 * S39 — bind/unbind a RAG collection to an agent. Pass `null` to
 * detach. Cross-tenant binds are refused at the repo layer (404).
 */
export function useBindAgentCollection(agentId: string) {
  return useAppMutation<Agent, { collection_id: string | null }>(
    (body) => put<Agent>(`/api/v1/agents/${agentId}/knowledge-collection`, body),
    {
      invalidate: [['copilot', 'agents']],
      errorContext: 'copilot.agent.knowledge_bind',
    }
  )
}

// -------- S40 triggers ----------------------------------------------

/**
 * S40 — filter DSL. Matches ai.FilterSpec on the backend:
 *   { all: [{field,op,value}], any: [{...}] }
 * Empty filter (both blocks empty) matches every lead — used by
 * catch-all rules at a low priority.
 */
export interface TriggerPredicate {
  field: string
  op: 'eq' | 'neq' | 'contains' | 'in' | 'present' | 'absent'
  value?: unknown
}

export interface TriggerFilter {
  all?: TriggerPredicate[]
  any?: TriggerPredicate[]
}

export interface AgentTrigger {
  id: string
  agent_id: string
  name: string
  description?: string | null
  priority: number
  filter: TriggerFilter
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface CreateTriggerPayload {
  name: string
  description?: string
  priority?: number
  filter: TriggerFilter
}

export interface UpdateTriggerPayload {
  name?: string
  description?: string | null
  priority?: number
  filter?: TriggerFilter
  is_active?: boolean
}

function triggerKeys() {
  return {
    list: (agentId: string) => ['copilot', 'agents', agentId, 'triggers'] as const,
  }
}

export function useAgentTriggers(agentId: string | undefined) {
  const client = useQueryClient()

  useWSSubscribe<AgentTrigger>(
    ['agent_trigger.created', 'agent_trigger.updated', 'agent_trigger.deleted'],
    () => {
      if (agentId) void client.invalidateQueries({ queryKey: triggerKeys().list(agentId) })
    },
    [agentId]
  )

  return useQuery<AgentTrigger[]>({
    queryKey: agentId ? triggerKeys().list(agentId) : ['copilot', 'triggers', 'disabled'],
    enabled: Boolean(agentId),
    queryFn: async () =>
      (await get<{ data: AgentTrigger[] }>(`/api/v1/agents/${agentId}/triggers`)).data,
  })
}

export function useCreateTrigger(agentId: string) {
  return useAppMutation<AgentTrigger, CreateTriggerPayload>(
    (body) => post<AgentTrigger>(`/api/v1/agents/${agentId}/triggers`, body),
    {
      invalidate: [['copilot', 'agents', agentId, 'triggers']],
      errorContext: 'copilot.trigger.create',
    }
  )
}

export function useUpdateTrigger(agentId: string, triggerId: string) {
  return useAppMutation<AgentTrigger, UpdateTriggerPayload>(
    (body) => patch<AgentTrigger>(`/api/v1/triggers/${triggerId}`, body),
    {
      invalidate: [['copilot', 'agents', agentId, 'triggers']],
      errorContext: 'copilot.trigger.update',
    }
  )
}

export function useDeleteTrigger(agentId: string, triggerId: string) {
  return useAppMutation<void, void>(
    () => del<void>(`/api/v1/triggers/${triggerId}`),
    {
      invalidate: [['copilot', 'agents', agentId, 'triggers']],
      errorContext: 'copilot.trigger.delete',
    }
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
