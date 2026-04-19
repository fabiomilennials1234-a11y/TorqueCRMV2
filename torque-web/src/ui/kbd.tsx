import type { HTMLAttributes } from 'react'
import { cn } from '@/lib/utils'

export function Kbd({ className, children, ...props }: HTMLAttributes<HTMLSpanElement>) {
  return (
    <kbd
      className={cn(
        'inline-flex h-[18px] min-w-[18px] items-center justify-center rounded-xs px-1',
        'font-mono text-[0.6875rem] font-medium text-ink-muted',
        'bg-elevated/70 shadow-hairline',
        className
      )}
      {...props}
    >
      {children}
    </kbd>
  )
}
