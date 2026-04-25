import { useEffect, useRef, useState } from 'react'
import {
  ArrowUpRight,
  Check,
  ExternalLink,
  Flame,
  Pause,
  Play,
  Sparkles,
  Timer,
} from 'lucide-react'
import { ClayPanel } from './clay/ClayPanel'
import { ClayButton } from './clay/ClayButton'
import { ClayChip } from './clay/ClayChip'
import { cn } from '@/lib/utils'
import type { LeadSummary, Task } from '@/contracts/manual'
import { useTaskActions } from '@/hooks/useTaskActions'
import {
  KIND_ICON,
  KIND_LABEL,
  PRIORITY_LABEL,
  formatDueRelative,
  formatElapsedSince,
  priorityTone,
} from './utils'

interface ActiveTaskPanelProps {
  task: Task | null
  lead: LeadSummary | null
  hasQueue: boolean
  firstQueuedId: string | null
}

export function ActiveTaskPanel({ task, lead, hasQueue, firstQueuedId }: ActiveTaskPanelProps) {
  const { completeTask, pauseTask, startTask } = useTaskActions()
  const [note, setNote] = useState('')
  const textareaRef = useRef<HTMLTextAreaElement | null>(null)

  // Reseta nota quando muda a task ativa.
  useEffect(() => {
    setNote('')
  }, [task?.id])

  if (!task || !lead) {
    return <EmptyActive hasQueue={hasQueue} firstQueuedId={firstQueuedId} onStart={startTask} />
  }

  const Icon = KIND_ICON[task.kind]
  const due = formatDueRelative(task.dueAt)

  return (
    <ClayPanel variant="floating" size="none" className="flex h-full min-h-0 flex-col p-8">
      {/* Header */}
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-center gap-2.5">
          <div className="flex h-9 w-9 items-center justify-center rounded-full bg-[hsl(var(--accent)/0.14)] text-[hsl(var(--accent))] shadow-[inset_0_1px_0_0_hsl(var(--accent)/0.4),_inset_0_-1px_0_0_hsl(0_0%_0%/0.3)]">
            <Icon className="h-4 w-4" strokeWidth={2} />
          </div>
          <div>
            <div className="text-ink-dim text-[10px] tracking-[0.14em] uppercase">
              {KIND_LABEL[task.kind]}
            </div>
            <Chronometer startedAt={task.startedAt} />
          </div>
        </div>

        <div className="flex items-center gap-1.5">
          {task.priority === 'urgent' && (
            <ClayChip tone="danger" size="xs">
              <Flame className="h-3 w-3" strokeWidth={2.2} />
              Urgente
            </ClayChip>
          )}
          {task.priority !== 'urgent' && (
            <ClayChip tone={priorityTone(task.priority)} size="xs">
              {PRIORITY_LABEL[task.priority]}
            </ClayChip>
          )}
        </div>
      </div>

      {/* Title + description */}
      <div className="mt-6">
        <h1 className="font-display tracking-tightest text-ink text-[1.875rem] leading-[1.12]">
          {task.title}
        </h1>
        {task.description && (
          <p className="text-ink-muted mt-3 max-w-prose text-[0.9375rem] leading-relaxed">
            {task.description}
          </p>
        )}
      </div>

      {/* Context chips */}
      <div className="mt-5 flex flex-wrap items-center gap-1.5">
        <button
          type="button"
          className="text-ink inline-flex items-center gap-1.5 rounded-full bg-[hsl(var(--clay-panel-down))] px-3 py-1.5 text-xs shadow-[var(--clay-shadow-sunken)] transition-colors hover:text-[hsl(var(--accent))]"
          aria-label={`Abrir lead ${lead.name}`}
        >
          <span className="inline-block h-1.5 w-1.5 rounded-full bg-[hsl(var(--channel-whatsapp))]" />
          <span className="max-w-[220px] truncate">{lead.name}</span>
          <ExternalLink className="text-ink-dim h-3 w-3" strokeWidth={1.75} />
        </button>

        <ClayChip tone="info" size="sm">
          {lead.pipeName} · {lead.stageName}
        </ClayChip>

        <ClayChip
          tone={due.tone === 'past' ? 'danger' : due.tone === 'urgent' ? 'warning' : 'neutral'}
          size="sm"
        >
          <Timer className="h-3 w-3" strokeWidth={2} />
          {due.label}
        </ClayChip>

        <HeatBadge heat={lead.heat} />
      </div>

      {/* Spacer */}
      <div className="flex-1" />

      {/* Result note + actions */}
      <div className="mt-6 space-y-3">
        <label className="block">
          <span className="text-ink-dim mb-2 block text-[11px] tracking-[0.12em] uppercase">
            Resultado (opcional)
          </span>
          <textarea
            ref={textareaRef}
            value={note}
            onChange={(e) => setNote(e.target.value)}
            rows={2}
            placeholder="Anote o desfecho, próximo passo, ou deixe vazio."
            className={cn(
              'w-full resize-none rounded-[var(--clay-radius-sm)] bg-[hsl(var(--clay-panel-down))] px-3.5 py-2.5',
              'text-ink placeholder:text-ink-dim text-sm',
              'shadow-[var(--clay-shadow-sunken)]',
              'transition-shadow focus:shadow-[var(--clay-ring-focus)] focus:outline-none'
            )}
          />
        </label>

        <div className="flex items-center gap-2">
          <ClayButton
            variant="primary"
            size="lg"
            onClick={() => completeTask(task.id, note.trim() || undefined)}
            className="flex-1"
          >
            <Check className="h-4 w-4" strokeWidth={2.5} />
            Concluir
          </ClayButton>
          <ClayButton
            variant="secondary"
            size="lg"
            onClick={() => pauseTask(task.id)}
            aria-label="Pausar task"
          >
            <Pause className="h-4 w-4" strokeWidth={2} />
            Pausar
          </ClayButton>
          <ClayButton variant="ghost" size="lg" aria-label="Abrir lead completo">
            <ArrowUpRight className="h-4 w-4" strokeWidth={2} />
            Lead
          </ClayButton>
        </div>
      </div>
    </ClayPanel>
  )
}

