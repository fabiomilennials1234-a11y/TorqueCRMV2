---
tags:
  - sistema-base
  - checklist
  - qualidade
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Checklist Sistema Base

Checklist para declarar o Sistema Base "pronto". **Só avança para [[00 - Mapa de Features]] após 100% dos itens marcados.**

Toda linha aqui é verificável objetivamente. Se um item é ambíguo, ele é reescrito — não deixado ao julgamento.

## Tokens e estilo

- [ ] Tokens HSL canvas completos (background, surface, elevated, overlay)
- [ ] Tokens HSL type completos (primary, secondary, tertiary, disabled, inverse)
- [ ] Tokens HSL accent completos (base, hover, active, soft, ring)
- [ ] Tokens HSL semânticos (success, warning, danger, info) com variantes soft
- [ ] Tokens stages por pipe definidos
- [ ] Tokens calor (frio / morno / quente / muito quente / perdido)
- [ ] Tokens canal (whatsapp, instagram, messenger, sz_chat)
- [ ] Tokens countdown (safe D-5, warn D-3, urgent D-1)
- [ ] Tokens job status (pending, running, completed, failed)
- [ ] Fontes self-hosted (Fraunces, Instrument Sans, JetBrains Mono) via `@fontsource`
- [ ] Texturas grain e vignette aplicáveis como utility classes
- [ ] Classes utilitárias `shadow-hairline-*` substituindo uso de `border`

## Primitivos UI

- [ ] Audit dos 17 primitivos existentes (props, a11y, states)
- [ ] `Button` — snapshot + a11y
- [ ] `Input` — snapshot + a11y
- [ ] `Select` — snapshot + a11y
- [ ] `Dialog` — snapshot + focus trap
- [ ] `Drawer` — snapshot + focus trap
- [ ] `Tooltip` — snapshot + a11y
- [ ] `Popover` — snapshot + a11y
- [ ] `Badge` — snapshot
- [ ] `Avatar` — snapshot
- [ ] `Card` — snapshot
- [ ] `Tabs` — snapshot + keyboard nav
- [ ] `Menu` — snapshot + keyboard nav
- [ ] `Switch` — snapshot + a11y
- [ ] `Checkbox` — snapshot + a11y
- [ ] `RadioGroup` — snapshot + a11y
- [ ] `Skeleton` — snapshot
- [ ] `Toast` — snapshot + queue + a11y
- [ ] `StepProgress` (novo) — implementado + testado
- [ ] `QuotaGauge` (novo) — implementado + testado
- [ ] `ChannelBadge` (novo) — implementado + testado

## Estrutura de pastas

- [ ] `src/features/` criado (vazio)
- [ ] `src/hooks/` criado
- [ ] `src/api/` criado
- [ ] `src/contracts/` criado
- [ ] `src/lib/{fetch,ws,domain}/` criados
- [ ] `src/providers/` criado

## Contratos

- [ ] Script `generate:types` com `openapi-typescript` funcional
- [ ] `src/contracts/api.gen.ts` existe (vazio inicial aceitável)
- [ ] `src/contracts/manual.ts` com rascunho das entidades fundamentais
- [ ] Conversor `snake_case` ↔ `camelCase` + paginação cursor-based

## HTTP client

- [ ] `credentials: 'include'` default
- [ ] Interceptor 401 → refresh automático
- [ ] Refresh endpoint integrado (`POST /auth/refresh`)
- [ ] Strip de `organization_id` em mutations (enforçado no client)
- [ ] Error mapping uniforme (`AppError` discriminado)
- [ ] Retry exponencial em 5xx/network errors

## Auth

- [ ] Página `/login` real (email + senha + validação + loading + error)
- [ ] `AuthContext` + `AuthProvider`
- [ ] `useSession()` expondo user, org, permissions, isAuthenticated
- [ ] `ProtectedRoute` redireciona não autenticados
- [ ] `MasterRoute` bloqueia não-master
- [ ] `GET /auth/me` retorna bundle (session + permissions + org)
- [ ] `useCanPerformAction` + `usePermission` funcionais
- [ ] `<PermissionGate>` com fallback configurável

## Realtime

- [ ] `src/lib/ws.ts` singleton
- [ ] Reconexão exponencial com jitter
- [ ] Heartbeat / ping-pong
- [ ] `useWSStatus()` com estados (connecting, connected, disconnected, reconnecting)
- [ ] Badge visível no `TopBar`

## Estado global

- [ ] `QueryClient` com defaults corretos (staleTime 5min, gcTime 10min, sem refetch-on-focus)
- [ ] `AuthContext` provido na raiz
- [ ] `OrgContext` provido na raiz
- [ ] `ThemeContext` (dark-only por enquanto, preparado para multi-tema)

## Segurança

- [ ] CSP estrita nonce-based aplicada nos headers do servidor
- [ ] HSTS `max-age` de 1 ano com preload
- [ ] `X-Frame-Options: DENY`
- [ ] `X-Content-Type-Options: nosniff`
- [ ] `Referrer-Policy: strict-origin-when-cross-origin`
- [ ] `Permissions-Policy` restritiva (geolocation, camera, microphone, etc.)
- [ ] CORS com whitelist explícita
- [ ] `GET /api/bootstrap` entrega config sensível (sem `VITE_*` com secret)
- [ ] ESLint rule proibindo `dangerouslySetInnerHTML` sem allowlist
- [ ] Sentry com `beforeSend` fazendo scrubbing de PII

## Observabilidade

- [ ] Sentry inicializado com DSN via bootstrap
- [ ] Logger cliente estruturado (níveis: debug, info, warn, error)
- [ ] Badge WS status no `TopBar` reativo
- [ ] Identificação de usuário no Sentry (user id apenas, sem email)

## Páginas-casca

- [ ] `/login` funcional
- [ ] `/` (dashboard vazio com shell completo)
- [ ] `/404` com link de volta
- [ ] `/401-403` com mensagem clara + logout

## Qualidade

- [ ] ESLint com regras reais (`no-restricted-imports`, a11y, hooks, etc.)
- [ ] Prettier configurado e integrado com ESLint
- [ ] TS `strict: true` + `noUncheckedIndexedAccess` + `exactOptionalPropertyTypes`
- [ ] Testes dos primitivos passando (Vitest + Testing Library)
- [ ] `src/lib/vocabulary.ts` com termos canônicos do produto
- [ ] Pipeline CI: lint + typecheck + test + build

---

## Critério de saída

**100% marcado. Sem exceção.**

Ao bater 100%, abrir [[00 - Mapa de Features]] e iniciar F01.
