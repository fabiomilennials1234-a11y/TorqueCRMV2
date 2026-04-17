import { MessageSquare, StickyNote, CheckCircle2 } from "lucide-react";
import { Avatar } from "@/ui/avatar";
import { Badge } from "@/ui/badge";
import { ScoreMeter } from "@/ui/score-meter";
import { formatRelative, cn } from "@/lib/utils";
import type { Lead } from "@/lib/seed";

export function LeadCard({ lead, onClick }: { lead: Lead; onClick: () => void }) {
  return (
    <article
      onClick={onClick}
      className={cn(
        "group relative cursor-pointer rounded-xl p-3",
        "clay-surface shadow-clay-1",
        "transition-all duration-200 ease-[var(--ease-out-soft)]",
        "hover:shadow-clay-2 hover:-translate-y-px",
        "active:shadow-clay-pressed active:translate-y-0 active:scale-[0.985]",
      )}
    >
      {/* Left accent bar appears on hot leads */}
      {lead.score >= 80 && (
        <span className="absolute left-0 top-2 bottom-2 w-[2px] rounded-full bg-accent" />
      )}

      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            {lead.unread && lead.unread > 0 && (
              <span className="relative flex h-1.5 w-1.5 shrink-0">
                <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-accent opacity-60" />
                <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-accent" />
              </span>
            )}
            <h4 className="truncate text-[0.875rem] leading-tight font-medium text-ink">
              {lead.name}
            </h4>
          </div>
          <div className="mt-0.5 truncate text-2xs uppercase tracking-[0.1em] text-ink-dim">
            {lead.company}
          </div>
        </div>
        <ScoreMeter value={lead.score} size={32} strokeWidth={2} />
      </div>

      {/* Value */}
      <div className="mt-2 flex items-baseline gap-1">
        <span className="text-2xs text-ink-dim">R$</span>
        <span className="font-metric text-[0.9375rem] tabular-nums text-ink">
          {lead.value.toLocaleString("pt-BR", { maximumFractionDigits: 0 })}
        </span>
      </div>

      {/* Tags */}
      {lead.tags.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-1">
          {lead.tags.slice(0, 2).map((t) => (
            <Badge
              key={t}
              tone={t === "Hot" ? "accent" : t === "Decisor" ? "info" : "neutral"}
            >
              {t}
            </Badge>
          ))}
          {lead.tags.length > 2 && (
            <Badge tone="neutral">+{lead.tags.length - 2}</Badge>
          )}
        </div>
      )}

      {/* Footer */}
      <div className="mt-3 flex items-center justify-between gap-2 pt-2 shadow-[inset_0_1px_0_0_hsl(var(--hairline))]">
        <div className="flex items-center gap-2 text-2xs text-ink-dim">
          {lead.notesCount ? (
            <span className="inline-flex items-center gap-1">
              <StickyNote className="h-3 w-3" />
              {lead.notesCount}
            </span>
          ) : null}
          {lead.tasksDue ? (
            <span className="inline-flex items-center gap-1 text-warning">
              <CheckCircle2 className="h-3 w-3" />
              {lead.tasksDue}
            </span>
          ) : null}
          {lead.unread ? (
            <span className="inline-flex items-center gap-1 text-accent">
              <MessageSquare className="h-3 w-3" />
              {lead.unread}
            </span>
          ) : null}
          <span className="font-metric tabular-nums">
            {formatRelative(lead.lastTouch)}
          </span>
        </div>
        <Avatar size="sm" fallback={lead.owner.initials} />
      </div>
    </article>
  );
}