function Chronometer({ startedAt }: { startedAt: string | null }) {
  const [, setTick] = useState(0)
  useEffect(() => {
    const id = setInterval(() => setTick((t) => t + 1), 1000)
    return () => clearInterval(id)
  }, [])
  if (!startedAt) return null
  return (
    <div className="text-ink mt-0.5 font-mono text-[0.8125rem] tabular-nums">
      {formatElapsedSince(startedAt)}
    </div>
  )
}

function HeatBadge({ heat }: { heat: 1 | 2 | 3 | 4 | 5 }) {
  return (
    <span
      className="inline-flex h-7 items-center gap-1 rounded-full bg-[hsl(var(--clay-panel-down))] px-2.5 shadow-[var(--clay-shadow-sunken)]"
      aria-label={`Calor ${heat} de 5`}
    >
      <span className="text-ink-dim text-[10px] tracking-[0.1em] uppercase">Calor</span>
      <span className="flex gap-0.5">
        {[1, 2, 3, 4, 5].map((n) => (
          <span
            key={n}
            className={cn(
              'h-1.5 w-1.5 rounded-full',
              n <= heat ? `bg-[hsl(var(--heat-${heat}))]` : 'bg-[hsl(var(--clay-rim)/0.3)]'
            )}
          />
        ))}
      </span>
    </span>
  )
}

function EmptyActive({
  hasQueue,
  firstQueuedId,
  onStart,
}: {
  hasQueue: boolean
  firstQueuedId: string | null
  onStart: (id: string) => void
}) {
  return (
    <ClayPanel
      variant="sunken"
      size="none"
      className="flex h-full min-h-0 flex-col items-center justify-center p-10"
    >
      <div className="flex h-16 w-16 items-center justify-center rounded-full bg-[hsl(var(--clay-panel))] shadow-[var(--clay-shadow-raised)]">
        <Sparkles className="h-6 w-6 text-[hsl(var(--accent))]" strokeWidth={1.6} />
      </div>
      <h2 className="font-display tracking-tightest text-ink mt-6 text-[1.5rem]">Respire fundo.</h2>
      <p className="text-ink-muted mt-2 max-w-[320px] text-center text-sm">
        {hasQueue
          ? 'Sua fila está pronta. Inicie a próxima quando estiver.'
          : 'Nenhuma task na fila. Adicione do backlog ou aguarde atribuições.'}
      </p>
      {hasQueue && firstQueuedId && (
        <ClayButton
          variant="primary"
          size="lg"
          className="mt-7"
          onClick={() => onStart(firstQueuedId)}
        >
          <Play className="h-4 w-4" strokeWidth={2.5} />
          Iniciar próxima
        </ClayButton>
      )}
    </ClayPanel>
  )
}
