/**
 * AppError → user-facing message pipeline.
 *
 * The client throws AppError with a machine `code` + server `message`. For
 * known codes we prefer a hand-crafted PT-BR string; for everything else we
 * fall back to the server message (already localizable on the backend side).
 *
 * `notifyAppError` is the one place that touches the toast surface. Call it
 * from error boundaries, mutation `onError`, and effect error handlers — do
 * NOT scatter `alert()` or ad-hoc logging in components.
 */

import { AppError } from './client'

// Fallback map — tune as we discover common UX-affecting codes.
const FRIENDLY_MESSAGES: Record<string, string> = {
  AUTH_EXPIRED: 'Sua sessão expirou. Faça login novamente.',
  UNAUTHENTICATED: 'Autenticação necessária.',
  PERMISSION_DENIED: 'Você não tem permissão para essa ação.',
  CSRF_MISMATCH: 'Token de segurança inválido. Recarregue a página e tente novamente.',
  CSRF_MISSING_COOKIE: 'Cookie de segurança ausente. Recarregue a página.',
  CSRF_MISSING_HEADER: 'Requisição rejeitada por segurança. Recarregue a página.',
  RATE_LIMITED: 'Muitas requisições. Aguarde um instante antes de tentar de novo.',
  TENANT_FIELD_FORBIDDEN: 'Erro interno: tente de novo em alguns segundos.',
  NO_TENANT: 'Sua conta não está associada a nenhuma organização ativa.',
  NO_MEMBERSHIP: 'Sua conta não está associada a nenhuma organização ativa.',
  INVALID_CREDENTIALS: 'E-mail ou senha incorretos.',
  INVALID_BODY: 'Não foi possível processar a requisição.',
  NOT_FOUND: 'Não encontrado.',
  INTERNAL: 'Algo deu errado. Tente de novo; se persistir, fale com o suporte.',
  // S51 — quota runtime.
  QUOTA_EXCEEDED:
    'Você atingiu o limite do seu plano para esse recurso. Atualize o plano ou compre add-ons em Configurações → Plano e faturamento.',
  QUOTA_LOOKUP_FAILED: 'Não foi possível conferir o limite do plano. Tente de novo em instantes.',
}

export function friendlyMessage(err: unknown): string {
  if (err instanceof AppError) {
    return FRIENDLY_MESSAGES[err.code] ?? err.message
  }
  if (err instanceof Error) {
    return err.message
  }
  return 'Ocorreu um erro inesperado.'
}

/**
 * Dispatches a custom event that a toast provider picks up. We avoid a direct
 * dependency on any toast library here so the helper works in any surface
 * (including tests, where the event is asserted as-is).
 */
export function notifyAppError(err: unknown, context?: string): void {
  if (typeof window === 'undefined') return
  const message = friendlyMessage(err)
  const code = err instanceof AppError ? err.code : 'UNKNOWN'
  const status = err instanceof AppError ? err.status : 0
  window.dispatchEvent(
    new CustomEvent('torque:toast', {
      detail: { level: 'error', code, status, message, context: context ?? '' },
    })
  )
}

/**
 * Distinguish recoverable errors we should silently retry from ones the user
 * must acknowledge. 5xx = yes, retry; 4xx = no, surface.
 */
export function isRetryable(err: unknown): boolean {
  if (!(err instanceof AppError)) return false
  if (err.status === 429) return true
  return err.status >= 500 && err.status < 600
}
