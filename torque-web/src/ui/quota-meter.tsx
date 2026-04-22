/**
 * QuotaMeter — compact progress indicator for (used / limit). S51.
 *
 * Displayed on resource list pages (leads, agents, workflows…) so the
 * user sees cap approach before the 402 lands on a mutation.
 *
 * Tones shift as usage approaches the cap:
 *   ≤ 80%  → neutral
 *   80-99% → warning
 *   100%   → danger
 */

import type { Quota } from '@/hooks/useQuotas'
import { cn } from '@/lib/utils'

interface QuotaMeterProps {
  quota: Quota | undefined
  label?: string
  className?: string
}

export function QuotaMeter({ quota, label, className }: QuotaMeterProps) {
  if (!quota || quota.effective_limit <= 0) {
    return null
  }
  const pct = Math.min(100, Math.round((quota.current_usage / quota.effective_limit) * 100))
  const tone = pct >= 100 ? 'danger' : pct >= 80 ? 'warning' : 'neutral'
  const toneClass = {
    danger: 'bg-danger',
    warning: 'bg-warning',
    neutral: 'bg-accent',
  }[tone]

  return (
    <div
      className={cn('flex flex-col gap-1', className)}
      role="meter"
      aria-valuemin={0}
      aria-valuemax={quota.effective_limit}
      aria-valuenow={quota.current_usage}
      aria-label={label ?? `uso de ${quota.resource}`}
    >
      <div className="flex items-baseline justify-between text-[11px] text-ink-muted">
        <span>
          {label ?? quota.resource} —{' '}
          <span className="text-ink">{quota.current_usage}</span>
          <span className="text-ink-subtle"> / {quota.effective_limit}</span>
        </span>
        <span
          className={cn(
            'font-metric tabular-nums',
            tone === 'danger' && 'text-danger',
            tone === 'warning' && 'text-warning'
          )}
        >
          {pct}%
        </span>
      </div>
      <div className="h-1 overflow-hidden rounded-xs bg-elevated">
        <div
          className={cn('h-full transition-[width] duration-500', toneClass)}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  )
}
