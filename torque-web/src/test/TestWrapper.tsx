import type { ReactNode } from 'react'
import { TooltipProvider } from '@/ui/tooltip'

/**
 * Minimal provider wrapper for component tests.
 * Adds only the providers primitives that require (TooltipProvider for
 * any UI that uses the Tooltip primitive). Keep thin — if a test needs
 * Query/Intl/Auth, mount those locally so the default remains fast.
 */
export function TestWrapper({ children }: { children: ReactNode }) {
  return <TooltipProvider delayDuration={0}>{children}</TooltipProvider>
}
