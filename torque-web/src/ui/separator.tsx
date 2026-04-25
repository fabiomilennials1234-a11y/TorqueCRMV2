import * as RS from '@radix-ui/react-separator'
import type { ComponentPropsWithoutRef } from 'react'
import { cn } from '@/lib/utils'

export function Separator({
  className,
  orientation = 'horizontal',
  ...props
}: ComponentPropsWithoutRef<typeof RS.Root>) {
  return (
    <RS.Root
      orientation={orientation}
      className={cn(
        'bg-hairline shrink-0',
        orientation === 'horizontal' ? 'h-px w-full' : 'h-full w-px',
        className
      )}
      {...props}
    />
  )
}
