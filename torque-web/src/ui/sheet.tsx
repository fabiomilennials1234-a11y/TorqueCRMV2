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
      <RD.Overlay className="bg-bg/50 data-[state=open]:animate-fade-in fixed inset-0 z-40 backdrop-blur-[16px]" />
      <RD.Content
        className={cn(
          'bg-surface fixed top-0 z-50 h-full w-full max-w-[520px]',
          'shadow-elev-3 shadow-[inset_1px_0_0_0_hsl(var(--hairline))]',
          'data-[state=open]:animate-in data-[state=open]:slide-in-from-right data-[state=open]:duration-300',
          'data-[state=closed]:animate-out data-[state=closed]:slide-out-to-right data-[state=closed]:duration-200',
          side === 'left' &&
            'data-[state=closed]:slide-out-to-left data-[state=open]:slide-in-from-left left-0 shadow-[inset_-1px_0_0_0_hsl(var(--hairline))]',
          side === 'right' && 'right-0',
          'flex flex-col',
          className
        )}
        {...props}
      >
        {(title || description) && (
          <div className="shadow-hairline-b flex items-start justify-between gap-4 px-6 py-5">
            <div>
              {title && (
                <RD.Title className="font-display tracking-tightest text-ink text-lg">
                  {title}
                </RD.Title>
              )}
              {description && (
                <RD.Description className="text-ink-muted mt-1 text-sm">
                  {description}
                </RD.Description>
              )}
            </div>
            <RD.Close className="text-ink-dim hover:bg-elevated hover:text-ink rounded-sm p-1 transition-colors">
              <X className="h-4 w-4" />
            </RD.Close>
          </div>
        )}
        {children}
      </RD.Content>
    </RD.Portal>
  )
}
