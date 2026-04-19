import { Slot } from '@radix-ui/react-slot'
import { cva, type VariantProps } from 'class-variance-authority'
import { forwardRef, type ButtonHTMLAttributes } from 'react'
import { cn } from '@/lib/utils'

const button = cva(
  'inline-flex items-center justify-center gap-2 whitespace-nowrap font-medium tracking-tight transition-[color,background,box-shadow,transform] duration-150 ease-[var(--ease-out-soft)] focus-visible:outline-none disabled:pointer-events-none disabled:opacity-40 select-none',
  {
    variants: {
      variant: {
        primary:
          'bg-accent text-[hsl(222_20%_8%)] hover:bg-accent/90 active:scale-[0.98] shadow-btn-primary',
        secondary:
          'bg-elevated text-ink shadow-hairline hover:bg-elevated/70 hover:shadow-[inset_0_0_0_1px_hsl(var(--ink-dim)/0.4)]',
        ghost: 'text-ink-muted hover:text-ink hover:bg-elevated/60',
        outline:
          'shadow-hairline bg-transparent text-ink hover:shadow-[inset_0_0_0_1px_hsl(var(--ink-dim)/0.4)] hover:bg-elevated/40',
        danger:
          'bg-danger/10 text-danger shadow-[inset_0_0_0_1px_hsl(var(--danger)/0.35)] hover:bg-danger/15',
        link: 'text-accent underline-offset-4 hover:underline decoration-accent/40',
      },
      size: {
        xs: 'h-6 px-2 text-xs rounded-sm',
        sm: 'h-8 px-3 text-[0.8125rem] rounded',
        md: 'h-9 px-4 text-sm rounded-md',
        lg: 'h-11 px-6 text-[0.9375rem] rounded-md',
        icon: 'h-8 w-8 rounded-sm',
        'icon-lg': 'h-10 w-10 rounded-md',
      },
    },
    defaultVariants: { variant: 'secondary', size: 'md' },
  }
)

export interface ButtonProps
  extends ButtonHTMLAttributes<HTMLButtonElement>, VariantProps<typeof button> {
  asChild?: boolean
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild, ...props }, ref) => {
    const Comp = asChild ? Slot : 'button'
    return <Comp ref={ref} className={cn(button({ variant, size }), className)} {...props} />
  }
)
Button.displayName = 'Button'

export { button as buttonVariants }
