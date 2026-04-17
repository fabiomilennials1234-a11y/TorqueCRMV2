---
tags: [arquitetura, sistema, overview]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Arquitetura do Sistema

Torque é um SaaS B2B multi-tenant com frontend em React/TS/Vite e backend em Go. O vault documenta a intenção arquitetural agnóstica de stack: as decisões valem mesmo se amanhã trocarmos o runtime.

## Camadas

```
+---------------------------------------------------------------+
|  Apresentacao        React SPA (Vite, TS, React Query)        |
|                      src/features + src/hooks + src/api       |
+---------------------------------------------------------------+
              |  HTTPS/2 (fetch)           |  WSS (hub)
              v                            v
+---------------------------------------------------------------+
|  API Gateway         Go HTTP/2 server (chi/fiber)             |
|                      auth middleware, tenant-scope, RBAC      |
+---------------------------------------------------------------+
                          |
                          v
+---------------------------------------------------------------+
|  Dominio             Go services (use cases, invariantes)     |
|                      regras de negocio, permissoes efetivas   |
+---------------------------------------------------------------+
           |                    |                     |
           v                    v                     v
+-------------------+  +------------------+  +-------------------+
|  Persistencia     |  |  Realtime Hub    |  |  Jobs/Worker Pool |
|  Postgres + OS    |  |  WS fanout       |  |  async + poll     |
+-------------------+  +------------------+  +-------------------+
```

## Request flow (write)

```
[User click]
    |
    v
[Feature component]  -- dispara mutation do hook React Query
    |
    v
[useUpdateLead]      -- hook em src/hooks/, encapsula cache + optimistic
    |
    v
[api/leads.ts]       -- funcao tipada, serializa camelCase -> snake_case
    |
    v
[lib/fetch]          -- credentials:'include', interceptor 401, CSRF
    |
    v  HTTPS/2
[Go API Gateway]     -- valida JWT, extrai organization_id, RBAC
    |
    v
[Domain service]     -- valida invariantes, executa caso de uso
    |
    v
[Postgres]           -- commit transacional, emite evento
    |
    v
[Realtime Hub]       -- fanout WS para todos os clientes da org
    |
    v
[lib/ws.ts]          -- recebe patch, atualiza React Query cache
    |
    v
[UI re-render]       -- sem refetch, sem flicker
```

## Separacao de modulos do frontend

| Camada | Diretorio | Responsabilidade | Proibicoes |
|---|---|---|---|
| UI pura | `src/features/` | Componentes, layout, interacao | Nao chama fetch. Nao conhece backend. Nao tem logica de negocio. |
| Estado servidor | `src/hooks/` | React Query hooks (`useLeads`, `useUpdateLead`) | Nao manipula DOM. Nao contem regras de negocio alem de cache. |
| Cliente HTTP | `src/api/` | Funcoes tipadas que falam com endpoints | Nao tem estado. Nao conhece componentes. |
| Contratos | `src/contracts/` | Tipos gerados do OpenAPI (`api.gen.ts`) | **Nunca editado manualmente.** |
| Infra | `src/lib/fetch` `src/lib/ws` `src/lib/domain` | fetch base, socket, utilitarios de dominio puro | Sem dependencias de React. |

A dependencia flui em um sentido: `features -> hooks -> api -> contracts/lib`. Nunca o contrario.

## Proibicoes absolutas

- Nenhum SDK do backend anterior (Supabase/Edge Functions) em nenhum lugar do codigo. A migracao e total, nao gradual.
- Zero logica de negocio em componentes. Se precisa de `if role === 'admin'` em JSX, e sintoma: mova para `useCanPerformAction` ou componente guard.
- Tipos de seed/mock nunca vazam para hooks reais. Mocks vivem em `src/features/*/__mocks__` e sao consumidos apenas por Storybook/testes.
- `organization_id` nunca sai do frontend em mutations. O servidor extrai do JWT. Ver [[Multi-tenancy]].
- Proibido `any`, `as unknown as`, `@ts-ignore`. Se precisa, o contrato esta errado: corrija o OpenAPI.
- Proibido setState dentro de `useEffect` que dependa de dados de servidor. Use React Query.

## Decisoes transversais

- **Dark-first.** Ver [[Principios de Identidade Visual]]. Tema claro e override opcional, nao paridade visual.
- **Cursor-based pagination** em todas as listas. Offset e proibido. Ver [[Contratos e Boundaries]].
- **WS + optimistic update** e o caminho feliz. Polling so existe para long-running operations. Ver [[Realtime e Jobs]].
- **Servidor dita headers de seguranca.** Vite nao configura CSP. Ver [[Seguranca Web]].
- **JWT em cookie httpOnly.** Front nunca le token. Ver [[Autenticacao e Autorizacao]].

## Referencias

- [[Contratos e Boundaries]]
- [[Autenticacao e Autorizacao]]
- [[Realtime e Jobs]]
- [[Seguranca Web]]
- [[Multi-tenancy]]
- [[Glossario e Vocabulario]]
- [[Principios de Identidade Visual]]
