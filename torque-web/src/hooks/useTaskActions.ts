import { useCallback } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { COCKPIT_QUERY_KEY } from '@/hooks/useCockpit'
import type { Task, TaskCockpitBundle } from '@/contracts/manual'

type Bundle = TaskCockpitBundle

function removeById(list: Task[], id: string): Task[] {
  return list.filter((t) => t.id !== id)
}

function recomputeCounts(b: Bundle): Bundle {
  return {
    ...b,
    counts: {
      ...b.counts,
      queue: b.queue.length,
      backlog: b.backlog.length,
      missed: b.missed.length,
    },
  }
}

function resequenceQueue(queue: Task[]): Task[] {
  return queue.map((t, i) => ({
    ...t,
    queuePosition: i + 1,
  }))
}

export function useTaskActions() {
  const qc = useQueryClient()

  const update = useCallback(
    (mutator: (prev: Bundle) => Bundle) => {
      qc.setQueryData<Bundle>(COCKPIT_QUERY_KEY, (prev) => {
        if (!prev) return prev
        return recomputeCounts(mutator(prev))
      })
    },
    [qc]
  )

  const startTask = useCallback(
    (taskId: string) => {
      update((prev) => {
        const fromQueue = prev.queue.find((t) => t.id === taskId)
        const fromBacklog = prev.backlog.find((t) => t.id === taskId)
        const candidate = fromQueue ?? fromBacklog
        if (!candidate) return prev

        const now = new Date().toISOString()
        const paused = prev.inProgress
          ? {
              ...prev.inProgress,
              status: 'pending' as const,
              inQueue: true,
              queuePosition: 1,
              startedAt: prev.inProgress.startedAt,
            }
          : null

        const next: Task = {
          ...candidate,
          status: 'in_progress',
          inQueue: false,
          queuePosition: null,
          startedAt: now,
        }

        const nextQueue = paused
          ? resequenceQueue([paused, ...removeById(prev.queue, candidate.id)])
          : resequenceQueue(removeById(prev.queue, candidate.id))

        return {
          ...prev,
          inProgress: next,
          queue: nextQueue,
          backlog: removeById(prev.backlog, candidate.id),
        }
      })
    },
    [update]
  )

  const completeTask = useCallback(
    (taskId: string, resultNote?: string) => {
      update((prev) => {
        if (!prev.inProgress || prev.inProgress.id !== taskId) return prev
        void resultNote // TODO: enviar ao backend em POST /tasks/:id/complete
        return {
          ...prev,
          inProgress: null,
          counts: {
            ...prev.counts,
            completedToday: prev.counts.completedToday + 1,
          },
          // Task concluída sai do cockpit. Histórico via endpoint separado.
          activeLead: prev.activeLead,
        }
      })
    },
    [update]
  )

  const pauseTask = useCallback(
    (taskId: string) => {
      update((prev) => {
        if (!prev.inProgress || prev.inProgress.id !== taskId) return prev
        const paused: Task = {
          ...prev.inProgress,
          status: 'pending',
          inQueue: true,
          queuePosition: 1,
        }
        return {
          ...prev,
          inProgress: null,
          queue: resequenceQueue([paused, ...prev.queue]),
        }
      })
    },
    [update]
  )

  const enqueueTask = useCallback(
    (taskId: string) => {
      update((prev) => {
        const t = prev.backlog.find((x) => x.id === taskId)
        if (!t) return prev
        const promoted: Task = {
          ...t,
          inQueue: true,
          queuePosition: prev.queue.length + 1,
        }
        return {
          ...prev,
          queue: [...prev.queue, promoted],
          backlog: removeById(prev.backlog, taskId),
        }
      })
    },
    [update]
  )

  const dequeueTask = useCallback(
    (taskId: string) => {
      update((prev) => {
        const t = prev.queue.find((x) => x.id === taskId)
        if (!t) return prev
        const demoted: Task = {
          ...t,
          inQueue: false,
          queuePosition: null,
        }
        return {
          ...prev,
          queue: resequenceQueue(removeById(prev.queue, taskId)),
          backlog: [demoted, ...prev.backlog],
        }
      })
    },
    [update]
  )

  const reorderQueue = useCallback(
    (orderedIds: string[]) => {
      update((prev) => {
        const map = new Map(prev.queue.map((t) => [t.id, t]))
        const reordered = orderedIds.map((id) => map.get(id)).filter((t): t is Task => Boolean(t))
        return {
          ...prev,
          queue: resequenceQueue(reordered),
        }
      })
    },
    [update]
  )

  const reopenMissed = useCallback(
    (taskId: string) => {
      update((prev) => {
        const t = prev.missed.find((x) => x.id === taskId)
        if (!t) return prev
        const reopened: Task = {
          ...t,
          status: 'pending',
          inQueue: true,
          queuePosition: 1,
          missedReason: null,
        }
        return {
          ...prev,
          missed: removeById(prev.missed, taskId),
          queue: resequenceQueue([reopened, ...prev.queue]),
        }
      })
    },
    [update]
  )

  return {
    startTask,
    completeTask,
    pauseTask,
    enqueueTask,
    dequeueTask,
    reorderQueue,
    reopenMissed,
  }
}
