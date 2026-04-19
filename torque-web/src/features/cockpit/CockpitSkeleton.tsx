export function CockpitSkeleton() {
  return (
    <div
      className="grid h-full min-h-0 gap-4 pt-2"
      style={{
        gridTemplateColumns: 'minmax(280px, 320px) minmax(420px, 1fr) minmax(300px, 360px)',
      }}
      aria-label="Carregando cockpit…"
    >
      <ShimmerBlock />
      <ShimmerBlock />
      <div className="flex min-h-0 flex-col gap-4">
        <ShimmerBlock className="h-60 flex-none" />
        <ShimmerBlock className="flex-1" />
      </div>
    </div>
  )
}

function ShimmerBlock({ className = '' }: { className?: string }) {
  return (
    <div
      className={
        'rounded-[var(--clay-radius-lg)] bg-[hsl(var(--clay-panel)/0.6)] shadow-[var(--clay-shadow-sunken)] ' +
        className
      }
    >
      <div className="h-full w-full animate-shimmer rounded-[inherit] bg-gradient-to-r from-transparent via-[hsl(var(--clay-panel-up)/0.5)] to-transparent bg-[length:200%_100%]" />
    </div>
  )
}
