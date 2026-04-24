/**
 * MessageComposer — footer do ConversationPane (S35).
 *
 * Textarea com auto-grow + envio via Enter (Shift+Enter quebra linha).
 * Template picker como popover simples consumindo useMessageTemplates.
 * A resolução de variáveis é client-side: o usuário vê o texto final
 * na textarea antes de mandar (WYSIWYG).
 *
 * Comparado com v8 ChatWhatsApp.tsx onde o compose ocupa ~400 LOC, este
 * fica em ~230 — só o essencial: texto, template, envio. Anexos + emoji
 * + áudio nativo ficam para uma iteração de polish (TODO explícito).
 */

import { FileText, Send } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'

import { Button } from '@/ui/button'
import {
  renderTemplate,
  useMessageTemplates,
  type MessageTemplate,
} from '@/hooks/useMessageTemplates'
import { useSendMessage } from '@/hooks/useInbox'

export interface MessageComposerProps {
  conversationId: string
  /** Nome/empresa do contato para pré-preencher variáveis do template. */
  contactName?: string | null
  /** Desabilita composer quando conversa está em estado resolvido/arquivado. */
  disabled?: boolean
}

export function MessageComposer({ conversationId, contactName, disabled }: MessageComposerProps) {
  const [body, setBody] = useState('')
  const [showTemplates, setShowTemplates] = useState(false)
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const send = useSendMessage(conversationId)

  // Auto-resize textarea até ~8 linhas. Além disso, scroll interno.
  useEffect(() => {
    const el = textareaRef.current
    if (!el) return
    el.style.height = 'auto'
    el.style.height = `${Math.min(el.scrollHeight, 192)}px`
  }, [body])

  const canSend = body.trim().length > 0 && !send.isPending && !disabled

  async function handleSend() {
    if (!canSend) return
    const text = body.trim()
    setBody('')
    try {
      await send.mutateAsync({ kind: 'text', body: text })
    } catch {
      // useAppMutation already surfaced the error via toast; restore draft.
      setBody(text)
    }
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      void handleSend()
    }
  }

  function handleTemplate(t: MessageTemplate) {
    const resolved = renderTemplate(t.body, {
      nome: contactName ?? '',
      contato: contactName ?? '',
    })
    setBody(resolved)
    setShowTemplates(false)
    requestAnimationFrame(() => textareaRef.current?.focus())
  }

  return (
    <div className="relative shrink-0 border-t border-hairline px-6 py-3">
      {showTemplates && (
        <TemplatePicker onPick={handleTemplate} onDismiss={() => setShowTemplates(false)} />
      )}

      <div className="flex items-end gap-2">
        <Button
          variant="ghost"
          size="sm"
          type="button"
          aria-label="Inserir template"
          disabled={disabled}
          onClick={() => setShowTemplates((v) => !v)}
        >
          <FileText className="h-4 w-4" />
        </Button>

        <textarea
          ref={textareaRef}
          value={body}
          onChange={(e) => setBody(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={disabled}
          placeholder={
            disabled
              ? 'Conversa encerrada'
              : 'Digite uma mensagem… (Enter envia, Shift+Enter quebra linha)'
          }
          rows={1}
          className="flex-1 resize-none rounded-md bg-elevated/40 px-3 py-2 text-sm text-ink shadow-hairline placeholder:text-ink-dim focus:outline-none focus:ring-1 focus:ring-accent/50"
        />

        <Button
          variant="primary"
          size="sm"
          type="button"
          disabled={!canSend}
          onClick={() => void handleSend()}
        >
          {send.isPending ? (
            'Enviando…'
          ) : (
            <>
              Enviar <Send className="ml-1.5 h-3.5 w-3.5" />
            </>
          )}
        </Button>
      </div>
    </div>
  )
}

function TemplatePicker({
  onPick,
  onDismiss,
}: {
  onPick: (t: MessageTemplate) => void
  onDismiss: () => void
}) {
  const query = useMessageTemplates(true)

  return (
    <div
      role="dialog"
      aria-label="Escolher template"
      className="absolute bottom-full left-6 mb-2 w-80 rounded-lg bg-surface p-2 shadow-elev-2"
    >
      <div className="flex items-center justify-between px-2 py-1">
        <span className="text-xs text-ink-muted">Templates</span>
        <Button variant="ghost" size="xs" onClick={onDismiss}>
          Fechar
        </Button>
      </div>
      <ul className="max-h-72 overflow-y-auto">
        {query.isLoading && <li className="px-3 py-2 text-xs text-ink-dim">Carregando…</li>}
        {query.isSuccess && query.data.length === 0 && (
          <li className="px-3 py-4 text-center text-xs text-ink-dim">
            Nenhum template ativo. Admins podem criar em Configurações.
          </li>
        )}
        {query.isSuccess &&
          query.data.map((t) => (
            <li key={t.id}>
              <button
                type="button"
                onClick={() => onPick(t)}
                className="flex w-full flex-col items-start rounded-md px-3 py-2 text-left hover:bg-elevated/40"
              >
                <span className="text-sm text-ink">{t.name}</span>
                <span className="line-clamp-2 text-xs text-ink-dim">{t.body}</span>
              </button>
            </li>
          ))}
      </ul>
    </div>
  )
}
