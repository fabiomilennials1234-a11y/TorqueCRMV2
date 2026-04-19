import { cn } from '@/lib/utils'

/**
 * Inline sparkline. Deliberate: hairline stroke + single accent dot on latest point.
 */
export function Sparkline({
  data,
  width = 120,
  height = 32,
  color = 'hsl(var(--ink-muted))',
  accent = 'hsl(var(--accent))',
  className,
}: {
  data: number[]
  width?: number
  height?: number
  color?: string
  accent?: string
  className?: string
}) {
  if (data.length < 2) return null
  const min = Math.min(...data)
  const max = Math.max(...data)
  const range = max - min || 1
  const stepX = width / (data.length - 1)
  const points = data.map((v, i) => {
    const x = i * stepX
    const y = height - ((v - min) / range) * height
    return [x, y] as const
  })
  const d = points.map(([x, y], i) => (i === 0 ? `M ${x} ${y}` : `L ${x} ${y}`)).join(' ')
  const last = points[points.length - 1]!
  const areaD = d + ` L ${last[0]} ${height} L 0 ${height} Z`

  return (
    <svg width={width} height={height} className={cn('overflow-visible', className)}>
      <defs>
        <linearGradient id="spark-fill" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={accent} stopOpacity="0.18" />
          <stop offset="100%" stopColor={accent} stopOpacity="0" />
        </linearGradient>
      </defs>
      <path d={areaD} fill="url(#spark-fill)" />
      <path
        d={d}
        fill="none"
        stroke={color}
        strokeWidth={1.25}
        strokeLinejoin="round"
        strokeLinecap="round"
      />
      <circle cx={last[0]} cy={last[1]} r={2} fill={accent} />
    </svg>
  )
}
