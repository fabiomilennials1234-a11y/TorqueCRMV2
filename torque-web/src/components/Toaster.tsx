/**
 * Toaster — minimal DOM toast surface.
 *
 * Listens for `torque:toast` CustomEvents emitted by `notifyAppError` (and
 * anyone else who wants a transient message). Keeps a small FIFO queue, auto-
 * dismisses after 5 seconds, and never blocks the main thread.
 *
 * Rendered once at the root of AppShell so every page inherits it.
 */

import { useEffect, useState } from 'react'
import { AlertTriangle, CheckCircle2, Info, X } from 'lucide-react'

import { cn } from '@/lib/utils'

interface Toast {
  id: string
  level: 'error' | 'info' | 'success'
  code?: string
  message: string
  context?: string
}

interface TorqueToastDetail {
  level?: 'error' | 'info' | 'success'
  code?: string
  message: string
  context?: string
}

const DISMISS_MS = 5000
const MAX_TOASTS = 4

export function Toaster() {
  const [toasts, setToasts] = useState<Toast[]>([])

  useEffect(() => {
    const handler = (ev: Event) => {
      const detail = (ev as CustomEvent<TorqueToastDetail>).detail
      if (!detail?.message) return
      // With exactOptionalPropertyTypes=true, omit undefined fields rather
      // than assigning `undefined` to an optional string property.
      const toast: Toast = {
        id: typeof crypto !== 'undefined' && crypto.randomUUID ? crypto.randomUUID() : String(Math.random()),
        level: detail.level ?? 'info',
        message: detail.message,
        ...(detail.code !== undefined ? { code: detail.code } : {}),
        ...(detail.context !== undefined ? { context: detail.context } : {}),
      }
      setToasts((prev) => [...prev.slice(-(MAX_TOASTS - 1)), toast])
      window.setTimeout(() => {
        setToasts((prev) => prev.filter((t) => t.id !== toast.id))
      }, DISMISS_MS)
    }
    window.addEventListener('torque:toast', handler)
    return () => window.removeEventListener('torque:toast', handler)
  }, [])

  if (toasts.length === 0) return null

  return (
    <div
      aria-live="polite"
      className="pointer-events-none fixed bottom-4 right-4 z-50 flex w-80 flex-col gap-2"
    >
      {toasts.map((t) => (
        <div
          key={t.id}
          role={t.level === 'error' ? 'alert' : 'status'}
          className={cn(
            'pointer-events-auto flex items-start gap-3 rounded-md bg-elevated p-3 shadow-hairline',
            t.level === 'error' && 'shadow-[inset_0_0_0_1px_hsl(var(--danger)/0.5)]'
          )}
        >
          <Icon level={t.level} />
          <div className="min-w-0 flex-1">
            <p className="text-sm leading-snug text-ink">{t.message}</p>
            {t.context && (
              <p className="mt-0.5 text-2xs uppercase tracking-[0.14em] text-ink-dim">
                {t.context}
              </p>
            )}
          </div>
          <button
            type="button"
            aria-label="Fechar notificação"
            onClick={() => setToasts((prev) => prev.filter((x) => x.id !== t.id))}
            className="rounded-sm text-ink-dim transition hover:text-ink"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      ))}
    </div>
  )
}

function Icon({ level }: { level: Toast['level'] }) {
  const common = 'mt-0.5 h-4 w-4 shrink-0'
  if (level === 'error') return <AlertTriangle className={cn(common, 'text-danger')} />
  if (level === 'success') return <CheckCircle2 className={cn(common, 'text-success')} />
  return <Info className={cn(common, 'text-ink-muted')} />
}
