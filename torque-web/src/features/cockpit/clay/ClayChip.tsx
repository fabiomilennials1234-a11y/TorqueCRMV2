import { forwardRef, type HTMLAttributes } from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/utils'

const claychip = cva(
  cn(
    'inline-flex items-center gap-1.5 whitespace-nowrap',
    'rounded-full font-medium tracking-tight',
    'shadow-[var(--clay-shadow-sunken)]'
  ),
  {
    variants: {
      tone: {
        neutral: 'bg-[hsl(var(--clay-panel-down))] text-ink-muted',
        accent:
          'bg-[hsl(var(--accent)/0.14)] text-[hsl(var(--accent))] shadow-[inset_0_1px_0_0_hsl(var(--accent)/0.3),_inset_0_-1px_0_0_hsl(0_0%_0%/0.2)]',
        success:
          'bg-[hsl(var(--success)/0.14)] text-[hsl(var(--success))] shadow-[inset_0_1px_0_0_hsl(var(--success)/0.3),_inset_0_-1px_0_0_hsl(0_0%_0%/0.2)]',
        warning:
          'bg-[hsl(var(--warning)/0.14)] text-[hsl(var(--warning))] shadow-[inset_0_1px_0_0_hsl(var(--warning)/0.3),_inset_0_-1px_0_0_hsl(0_0%_0%/0.2)]',
        danger:
          'bg-[hsl(var(--danger)/0.14)] text-[hsl(var(--danger))] shadow-[inset_0_1px_0_0_hsl(var(--danger)/0.3),_inset_0_-1px_0_0_hsl(0_0%_0%/0.2)]',
        info: 'bg-[hsl(var(--info)/0.14)] text-[hsl(var(--info))] shadow-[inset_0_1px_0_0_hsl(var(--info)/0.3),_inset_0_-1px_0_0_hsl(0_0%_0%/0.2)]',
      },
      size: {
        xs: 'h-5 px-2 text-[10px] uppercase tracking-[0.08em]',
        sm: 'h-6 px-2.5 text-[11px]',
        md: 'h-7 px-3 text-xs',
      },
    },
    defaultVariants: { tone: 'neutral', size: 'sm' },
  }
)

export interface ClayChipProps
  extends HTMLAttributes<HTMLSpanElement>, VariantProps<typeof claychip> {}

export const ClayChip = forwardRef<HTMLSpanElement, ClayChipProps>(
  ({ className, tone, size, ...props }, ref) => (
    <span ref={ref} className={cn(claychip({ tone, size }), className)} {...props} />
  )
)
ClayChip.displayName = 'ClayChip'
