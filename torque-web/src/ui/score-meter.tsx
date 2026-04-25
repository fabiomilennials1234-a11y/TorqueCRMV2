import { cn } from '@/lib/utils'

/**
 * Torque-meter: a radial score indicator that evokes a mechanical gauge.
 * 0..100. Accent colors by band: dim / warn / accent / success.
 */
export function ScoreMeter({
  value,
  size = 44,
  strokeWidth = 3,
  showLabel = true,
  className,
}: {
  value: number
  size?: number
  strokeWidth?: number
  showLabel?: boolean
  className?: string
}) {
  const clamped = Math.max(0, Math.min(100, value))
  const band =
    clamped < 30 ? 'ink-dim' : clamped < 60 ? 'warning' : clamped < 85 ? 'accent' : 'success'
  const radius = (size - strokeWidth) / 2
  const circ = 2 * Math.PI * radius
  // draw only 270° arc (gauge style); start at 135°
  const arc = circ * 0.75
  const dash = (clamped / 100) * arc

  return (
    <div
      className={cn('relative inline-flex items-center justify-center', className)}
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
          stroke={`hsl(var(--${band}))`}
          strokeWidth={strokeWidth}
          strokeDasharray={`${dash} ${circ}`}
          strokeLinecap="round"
          style={{ transition: 'stroke-dasharray 400ms var(--ease-out-soft)' }}
        />
      </svg>
      {showLabel && (
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className="font-metric text-ink text-[0.8125rem] leading-none">{clamped}</span>
        </div>
      )}
    </div>
  )
}
