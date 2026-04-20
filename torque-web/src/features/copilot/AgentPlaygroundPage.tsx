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

import { ArrowLeft, Book, Plus, RotateCcw, Send, Square } from 'lucide-react'
import { useEffect, useId, useMemo, useRef, useState } from 'react'
import { Link, Navigate, useParams } from 'react-router-dom'

import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { Input } from '@/ui/input'
import { PageHeader } from '@/ui/page-header'
import { Skeleton } from '@/ui/skeleton'
import { friendlyMessage } from '@/api/errors'
import {
  useAgent,
  useBindAgentCollection,
  useCreateCollection,
  useEnqueueSource,
  useKnowledgeCollections,
  useKnowledgeSources,
  useUpdateAgent,
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
      draft.max_output_tokens !== agent.max_output_tokens
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

        <KnowledgePanel agentId={agentId} collectionId={agent.knowledge_collection_id ?? null} />
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
