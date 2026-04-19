import { forwardRef, type HTMLAttributes, type KeyboardEvent } from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/utils'

const claycard = cva(
  cn(
    'group relative block w-full text-left',
    'bg-[hsl(var(--clay-panel))] shadow-[var(--clay-shadow-raised)]',
    'transition-[box-shadow,transform] ease-[var(--ease-out-soft)]',
    'focus-visible:outline-none focus-visible:shadow-[var(--clay-ring-focus)]'
  ),
  {
    variants: {
      interactive: {
        true: cn(
          'cursor-pointer',
          'hover:-translate-y-[1px] hover:bg-[hsl(var(--clay-panel-up))]',
          'active:translate-y-[0.5px] active:shadow-[var(--clay-shadow-sunken)]'
        ),
        false: '',
      },
      state: {
        idle: '',
        active: cn('bg-[hsl(var(--clay-panel-up))] shadow-[var(--clay-shadow-accent)]'),
        muted: 'opacity-60 hover:opacity-100',
      },
      size: {
        sm: 'rounded-[var(--clay-radius-sm)] p-3',
        md: 'rounded-[var(--clay-radius)] p-4',
      },
    },
    defaultVariants: {
      interactive: true,
      state: 'idle',
      size: 'sm',
    },
  }
)

export interface ClayCardProps
  extends HTMLAttributes<HTMLDivElement>, VariantProps<typeof claycard> {
  onActivate?: () => void
  draggable?: boolean
}

export const ClayCard = forwardRef<HTMLDivElement, ClayCardProps>(
  ({ className, interactive, state, size, onActivate, style, onKeyDown, ...props }, ref) => {
    function handleKeyDown(e: KeyboardEvent<HTMLDivElement>) {
      onKeyDown?.(e)
      if (!onActivate) return
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault()
        onActivate()
      }
    }

    return (
      <div
        ref={ref}
        role={onActivate ? 'button' : undefined}
        tabIndex={onActivate ? 0 : undefined}
        onClick={onActivate}
        onKeyDown={handleKeyDown}
        className={cn(claycard({ interactive, state, size }), className)}
        style={{ transitionDuration: 'var(--clay-transition)', ...style }}
        {...props}
      />
    )
  }
)
ClayCard.displayName = 'ClayCard'
