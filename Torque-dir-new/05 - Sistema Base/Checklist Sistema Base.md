---
tags:
  - sistema-base
  - checklist
  - qualidade
created: 2026-04-15
last_updated: 2026-04-18
status: draft
---

# Checklist Sistema Base

Checklist para declarar o Sistema Base "pronto". **Só avança para [[00 - Mapa de Features]] após 100% dos itens marcados.**

Toda linha aqui é verificável objetivamente. Se um item é ambíguo, ele é reescrito — não deixado ao julgamento.

> **Extensão ativa (ADR-007).** A seção **Modos de UI + Cockpit** no final deste checklist faz parte do Sistema Base e é pré-requisito para F01. Ver [[UI Modes - Vendedor e Gerente]] e [[ADR-007-modo-vendedor-gerente]].

## Tokens e estilo

- [x] Tokens HSL canvas completos (background, surface, elevated, overlay)
- [x] Tokens HSL type completos (primary, secondary, tertiary, disabled, inverse)
- [x] Tokens HSL accent completos (base, hover, active, soft, ring)
- [x] Tokens HSL semânticos (success, warning, danger, info) com variantes soft
- [x] Tokens stages por pipe definidos
- [x] Tokens calor (frio / morno / quente / muito quente / perdido)
- [x] Tokens canal (whatsapp, instagram, messenger, sz_chat)
- [x] Tokens countdown (safe D-5, warn D-3, urgent D-1)
- [x] Tokens job status (pending, running, completed, failed)
- [x] Fontes self-hosted (Fraunces, Instrument Sans, JetBrains Mono) via `@fontsource`
- [x] Texturas grain e vignette aplicáveis como utility classes
- [x] Classes utilitárias `shadow-hairline-*` substituindo uso de `border`

## Primitivos UI

- [x] Audit dos 17 primitivos existentes (props, a11y, states)
- [x] `Button` — snapshot + a11y
- [x] `Input` — snapshot + a11y
- [ ] `Select` — snapshot + a11y <!-- primitivo não existe no repo; manter até adoção -->
- [x] `Dialog` — snapshot (via Sheet baseado em Radix Dialog)
- [x] `Drawer` — snapshot + focus trap (Sheet)
- [x] `Tooltip` — snapshot + a11y
- [ ] `Popover` — snapshot + a11y <!-- primitivo não existe no repo; adotar se virar necessidade -->
- [x] `Badge` — snapshot
- [x] `Avatar` — snapshot
- [x] `Card` — snapshot
- [x] `Tabs` — snapshot + keyboard nav
- [x] `Menu` — snapshot + keyboard nav (dropdown)
- [ ] `Switch` — snapshot + a11y <!-- primitivo não existe no repo -->
- [ ] `Checkbox` — snapshot + a11y <!-- via dropdown CheckboxItem; primitivo isolado pendente -->
- [ ] `RadioGroup` — snapshot + a11y <!-- primitivo não existe no repo -->
- [x] `Skeleton` — snapshot
- [ ] `Toast` — snapshot + queue + a11y <!-- primitivo não existe no repo; adotar quando features pedirem -->
- [x] `StepProgress` (novo) — implementado + testado
- [x] `QuotaGauge` (novo) — implementado + testado
- [x] `ChannelBadge` (novo) — implementado + testado

## Estrutura de pastas

- [x] `src/features/` criado (vazio)
- [x] `src/hooks/` criado
- [x] `src/api/` criado
- [x] `src/contracts/` criado
- [x] `src/lib/{fetch,ws,domain}/` criados
- [x] `src/providers/` criado

## Contratos

- [x] Script `generate:types` com `openapi-typescript` funcional
- [x] `src/contracts/api.gen.ts` existe (vazio inicial aceitável)
- [x] `src/contracts/manual.ts` com rascunho das entidades fundamentais
- [x] Conversor `snake_case` ↔ `camelCase` + paginação cursor-based

