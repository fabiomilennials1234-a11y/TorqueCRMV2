import * as RT from '@radix-ui/react-tabs'
import type { ComponentPropsWithoutRef } from 'react'
import { cn } from '@/lib/utils'

export const Tabs = RT.Root

export function TabsList({ className, ...props }: ComponentPropsWithoutRef<typeof RT.List>) {
  return (
    <RT.List
      className={cn('shadow-hairline-b inline-flex items-center gap-1', className)}
      {...props}
    />
  )
}

export function TabsTrigger({ className, ...props }: ComponentPropsWithoutRef<typeof RT.Trigger>) {
  return (
    <RT.Trigger
      className={cn(
        'text-ink-muted relative inline-flex h-9 items-center gap-2 px-3 text-sm font-medium',
        'hover:text-ink transition-colors',
        'data-[state=active]:text-ink',
        'data-[state=active]:after:absolute data-[state=active]:after:inset-x-0 data-[state=active]:after:-bottom-px',
        'data-[state=active]:after:bg-accent data-[state=active]:after:h-px',
        'disabled:pointer-events-none disabled:opacity-40',
        className
      )}
      {...props}
    />
  )
}

export function TabsContent({ className, ...props }: ComponentPropsWithoutRef<typeof RT.Content>) {
  return (
    <RT.Content
      className={cn(
        'data-[state=active]:animate-fade-in mt-4 focus-visible:outline-none',
        className
      )}
      {...props}
    />
  )
}
