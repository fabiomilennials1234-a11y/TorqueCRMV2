/**
 * F06 Copilot — AgentPlaygroundPage (S38).
 *
 * Layout estilo Vercel AI Playground: editor de configuração à esquerda,
 * chat de teste à direita. O editor persiste via `PATCH /agents/:id`
 * (useUpdateAgent); o chat consome SSE via `useAgentStream` lendo o
 * system_prompt+model+temp+max_tokens direto da última versão salva
 * (o backend refresca antes de dialar o provider).
 *
 * Design decisions:
 * - Draft local + dirty detection (sem auto-save em cada keystroke).
 * - Regenerate = stop() → send() com a mesma conversa (útil em dev).
 * - Kill-switch é gate server-side (409) que aparece como "error" no
 *   stream; a UI não replica o toggle — é via AgentListPage.
 */

import {
  ArrowLeft,
  Book,
  Play,
  Plus,
  RotateCcw,
  Send,
  Square,
  Trash2,
  Volume2,
  Zap,
} from 'lucide-react'
import { useEffect, useId, useMemo, useRef, useState } from 'react'
import { Link, Navigate, useParams } from 'react-router-dom'

import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { Input } from '@/ui/input'
import { PageHeader } from '@/ui/page-header'
import { Skeleton } from '@/ui/skeleton'
import { friendlyMessage } from '@/api/errors'
import {
  previewTTS,
  useAgent,
  useAgentTriggers,
  useBindAgentCollection,
  useCreateCollection,
  useCreateTrigger,
  useDeleteTrigger,
  useEnqueueSource,
  useKnowledgeCollections,
  useKnowledgeSources,
  useUpdateAgent,
  useUpdateTrigger,
  type TriggerFilter,
  type UpdateAgentPayload,
} from '@/hooks/useAgents'
import { useAgentStream } from '@/hooks/useAgentStream'

type ChatTurn = {
  role: 'user' | 'assistant'
  content: string
}

export function AgentPlaygroundPage() {
  const { id } = useParams<{ id: string }>()
  const query = useAgent(id)

  if (!id) return <Navigate to="/copilot" replace />
  if (query.isLoading) return <PlaygroundSkeleton />
  if (query.isError || !query.data) {
    return (
      <div className="mx-auto max-w-md p-12 text-center">
        <p className="text-sm text-danger">{friendlyMessage(query.error)}</p>
        <Link to="/copilot" className="mt-4 inline-block text-sm text-accent underline">
          Voltar
        </Link>
      </div>
    )
  }
  return <PlaygroundInner agent={query.data} agentId={id} />
}

