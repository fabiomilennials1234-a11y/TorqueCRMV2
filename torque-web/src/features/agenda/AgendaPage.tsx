/**
 * F13 Agenda — AgendaPage (S48).
 *
 * Vista de lista agrupada por dia dentro de uma janela configurável.
 * Optamos por lista em vez de grid-de-calendário (react-big-calendar é
 * 1385 LOC no v8 e pesa no bundle); a list atende a operação imediata
 * — "quais reuniões tenho nos próximos 30 dias?" — e o S49 que ligar
 * ao Google Calendar pode reutilizar os mesmos hooks.
 */

import { Plus, Trash2 } from 'lucide-react'
import { useMemo, useState } from 'react'

import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { EmptyState } from '@/ui/empty-state'
import { Input } from '@/ui/input'
import { PageHeader } from '@/ui/page-header'
import { Skeleton } from '@/ui/skeleton'
import {
  useCreateMeeting,
  useDeleteMeeting,
  useMeetings,
  type Meeting,
  type MeetingStatus,
} from '@/hooks/useMeetings'

function windowRange(daysBack: number, daysFwd: number): { from: string; to: string } {
  const now = new Date()
  const from = new Date(now)
  from.setDate(from.getDate() - daysBack)
  const to = new Date(now)
  to.setDate(to.getDate() + daysFwd)
  return { from: from.toISOString(), to: to.toISOString() }
}

const statusTone: Record<MeetingStatus, 'success' | 'danger' | 'neutral'> = {
  scheduled: 'neutral',
  completed: 'success',
  cancelled: 'danger',
  no_show: 'danger',
}

export function AgendaPage() {
  const range = useMemo(() => windowRange(7, 30), [])
  const query = useMeetings(range)
  const [creating, setCreating] = useState(false)

  const groups = useMemo(() => {
    const data = query.data?.data ?? []
    const map = new Map<string, Meeting[]>()
    for (const m of data) {
      const day = new Date(m.starts_at).toLocaleDateString('pt-BR', {
        weekday: 'long',
        day: '2-digit',
        month: 'long',
      })
      const arr = map.get(day) ?? []
      arr.push(m)
      map.set(day, arr)
    }
    return Array.from(map.entries())
  }, [query.data])

  return (
    <div className="mx-auto max-w-4xl px-8 py-8">
      <PageHeader
        eyebrow="Time"
        title="Agenda"
        description="Reuniões internas nos próximos 30 dias."
        actions={
          <Button type="button" variant="primary" size="sm" onClick={() => setCreating(true)}>
            <Plus className="mr-1 h-3.5 w-3.5" />
            Nova reunião
          </Button>
        }
      />

      {query.isLoading ? (
        <Skeleton className="mt-6 h-48" />
      ) : groups.length === 0 ? (
        <EmptyState
          title="Nenhuma reunião no período"
          description="Crie a primeira com o botão acima."
        />
      ) : (
        <div className="mt-6 space-y-6">
          {groups.map(([day, items]) => (
            <section key={day}>
              <h2 className="mb-2 text-xs uppercase tracking-wide text-ink-dim">{day}</h2>
              <ul className="space-y-1.5">
                {items.map((m) => (
                  <MeetingRow key={m.id} meeting={m} />
                ))}
              </ul>
            </section>
          ))}
        </div>
      )}

      {creating && <CreateMeetingModal onClose={() => setCreating(false)} />}
    </div>
  )
}

function MeetingRow({ meeting }: { meeting: Meeting }) {
  const remove = useDeleteMeeting(meeting.id)
  const start = new Date(meeting.starts_at)
  const end = new Date(meeting.ends_at)
  const time = `${start.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })} – ${end.toLocaleTimeString(
    'pt-BR',
    { hour: '2-digit', minute: '2-digit' }
  )}`
  return (
    <li className="flex items-center gap-3 rounded-md bg-surface px-3 py-2 shadow-hairline">
      <span className="font-mono text-2xs text-ink-dim">{time}</span>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="truncate text-sm text-ink">{meeting.title}</span>
          <Badge tone={statusTone[meeting.status]}>{meeting.status}</Badge>
        </div>
        {meeting.location && <div className="truncate text-2xs text-ink-dim">{meeting.location}</div>}
      </div>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={() => void remove.mutateAsync()}
        disabled={remove.isPending}
        aria-label="Excluir reunião"
      >
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </li>
  )
}

function CreateMeetingModal({ onClose }: { onClose: () => void }) {
  const create = useCreateMeeting()
  const [title, setTitle] = useState('')
  const [location, setLocation] = useState('')
  const [startsAt, setStartsAt] = useState('')
  const [endsAt, setEndsAt] = useState('')

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!title.trim() || !startsAt || !endsAt) return
    const payload: Parameters<typeof create.mutateAsync>[0] = {
      title: title.trim(),
      starts_at: new Date(startsAt).toISOString(),
      ends_at: new Date(endsAt).toISOString(),
    }
    if (location.trim()) payload.location = location.trim()
    await create.mutateAsync(payload)
    onClose()
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-ink/60 p-4">
      <form
        onSubmit={(e) => void handleSubmit(e)}
        className="w-full max-w-md rounded-xl bg-surface p-6 shadow-elev-2"
      >
        <h2 className="mb-4 font-fraunces text-lg text-ink">Nova reunião</h2>
        <div className="space-y-3">
          <Input value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Título" />
          <Input
            value={location}
            onChange={(e) => setLocation(e.target.value)}
            placeholder="Local (opcional)"
          />
          <Input
            type="datetime-local"
            value={startsAt}
            onChange={(e) => setStartsAt(e.target.value)}
          />
          <Input
            type="datetime-local"
            value={endsAt}
            onChange={(e) => setEndsAt(e.target.value)}
          />
        </div>
        <div className="mt-5 flex items-center justify-between">
          <Button type="button" variant="ghost" size="sm" onClick={onClose}>
            Cancelar
          </Button>
          <Button type="submit" variant="primary" size="sm" disabled={create.isPending}>
            {create.isPending ? 'Criando…' : 'Criar'}
          </Button>
        </div>
      </form>
    </div>
  )
}
