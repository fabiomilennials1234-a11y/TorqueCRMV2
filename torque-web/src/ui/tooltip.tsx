import * as RT from '@radix-ui/react-tooltip'
import type { ReactNode } from 'react'
import { Kbd } from './kbd'
import { cn } from '@/lib/utils'

export const TooltipProvider = RT.Provider

export function Tooltip({
  children,
  content,
  shortcut,
  side = 'bottom',
  delayDuration = 250,
}: {
  children: ReactNode
  content: ReactNode
  shortcut?: string | undefined
  side?: 'top' | 'bottom' | 'left' | 'right'
  delayDuration?: number
}) {
  return (
    <RT.Root delayDuration={delayDuration}>
      <RT.Trigger asChild>{children}</RT.Trigger>
      <RT.Portal>
        <RT.Content
          side={side}
          sideOffset={6}
          className={cn(
            'z-50 select-none rounded-sm bg-elevated px-2 py-1 text-xs text-ink shadow-elev-2',
            'data-[state=delayed-open]:animate-scale-in',
            'flex items-center gap-2'
          )}
        >
          <span>{content}</span>
          {shortcut && <Kbd>{shortcut}</Kbd>}
        </RT.Content>
      </RT.Portal>
    </RT.Root>
  )
}
