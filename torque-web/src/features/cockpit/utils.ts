import {
  CalendarCheck,
  CheckSquare,
  Compass,
  FileText,
  MessageCircle,
  Phone,
  ShieldAlert,
  type LucideIcon,
} from 'lucide-react'
import type { TaskKind, TaskPriority } from '@/contracts/manual'

export const KIND_ICON: Record<TaskKind, LucideIcon> = {
  followup: MessageCircle,
  call: Phone,
  qualification: Compass,
  send_proposal: FileText,
  confirm_meeting: CalendarCheck,
  objection: ShieldAlert,
  generic: CheckSquare,
}

export const KIND_LABEL: Record<TaskKind, string> = {
  followup: 'Follow-up',
  call: 'Ligação',
  qualification: 'Qualificação',
  send_proposal: 'Proposta',
  confirm_meeting: 'Confirmação',
  objection: 'Objeção',
  generic: 'Tarefa',
}

export const PRIORITY_LABEL: Record<TaskPriority, string> = {
  low: 'Baixa',
  normal: 'Normal',
  high: 'Alta',
  urgent: 'Urgente',
}

export function priorityTone(
  p: TaskPriority
): 'neutral' | 'info' | 'warning' | 'danger' | 'accent' {
  switch (p) {
    case 'low':
      return 'neutral'
    case 'normal':
      return 'info'
    case 'high':
      return 'warning'
    case 'urgent':
      return 'danger'
    default:
      return 'neutral'
  }
}

export function formatDueRelative(
  iso: string | null,
  now = new Date()
): { label: string; tone: 'safe' | 'warn' | 'urgent' | 'past' } {
  if (!iso) return { label: 'sem prazo', tone: 'safe' }
  const target = new Date(iso)
  const diffMs = target.getTime() - now.getTime()
  const diffMin = Math.round(diffMs / 60_000)

  if (diffMin < 0) {
    const abs = Math.abs(diffMin)
    if (abs < 60) return { label: `${abs}m atrasada`, tone: 'past' }
    if (abs < 1440) return { label: `${Math.round(abs / 60)}h atrasada`, tone: 'past' }
    return { label: `${Math.round(abs / 1440)}d atrasada`, tone: 'past' }
  }
  if (diffMin < 60) return { label: `em ${diffMin}m`, tone: 'urgent' }
  if (diffMin < 1440)
    return { label: `em ${Math.round(diffMin / 60)}h`, tone: diffMin < 240 ? 'urgent' : 'warn' }
  return { label: `em ${Math.round(diffMin / 1440)}d`, tone: 'safe' }
}

export function formatElapsedSince(iso: string, now = new Date()): string {
  const diffMs = now.getTime() - new Date(iso).getTime()
  const s = Math.max(0, Math.floor(diffMs / 1000))
  const m = Math.floor(s / 60)
  const ss = String(s % 60).padStart(2, '0')
  const mm = String(m % 60).padStart(2, '0')
  const h = Math.floor(m / 60)
  if (h > 0) return `${h}:${mm}:${ss}`
  return `${mm}:${ss}`
}