function PlaygroundInner({ agent, agentId }: { agent: ReturnType<typeof useAgent>['data'] & {}; agentId: string }) {
  const update = useUpdateAgent(agentId)
  const stream = useAgentStream(agentId)

  const [draft, setDraft] = useState<UpdateAgentPayload>({
    name: agent.name,
    system_prompt: agent.system_prompt,
    model: agent.model,
    temperature: agent.temperature,
    max_output_tokens: agent.max_output_tokens,
    tts_enabled: agent.tts_enabled,
    tts_voice_id: agent.tts_voice_id ?? '',
  })

  // Conversa local do playground. Não persiste — a sessão formal vive
  // em /sessions/:id/messages e chega em sprint posterior.
  const [conversation, setConversation] = useState<ChatTurn[]>([])
  const [input, setInput] = useState('')

  const nameId = useId()
  const promptId = useId()
  const modelId = useId()
  const tempId = useId()
  const tokensId = useId()

  const dirty = useMemo(() => {
    return (
      draft.name !== agent.name ||
      draft.system_prompt !== agent.system_prompt ||
      draft.model !== agent.model ||
      draft.temperature !== agent.temperature ||
      draft.max_output_tokens !== agent.max_output_tokens ||
      draft.tts_enabled !== agent.tts_enabled ||
      (draft.tts_voice_id ?? '') !== (agent.tts_voice_id ?? '')
    )
  }, [agent, draft])

  async function handleSave() {
    if (!dirty || update.isPending) return
    await update.mutateAsync(draft)
  }

  async function handleSend() {
    const text = input.trim()
    if (!text || stream.state === 'streaming') return
    const nextTurn: ChatTurn = { role: 'user', content: text }
    const nextConv: ChatTurn[] = [...conversation, nextTurn]
    setConversation(nextConv)
    setInput('')

    let assistantBuffer = ''
    await stream.send({
      messages: nextConv,
      onToken: (delta) => {
        assistantBuffer += delta
      },
      onDone: () => {
        setConversation((c) => [...c, { role: 'assistant', content: assistantBuffer }])
      },
      onError: () => {
        // Mantém a user turn mas não append assistant (error já exposto).
      },
    })
  }

  function handleRegenerate() {
    if (stream.state === 'streaming') stream.stop()
    if (conversation.length === 0) return
    // Remove último turn assistant se houver, mantém o user, re-envia.
    const last = conversation[conversation.length - 1]
    const base = last && last.role === 'assistant' ? conversation.slice(0, -1) : conversation
    setConversation(base)
    let assistantBuffer = ''
    void stream.send({
      messages: base,
      onToken: (d) => (assistantBuffer += d),
      onDone: () => setConversation((c) => [...c, { role: 'assistant', content: assistantBuffer }]),
    })
  }

  return (
    <div className="mx-auto grid h-[calc(100vh-56px)] max-w-[1600px] grid-cols-[minmax(320px,420px)_1fr] gap-6 px-8">
      {/* Editor */}
      <aside className="flex min-h-0 flex-col overflow-y-auto py-6">
        <Link to="/copilot" className="mb-3 inline-flex items-center gap-1.5 text-xs text-ink-dim hover:text-ink-muted">
          <ArrowLeft className="h-3.5 w-3.5" />
          Agentes
        </Link>
        <PageHeader eyebrow="Copilot" title={agent.name} description={agent.description ?? undefined} />

        <div className="mt-6 flex items-center gap-2">
          <Badge tone={agent.status === 'active' ? 'success' : 'neutral'}>{agent.status}</Badge>
          {agent.kill_switch && <Badge tone="danger">Kill-switch</Badge>}
        </div>

        <form
          className="mt-6 space-y-4"
          onSubmit={(e) => {
            e.preventDefault()
            void handleSave()
          }}
        >
          <div>
            <label htmlFor={nameId} className="mb-1 block text-xs text-ink-muted">
              Nome
            </label>
            <Input
              id={nameId}
              value={draft.name ?? ''}
              onChange={(e) => setDraft((d) => ({ ...d, name: e.target.value }))}
            />
          </div>
          <div>
            <label htmlFor={promptId} className="mb-1 block text-xs text-ink-muted">
              System prompt
            </label>
            <textarea
              id={promptId}
              rows={10}
              value={draft.system_prompt ?? ''}
              onChange={(e) => setDraft((d) => ({ ...d, system_prompt: e.target.value }))}
              className="w-full resize-y rounded-md bg-elevated/40 px-3 py-2 text-sm text-ink shadow-hairline placeholder:text-ink-dim focus:outline-none focus:ring-1 focus:ring-accent/50"
            />
            <p className="mt-1 text-2xs text-ink-dim">
              10 a 16.000 caracteres. Define tom, regras de negócio, restrições.
            </p>
          </div>
          <div>
            <label htmlFor={modelId} className="mb-1 block text-xs text-ink-muted">
              Modelo
            </label>
            <Input
              id={modelId}
              value={draft.model ?? ''}
              onChange={(e) => setDraft((d) => ({ ...d, model: e.target.value }))}
            />
            <p className="mt-1 text-2xs text-ink-dim">
              Qualquer identifier aceito pelo OpenRouter (ex.: anthropic/claude-sonnet-4).
            </p>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label htmlFor={tempId} className="mb-1 block text-xs text-ink-muted">
                Temperatura
              </label>
              <Input
                id={tempId}
                type="number"
                step="0.1"
                min={0}
                max={2}
                value={draft.temperature ?? 0}
                onChange={(e) =>
                  setDraft((d) => ({ ...d, temperature: Number(e.target.value) }))
                }
              />
            </div>
            <div>
              <label htmlFor={tokensId} className="mb-1 block text-xs text-ink-muted">
                Max tokens
              </label>
              <Input
                id={tokensId}
                type="number"
                min={16}
                max={8192}
                value={draft.max_output_tokens ?? 0}
                onChange={(e) =>
                  setDraft((d) => ({ ...d, max_output_tokens: Number(e.target.value) }))
                }
              />
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Button type="submit" variant="primary" size="sm" disabled={!dirty || update.isPending}>
              {update.isPending ? 'Salvando…' : 'Salvar alterações'}
            </Button>
            {dirty && (
              <span className="text-2xs text-ink-dim">Alterações pendentes — chat usa a versão salva.</span>
            )}
          </div>
        </form>

        <TTSPanel
          agentId={agentId}
          enabled={draft.tts_enabled ?? false}
          voiceId={draft.tts_voice_id ?? ''}
          onEnabledChange={(v) => setDraft((d) => ({ ...d, tts_enabled: v }))}
          onVoiceIdChange={(v) => setDraft((d) => ({ ...d, tts_voice_id: v }))}
        />
        <KnowledgePanel agentId={agentId} collectionId={agent.knowledge_collection_id ?? null} />
        <TriggersPanel agentId={agentId} />
      </aside>

      {/* Chat pane */}
      <section className="flex min-h-0 flex-col overflow-hidden rounded-lg bg-surface shadow-elev-1">
        <ChatTranscript
          conversation={conversation}
          pending={stream.state === 'streaming' ? stream.pending : ''}
        />

        {stream.state === 'error' && stream.error && (
          <div
            role="alert"
            className="border-t border-danger/30 bg-danger/10 px-6 py-2 text-xs text-danger"
          >
            {stream.error.code}: {stream.error.message}
          </div>
        )}

        <form
          className="flex items-end gap-2 border-t border-hairline px-6 py-3"
          onSubmit={(e) => {
            e.preventDefault()
            void handleSend()
          }}
        >
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault()
                void handleSend()
              }
            }}
            placeholder="Mensagem do usuário — Enter envia, Shift+Enter quebra linha"
            rows={2}
            className="flex-1 resize-none rounded-md bg-elevated/40 px-3 py-2 text-sm text-ink shadow-hairline placeholder:text-ink-dim focus:outline-none focus:ring-1 focus:ring-accent/50"
            disabled={stream.state === 'streaming'}
          />
          {stream.state === 'streaming' ? (
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => stream.stop()}
            >
              <Square className="mr-1 h-3.5 w-3.5" />
              Parar
            </Button>
          ) : (
            <Button
              type="submit"
              variant="primary"
              size="sm"
              disabled={!input.trim()}
            >
              <Send className="mr-1 h-3.5 w-3.5" />
              Enviar
            </Button>
          )}
          {conversation.length > 0 && stream.state !== 'streaming' && (
            <Button type="button" variant="ghost" size="sm" onClick={handleRegenerate}>
              <RotateCcw className="mr-1 h-3.5 w-3.5" />
              Regenerar
            </Button>
          )}
        </form>
      </section>
    </div>
  )
}

