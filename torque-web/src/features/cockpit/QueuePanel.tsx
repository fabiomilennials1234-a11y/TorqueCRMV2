import { useState } from 'react'
import type { DragEvent } from 'react'
import { ArrowDownUp, Check, Flame, GripVertical, Play, X } from 'lucide-react'
import { ClayCard } from './clay/ClayCard'
import { ClayChip } from './clay/ClayChip'
import { ClayButton } from './clay/ClayButton'
import { cn } from '@/lib/utils'
import type { Task } from '@/contracts/manual'
import { useTaskActions } from '@/hooks/useTaskActions'
import { KIND_ICON, formatDueRelative, priorityTone } from './utils'

interface QueuePanelProps {
  tasks: Task[]
  inProgressId: string | null
}

export function QueuePanel({ tasks, inProgressId }: QueuePanelProps) {
  const { startTask, reorderQueue, dequeueTask } = useTaskActions()
  const [draggingId, setDraggingId] = useState<string | null>(null)
  const [hoverId, setHoverId] = useState<string | null>(null)

  function onDragStart(e: DragEvent<HTMLDivElement>, id: string) {
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.setData('text/task-id', id)
    setDraggingId(id)
  }
  function onDragOver(e: DragEvent<HTMLDivElement>, id: string) {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
    setHoverId(id)
  }
  function onDrop(e: DragEvent<HTMLDivElement>, targetId: string) {
    e.preventDefault()
    const draggedId = e.dataTransfer.getData('text/task-id')
    if (!draggedId || draggedId === targetId) {
      setDraggingId(null)
      setHoverId(null)
      return
    }
    const ids = tasks.map((t) => t.id)
    const from = ids.indexOf(draggedId)
    const to = ids.indexOf(targetId)
    if (from === -1 || to === -1) return
    const reordered = [...ids]
    reordered.splice(from, 1)
    reordered.splice(to, 0, draggedId)
    reorderQueue(reordered)
    setDraggingId(null)
    setHoverId(null)
  }
  function onDragEnd() {
    setDraggingId(null)
    setHoverId(null)
  }

  return (
    <section className="flex h-full min-h-0 flex-col" aria-label="Tasks a fazer">
      <header className="flex items-center justify-between px-1 pb-4">
        <div className="flex items-baseline gap-2">
          <h2 className="font-display tracking-tightest text-ink text-[1.25rem] leading-none">
            Tasks a fazer
          </h2>
          <span className="text-ink-dim font-mono text-xs tabular-nums">{tasks.length}</span>
        </div>
        <ArrowDownUp className="text-ink-dim h-3.5 w-3.5" strokeWidth={1.75} aria-hidden />
      </header>

      {tasks.length === 0 ? (
        <EmptyQueue />
      ) : (
        <div className="flex min-h-0 flex-1 flex-col gap-2.5 overflow-y-auto pr-1">
          {tasks.map((t, idx) => {
            const Icon = KIND_ICON[t.kind]
            const due = formatDueRelative(t.dueAt)
            const isDragging = draggingId === t.id
            const isHover = hoverId === t.id && draggingId && draggingId !== t.id
            const isActive = inProgressId === t.id

            return (
              <div
                key={t.id}
                draggable
                onDragStart={(e) => onDragStart(e, t.id)}
                onDragOver={(e) => onDragOver(e, t.id)}
                onDrop={(e) => onDrop(e, t.id)}
                onDragEnd={onDragEnd}
                className={cn('transition-opacity', isDragging && 'opacity-40')}
              >
                <ClayCard
                  state={isActive ? 'active' : 'idle'}
                  onActivate={() => startTask(t.id)}
                  className={cn('group', isHover && 'ring-2 ring-[hsl(var(--accent)/0.6)]')}
                >
                  <div className="flex items-start gap-2.5">
                    <GripVertical
                      className="text-ink-dim mt-0.5 h-3.5 w-3.5 shrink-0 opacity-0 transition-opacity group-hover:opacity-100"
                      strokeWidth={1.75}
                      aria-hidden
                    />
                    <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-[hsl(var(--clay-panel-down))] shadow-[var(--clay-shadow-sunken)]">
                      <Icon className="text-ink-muted h-3.5 w-3.5" strokeWidth={1.75} />
                    </div>

                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-1.5">
                        <span className="text-ink-dim font-mono text-[10px] tabular-nums">
                          {String(idx + 1).padStart(2, '0')}
                        </span>
                        {t.priority === 'urgent' && (
                          <Flame
                            className="h-3 w-3 text-[hsl(var(--danger))]"
                            strokeWidth={2}
                            aria-label="Urgente"
                          />
                        )}
                      </div>
                      <p className="clamp-2 text-ink mt-0.5 text-[13px] leading-snug">{t.title}</p>
                      <div className="mt-2 flex items-center gap-1.5">
                        <ClayChip tone={priorityTone(t.priority)} size="xs">
                          {t.priority}
                        </ClayChip>
                        {t.dueAt && (
                          <span
                            className={cn(
                              'font-mono text-[10px] tabular-nums',
                              due.tone === 'past' && 'text-[hsl(var(--danger))]',
                              due.tone === 'urgent' && 'text-[hsl(var(--warning))]',
                              due.tone === 'warn' && 'text-[hsl(var(--warning)/0.75)]',
                              due.tone === 'safe' && 'text-ink-dim'
                            )}
                          >
                            {due.label}
                          </span>
                        )}
                      </div>
                    </div>

                    <div className="flex flex-col gap-1 opacity-0 transition-opacity group-hover:opacity-100">
                      <button
                        type="button"
                        aria-label="Iniciar task"
                        onClick={(e) => {
                          e.stopPropagation()
                          startTask(t.id)
                        }}
                        className="text-ink-dim rounded-full p-1 hover:text-[hsl(var(--accent))]"
                      >
                        <Play className="h-3 w-3" strokeWidth={2} />
                      </button>
                      <button
                        type="button"
                        aria-label="Remover da fila"
                        onClick={(e) => {
                          e.stopPropagation()
                          dequeueTask(t.id)
                        }}
                        className="text-ink-dim rounded-full p-1 hover:text-[hsl(var(--danger))]"
                      >
                        <X className="h-3 w-3" strokeWidth={2} />
                      </button>
                    </div>
                  </div>
                </ClayCard>
              </div>
            )
          })}
        </div>
      )}
    </section>
  )
}

function EmptyQueue() {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-3 px-6 text-center">
      <div className="flex h-10 w-10 items-center justify-center rounded-full bg-[hsl(var(--clay-panel))] shadow-[var(--clay-shadow-sunken)]">
        <Check className="text-ink-dim h-4 w-4" strokeWidth={1.75} />
      </div>
      <div>
        <p className="text-ink text-sm">Fila limpa</p>
        <p className="text-ink-dim mt-1 text-xs">Arraste tasks do backlog para priorizar.</p>
      </div>
      <ClayButton variant="ghost" size="sm">
        Pegar da lista em aberto
      </ClayButton>
    </div>
  )
}
