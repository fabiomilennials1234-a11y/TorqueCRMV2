import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

export function PageHeader({
  eyebrow,
  title,
  description,
  actions,
  className,
}: {
  eyebrow?: string
  title: ReactNode
  description?: ReactNode
  actions?: ReactNode
  className?: string
}) {
  return (
    <header
      className={cn(
        'flex items-start justify-between gap-6 pt-8 pb-6',
        'shadow-hairline-b',
        className
      )}
    >
      <div className="min-w-0">
        {eyebrow && (
          <div className="text-2xs text-ink-dim mb-2 font-medium tracking-[0.14em] uppercase">
            {eyebrow}
          </div>
        )}
        <h1 className="font-display tracking-tightest text-ink text-[2rem] leading-[1.05]">
          {title}
        </h1>
        {description && (
          <p className="text-ink-muted mt-2 max-w-2xl text-[0.9375rem] leading-relaxed">
            {description}
          </p>
        )}
      </div>
      {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
    </header>
  )
}