function ChatTranscript({
  conversation,
  pending,
}: {
  conversation: ChatTurn[]
  pending: string
}) {
  const scrollRef = useRef<HTMLDivElement>(null)
  useEffect(() => {
    const el = scrollRef.current
    if (el) el.scrollTop = el.scrollHeight
  }, [conversation, pending])

  return (
    <div ref={scrollRef} className="flex-1 space-y-3 overflow-y-auto px-6 py-4">
      {conversation.length === 0 && !pending && (
        <div className="flex h-full items-center justify-center text-sm text-ink-dim">
          Envie uma mensagem para testar o agente.
        </div>
      )}
      {conversation.map((turn, i) => (
        <TranscriptBubble key={i} speaker={turn.role} content={turn.content} />
      ))}
      {pending && <TranscriptBubble speaker="assistant" content={pending} streaming />}
    </div>
  )
}

function TranscriptBubble({
  speaker,
  content,
  streaming,
}: {
  speaker: 'user' | 'assistant'
  content: string
  streaming?: boolean
}) {
  const isUser = speaker === 'user'
  return (
    <div className={isUser ? 'flex justify-end' : 'flex justify-start'}>
      <div
        className={
          'max-w-[72%] rounded-lg px-3 py-2 text-sm shadow-elev-1 ' +
          (isUser ? 'bg-accent/15 text-ink' : 'bg-elevated/40 text-ink-muted')
        }
      >
        <p className="whitespace-pre-wrap break-words">{content}</p>
        {streaming && (
          <span className="ml-1 inline-block h-3 w-0.5 animate-pulse bg-ink align-text-bottom" />
        )}
      </div>
    </div>
  )
}

