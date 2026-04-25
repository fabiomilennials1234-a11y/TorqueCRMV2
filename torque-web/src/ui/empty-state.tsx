import type { LucideIcon } from 'lucide-react'
import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

export function EmptyState({
  icon: Icon,
  title,
  description,
  action,
  className,
}: {
  icon?: LucideIcon
  title: string
  description?: string
  action?: ReactNode
  className?: string
}) {
  return (
    <div
      className={cn('flex flex-col items-center justify-center px-6 py-16 text-center', className)}
    >
      {Icon && (
        <div className="bg-elevated shadow-hairline mx-auto mb-4 flex h-11 w-11 items-center justify-center rounded-md">
          <Icon className="text-ink-dim h-5 w-5" strokeWidth={1.5} />
        </div>
      )}
      <h3 className="font-display tracking-tightest text-ink text-lg">{title}</h3>
      {description && <p className="text-ink-muted mx-auto mt-2 max-w-sm text-sm">{description}</p>}
      {action && <div className="mt-5">{action}</div>}
    </div>
  )
}
