import { useEffect, useState } from 'react'
import { Check, ChevronDown, Layers, Loader2 } from 'lucide-react'
import { ClayPanel } from './clay/ClayPanel'
import { cn } from '@/lib/utils'
import type { LeadSummary, PipeStageSnapshot } from '@/contracts/manual'

interface PipeSnapshotPanelProps {
  stages: PipeStageSnapshot[]
  lead: LeadSummary | null
  taskId: string | null
}

export function PipeSnapshotPanel({ stages, lead, taskId }: PipeSnapshotPanelProps) {
  const activeFromLead = stages.find((s) => s.isActive)?.id ?? null
  const [activeStageId, setActiveStageId] = useState<string | null>(activeFromLead)
  const [pendingId, setPendingId] = useState<string | null>(null)

  // Reseta stage ativo quando a task ativa muda.
  useEffect(() => {
    setActiveStageId(activeFromLead)
  }, [activeFromLead, taskId])

  async function moveTo(stageId: string) {
    if (!lead || stageId === activeStageId) return
    setPendingId(stageId)
    // TODO: backend — PATCH /leads/:id/stage com otimistic update do WS.
    await new Promise((r) => setTimeout(r, 280))
    setActiveStageId(stageId)
    setPendingId(null)
  }

  return (
    <ClayPanel variant="surface" size="none" className="flex flex-none flex-col p-4">
      <header className="flex items-start justify-between pb-3">
        <div className="flex items-center gap-2">
          <Layers className="h-3.5 w-3.5 text-[hsl(var(--accent))]" strokeWidth={2} aria-hidden />
          <h3 className="text-ink text-[0.8125rem] font-medium">{lead?.pipeName ?? 'Pipe'}</h3>
        </div>
        <span className="text-ink-dim text-[10px] tracking-[0.1em] uppercase">Mover</span>
      </header>

      {!lead ? (
        <p className="text-ink-dim py-6 text-center text-xs">Nenhuma task ativa.</p>
      ) : (
        <ol className="flex flex-col gap-1" role="radiogroup" aria-label="Etapas do kanban">
          {stages.map((s) => {
            const isActive = activeStageId === s.id
            const isPending = pendingId === s.id

            return (
              <li key={s.id}>
                <button
                  type="button"
                  role="radio"
                  aria-checked={isActive}
                  onClick={() => void moveTo(s.id)}
                  disabled={isPending || isActive}
                  className={cn(
                    'group w-full rounded-[var(--clay-radius-sm)] px-2.5 py-2',
                    'flex items-center gap-2 text-left',
                    'transition-[background,box-shadow,transform]',
                    'focus-visible:shadow-[var(--clay-ring-focus)] focus-visible:outline-none',
                    'disabled:cursor-default',
                    isActive
                      ? 'bg-[hsl(var(--clay-panel-up))] shadow-[var(--clay-shadow-accent)]'
                      : 'bg-transparent hover:-translate-y-[0.5px] hover:bg-[hsl(var(--clay-panel-up)/0.6)] active:translate-y-[0.5px] active:shadow-[var(--clay-shadow-sunken)]'
                  )}
                >
                  <span
                    aria-hidden
                    className={cn(
                      'h-1.5 w-1.5 shrink-0 rounded-full',
                      isActive
                        ? 'bg-[hsl(var(--accent))] shadow-[0_0_8px_0_hsl(var(--accent)/0.8)]'
                        : 'bg-[hsl(var(--clay-rim))] group-hover:bg-[hsl(var(--ink-dim))]'
                    )}
                  />
                  <span
                    className={cn(
                      'min-w-0 flex-1 truncate text-[12px]',
                      isActive ? 'text-ink' : 'text-ink-muted'
                    )}
                  >
                    {s.name}
                  </span>
                  <span
                    className={cn(
                      'font-mono text-[10px] tabular-nums',
                      isActive ? 'text-[hsl(var(--accent))]' : 'text-ink-dim'
                    )}
                  >
                    {s.leadCount}
                  </span>
                  {isPending ? (
                    <Loader2
                      className="h-3 w-3 shrink-0 animate-spin text-[hsl(var(--accent))]"
                      strokeWidth={2}
                      aria-label="Movendo"
                    />
                  ) : isActive ? (
                    <Check
                      className="h-3 w-3 shrink-0 text-[hsl(var(--accent))]"
                      strokeWidth={2.5}
                      aria-hidden
                    />
                  ) : (
                    <ChevronDown
                      className="text-ink-dim h-3 w-3 shrink-0 -rotate-90 opacity-0 transition-opacity group-hover:opacity-100"
                      strokeWidth={1.75}
                      aria-hidden
                    />
                  )}
                </button>
              </li>
            )
          })}
        </ol>
      )}

      {lead && (
        <p className="text-ink-dim mt-3 border-t border-[hsl(var(--clay-rim)/0.35)] pt-3 text-[11px] leading-relaxed">
          <span className="text-ink">{lead.name.split(' · ')[0]}</span> em{' '}
          <span className="text-ink">
            {stages.find((s) => s.id === activeStageId)?.name ?? lead.stageName}
          </span>
          .
        </p>
      )}
    </ClayPanel>
  )
}
