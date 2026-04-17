---
tags: [arquitetura, contratos, api, openapi]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Contratos e Boundaries

A fronteira entre frontend e backend e um contrato. Nao uma sugestao, nao uma convencao verbal. Um arquivo versionado.

## Fonte unica de verdade

O backend Go gera o `openapi.yaml` a partir das definicoes de rota. O frontend **consome**, nunca define.

```
Go handlers + structs  -->  openapi.yaml (commitado)  -->  src/contracts/api.gen.ts
```

- Geracao client-side via `openapi-typescript` em step de build (`pnpm gen:api`).
- `api.gen.ts` esta no `.gitignore`? **Nao.** E commitado para CI lint e para PRs mostrarem drift de contrato.
- Editar `api.gen.ts` manualmente e erro de revisao. CI falha se hash do arquivo nao bate com hash gerado.

## Serializacao

- Wire format: JSON `snake_case`. Sempre.
- No cliente: tipos gerados sao `snake_case` (espelho fiel do wire), transformacao para `camelCase` acontece na camada `src/api/`.
- Camada `api/` e a unica que ve `snake_case`. Hooks e componentes so veem `camelCase`.
- Datas: ISO 8601 UTC com sufixo `Z`. Nunca epoch, nunca timezone local no wire.
- Dinheiro: inteiro em centavos (`amount_cents: number`), `currency: "BRL"`. Zero float em dinheiro.

## Paginacao

**Cursor-based obrigatoria.** Offset/limit e proibido (nao escala, duplica em writes concorrentes).

```ts
type Page<T> = {
  items: T[];
  next_cursor: string | null;
  total?: number; // opcional, backend so inclui se barato
};
```

Request: `GET /leads?cursor=<opaque>&limit=50`. Cursor e opaco (base64 de `{id, created_at}`), cliente nao interpreta.

## Filtros e ordenacao

- Filtros: `filter[campo]=valor`. Multiplos valores: `filter[stage_id]=a,b,c`.
- Ranges: `filter[created_at.gte]=2026-01-01`.
- Sort: `sort=-created_at,name`. Prefixo `-` e descendente.
- Busca textual: `q=texto` (backend decide fulltext vs ilike).

## Long-running operations

Qualquer operacao que passe de ~800ms **nunca** bloqueia a requisicao HTTP. Padrao:

```
POST /campaigns/{id}/dispatch
  -> 202 Accepted
     { "operation_id": "op_...", "status": "queued" }

GET /operations/op_...
  -> 200 { id, type, status, started_at, ended_at, error, progress }
```

- Cliente faz poll a cada 2s, timeout de 5min, backoff em erro.
- Para batches curtos conhecidos (< 10s), request aceita `?push=true`: servidor responde 202 imediatamente e emite WS event `operation.completed` com o resultado. Cliente escolhe entre poll e push.
- Hook padrao: `useOperation(operationId)` encapsula poll + push + cleanup.
- Ver [[Realtime e Jobs]].

## Erros

Formato unico:

```json
{
  "error": {
    "code": "quota_exceeded",
    "message": "Limite de leads atingido para o plano atual.",
    "details": { "resource": "leads", "limit": 1000, "current": 1000 }
  }
}
```

- `code` e enum estavel (snake_case), usado para i18n e branching no cliente.
- `message` e PT-BR, pronto para exibir. Nao inventar mensagem no cliente.
- HTTP status reflete categoria (400/401/403/404/409/422/429/500).

## Entidades canonicas

| Entidade | Campos-chave | Endpoints |
|---|---|---|
| Organization | `id`, `name`, `plan_id`, `created_at` | `GET /org`, `PATCH /org` |
| TeamMember | `id`, `user_id`, `organization_id`, `role` (admin/membro), `created_at` | `GET /team`, `POST /team/invite`, `PATCH /team/:id`, `DELETE /team/:id` |
| Lead | `id`, `organization_id`, `stage_id`, `owner_id`, `name`, `phone`, `email`, `score`, `source`, `version` | `GET/POST /leads`, `PATCH/DELETE /leads/:id`, `POST /leads/:id/move` |
| PipeWhatsApp | `id`, `organization_id`, `lead_id`, `status`, `last_message_at` | `GET /pipes/whatsapp`, `PATCH /pipes/whatsapp/:id` |
| PipeConfirmacao | `id`, `organization_id`, `lead_id`, `stage`, `confirmed_at` | `GET /pipes/confirmacao`, `POST /pipes/confirmacao/:id/confirm` |
| PipeProposta | `id`, `organization_id`, `lead_id`, `amount_cents`, `status`, `sent_at` | `GET /pipes/propostas`, `POST /pipes/propostas`, `PATCH /pipes/propostas/:id` |
| Conversation | `id`, `organization_id`, `lead_id`, `channel`, `last_message_at`, `unread_count` | `GET /conversations`, `GET /conversations/:id` |
| ChannelMessage | `id`, `conversation_id`, `direction`, `body`, `media_url`, `sent_at`, `read_at` | `GET /conversations/:id/messages`, `POST /conversations/:id/messages` |
| Workflow | `id`, `organization_id`, `name`, `trigger`, `steps`, `enabled` | `GET/POST /workflows`, `PATCH/DELETE /workflows/:id` |
| WorkflowExecution | `id`, `workflow_id`, `lead_id`, `status`, `started_at`, `ended_at` | `GET /workflows/:id/executions` |
| Campaign | `id`, `organization_id`, `name`, `audience_filter`, `template_id`, `scheduled_at`, `status` | `GET/POST /campaigns`, `POST /campaigns/:id/dispatch` |
| CopilotAgent | `id`, `organization_id`, `name`, `prompt`, `model`, `channels[]`, `enabled` | `GET/POST /copilot/agents`, `PATCH /copilot/agents/:id` |
| FollowUp | `id`, `organization_id`, `lead_id`, `owner_id`, `due_at`, `completed_at`, `note` | `GET/POST /followups`, `PATCH /followups/:id/complete` |
| Product | `id`, `organization_id`, `name`, `sku`, `price_cents`, `commission_pct` | `GET/POST /products`, `PATCH/DELETE /products/:id` |
| Commission | `id`, `organization_id`, `member_id`, `lead_id`, `amount_cents`, `paid_at` | `GET /commissions`, `POST /commissions/:id/pay` |
| OrgQuota | `organization_id`, `resource`, `plan_base`, `purchased_addons`, `admin_adjustment`, `current_usage` | `GET /org/quotas` (via `/auth/me`) |

Cada entidade com coluna `version` (monotonic bigint) participa do dedup de WebSocket. Ver [[Realtime e Jobs]].

## Referencias

- [[Arquitetura do Sistema]]
- [[Autenticacao e Autorizacao]]
- [[Realtime e Jobs]]
- [[Multi-tenancy]]
- [[Glossario e Vocabulario]]
