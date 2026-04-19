import { Navigate, Outlet } from 'react-router-dom'
import { useSession } from '@/hooks/useSession'

/**
 * Route guard that restricts access to master-only areas.
 * Non-master users are redirected to the root route.
 */
export function MasterRoute() {
  const { isMaster } = useSession()

  if (!isMaster) {
    return <Navigate to="/" replace />
  }

  return <Outlet />
}
