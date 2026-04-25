import * as RA from '@radix-ui/react-avatar'
import type { ComponentPropsWithoutRef } from 'react'
import { cn } from '@/lib/utils'

const sizes = {
  xs: 'h-5 w-5 text-[0.625rem]',
  sm: 'h-6 w-6 text-[0.6875rem]',
  md: 'h-8 w-8 text-xs',
  lg: 'h-10 w-10 text-sm',
} as const

export interface AvatarProps extends ComponentPropsWithoutRef<typeof RA.Root> {
  size?: keyof typeof sizes
  src?: string
  fallback: string
  ring?: boolean
}

export function Avatar({
  size = 'md',
  src,
  fallback,
  ring = false,
  className,
  ...props
}: AvatarProps) {
  return (
    <RA.Root
      className={cn(
        'font-metric relative inline-flex shrink-0 overflow-hidden rounded-full uppercase select-none',
        sizes[size],
        ring && 'shadow-hairline ring-bg ring-2',
        className
      )}
      {...props}
    >
      {src && <RA.Image src={src} alt={fallback} className="h-full w-full object-cover" />}
      <RA.Fallback
        className="bg-elevated text-ink-muted flex h-full w-full items-center justify-center"
        delayMs={200}
      >
        {fallback.slice(0, 2)}
      </RA.Fallback>
    </RA.Root>
  )
}
