import type { HTMLAttributes } from 'react'
import { cn } from '@/lib/utils'

export function Card({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        'bg-surface shadow-elev-1 relative overflow-hidden rounded-lg',
        'transition-shadow duration-200 ease-[var(--ease-out-soft)]',
        'hover:shadow-neu-hover',
        className
      )}
      {...props}
    />
  )
}

export function CardHeader({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn('flex items-start justify-between gap-4 px-6 pt-6 pb-3', className)}
      {...props}
    />
  )
}

export function CardTitle({ className, children, ...props }: HTMLAttributes<HTMLHeadingElement>) {
  return (
    <h3
      className={cn('text-2xs text-ink-dim font-medium tracking-[0.12em] uppercase', className)}
      {...props}
    >
      {children}
    </h3>
  )
}

export function CardBody({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('px-6 pb-6', className)} {...props} />
}

export function CardFooter({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        'text-ink-muted px-6 py-3 text-xs shadow-[inset_0_1px_0_0_hsl(var(--hairline)/0.5)]',
        className
      )}
      {...props}
    />
  )
}