function KnowledgePanel({
  agentId,
  collectionId,
}: {
  agentId: string
  collectionId: string | null
}) {
  const collections = useKnowledgeCollections()
  const sources = useKnowledgeSources(collectionId ?? undefined)
  const bind = useBindAgentCollection(agentId)
  const createCollection = useCreateCollection()
  const enqueue = useEnqueueSource(collectionId ?? '')

  const [creatingCollection, setCreatingCollection] = useState(false)
  const [collectionName, setCollectionName] = useState('')
  const [sourceTitle, setSourceTitle] = useState('')
  const [sourceContent, setSourceContent] = useState('')
  const collectionSelectId = useId()

  async function handleBind(value: string) {
    await bind.mutateAsync({ collection_id: value === '' ? null : value })
  }

  async function handleCreateCollection(e: React.FormEvent) {
    e.preventDefault()
    const name = collectionName.trim()
    if (!name) return
    const created = await createCollection.mutateAsync({ name })
    setCollectionName('')
    setCreatingCollection(false)
    await bind.mutateAsync({ collection_id: created.id })
  }

  async function handleAddSource(e: React.FormEvent) {
    e.preventDefault()
    if (!collectionId) return
    const title = sourceTitle.trim()
    const content = sourceContent.trim()
    if (!title || !content) return
    await enqueue.mutateAsync({ kind: 'text', title, content })
    setSourceTitle('')
    setSourceContent('')
  }

  return (
    <section className="mt-8 border-t border-hairline pt-6">
      <div className="mb-3 flex items-center gap-2">
        <Book className="h-4 w-4 text-ink-muted" />
        <h3 className="text-sm font-medium text-ink">Base de conhecimento</h3>
      </div>
      <p className="mb-3 text-2xs text-ink-dim">
        Ligue uma coleção para injetar contexto recuperado (topK=5) no system prompt antes de cada mensagem.
      </p>

      <div className="space-y-2">
        <label htmlFor={collectionSelectId} className="block text-xs text-ink-muted">
          Coleção ativa
        </label>
        <select
          id={collectionSelectId}
          className="w-full rounded-md bg-elevated/40 px-3 py-2 text-sm text-ink shadow-hairline focus:outline-none focus:ring-1 focus:ring-accent/50"
          value={collectionId ?? ''}
          onChange={(e) => void handleBind(e.target.value)}
          disabled={bind.isPending || collections.isLoading}
        >
          <option value="">— sem retrieval —</option>
          {(collections.data ?? []).map((c) => (
            <option key={c.id} value={c.id}>
              {c.name} ({c.source_count})
            </option>
          ))}
        </select>
      </div>

      {!creatingCollection ? (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="mt-2"
          onClick={() => setCreatingCollection(true)}
        >
          <Plus className="mr-1 h-3.5 w-3.5" />
          Nova coleção
        </Button>
      ) : (
        <form className="mt-2 flex items-center gap-2" onSubmit={handleCreateCollection}>
          <Input
            value={collectionName}
            onChange={(e) => setCollectionName(e.target.value)}
            placeholder="Nome da coleção"
          />
          <Button type="submit" variant="primary" size="sm" disabled={createCollection.isPending}>
            Criar
          </Button>
          <Button type="button" variant="ghost" size="sm" onClick={() => setCreatingCollection(false)}>
            Cancelar
          </Button>
        </form>
      )}

      {collectionId && (
        <div className="mt-6 space-y-3">
          <h4 className="text-xs font-medium text-ink-muted">Fontes</h4>
          {sources.isLoading ? (
            <Skeleton className="h-12 w-full" />
          ) : (sources.data ?? []).length === 0 ? (
            <p className="text-2xs text-ink-dim">Nenhuma fonte ingerida ainda.</p>
          ) : (
            <ul className="space-y-1.5">
              {sources.data!.map((s) => (
                <li
                  key={s.id}
                  className="flex items-center justify-between rounded-md bg-elevated/30 px-3 py-2 text-xs"
                >
                  <span className="truncate pr-2 text-ink">{s.title}</span>
                  <Badge
                    tone={
                      s.status === 'ready'
                        ? 'success'
                        : s.status === 'failed'
                          ? 'danger'
                          : 'neutral'
                    }
                  >
                    {s.status}
                  </Badge>
                </li>
              ))}
            </ul>
          )}

          <form className="space-y-2" onSubmit={handleAddSource}>
            <Input
              value={sourceTitle}
              onChange={(e) => setSourceTitle(e.target.value)}
              placeholder="Título da fonte (ex: FAQ 2026)"
            />
            <textarea
              value={sourceContent}
              onChange={(e) => setSourceContent(e.target.value)}
              rows={4}
              placeholder="Cole o texto aqui — será dividido em chunks e embeddado."
              className="w-full resize-y rounded-md bg-elevated/40 px-3 py-2 text-xs text-ink shadow-hairline placeholder:text-ink-dim focus:outline-none focus:ring-1 focus:ring-accent/50"
            />
            <Button
              type="submit"
              variant="outline"
              size="sm"
              disabled={enqueue.isPending || !sourceTitle.trim() || !sourceContent.trim()}
            >
              {enqueue.isPending ? 'Enviando…' : 'Adicionar fonte'}
            </Button>
          </form>
        </div>
      )}
    </section>
  )
}

