---
tags: [arquitetura, autenticacao, autorizacao, rbac, seguranca]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Autenticacao e Autorizacao

Autenticacao responde "quem voce e". Autorizacao responde "o que voce pode fazer". As duas camadas sao ortogonais, tratadas separadamente, e o frontend nunca inventa a resposta.

## Sessao: cookies, nao localStorage

- Access token: JWT em cookie **`__torque_session`**.
- Refresh token: rotativo em cookie **`__torque_refresh`**.
- Ambos: `HttpOnly; Secure; SameSite=Strict; Path=/`.
- Refresh `Path=/auth/refresh` para nao vazar em outras requisicoes.
- **O frontend nunca toca nos tokens.** Nao le, nao escreve, nao parseia, nao armazena. Esse e o ponto inteiro.
- Todo `fetch` usa `credentials: 'include'`. Sem excecao.

## Interceptor de 401

```ts
// src/lib/fetch.ts (pseudocodigo)
if (res.status === 401 && !req.url.endsWith('/auth/refresh')) {
  const refreshed = await attemptRefresh(); // POST /auth/refresh, uma vez
  if (refreshed) return retry(req);
  logout(); // limpa React Query, redireciona /login
}
```

- **Uma unica tentativa** de refresh por requisicao. Deduplicacao global: se 10 requisicoes cairem em 401 simultaneamente, apenas uma chama `/auth/refresh`.
- Falha no refresh: logout imediato + redirect + limpeza total do React Query cache.

## CSRF

Para mutations sensiveis (delete, pagamento, mudanca de billing, convite de membro):
- Header `X-CSRF-Token` com valor rotativo.
- Token vive em `<meta name="csrf-token" content="...">` injetado pelo servidor Go no HTML inicial.
- Backend valida double-submit: cookie nao-HttpOnly `__torque_csrf` + header devem bater.
- Rotaciona a cada `/auth/refresh`.

GET idempotentes nao precisam de CSRF. Mutations sempre precisam. O middleware do Go rejeita com 403 se ausente em POST/PATCH/DELETE.

## RBAC em 4 camadas

A decisao "pode fazer X?" e uma composicao em cascata. Cada camada pode **negar**, so a primeira pode **permitir implicitamente**.

1. **Master Admin** (`users_master`) — bypass cross-organization. Invisivel para clientes. Nao aparece em listagens de `team`. Usado para suporte e auditoria interna. Toda acao como master grava audit log especial (`actor_type: 'master'`).
2. **Admin da organizacao** (`team_members.role = 'admin'`) — pode tudo dentro da org, exceto o que a feature marcar como `master_only`.
3. **Feature permissions globais** (`feature_permissions`) — por feature_key, campos `is_admin_only: bool` e `default_value: bool`. Define o comportamento da feature no nivel da org.
4. **Member feature permissions** (`member_feature_permissions`) — override por membro, sobrescreve `default_value` da camada 3 para aquele usuario especifico.

Resolucao efetiva (ordem):
```
if is_master -> allow (com audit especial)
if is_admin and not feature.master_only -> allow
if feature.is_admin_only and not is_admin -> deny
if member_override exists -> member_override.value
else -> feature.default_value
```

## Invariante de roles

Em codigo, `role` so assume tres valores: **`admin`**, **`master`**, **`membro`**.

- SDR, Closer, BDR, Gerente, Vendedor sao **conceitos de negocio**, representados via tags, labels ou atribuicao funcional. **Nunca** sao roles.
- Se alguem propor `role: 'sdr'`, a resposta e nao. Isso inflaria o RBAC e quebraria a cascata.

## Hooks de frontend

| Hook | Retorno | Uso |
|---|---|---|
| `useSession()` | `{ user, org, role, is_master } \| null` | Gate de rotas, header, avatar |
| `useFeaturePermissions()` | `Record<FeatureKey, boolean>` | Ocultar itens de menu, desabilitar CTAs |
| `usePermission(featureKey)` | `boolean` | Shortcut para uma feature |
| `useCanPerformAction(action)` | `boolean` | Acoes compostas (ex: "delete_lead" = feature + ownership) |
| `useQuota(resource)` | `{ effective_limit, current_usage, can_add }` | CTAs de criacao, banners de limite |

Todos populados **uma unica vez** via `GET /auth/me` no boot. React Query `staleTime: Infinity`, invalidados apenas em:
- Logout.
- Evento WS `permissions.changed` (ou `quota.changed`).
- Refresh explicito via `queryClient.invalidateQueries(['session'])` apos mudancas que o proprio usuario fez.

## Bootstrap: um unico round-trip

```
AppShell monta
  -> dispara GET /auth/me
  -> loading screen minimalista (logo + shimmer, nao spinner)
  -> resposta popula QueryClient com tudo
  -> children renderizam com dados ja disponiveis
```

Nao e aceitavel waterfall: `useSession` -> `usePermissions` -> `useQuotas` em tres chamadas. Um bundle, um trip.

## Payload `/auth/me`

```json
{
  "user": {
    "id": "usr_...",
    "name": "Fabio",
    "email": "fabio@...",
    "avatar_url": "https://storage.torque.app/...",
    "is_master": false
  },
  "org": {
    "id": "org_...",
    "name": "Acme",
    "plan_id": "growth",
    "logo_url": "https://storage.torque.app/..."
  },
  "role": "admin",
  "feature_permissions": {
    "leads.create": true,
    "leads.delete": true,
    "billing.manage": true,
    "workflows.edit": true,
    "copilot.configure": false
  },
  "quotas": {
    "leads": { "effective_limit": 5000, "current_usage": 1243, "can_add": true },
    "team_members": { "effective_limit": 10, "current_usage": 7, "can_add": true },
    "workflows": { "effective_limit": 20, "current_usage": 20, "can_add": false }
  },
  "csrf_token": "..."
}
```

## Logout

- `POST /auth/logout` invalida refresh token no servidor (blacklist).
- Cliente: `queryClient.clear()`, remove CSRF meta, redirect hard `/login` (nao SPA nav, forca reload para descartar qualquer estado residual).

## Referencias

- [[Arquitetura do Sistema]]
- [[Contratos e Boundaries]]
- [[Multi-tenancy]]
- [[Seguranca Web]]
- [[Realtime e Jobs]]
- [[Glossario e Vocabulario]]
