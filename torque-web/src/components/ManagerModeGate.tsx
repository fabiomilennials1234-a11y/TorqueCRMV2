import { Navigate, Outlet } from 'react-router-dom'
import { useUiMode } from '@/providers/UiModeProvider'

/**
 * Gate frontend do modo Gerente. Redireciona para /cockpit quando:
 * - o usuário não pode alternar para manager (falta ui.view_manager_mode);
 * - a preferência ativa é salesperson.
 *
 * Nota: apenas UX. O servidor aplica RBAC independente do modo.
 */
export function ManagerModeGate() {
  const { mode, canSwitchToManager } = useUiMode()

  if (!canSwitchToManager) {
    return <Navigate to="/cockpit" replace />
  }
  if (mode === 'salesperson') {
    return <Navigate to="/cockpit" replace />
  }
  return <Outlet />
}
