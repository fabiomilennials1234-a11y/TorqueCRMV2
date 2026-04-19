import type { HTMLAttributes } from 'react'
import { cn } from '@/lib/utils'

/**
 * Pill — compact round-shouldered tag. Hairline outlined, dimmed by default.
 * Different from Badge (uppercase, severe). Pill is for free-form tags/filters.
 */
export function Pill({
  className,
  active,
  ...props
}: HTMLAttributes<HTMLButtonElement> & { active?: boolean }) {
  return (
    <button
      type="button"
      className={cn(
        'inline-flex h-6 items-center gap-1.5 rounded-full px-2.5 text-[0.75rem] font-medium',
        'whitespace-nowrap shadow-hairline transition-colors',
        active
          ? 'bg-accent/15 text-accent shadow-[inset_0_0_0_1px_hsl(var(--accent)/0.4)]'
          : 'text-ink-muted hover:bg-elevated/60 hover:text-ink',
        className
      )}
      {...props}
    />
  )
}
