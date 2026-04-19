import type { HTMLAttributes } from 'react'
import { cn } from '@/lib/utils'

export function Card({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        'relative overflow-hidden rounded-lg bg-surface shadow-elev-1',
        'ease-[var(--ease-out-soft)] transition-shadow duration-200',
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
      className={cn('flex items-start justify-between gap-4 px-6 pb-3 pt-6', className)}
      {...props}
    />
  )
}

export function CardTitle({ className, children, ...props }: HTMLAttributes<HTMLHeadingElement>) {
  return (
    <h3
      className={cn('text-2xs font-medium uppercase tracking-[0.12em] text-ink-dim', className)}
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
        'px-6 py-3 text-xs text-ink-muted shadow-[inset_0_1px_0_0_hsl(var(--hairline)/0.5)]',
        className
      )}
      {...props}
    />
  )
}
