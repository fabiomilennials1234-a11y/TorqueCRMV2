import { useMemo, useState } from 'react'
import { AlertTriangle, ArrowLeftCircle } from 'lucide-react'
import { ClayPanel } from './clay/ClayPanel'
import { ClayChip } from './clay/ClayChip'
import { cn } from '@/lib/utils'
import type { Task } from '@/contracts/manual'
import { useTaskActions } from '@/hooks/useTaskActions'
import { KIND_ICON, formatDueRelative, priorityTone } from './utils'

interface BacklogPanelProps {
  backlog: Task[]
  missed: Task[]
}

type Filter = 'all' | 'today' | 'overdue' | 'high'

const FILTER_LABEL: Record<Filter, string> = {
  all: 'Todas',
  today: 'Hoje',
  overdue: 'Atrasadas',
  high: 'Alta',
}

function isToday(iso: string | null): boolean {
  if (!iso) return false
  const a = new Date(iso)
  const b = new Date()
  return (
    a.getDate() === b.getDate() &&
    a.getMonth() === b.getMonth() &&
    a.getFullYear() === b.getFullYear()
  )
}

function isOverdue(iso: string | null): boolean {
  if (!iso) return false
  return new Date(iso).getTime() < Date.now()
}

export function BacklogPanel({ backlog, missed }: BacklogPanelProps) {
  const { enqueueTask, reopenMissed } = useTaskActions()
  const [filter, setFilter] = useState<Filter>('all')

  const filtered = useMemo(() => {
    switch (filter) {
      case 'today':
        return backlog.filter((t) => isToday(t.dueAt))
      case 'overdue':
        return backlog.filter((t) => isOverdue(t.dueAt))
      case 'high':
        return backlog.filter((t) => t.priority === 'high' || t.priority === 'urgent')
      default:
        return backlog
    }
  }, [backlog, filter])

  return (
    <ClayPanel variant="surface" size="none" className="flex min-h-0 flex-1 flex-col p-4">
      <header className="flex items-center justify-between pb-3">
        <div className="flex items-baseline gap-2">
          <h3 className="text-[0.8125rem] font-medium text-ink">Tasks em aberto</h3>
          <span className="font-mono text-[10px] tabular-nums text-ink-dim">{backlog.length}</span>
        </div>
      </header>

      <div className="flex items-center gap-1 pb-3">
        {(Object.keys(FILTER_LABEL) as Filter[]).map((f) => (
          <button
            key={f}
            type="button"
            onClick={() => setFilter(f)}
            className={cn(
              'h-6 rounded-full px-2.5 text-[10px] uppercase tracking-[0.08em] transition-colors',
              filter === f
                ? 'bg-[hsl(var(--clay-panel-up))] text-ink shadow-[var(--clay-shadow-raised)]'
                : 'text-ink-dim hover:text-ink'
            )}
          >
            {FILTER_LABEL[f]}
          </button>
        ))}
      </div>

      <div className="flex min-h-0 flex-1 flex-col gap-1.5 overflow-y-auto pr-1">
        {missed.length > 0 && (
          <div className="mb-2">
            <div className="mb-1.5 flex items-center gap-1.5 px-1 text-[10px] uppercase tracking-[0.1em] text-[hsl(var(--danger))]">
              <AlertTriangle className="h-3 w-3" strokeWidth={2} />
              Atrasadas perdidas · {missed.length}
            </div>
            {missed.map((t) => (
              <BacklogRow
                key={t.id}
                task={t}
                variant="missed"
                onEnqueue={() => reopenMissed(t.id)}
              />
            ))}
          </div>
        )}

        {filtered.length === 0 ? (
          <p className="px-1 py-6 text-center text-xs text-ink-dim">Nada por aqui. Respire.</p>
        ) : (
          filtered.map((t) => (
            <BacklogRow key={t.id} task={t} onEnqueue={() => enqueueTask(t.id)} />
          ))
        )}
      </div>
    </ClayPanel>
  )
}

function BacklogRow({
  task,
  onEnqueue,
  variant = 'idle',
}: {
  task: Task
  onEnqueue: () => void
  variant?: 'idle' | 'missed'
}) {
  const Icon = KIND_ICON[task.kind]
  const due = formatDueRelative(task.dueAt)

  return (
    <div
      role="group"
      className={cn(
        'group relative flex items-center gap-2 rounded-[var(--clay-radius-sm)] px-2.5 py-2',
        'transition-[background,box-shadow]',
        variant === 'missed'
          ? 'bg-[hsl(var(--danger)/0.07)] shadow-[inset_0_0_0_1px_hsl(var(--danger)/0.25)]'
          : 'hover:bg-[hsl(var(--clay-panel-up)/0.55)]'
      )}
    >
      <span
        aria-hidden
        className={cn(
          'h-1.5 w-1.5 shrink-0 rounded-full',
          task.priority === 'urgent' && 'bg-[hsl(var(--danger))]',
          task.priority === 'high' && 'bg-[hsl(var(--warning))]',
          task.priority === 'normal' && 'bg-[hsl(var(--info))]',
          task.priority === 'low' && 'bg-[hsl(var(--ink-dim))]'
        )}
      />
      <Icon className="h-3 w-3 shrink-0 text-ink-dim" strokeWidth={1.75} aria-hidden />
      <p className="clamp-1 min-w-0 flex-1 text-[12px] leading-tight text-ink">{task.title}</p>
      {task.dueAt && (
        <span
          className={cn(
            'font-mono text-[10px] tabular-nums',
            due.tone === 'past' && 'text-[hsl(var(--danger))]',
            due.tone === 'urgent' && 'text-[hsl(var(--warning))]',
            due.tone === 'warn' && 'text-[hsl(var(--warning)/0.8)]',
            due.tone === 'safe' && 'text-ink-dim'
          )}
        >
          {due.label}
        </span>
      )}

      <button
        type="button"
        onClick={onEnqueue}
        aria-label={variant === 'missed' ? 'Reabrir na fila' : 'Adicionar à fila'}
        className={cn(
          'opacity-0 transition-opacity group-hover:opacity-100',
          'rounded-full p-1 text-ink-dim hover:text-[hsl(var(--accent))]'
        )}
      >
        <ArrowLeftCircle className="h-3.5 w-3.5" strokeWidth={1.75} />
      </button>

      <ClayChip tone={priorityTone(task.priority)} size="xs" className="hidden">
        {task.priority}
      </ClayChip>
    </div>
  )
}
