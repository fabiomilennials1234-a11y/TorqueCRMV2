import type { HTMLAttributes } from 'react'
import { cn } from '@/lib/utils'

export function Skeleton({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        'bg-elevated/60 relative overflow-hidden rounded-sm',
        'animate-shimmer bg-[linear-gradient(110deg,transparent_0%,hsl(var(--ink-dim)/0.12)_50%,transparent_100%)] bg-[length:200%_100%]',
        className
      )}
      {...props}
    />
  )
}
