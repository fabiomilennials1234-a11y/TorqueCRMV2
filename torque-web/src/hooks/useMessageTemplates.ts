/**
 * Message templates hooks (S35).
 *
 * Leitura é member-wide; mutações (create/update/delete) exigem role
 * admin — o backend retorna 403 para membros comuns, que flui via
 * `notifyAppError` como toast.
 */

import { useQuery } from '@tanstack/react-query'

import { del, get, patch, post } from '@/api/client'
import { useAppMutation } from '@/hooks/useAppMutation'

export interface MessageTemplate {
  id: string
  name: string
  body: string
  variables: string[]
  is_active: boolean
  created_at: string
  updated_at: string
}

/** Lista templates do tenant; activeOnly é o padrão para o ComposerBar. */
export function useMessageTemplates(activeOnly = true) {
  return useQuery<MessageTemplate[]>({
    queryKey: ['message-templates', { activeOnly }],
    queryFn: async () => {
      const suffix = activeOnly ? '?active_only=1' : ''
      const res = await get<{ data: MessageTemplate[] }>(`/api/v1/message-templates${suffix}`)
      return res.data
    },
    staleTime: 60 * 1000,
  })
}

export function useCreateTemplate() {
  return useAppMutation<MessageTemplate, { name: string; body: string; variables?: string[] }>(
    (body) => post<MessageTemplate>('/api/v1/message-templates', body),
    {
      invalidate: [['message-templates']],
      errorContext: 'template.create',
    }
  )
}

export function useUpdateTemplate(id: string) {
  return useAppMutation<
    MessageTemplate,
    { name?: string; body?: string; variables?: string[]; is_active?: boolean }
  >((body) => patch<MessageTemplate>(`/api/v1/message-templates/${id}`, body), {
    invalidate: [['message-templates']],
    errorContext: 'template.update',
  })
}

export function useDeleteTemplate(id: string) {
  return useAppMutation<void, void>(() => del<void>(`/api/v1/message-templates/${id}`), {
    invalidate: [['message-templates']],
    errorContext: 'template.delete',
  })
}

/**
 * Resolve variáveis client-side usando `{{chave}}` como placeholder.
 * Variáveis não encontradas em `values` ficam intocadas para o usuário
 * ver que precisa preencher antes de enviar.
 */
export function renderTemplate(body: string, values: Record<string, string>): string {
  return body.replace(/\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}/g, (_, key: string) => {
    const v = values[key]
    return v != null && v !== '' ? v : `{{${key}}}`
  })
}
