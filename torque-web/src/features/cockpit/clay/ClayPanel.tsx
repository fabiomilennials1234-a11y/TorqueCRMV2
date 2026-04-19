import { forwardRef, type HTMLAttributes } from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/utils'

const panel = cva(
  'relative isolate transition-[box-shadow,transform] ease-[var(--ease-out-soft)]',
  {
    variants: {
      variant: {
        surface: 'bg-[hsl(var(--clay-panel))] shadow-[var(--clay-shadow-raised)]',
        raised: 'bg-[hsl(var(--clay-panel-up))] shadow-[var(--clay-shadow-raised)]',
        floating: 'bg-[hsl(var(--clay-panel-up))] shadow-[var(--clay-shadow-floating)]',
        sunken: 'bg-[hsl(var(--clay-panel-down))] shadow-[var(--clay-shadow-sunken)]',
        accent: 'bg-[hsl(var(--clay-panel-up))] shadow-[var(--clay-shadow-accent)]',
      },
      size: {
        sm: 'rounded-[var(--clay-radius-sm)] p-3',
        md: 'rounded-[var(--clay-radius)] p-5',
        lg: 'rounded-[var(--clay-radius-lg)] p-7',
        none: 'rounded-[var(--clay-radius)]',
      },
    },
    defaultVariants: {
      variant: 'surface',
      size: 'md',
    },
  }
)

export interface ClayPanelProps
  extends HTMLAttributes<HTMLDivElement>, VariantProps<typeof panel> {}

export const ClayPanel = forwardRef<HTMLDivElement, ClayPanelProps>(
  ({ className, variant, size, style, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(panel({ variant, size }), className)}
      style={{ transitionDuration: 'var(--clay-transition)', ...style }}
      {...props}
    />
  )
)
ClayPanel.displayName = 'ClayPanel'