function TTSPanel({
  agentId,
  enabled,
  voiceId,
  onEnabledChange,
  onVoiceIdChange,
}: {
  agentId: string
  enabled: boolean
  voiceId: string
  onEnabledChange: (v: boolean) => void
  onVoiceIdChange: (v: string) => void
}) {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [audioUrl, setAudioUrl] = useState<string | null>(null)
  const enabledId = useId()
  const voiceIdField = useId()

  // Free the previous object URL when it changes or the component unmounts —
  // URL.createObjectURL holds the blob in memory until revoked.
  useEffect(() => {
    return () => {
      if (audioUrl) URL.revokeObjectURL(audioUrl)
    }
  }, [audioUrl])

  async function handlePreview() {
    if (!voiceId.trim()) return
    setLoading(true)
    setError(null)
    try {
      const url = await previewTTS(agentId, { voice_id: voiceId.trim() })
      if (audioUrl) URL.revokeObjectURL(audioUrl)
      setAudioUrl(url)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }

  return (
    <section className="mt-8 border-t border-hairline pt-6">
      <div className="mb-3 flex items-center gap-2">
        <Volume2 className="h-4 w-4 text-ink-muted" />
        <h3 className="text-sm font-medium text-ink">Voz (TTS)</h3>
      </div>
      <p className="mb-3 text-2xs text-ink-dim">
        Quando habilitado, respostas do agente são renderizadas como áudio (ElevenLabs). Preview não
        consome quota da conversa real.
      </p>

      <div className="flex items-center gap-2">
        <input
          id={enabledId}
          type="checkbox"
          checked={enabled}
          onChange={(e) => onEnabledChange(e.target.checked)}
          className="h-4 w-4"
        />
        <label htmlFor={enabledId} className="text-xs text-ink-muted">
          Habilitar síntese de voz
        </label>
      </div>

      <div className="mt-3">
        <label htmlFor={voiceIdField} className="mb-1 block text-xs text-ink-muted">
          Voice ID (ElevenLabs)
        </label>
        <Input
          id={voiceIdField}
          value={voiceId}
          onChange={(e) => onVoiceIdChange(e.target.value)}
          placeholder="Ex: 21m00Tcm4TlvDq8ikWAM"
        />
        <p className="mt-1 text-2xs text-ink-dim">
          Deixe vazio para desvincular. O ID fica em elevenlabs.io → Voice Library.
        </p>
      </div>

      <div className="mt-3 flex items-center gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => void handlePreview()}
          disabled={loading || !voiceId.trim()}
        >
          <Play className="mr-1 h-3.5 w-3.5" />
          {loading ? 'Gerando…' : 'Ouvir preview'}
        </Button>
      </div>

      {error && (
        <p role="alert" className="mt-2 text-2xs text-danger">
          {error}
        </p>
      )}
      {audioUrl && (
        <audio
          className="mt-2 w-full"
          src={audioUrl}
          controls
          autoPlay
          onError={() => setError('Falha ao reproduzir o áudio.')}
        >
          <track kind="captions" />
        </audio>
      )}
    </section>
  )
}

function TriggersPanel({ agentId }: { agentId: string }) {
  const triggers = useAgentTriggers(agentId)
  const create = useCreateTrigger(agentId)

  const [showForm, setShowForm] = useState(false)
  const [name, setName] = useState('')
  const [priority, setPriority] = useState(100)
  const [field, setField] = useState('origin')
  const [op, setOp] = useState<'eq' | 'neq' | 'contains' | 'in' | 'present' | 'absent'>('eq')
  const [value, setValue] = useState('')
  const priorityId = useId()

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    const trimmedName = name.trim()
    if (!trimmedName) return
    const filter: TriggerFilter =
      op === 'present' || op === 'absent'
        ? { all: [{ field, op }] }
        : op === 'in'
          ? {
              all: [
                {
                  field,
                  op,
                  value: value
                    .split(',')
                    .map((v) => v.trim())
                    .filter(Boolean),
                },
              ],
            }
          : { all: [{ field, op, value: value.trim() }] }
    await create.mutateAsync({ name: trimmedName, priority, filter })
    setName('')
    setValue('')
    setShowForm(false)
  }

  return (
    <section className="mt-8 border-t border-hairline pt-6">
      <div className="mb-3 flex items-center gap-2">
        <Zap className="h-4 w-4 text-ink-muted" />
        <h3 className="text-sm font-medium text-ink">Gatilhos de ativação</h3>
      </div>
      <p className="mb-3 text-2xs text-ink-dim">
        Regras priorizadas que atribuem este agent a novas conversas quando o lead bate o filtro.
        Menor prioridade ganha (1 = mais alta).
      </p>

      {triggers.isLoading ? (
        <Skeleton className="h-10 w-full" />
      ) : (triggers.data ?? []).length === 0 ? (
        <p className="text-2xs text-ink-dim">Nenhum gatilho configurado.</p>
      ) : (
        <ul className="space-y-1.5">
          {triggers.data!.map((t) => (
            <TriggerRow key={t.id} agentId={agentId} trigger={t} />
          ))}
        </ul>
      )}

      {!showForm ? (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="mt-2"
          onClick={() => setShowForm(true)}
        >
          <Plus className="mr-1 h-3.5 w-3.5" />
          Novo gatilho
        </Button>
      ) : (
        <form className="mt-3 space-y-2 rounded-md bg-elevated/20 p-3" onSubmit={handleCreate}>
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Nome (ex: Leads Meta Ads)"
          />
          <div className="grid grid-cols-2 gap-2">
            <Input
              value={field}
              onChange={(e) => setField(e.target.value)}
              placeholder="Campo (origin, segment, tags, ...)"
            />
            <select
              className="rounded-md bg-elevated/40 px-2 py-2 text-sm text-ink shadow-hairline focus:outline-none focus:ring-1 focus:ring-accent/50"
              value={op}
              onChange={(e) => setOp(e.target.value as typeof op)}
            >
              <option value="eq">eq</option>
              <option value="neq">neq</option>
              <option value="contains">contains</option>
              <option value="in">in (CSV)</option>
              <option value="present">present</option>
              <option value="absent">absent</option>
            </select>
          </div>
          {op !== 'present' && op !== 'absent' && (
            <Input
              value={value}
              onChange={(e) => setValue(e.target.value)}
              placeholder={op === 'in' ? 'valor1, valor2, ...' : 'Valor de comparação'}
            />
          )}
          <div className="flex items-center gap-2">
            <label htmlFor={priorityId} className="text-2xs text-ink-muted">
              Prioridade
            </label>
            <Input
              id={priorityId}
              type="number"
              min={1}
              max={10000}
              value={priority}
              onChange={(e) => setPriority(Number(e.target.value))}
              className="w-24"
            />
            <Button type="submit" variant="primary" size="sm" disabled={create.isPending}>
              Criar
            </Button>
            <Button type="button" variant="ghost" size="sm" onClick={() => setShowForm(false)}>
              Cancelar
            </Button>
          </div>
        </form>
      )}
    </section>
  )
}