## HTTP client

- [x] `credentials: 'include'` default
- [x] Interceptor 401 → refresh automático
- [x] Refresh endpoint integrado (`POST /auth/refresh`)
- [ ] Strip de `organization_id` em mutations (enforçado no client) <!-- aguarda backend S01 para definir formato exato -->
- [x] Error mapping uniforme (`AppError` discriminado)
- [ ] Retry exponencial em 5xx/network errors <!-- aguarda S01 para calibrar contra backend real -->

## Auth

- [x] Página `/login` real (email + senha + validação + loading + error)
- [x] `AuthContext` + `AuthProvider`
- [x] `useSession()` expondo user, org, permissions, isAuthenticated (via `useAuth`)
- [x] `ProtectedRoute` redireciona não autenticados
- [ ] `MasterRoute` bloqueia não-master <!-- aguarda endpoint /master/* em S01 -->
- [ ] `GET /auth/me` retorna bundle (session + permissions + org) <!-- aguarda S01; mock já completo no cliente -->
- [ ] `useCanPerformAction` + `usePermission` funcionais <!-- aguarda payload real -->
- [ ] `<PermissionGate>` com fallback configurável <!-- aguarda payload real -->

## Realtime

- [x] `src/lib/ws.ts` singleton
- [x] Reconexão exponencial com jitter
- [x] Heartbeat / ping-pong
- [x] `useWSStatus()` com estados (connecting, connected, disconnected, reconnecting)
- [x] Badge visível no `TopBar`

## Estado global

- [x] `QueryClient` com defaults corretos (staleTime 5min, gcTime 10min, sem refetch-on-focus)
- [x] `AuthContext` provido na raiz
- [ ] `OrgContext` provido na raiz <!-- org vive dentro de SessionBundle até surgir necessidade de split -->
- [x] `ThemeContext` (dark-only por enquanto, preparado para multi-tema)

## Segurança

- [ ] CSP estrita nonce-based aplicada nos headers do servidor <!-- aguarda S01 -->
- [ ] HSTS `max-age` de 1 ano com preload <!-- aguarda S01 -->
- [ ] `X-Frame-Options: DENY` <!-- aguarda S01 -->
- [ ] `X-Content-Type-Options: nosniff` <!-- aguarda S01 -->
- [ ] `Referrer-Policy: strict-origin-when-cross-origin` <!-- aguarda S01 -->
- [ ] `Permissions-Policy` restritiva (geolocation, camera, microphone, etc.) <!-- aguarda S01 -->
- [ ] CORS com whitelist explícita <!-- aguarda S01 -->
- [ ] `GET /api/bootstrap` entrega config sensível (sem `VITE_*` com secret) <!-- aguarda S01 -->
- [x] ESLint rule proibindo `dangerouslySetInnerHTML` sem allowlist
- [ ] Sentry com `beforeSend` fazendo scrubbing de PII <!-- init ok; scrubbing será ligado com DSN real -->

## Observabilidade

- [ ] Sentry inicializado com DSN via bootstrap <!-- init condicional pronto; aguarda /api/bootstrap em S01 -->
- [ ] Logger cliente estruturado (níveis: debug, info, warn, error) <!-- adiado para S02 -->
- [x] Badge WS status no `TopBar` reativo
- [ ] Identificação de usuário no Sentry (user id apenas, sem email) <!-- aguarda S01 -->

## Páginas-casca

- [x] `/login` funcional
- [x] `/` (dashboard vazio com shell completo)
- [x] `/404` com link de volta
- [x] `/401-403` com mensagem clara + logout

## Qualidade

- [x] ESLint com regras reais (`no-restricted-imports`, a11y, hooks, etc.)
- [x] Prettier configurado e integrado com ESLint
- [x] TS `strict: true` + `noUncheckedIndexedAccess` + `exactOptionalPropertyTypes`
- [x] Testes dos primitivos passando (Vitest + Testing Library) — 51 testes, 36 snapshots
- [x] `src/i18n/vocabulary.ts` com termos canônicos do produto
- [x] Pipeline CI: lint + typecheck + test + build (GitHub Actions `.github/workflows/ci.yml`)

## Modos de UI + Cockpit (ADR-007)

Extensão do Sistema Base. Spec operacional completa em [[UI Modes - Vendedor e Gerente]]; decisão em [[ADR-007-modo-vendedor-gerente]].

### Estado de UI e roteamento

- [x] `UiModeProvider` na árvore de providers (após AuthProvider)
- [x] `UiModeToggle` no `TopBar` (canto superior direito)
- [x] `ManagerModeGate` redireciona `membro` sem `ui.view_manager_mode` para `/cockpit`
- [x] Rota `/cockpit` com `CockpitShell` próprio (sem AppShell)
- [x] Roteamento paralelo `/` (Gerente) vs `/cockpit` (Vendedor)
- [x] Pós-login: `ui_mode='salesperson'` → `navigate('/cockpit', { replace: true })` após `/auth/me`
- [x] Fallback `localStorage.torque.ui_mode` antes da hidratação de `/auth/me`

### Tokens claymorphism (escopo `.cockpit-theme`)

- [x] Família `--clay-*` declarada em `globals.css`
- [x] Aplicação exclusiva via classe raiz `.cockpit-theme` no `CockpitShell`
- [x] Accent gold canônico preservado (`44 93% 54%`)
- [x] Motion mais suave (380ms) dentro do cockpit
- [x] Revisão visual: zero vazamento de `--clay-*` fora do cockpit (legacy tokens renomeados para `--card-*` / `.tactile-*`)
- [x] `prefers-reduced-motion` respeitado nas transições clay

### Cockpit — 4 painéis

- [x] `CockpitHeader`
- [x] `QueuePanel` (Tasks "A fazer")
- [x] `ActiveTaskPanel` (Task `in_progress`)
- [x] `PipeSnapshotPanel` (Kanban enxuto do funil ativo)
- [x] `BacklogPanel` (Tasks "Em aberto")
- [x] `CockpitSkeleton` (loading state)
- [x] Primitivos `clay/` (ClayPanel, ClayButton, ClayCard, ClayChip)
- [x] `__fixtures__/cockpit-mock.ts` para hidratar enquanto backend é 0%

### Hooks e tipos

- [x] `useCockpit` (React Query, consome fixture → futuro `GET /tasks/me/cockpit`)
- [x] `useTaskActions` (start, pause, complete, cancel, enqueue, reorder)
- [x] `useUiMode` exposto pelo `UiModeProvider`
- [x] Tipos `Task`, `TaskStatus`, `TaskKind`, `TaskPriority`, `TaskOrigin`, `UiMode`, `TaskCockpitBundle` em `src/contracts/manual.ts`

### i18n

- [x] Strings do cockpit em `src/i18n/pt-BR.ts`
- [x] Toggle Gerente/Vendedor traduzido

### Integração futura (backend + banco)

- [ ] Migration `0003_tasks_and_ui_preferences` aplicada (enums, tabela `tasks`, `users.ui_mode_preference`, índices, constraints parciais únicas)
- [ ] Endpoints `/tasks/*`, `/tasks/me/cockpit`, `/me/preferences` expostos e consumidos
- [ ] Eventos WS `task.created|updated|started|completed|cancelled|missed|queue_reordered|reassigned` com filtro tenant + assignee
- [ ] Seed de `feature_permissions`: `tasks.*`, `ui.view_manager_mode`, `ui.view_salesperson_mode`
- [ ] Rate limit leve no `PATCH /me/preferences` (anti-flood de toggle)

---

## Critério de saída

**100% marcado. Sem exceção.**

Ao bater 100%, abrir [[00 - Mapa de Features]] e iniciar F01.
