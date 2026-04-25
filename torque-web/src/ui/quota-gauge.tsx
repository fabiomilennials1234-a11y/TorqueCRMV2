import { Lock } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface QuotaGaugeProps {
  current: number
  limit: number
  canAdd?: boolean
  label?: string
  variant?: 'bar' | 'ring' | 'inline'
  className?: string
}

function resolveTone(pct: number): { color: string; hsl: string } {
  if (pct > 90) return { color: 'danger', hsl: '--danger' }
  if (pct > 70) return { color: 'warning', hsl: '--warning' }
  return { color: 'success', hsl: '--success' }
}

/* ─── Bar variant ─── */
function BarGauge({ current, limit, label }: Pick<QuotaGaugeProps, 'current' | 'limit' | 'label'>) {
  const pct = limit === 0 ? 100 : Math.min((current / limit) * 100, 100)
  const { hsl } = resolveTone(pct)

  return (
    <div className="flex flex-col gap-1.5">
      {label && (
        <div className="flex items-center justify-between">
          <span className="text-2xs text-ink-muted font-medium">{label}</span>
          <span className="font-metric text-2xs text-ink-muted tabular-nums">
            {current}
            <span className="text-ink-dim"> / {limit}</span>
          </span>
        </div>
      )}
      <div className="bg-hairline/40 relative h-1.5 w-full overflow-hidden rounded-full">
        <div
          className="absolute inset-y-0 left-0 rounded-full transition-[width,background-color] duration-[280ms] ease-[var(--ease-out-soft)]"
          style={{
            width: `${pct}%`,
            backgroundColor: `hsl(var(${hsl}))`,
          }}
        />
      </div>
    </div>
  )
}

/* ─── Ring variant ─── */
function RingGauge({ current, limit }: Pick<QuotaGaugeProps, 'current' | 'limit'>) {
  const size = 52
  const strokeWidth = 3.5
  const pct = limit === 0 ? 100 : Math.min((current / limit) * 100, 100)
  const { hsl } = resolveTone(pct)
  const radius = (size - strokeWidth) / 2
  const circ = 2 * Math.PI * radius
  // 270-degree arc, same as ScoreMeter
  const arc = circ * 0.75
  const dash = (pct / 100) * arc

  return (
    <div
      className="relative inline-flex items-center justify-center"
      style={{ width: size, height: size }}
    >
      <svg width={size} height={size} className="-rotate-[135deg]">
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke="hsl(var(--hairline))"
          strokeWidth={strokeWidth}
          strokeDasharray={`${arc} ${circ}`}
          strokeLinecap="round"
        />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke={`hsl(var(${hsl}))`}
          strokeWidth={strokeWidth}
          strokeDasharray={`${dash} ${circ}`}
          strokeLinecap="round"
          style={{
            transition:
              'stroke-dasharray 280ms var(--ease-out-soft), stroke 280ms var(--ease-out-soft)',
          }}
        />
      </svg>
      <div className="absolute inset-0 flex flex-col items-center justify-center">
        <span className="font-metric text-ink text-[0.8125rem] leading-none tabular-nums">
          {current}
        </span>
      </div>
    </div>
  )
}

/* ─── Inline variant ─── */
function InlineGauge({
  current,
  limit,
  canAdd,
}: Pick<QuotaGaugeProps, 'current' | 'limit' | 'canAdd'>) {
  const pct = limit === 0 ? 100 : Math.min((current / limit) * 100, 100)
  const { hsl } = resolveTone(pct)

  return (
    <span className="inline-flex items-center gap-1.5">
      <span
        className="inline-block h-1.5 w-1.5 rounded-full transition-colors duration-[280ms] ease-[var(--ease-out-soft)]"
        style={{ backgroundColor: `hsl(var(${hsl}))` }}
      />
      <span className="font-metric text-ink text-sm tabular-nums">
        {current}
        <span className="text-ink-dim"> / {limit}</span>
      </span>
      {canAdd === false && <Lock className="text-ink-dim h-3 w-3" strokeWidth={1.75} />}
    </span>
  )
}

/* ─── Shared lock overlay ─── */
function LockOverlay() {
  return (
    <div className="bg-bg/60 pointer-events-none absolute inset-0 flex items-center justify-center rounded-[inherit]">
      <Lock className="text-ink-dim h-3.5 w-3.5" strokeWidth={1.75} />
    </div>
  )
}

/* ─── Exported component ─── */
export function QuotaGauge({ variant = 'bar', className, canAdd, ...rest }: QuotaGaugeProps) {
  const isLocked = canAdd === false

  return (
    <div className={cn('relative', isLocked && variant !== 'inline' && 'opacity-60', className)}>
      {variant === 'bar' && <BarGauge {...rest} />}
      {variant === 'ring' && <RingGauge current={rest.current} limit={rest.limit} />}
      {variant === 'inline' && (
        <InlineGauge
          current={rest.current}
          limit={rest.limit}
          {...(canAdd !== undefined ? { canAdd } : {})}
        />
      )}
      {isLocked && variant !== 'inline' && <LockOverlay />}
    </div>
  )
}
