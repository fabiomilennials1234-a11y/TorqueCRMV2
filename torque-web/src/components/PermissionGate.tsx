import type { ReactNode } from 'react'
import { usePermission } from '@/hooks/usePermission'

interface PermissionGateProps {
  featureKey: string
  fallback?: ReactNode
  children: ReactNode
}

/**
 * Conditionally renders children based on a feature permission.
 * If the user lacks the permission, renders `fallback` (defaults to null).
 */
export function PermissionGate({ featureKey, fallback = null, children }: PermissionGateProps) {
  const allowed = usePermission(featureKey)

  if (!allowed) {
    return <>{fallback}</>
  }

  return <>{children}</>
}
