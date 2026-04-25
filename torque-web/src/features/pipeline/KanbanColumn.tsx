import { MoreHorizontal, Plus } from 'lucide-react'
import { Button } from '@/ui/button'
import { LeadCard } from './LeadCard'
import type { Lead, Stage } from '@/lib/seed'

export function KanbanColumn({
  stage,
  meta,
  leads,
  onCardClick,
}: {
  stage: Stage
  meta: { label: string; color: string; order: number }
  leads: Lead[]
  onCardClick: (l: Lead) => void
}) {
  const total = leads.reduce((s, l) => s + l.value, 0)
  const isTerminal = stage === 'vendido' || stage === 'perdido'

  return (
    <div className="tactile-surface shadow-clay-1 flex h-full w-[308px] shrink-0 flex-col rounded-xl backdrop-blur-sm">
      {/* Column header */}
      <div className="flex shrink-0 items-center gap-2 px-3 pt-3 pb-2">
        <span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: meta.color }} />
        <div className="text-ink text-[0.8125rem] font-medium">{meta.label}</div>
        <span className="font-metric text-2xs text-ink-dim tabular-nums">{leads.length}</span>
        <div className="ml-auto flex items-center gap-0.5">
          <Button variant="ghost" size="icon" className="h-6 w-6">
            <Plus className="h-3.5 w-3.5" />
          </Button>
          <Button variant="ghost" size="icon" className="h-6 w-6">
            <MoreHorizontal className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>

      <div className="shadow-hairline-b shrink-0 px-3 pb-2">
        <div className="font-metric text-2xs text-ink-muted tabular-nums">
          {isTerminal ? (stage === 'vendido' ? 'Receita · ' : 'Perdido · ') : 'Valor · '}
          R$ {total.toLocaleString('pt-BR', { maximumFractionDigits: 0 })}
        </div>
      </div>

      {/* Cards */}
      <div className="min-h-0 flex-1 space-y-2 overflow-y-auto p-2">
        {leads.length === 0 ? (
          <div className="text-2xs text-ink-dim flex h-24 items-center justify-center rounded-md tracking-[0.12em] uppercase shadow-[inset_0_0_0_1px_hsl(var(--hairline))]">
            vazio
          </div>
        ) : (
          leads.map((l) => <LeadCard key={l.id} lead={l} onClick={() => onCardClick(l)} />)
        )}
      </div>
    </div>
  )
}
