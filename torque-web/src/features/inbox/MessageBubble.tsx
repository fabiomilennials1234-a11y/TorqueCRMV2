/**
 * MessageBubble — primitive de uma mensagem no fluxo (S34).
 *
 * Renderiza conforme `kind` (text, image, audio, video, document, sticker,
 * system). Direction (inbound/outbound) troca alinhamento + cor de fundo.
 * Status icon só aparece no outbound (usuário não precisa ver "entregue"
 * em mensagens recebidas).
 *
 * Design choice: evita dependência de componentes heavy (react-player,
 * react-audio). Image usa <img>; audio usa <audio>; document é link
 * com ícone. Suficiente para paridade v8 sem carregar 200 KB de libs.
 */

import { Check, CheckCheck, Clock, TriangleAlert } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

import { cn, formatRelative } from '@/lib/utils'
import type { Message, MessageDirection, MessageStatus } from '@/hooks/useInbox'

const STATUS_ICONS: Record<MessageStatus, LucideIcon> = {
  queued: Clock,
  sent: Check,
  delivered: CheckCheck,
  read: CheckCheck,
  failed: TriangleAlert,
}

const STATUS_LABELS: Record<MessageStatus, string> = {
  queued: 'Enviando…',
  sent: 'Enviada',
  delivered: 'Entregue',
  read: 'Lida',
  failed: 'Falha ao enviar',
}

export interface MessageBubbleProps {
  message: Message
}

export function MessageBubble({ message }: MessageBubbleProps) {
  const m = message
  const isOutbound = m.direction === 'outbound'
  const isSystem = m.kind === 'system'

  // System messages sit centered with a muted tone — not a bubble.
  if (isSystem) {
    return (
      <div className="my-2 flex justify-center">
        <span className="rounded-full bg-elevated/40 px-3 py-1 text-2xs text-ink-dim">
          {m.body ?? 'Evento do sistema'}
        </span>
      </div>
    )
  }

  return (
    <div className={cn('flex w-full', isOutbound ? 'justify-end' : 'justify-start')}>
      <div
        className={cn(
          'max-w-[72%] rounded-lg px-3 py-2 text-sm shadow-elev-1',
          isOutbound ? 'bg-accent/15 text-ink' : 'bg-surface text-ink-muted'
        )}
      >
        <MessageBody message={m} />
        <MessageFooter direction={m.direction} status={m.status} occurredAt={m.occurred_at} />
      </div>
    </div>
  )
}

function MessageBody({ message }: { message: Message }) {
  const m = message
  switch (m.kind) {
    case 'image':
      return (
        <figure>
          {m.media_url ? (
            <img
              src={m.media_url}
              alt={m.body ?? 'Imagem recebida'}
              className="mb-1 max-h-72 w-full rounded-md object-cover"
              loading="lazy"
            />
          ) : (
            <div className="mb-1 h-24 w-48 rounded-md bg-elevated/40" />
          )}
          {m.body && <figcaption className="text-xs text-ink-muted">{m.body}</figcaption>}
        </figure>
      )
    case 'audio':
      return m.media_url ? (
        <audio controls src={m.media_url} className="max-w-[240px]">
          <track kind="captions" />
        </audio>
      ) : (
        <span className="text-xs text-ink-dim">Áudio indisponível.</span>
      )
    case 'video':
      return m.media_url ? (
        <video controls src={m.media_url} className="max-h-72 max-w-full rounded-md">
          <track kind="captions" />
        </video>
      ) : (
        <span className="text-xs text-ink-dim">Vídeo indisponível.</span>
      )
    case 'document':
      return (
        <a
          href={m.media_url ?? '#'}
          target="_blank"
          rel="noopener noreferrer"
          className="text-sm text-accent underline"
        >
          {m.body ?? 'Documento'}
        </a>
      )
    case 'sticker':
      return m.media_url ? (
        <img src={m.media_url} alt="Sticker" className="h-24 w-24 object-contain" />
      ) : (
        <span className="text-xs text-ink-dim">Sticker indisponível.</span>
      )
    case 'text':
    default:
      return <p className="whitespace-pre-wrap break-words">{m.body ?? ''}</p>
  }
}

function MessageFooter({
  direction,
  status,
  occurredAt,
}: {
  direction: MessageDirection
  status: MessageStatus
  occurredAt: string
}) {
  const StatusIcon = STATUS_ICONS[status]
  const isOutbound = direction === 'outbound'

  return (
    <div className="mt-1 flex items-center justify-end gap-1 text-2xs text-ink-dim">
      <time dateTime={occurredAt}>{formatRelative(new Date(occurredAt))}</time>
      {isOutbound && (
        <span
          aria-label={STATUS_LABELS[status]}
          className={cn(
            'inline-flex items-center',
            status === 'read' ? 'text-info' : '',
            status === 'failed' ? 'text-danger' : ''
          )}
        >
          <StatusIcon className="h-3 w-3" />
        </span>
      )}
    </div>
  )
}
