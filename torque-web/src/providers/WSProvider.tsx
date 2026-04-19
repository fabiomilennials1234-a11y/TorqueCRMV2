import { createContext, useContext, useState, type ReactNode } from 'react'
import type { WSStatus } from '@/lib/ws'

// ---------------------------------------------------------------------------
// Context
// ---------------------------------------------------------------------------

interface WSContextValue {
  status: WSStatus
}

const WSContext = createContext<WSContextValue | null>(null)

// ---------------------------------------------------------------------------
// Provider
//
// Mock-mode: exposes "disconnected" until the backend WebSocket endpoint
// exists and `TorqueWS.connect()` is wired up.
// ---------------------------------------------------------------------------

interface WSProviderProps {
  children: ReactNode
}

export function WSProvider({ children }: WSProviderProps) {
  const [status] = useState<WSStatus>('disconnected')

  return <WSContext.Provider value={{ status }}>{children}</WSContext.Provider>
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export function useWS(): WSContextValue {
  const ctx = useContext(WSContext)
  if (!ctx) {
    // Graceful fallback — safe to use outside provider
    return { status: 'disconnected' }
  }
  return ctx
}
