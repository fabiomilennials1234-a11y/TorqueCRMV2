/**
 * Torque brand mark — a precise gear-inspired glyph, not a literal gear.
 * Pure SVG, hairline-stroked, accent-tinted.
 */
export function TorqueMark({ size = 22 }: { size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      aria-hidden
      className="shrink-0"
    >
      <defs>
        <linearGradient id="torque-mark-grad" x1="0" y1="0" x2="24" y2="24">
          <stop offset="0%" stopColor="hsl(var(--accent))" />
          <stop offset="100%" stopColor="hsl(34 80% 58%)" />
        </linearGradient>
      </defs>
      {/* outer rim */}
      <circle
        cx="12"
        cy="12"
        r="9"
        stroke="hsl(var(--ink-dim))"
        strokeWidth="1"
      />
      {/* tick marks every 45° */}
      {[0, 45, 90, 135, 180, 225, 270, 315].map((deg) => (
        <line
          key={deg}
          x1="12"
          y1="3"
          x2="12"
          y2="4.5"
          stroke="hsl(var(--ink-muted))"
          strokeWidth="1"
          strokeLinecap="round"
          transform={`rotate(${deg} 12 12)`}
        />
      ))}
      {/* inner dial */}
      <circle cx="12" cy="12" r="5" fill="hsl(var(--surface))" stroke="hsl(var(--hairline))" />
      {/* needle */}
      <line
        x1="12"
        y1="12"
        x2="16"
        y2="8"
        stroke="url(#torque-mark-grad)"
        strokeWidth="1.5"
        strokeLinecap="round"
      />
      <circle cx="12" cy="12" r="1.2" fill="url(#torque-mark-grad)" />
    </svg>
  );
}
