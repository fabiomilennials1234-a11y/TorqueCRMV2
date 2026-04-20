/**
 * Forces unfinished onboarding through /onboarding before any inner route
 * is rendered. Runs inside ProtectedRoute so the session is guaranteed.
 *
 * If the onboarding fetch fails we *do not* block — we let the user into
 * the app with the error surfaced globally. A hard block on a transient
 * network error would be a nasty UX regression on retention.
 */

import { Navigate, Outlet, useLocation } from 'react-router-dom'

import { shouldGate, useOnboarding } from '@/hooks/useOnboarding'

export function OnboardingGate() {
  const location = useLocation()
  const status = useOnboarding()

  // Don't block while loading — the dashboard skeleton is cheap. If the
  // user is already on /onboarding, show the outlet so we don't redirect
  // into ourselves.
  if (location.pathname.startsWith('/onboarding')) {
    return <Outlet />
  }

  // Hard errors / loading → let the app through. Toast carries the error.
  if (status.isError || !status.data) {
    return <Outlet />
  }

  if (shouldGate(status.data)) {
    return <Navigate to="/onboarding" replace />
  }

  return <Outlet />
}
