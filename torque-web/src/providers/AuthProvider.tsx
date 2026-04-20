/**
 * AuthProvider — real-backend edition (S06).
 *
 * The provider owns nothing except the `SessionBundle | null` and the
 * `isLoading` flag during the initial `/auth/me` fetch. TanStack Query
 * is the cache of record; `queryKeys.session.me()` holds the authoritative
 * copy, and consumers may either read from this provider (ergonomic) or
 * query the key directly (when they need `isFetching`, etc).
 *
 * The mock session from earlier sprints is gone. A 401 from `/auth/me`
 * (either on bootstrap or after a refresh loss) sets the state to null
 * and redirects to `/login`. No silent fall-through to fake data — the
 * UI must know when it is logged out.
 */

import { createContext, useCallback, useContext, useEffect, useMemo, type ReactNode } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'

import { AppError, get, post } from '@/api/client'
import { queryKeys } from '@/api/queryKeys'
import { DEV_SESSION, isDevAuthEnabled } from '@/providers/devSession'
import { queryClient } from '@/providers/QueryProvider'
import { router } from '@/routes'
import type { SessionBundle } from '@/contracts/manual'

interface AuthContextValue {
  session: SessionBundle | null
  isLoading: boolean
  isAuthenticated: boolean
  logout: () => Promise<void>
  refresh: () => Promise<SessionBundle | null>
}

const AuthContext = createContext<AuthContextValue | null>(null)

function navigateToLogin(): void {
  void router.navigate('/login', { replace: true })
}

/**
 * The server returns `/auth/me` in the shape defined by OpenAPI: user +
 * organization + permissions + csrf_token + is_master. We convert that to
 * the legacy `SessionBundle` so the rest of the shell keeps working during
 * the integration sprint. Once pages consume the server shape natively,
 * `SessionBundle` can be retired.
 */
interface MeResponse {
  user: {
    id: string
    email: string
    display_name: string
    role: 'admin' | 'membro'
    ui_mode: 'manager' | 'salesperson'
  }
  organization: {
    id: string
    slug: string
    name: string
    plan_id: string | null
    payment_status: 'active' | 'overdue' | 'suspended' | 'cancelled'
    logo_url: string | null
  }
  permissions: Array<{ key: string; allowed: boolean; source: string }>
  csrf_token: string
  is_master: boolean
}

function toSessionBundle(me: MeResponse): SessionBundle {
  const featurePermissions: Record<string, boolean> = {}
  for (const p of me.permissions) featurePermissions[p.key] = p.allowed

  return {
    user: {
      id: me.user.id,
      email: me.user.email,
      displayName: me.user.display_name,
    },
    org: {
      id: me.organization.id,
      name: me.organization.name,
      slug: me.organization.slug,
      planId: me.organization.plan_id,
      paymentStatus: me.organization.payment_status,
      logoUrl: me.organization.logo_url,
    },
    role: me.is_master ? 'admin' : me.user.role === 'admin' ? 'admin' : 'membro',
    isMaster: me.is_master,
    featurePermissions,
    // Quotas arrive via a separate endpoint in S11+; empty for now.
    quotas: {},
  }
}

interface AuthProviderProps {
  children: ReactNode
}

export function AuthProvider({ children }: AuthProviderProps) {
  const client = useQueryClient()
  const devBypass = isDevAuthEnabled()

  const query = useQuery<SessionBundle, AppError>({
    queryKey: queryKeys.session.me(),
    queryFn: async () => {
      // Dev-only: skip the network entirely. The `if` is constant-folded
      // in prod builds because `isDevAuthEnabled` returns false when
      // `import.meta.env.DEV` is false.
      if (devBypass) return DEV_SESSION
      const me = await get<MeResponse>('/api/v1/auth/me')
      return toSessionBundle(me)
    },
    staleTime: 5 * 60 * 1000,
    retry: (failureCount, err) => {
      // 401 is never retried — RequireAuth + refresh handle in client.ts
      if (err instanceof AppError && err.status === 401) return false
      return failureCount < 2
    },
  })

  useEffect(() => {
    const handleForceLogout = () => {
      client.setQueryData<SessionBundle | null>(queryKeys.session.me(), null)
      client.clear()
      navigateToLogin()
    }
    window.addEventListener('auth:logout', handleForceLogout)
    return () => window.removeEventListener('auth:logout', handleForceLogout)
  }, [client])

  const logout = useCallback(async () => {
    try {
      await post('/api/v1/auth/logout')
    } catch {
      // Best-effort — clear local state regardless.
    }
    client.setQueryData<SessionBundle | null>(queryKeys.session.me(), null)
    client.clear()
    navigateToLogin()
  }, [client])

  const refresh = useCallback(async (): Promise<SessionBundle | null> => {
    const result = await query.refetch()
    return result.data ?? null
  }, [query])

  const session =
    query.isSuccess && query.data
      ? query.data
      : query.error instanceof AppError && query.error.status === 401
        ? null
        : null

  const value = useMemo<AuthContextValue>(
    () => ({
      session,
      isLoading: query.isLoading,
      isAuthenticated: session !== null,
      logout,
      refresh,
    }),
    [session, query.isLoading, logout, refresh]
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error('useAuth must be used within <AuthProvider>')
  }
  return ctx
}

// `queryClient` is re-exported for consumers that need to invalidate the
// session key imperatively after a login mutation. Import from here so the
// key factory stays colocated with the provider that owns it.
export { queryClient }
