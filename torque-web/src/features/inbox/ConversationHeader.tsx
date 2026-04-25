/**
 * Inbox — ConversationHeader (Fase B / S33).
 *
 * Fica no topo do ConversationPane: identifica quem está do outro lado,
 * mostra o canal e expõe 3 ações rápidas (Assumir, Resolver, Reabrir).
 *
 * Optou-se por *não* fazer chat-UI aqui — o MessageList + ComposerBar
 * entregam a conversa em si (S34/S35). Este componente é só o shell
 * cabeçalho; mantém <200 LOC para ficar auditável.
 */

import { CheckCircle2, RotateCcw, UserPlus2 } from 'lucide-react'

import { Avatar } from '@/ui/avatar'
import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { ChannelBadge } from '@/ui/channel-badge'
import { initials } from '@/lib/utils'
import type { Conversation } from '@/hooks/useInbox'
import { useAssignConversation, useSetConversationState } from '@/hooks/useInbox'

export interface ConversationHeaderProps {
  conversation: Conversation
  /**
   * Current caller's team_member id. When set we can show "Assigned to
   * me" state and offer the takeover button conditionally.
   */
  currentMemberId?: string
}

export function ConversationHeader({ conversation, currentMemberId }: ConversationHeaderProps) {
  const assign = useAssignConversation(conversation.id)
  const setState = useSetConversationState(conversation.id)

  const name = conversation.contact_name ?? conversation.contact_handle ?? 'Sem nome'
  const isMine = currentMemberId != null && conversation.assigned_to === currentMemberId
  const isResolved = conversation.state === 'resolved'

  return (
    <header className="flex items-center gap-4 px-6 py-3 shadow-[inset_0_-1px_0_0_hsl(var(--hairline))]">
      <Avatar size="md" fallback={initials(name)} />

      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="text-ink truncate text-sm">{name}</span>
          <ChannelBadge
            channel={channelOf(conversation.channel_kind)}
            variant="icon-only"
            size="sm"
          />
          {isMine && <Badge tone="accent">Minha</Badge>}
          {isResolved && <Badge tone="success">Resolvida</Badge>}
        </div>
        {conversation.contact_handle && conversation.contact_name && (
          <p className="font-metric text-2xs text-ink-dim truncate">
            {conversation.contact_handle}
          </p>
        )}
      </div>

      <div className="flex items-center gap-1.5">
        {!isMine && !isResolved && (
          <Button
            size="sm"
            variant="outline"
            disabled={assign.isPending}
            onClick={() =>
              currentMemberId && void assign.mutateAsync({ assigned_to: currentMemberId })
            }
          >
            <UserPlus2 className="mr-1.5 h-3.5 w-3.5" />
            Assumir
          </Button>
        )}
        {!isResolved ? (
          <Button
            size="sm"
            variant="outline"
            disabled={setState.isPending}
            onClick={() => void setState.mutateAsync({ state: 'resolved' })}
          >
            <CheckCircle2 className="mr-1.5 h-3.5 w-3.5" />
            Resolver
          </Button>
        ) : (
          <Button
            size="sm"
            variant="outline"
            disabled={setState.isPending}
            onClick={() => void setState.mutateAsync({ state: 'open' })}
          >
            <RotateCcw className="mr-1.5 h-3.5 w-3.5" />
            Reabrir
          </Button>
        )}
      </div>
    </header>
  )
}

function channelOf(kind: string): 'whatsapp' | 'messenger' | 'instagram' | 'sz_chat' {
  if (kind === 'whatsapp' || kind === 'messenger' || kind === 'instagram' || kind === 'sz_chat') {
    return kind
  }
  return 'sz_chat'
}
