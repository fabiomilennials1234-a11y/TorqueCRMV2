import { useEffect } from 'react'
import { useCockpit } from '@/hooks/useCockpit'
import { useTaskActions } from '@/hooks/useTaskActions'
import { QueuePanel } from './QueuePanel'
import { ActiveTaskPanel } from './ActiveTaskPanel'
import { PipeSnapshotPanel } from './PipeSnapshotPanel'
import { BacklogPanel } from './BacklogPanel'
import { CockpitSkeleton } from './CockpitSkeleton'

export function CockpitView() {
  const { data, isLoading, isError } = useCockpit()
  const { completeTask, pauseTask } = useTaskActions()

  useEffect(() => {
    function handler(e: KeyboardEvent) {
      if (e.target instanceof HTMLElement) {
        const tag = e.target.tagName
        if (tag === 'INPUT' || tag === 'TEXTAREA' || e.target.isContentEditable) {
          return
        }
      }
      if (!data?.inProgress) return
      if (e.key.toLowerCase() === 'c') {
        e.preventDefault()
        completeTask(data.inProgress.id)
      } else if (e.key.toLowerCase() === 'p') {
        e.preventDefault()
        pauseTask(data.inProgress.id)
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [data?.inProgress, completeTask, pauseTask])

  if (isLoading || !data) {
    return (
      <div className="flex h-full w-full overflow-hidden p-4">
        <CockpitSkeleton />
      </div>
    )
  }
  if (isError) {
    return (
      <div className="text-ink-muted flex h-full items-center justify-center text-sm">
        Não foi possível carregar o cockpit.
      </div>
    )
  }

  return (
    <div
      className="grid h-full min-h-0 w-full gap-4 overflow-hidden p-4"
      style={{
        gridTemplateColumns: '232px minmax(420px, 1fr) minmax(300px, 340px)',
        gridTemplateRows: '1fr',
      }}
    >
      {/* Sidebar esquerda — tasks a fazer (substitui Sidebar do AppShell) */}
      <aside
        className="flex min-h-0 flex-col rounded-[var(--clay-radius-lg)] bg-[hsl(var(--clay-panel)/0.55)] p-4 shadow-[var(--clay-shadow-sunken)]"
        aria-label="Tasks a fazer"
      >
        <QueuePanel tasks={data.queue} inProgressId={data.inProgress?.id ?? null} />
      </aside>

      {/* Centro — task em foco */}
      <section className="flex min-h-0 flex-col" aria-label="Task ativa">
        <ActiveTaskPanel
          task={data.inProgress}
          lead={data.activeLead}
          hasQueue={data.queue.length > 0}
          firstQueuedId={data.queue[0]?.id ?? null}
        />
      </section>

      {/* Sidebar direita — dividida: kanban (topo) + backlog (base) */}
      <aside className="flex min-h-0 flex-col gap-4" aria-label="Contexto e backlog">
        <PipeSnapshotPanel
          stages={data.pipeStages}
          lead={data.activeLead}
          taskId={data.inProgress?.id ?? null}
        />
        <BacklogPanel backlog={data.backlog} missed={data.missed} />
      </aside>
    </div>
  )
}
