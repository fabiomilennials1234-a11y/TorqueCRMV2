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
  // S56 — `bottom` slides from bottom (mobile sheet). `fullscreen` covers viewport (mobile dialog).
  side?: 'left' | 'right' | 'bottom' | 'fullscreen'
  title?: string
  description?: string
  children: ReactNode
}) {
  const isHorizontal = side === 'left' || side === 'right'
  return (
    <RD.Portal>
      <RD.Overlay className="bg-bg/50 data-[state=open]:animate-fade-in fixed inset-0 z-40 backdrop-blur-[16px]" />
      <RD.Content
        className={cn(
          'bg-surface fixed z-50 flex flex-col',
          'shadow-elev-3',
          // Horizontal side sheets (default desktop pattern).
          isHorizontal && 'top-0 h-full w-full max-w-[520px] shadow-[inset_1px_0_0_0_hsl(var(--hairline))]',
          side === 'right' &&
            'right-0 data-[state=open]:animate-in data-[state=open]:slide-in-from-right data-[state=open]:duration-300 data-[state=closed]:animate-out data-[state=closed]:slide-out-to-right data-[state=closed]:duration-200',
          side === 'left' &&
            'left-0 shadow-[inset_-1px_0_0_0_hsl(var(--hairline))] data-[state=open]:animate-in data-[state=open]:slide-in-from-left data-[state=open]:duration-300 data-[state=closed]:animate-out data-[state=closed]:slide-out-to-left data-[state=closed]:duration-200',
          // S56 — bottom sheet (mobile-first). Pinned to bottom, max 90dvh, rounded top, safe-area padding.
          side === 'bottom' &&
            'pb-safe inset-x-0 bottom-0 max-h-[90dvh] w-full rounded-t-xl shadow-[inset_0_1px_0_0_hsl(var(--hairline))] data-[state=open]:animate-in data-[state=open]:slide-in-from-bottom data-[state=open]:duration-300 data-[state=closed]:animate-out data-[state=closed]:slide-out-to-bottom data-[state=closed]:duration-200',
          // S56 — fullscreen sheet (mobile dialog substitute). 100dvh, edge-to-edge, safe-area on all sides.
          side === 'fullscreen' &&
            'pt-safe pb-safe inset-0 h-[100dvh] w-full data-[state=open]:animate-in data-[state=open]:fade-in data-[state=open]:duration-200 data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=closed]:duration-150',
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
            <RD.Close className="text-ink-dim hover:bg-elevated hover:text-ink min-h-[44px] min-w-[44px] rounded-sm p-1 transition-colors">
              <X className="mx-auto h-4 w-4" />
            </RD.Close>
          </div>
        )}
        {children}
      </RD.Content>
    </RD.Portal>
  )
}
