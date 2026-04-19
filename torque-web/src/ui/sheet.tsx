import * as RD from '@radix-ui/react-dialog'
import { X } from 'lucide-react'
import type { ComponentPropsWithoutRef, ReactNode } from 'react'
import { cn } from '@/lib/utils'

export const Sheet = RD.Root
export const SheetTrigger = RD.Trigger
export const SheetClose = RD.Close

export function SheetContent({
  className,
  children,
  side = 'right',
  title,
  description,
  ...props
}: ComponentPropsWithoutRef<typeof RD.Content> & {
  side?: 'left' | 'right'
  title?: string
  description?: string
  children: ReactNode
}) {
  return (
    <RD.Portal>
      <RD.Overlay className="fixed inset-0 z-40 bg-bg/50 backdrop-blur-[16px] data-[state=open]:animate-fade-in" />
      <RD.Content
        className={cn(
          'fixed top-0 z-50 h-full w-full max-w-[520px] bg-surface',
          'shadow-[inset_1px_0_0_0_hsl(var(--hairline))] shadow-elev-3',
          'data-[state=open]:duration-300 data-[state=open]:animate-in data-[state=open]:slide-in-from-right',
          'data-[state=closed]:duration-200 data-[state=closed]:animate-out data-[state=closed]:slide-out-to-right',
          side === 'left' &&
            'left-0 shadow-[inset_-1px_0_0_0_hsl(var(--hairline))] data-[state=closed]:slide-out-to-left data-[state=open]:slide-in-from-left',
          side === 'right' && 'right-0',
          'flex flex-col',
          className
        )}
        {...props}
      >
        {(title || description) && (
          <div className="flex items-start justify-between gap-4 px-6 py-5 shadow-hairline-b">
            <div>
              {title && (
                <RD.Title className="font-display text-lg tracking-tightest text-ink">
                  {title}
                </RD.Title>
              )}
              {description && (
                <RD.Description className="mt-1 text-sm text-ink-muted">
                  {description}
                </RD.Description>
              )}
            </div>
            <RD.Close className="rounded-sm p-1 text-ink-dim transition-colors hover:bg-elevated hover:text-ink">
              <X className="h-4 w-4" />
            </RD.Close>
          </div>
        )}
        {children}
      </RD.Content>
    </RD.Portal>
  )
}
