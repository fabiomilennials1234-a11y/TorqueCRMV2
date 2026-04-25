import { Navigate, Outlet } from 'react-router-dom'
import { useAuth } from '@/providers/AuthProvider'

/**
 * Route guard that redirects unauthenticated users to /login.
 * While the session is loading, renders a fullscreen shimmer skeleton.
 */
export function ProtectedRoute() {
  const { isAuthenticated, isLoading } = useAuth()

  if (isLoading) {
    return (
      <div className="bg-bg flex h-screen w-screen items-center justify-center">
        <div
          className="animate-shimmer bg-surface from-surface via-elevated to-surface h-8 w-48 rounded bg-gradient-to-r bg-[length:200%_100%]"
          aria-label="Carregando..."
        />
      </div>
    )
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return <Outlet />
}
