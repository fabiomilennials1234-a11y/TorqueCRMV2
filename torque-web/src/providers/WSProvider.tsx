/**
 * WSProvider — real-backend edition (S06).
 *
 * Boot sequence:
 *   1. `useBootstrap()` resolves `/api/bootstrap` → provides `ws_url`.
 *   2. The moment `ws_url` is known AND the user is authenticated, we call
 *      `TorqueWS.getInstance().connect(ws_url)`.
 *   3. On logout (auth:logout event) or unauthenticated state, we tear the
 *      socket down.
 *
 * The provider publishes the current status; consumers render a badge
 * ("Conectado" / "Reconectando" / "Desconectado") from it.
 */

import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'

import { useBootstrap } from '@/hooks/useBootstrap'
import { TorqueWS, type WSStatus } from '@/lib/ws'
import { useAuth } from '@/providers/AuthProvider'

interface WSContextValue {
  status: WSStatus
}

const WSContext = createContext<WSContextValue | null>(null)

interface WSProviderProps {
  children: ReactNode
}

export function WSProvider({ children }: WSProviderProps) {
  const { isAuthenticated } = useAuth()
  const bootstrap = useBootstrap()
  const [status, setStatus] = useState<WSStatus>('disconnected')

  const wsURL = bootstrap.data?.ws_url
  const shouldConnect = isAuthenticated && Boolean(wsURL)

  useEffect(() => {
    const ws = TorqueWS.getInstance()
    const unsub = ws.onStatusChange(setStatus)
    return unsub
  }, [])

  useEffect(() => {
    const ws = TorqueWS.getInstance()
    if (shouldConnect && wsURL) {
      ws.connect(wsURL)
    } else {
      ws.close()
    }
  }, [shouldConnect, wsURL])

  useEffect(() => {
    const handleForceLogout = () => TorqueWS.getInstance().close()
    window.addEventListener('auth:logout', handleForceLogout)
    return () => window.removeEventListener('auth:logout', handleForceLogout)
  }, [])

  return <WSContext.Provider value={{ status }}>{children}</WSContext.Provider>
}

export function useWS(): WSContextValue {
  const ctx = useContext(WSContext)
  if (!ctx) return { status: 'disconnected' }
  return ctx
}
