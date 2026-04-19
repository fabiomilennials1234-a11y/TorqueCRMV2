/**
 * `useLogin` — login mutation.
 *
 * Posts to `/api/v1/auth/login`. On success, invalidates the session key so
 * `AuthProvider` refetches `/auth/me`. On failure, the AppError surfaces via
 * the standard toast pipeline (or inline, if the caller reads mutation.error).
 */

import { post } from '@/api/client'
import { queryKeys } from '@/api/queryKeys'
import { useAppMutation } from '@/hooks/useAppMutation'

export interface LoginPayload {
  email: string
  password: string
  organization_slug?: string
}

export interface LoginResponse {
  user: { id: string; email: string; display_name: string }
  organization: { id: string; slug: string; name: string }
}

export function useLogin() {
  return useAppMutation<LoginResponse, LoginPayload>(
    (vars) => post<LoginResponse>('/api/v1/auth/login', vars),
    {
      invalidate: [queryKeys.session.me(), queryKeys.session.bootstrap()],
      errorContext: 'auth.login',
      // Keep the error available for inline rendering in <LoginPage>.
      silent: true,
    }
  )
}