function TriggerRow({
  agentId,
  trigger,
}: {
  agentId: string
  trigger: ReturnType<typeof useAgentTriggers>['data'] extends readonly (infer T)[] | undefined ? T : never
}) {
  const update = useUpdateTrigger(agentId, trigger.id)
  const remove = useDeleteTrigger(agentId, trigger.id)

  async function handleToggle() {
    await update.mutateAsync({ is_active: !trigger.is_active })
  }

  async function handleDelete() {
    await remove.mutateAsync()
  }

  const summary = useMemo(() => {
    const parts: string[] = []
    for (const p of trigger.filter.all ?? []) {
      let v = ''
      if (p.value == null) {
        v = ''
      } else if (typeof p.value === 'string') {
        v = p.value
      } else if (typeof p.value === 'number' || typeof p.value === 'boolean') {
        v = String(p.value)
      } else {
        v = JSON.stringify(p.value)
      }
      parts.push(`${p.field} ${p.op}${v ? ` "${v}"` : ''}`)
    }
    if (parts.length === 0) return 'catch-all'
    return parts.join(' AND ')
  }, [trigger.filter])

  return (
    <li className="flex items-center justify-between rounded-md bg-elevated/30 px-3 py-2 text-xs">
      <div className="min-w-0 flex-1 pr-2">
        <div className="flex items-center gap-2">
          <span className="font-medium text-ink">{trigger.name}</span>
          <Badge tone="neutral">p{trigger.priority}</Badge>
          {!trigger.is_active && <Badge tone="neutral">inativo</Badge>}
        </div>
        <div className="mt-0.5 truncate text-2xs text-ink-dim">{summary}</div>
      </div>
      <div className="flex items-center gap-1">
        <Button type="button" variant="ghost" size="sm" onClick={handleToggle} disabled={update.isPending}>
          {trigger.is_active ? 'Desativar' : 'Ativar'}
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={handleDelete}
          disabled={remove.isPending}
          aria-label="Remover gatilho"
        >
          <Trash2 className="h-3.5 w-3.5" />
        </Button>
      </div>
    </li>
  )
}

function PlaygroundSkeleton() {
  return (
    <div className="mx-auto grid h-[calc(100vh-56px)] max-w-[1600px] grid-cols-[minmax(320px,420px)_1fr] gap-6 px-8 py-6">
      <div className="space-y-3">
        <Skeleton className="h-6 w-40" />
        <Skeleton className="h-32 w-full" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
      </div>
      <Skeleton className="h-full w-full" />
    </div>
  )
}
