import { MessageSquare, StickyNote, CheckCircle2 } from 'lucide-react'
import { Avatar } from '@/ui/avatar'
import { Badge } from '@/ui/badge'
import { ScoreMeter } from '@/ui/score-meter'
import { formatRelative, cn } from '@/lib/utils'
import type { Lead } from '@/lib/seed'

export function LeadCard({ lead, onClick }: { lead: Lead; onClick: () => void }) {
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onClick()
        }
      }}
      className={cn(
        'group relative cursor-pointer rounded-xl p-3',
        'tactile-surface shadow-clay-1',
        'transition-all duration-200 ease-[var(--ease-out-soft)]',
        'hover:shadow-clay-2 hover:-translate-y-px',
        'active:shadow-clay-pressed active:translate-y-0 active:scale-[0.985]',
        'focus-visible:outline-none'
      )}
    >
      {/* Left accent bar appears on hot leads */}
      {lead.score >= 80 && (
        <span className="bg-accent absolute top-2 bottom-2 left-0 w-[2px] rounded-full" />
      )}

      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            {lead.unread && lead.unread > 0 && (
              <span className="relative flex h-1.5 w-1.5 shrink-0">
                <span className="bg-accent absolute inline-flex h-full w-full animate-ping rounded-full opacity-60" />
                <span className="bg-accent relative inline-flex h-1.5 w-1.5 rounded-full" />
              </span>
            )}
            <h4 className="text-ink truncate text-[0.875rem] leading-tight font-medium">
              {lead.name}
            </h4>
          </div>
          <div className="text-2xs text-ink-dim mt-0.5 truncate tracking-[0.1em] uppercase">
            {lead.company}
          </div>
        </div>
        <ScoreMeter value={lead.score} size={32} strokeWidth={2} />
      </div>

      {/* Value */}
      <div className="mt-2 flex items-baseline gap-1">
        <span className="text-2xs text-ink-dim">R$</span>
        <span className="font-metric text-ink text-[0.9375rem] tabular-nums">
          {lead.value.toLocaleString('pt-BR', { maximumFractionDigits: 0 })}
        </span>
      </div>

      {/* Tags */}
      {lead.tags.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-1">
          {lead.tags.slice(0, 2).map((t) => (
            <Badge key={t} tone={t === 'Hot' ? 'accent' : t === 'Decisor' ? 'info' : 'neutral'}>
              {t}
            </Badge>
          ))}
          {lead.tags.length > 2 && <Badge tone="neutral">+{lead.tags.length - 2}</Badge>}
        </div>
      )}

      {/* Footer */}
      <div className="mt-3 flex items-center justify-between gap-2 pt-2 shadow-[inset_0_1px_0_0_hsl(var(--hairline))]">
        <div className="text-2xs text-ink-dim flex items-center gap-2">
          {lead.notesCount ? (
            <span className="inline-flex items-center gap-1">
              <StickyNote className="h-3 w-3" />
              {lead.notesCount}
            </span>
          ) : null}
          {lead.tasksDue ? (
            <span className="text-warning inline-flex items-center gap-1">
              <CheckCircle2 className="h-3 w-3" />
              {lead.tasksDue}
            </span>
          ) : null}
          {lead.unread ? (
            <span className="text-accent inline-flex items-center gap-1">
              <MessageSquare className="h-3 w-3" />
              {lead.unread}
            </span>
          ) : null}
          <span className="font-metric tabular-nums">{formatRelative(lead.lastTouch)}</span>
        </div>
        <Avatar size="sm" fallback={lead.owner.initials} />
      </div>
    </div>
  )
}
