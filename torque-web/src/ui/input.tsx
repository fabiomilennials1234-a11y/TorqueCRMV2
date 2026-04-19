import { forwardRef, type InputHTMLAttributes } from 'react'
import { cn } from '@/lib/utils'

export const Input = forwardRef<HTMLInputElement, InputHTMLAttributes<HTMLInputElement>>(
  ({ className, ...props }, ref) => (
    <input
      ref={ref}
      className={cn(
        'flex h-9 w-full rounded-md bg-transparent px-3 py-2 text-sm',
        'shadow-hairline placeholder:text-ink-dim',
        'focus-visible:shadow-neu-focus',
        'transition-shadow duration-150 focus-visible:outline-none',
        'disabled:pointer-events-none disabled:opacity-40',
        className
      )}
      {...props}
    />
  )
)
Input.displayName = 'Input'
