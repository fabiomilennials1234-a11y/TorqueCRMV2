import { forwardRef, type ButtonHTMLAttributes } from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { Slot } from '@radix-ui/react-slot'
import { cn } from '@/lib/utils'

const claybutton = cva(
  cn(
    'inline-flex items-center justify-center gap-2 whitespace-nowrap font-medium',
    'transition-[box-shadow,transform,background-color,color] ease-[var(--ease-out-soft)]',
    'focus-visible:outline-none focus-visible:shadow-[var(--clay-ring-focus)]',
    'disabled:pointer-events-none disabled:opacity-40 select-none'
  ),
  {
    variants: {
      variant: {
        primary: cn(
          'bg-[hsl(var(--accent))] text-[hsl(222_20%_8%)]',
          'shadow-[var(--clay-shadow-accent)]',
          'hover:-translate-y-[1px] hover:brightness-[1.04]',
          'active:translate-y-[0.5px] active:shadow-[var(--clay-shadow-sunken)]'
        ),
        secondary: cn(
          'bg-[hsl(var(--clay-panel-up))] text-ink',
          'shadow-[var(--clay-shadow-raised)]',
          'hover:-translate-y-[1px]',
          'active:translate-y-[0.5px] active:shadow-[var(--clay-shadow-sunken)]'
        ),
        ghost: cn(
          'bg-transparent text-[hsl(var(--ink-muted))]',
          'hover:text-ink hover:bg-[hsl(var(--clay-panel-up))]',
          'active:shadow-[var(--clay-shadow-sunken)]'
        ),
        danger: cn(
          'bg-[hsl(var(--danger)/0.12)] text-[hsl(var(--danger))]',
          'shadow-[inset_0_1px_0_0_hsl(var(--danger)/0.35),_0_2px_8px_-2px_hsl(var(--danger)/0.25)]',
          'hover:-translate-y-[1px] hover:bg-[hsl(var(--danger)/0.18)]',
          'active:translate-y-[0.5px]'
        ),
      },
      size: {
        sm: 'h-8 px-3 text-[0.8125rem] rounded-full',
        md: 'h-10 px-5 text-sm rounded-full',
        lg: 'h-12 px-7 text-[0.9375rem] rounded-full',
        icon: 'h-10 w-10 rounded-full',
      },
    },
    defaultVariants: { variant: 'secondary', size: 'md' },
  }
)

export interface ClayButtonProps
  extends ButtonHTMLAttributes<HTMLButtonElement>, VariantProps<typeof claybutton> {
  asChild?: boolean
}

export const ClayButton = forwardRef<HTMLButtonElement, ClayButtonProps>(
  ({ className, variant, size, asChild, style, ...props }, ref) => {
    const Comp = asChild ? Slot : 'button'
    return (
      <Comp
        ref={ref}
        className={cn(claybutton({ variant, size }), className)}
        style={{ transitionDuration: 'var(--clay-transition)', ...style }}
        {...props}
      />
    )
  }
)
ClayButton.displayName = 'ClayButton'
