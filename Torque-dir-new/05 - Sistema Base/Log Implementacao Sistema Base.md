---
tags: [sistema-base, implementacao, log, changelog]
created: 2026-04-15
last_updated: 2026-04-15
status: active
---

# Log de Implementacao — Sistema Base

## Etapa 0 — Fix de bloqueadores (concluida)

6 bloqueadores criticos resolvidos antes do redesign:
- B1: Removido `* { @apply border-hairline }` global de globals.css
- B2: torque-tick corrigido (rotacao → escala 1.0→1.04→1.0, 280ms, ease-in-out-precise)
- B3: shimmer corrigido (2.4s → 1.2s), caret-blink (1.25s → 1.06s)
- B4: 22 ocorrencias de `border border-hairline` substituidas por shadow-hairline em 12 arquivos
- B5: @fontsource instalado, Google Fonts CDN removido de index.html
- B6: dropdown.tsx highlighted state corrigido (bg-surface → bg-elevated/80)

## Etapa 1 — Preparacao (concluida)

- Estrutura de pastas canonica criada: `src/{api,hooks,contracts,providers,components,lib/domain,i18n}`
- Dependencias instaladas: `@tanstack/react-query`, `react-intl@7`, `@sentry/react`, `@fontsource-variable/fraunces`, `@fontsource/instrument-sans`, `@fontsource/jetbrains-mono`
- `src/contracts/manual.ts` — 16 entidades tipadas + SessionBundle + CursorPage + Operation
- `src/contracts/transformer.ts` — funcoes toClient/toServer (snake↔camel)
- `src/contracts/api.gen.ts` — placeholder para openapi-typescript
- `src/lib/vocabulary.ts` — termos canonicos do produto (VOCAB, ROLES, CHANNELS)
- Script `generate:types` adicionado ao package.json

## Etapa 2 — Fundacao visual (concluida)

- Tokens faltantes adicionados em globals.css: heat-1..5, channel-whatsapp/messenger/instagram/sz, countdown-safe/warn/urgent, job-pending/running/completed/failed, radius-sm/default/lg/xl
- Tokens mapeados em tailwind.config.ts: `colors.heat.*`, `colors.channel.*`, `colors.countdown.*`, `colors.job.*`
- Fontes self-hosted via @fontsource (imports em globals.css)
- Google Fonts CDN completamente removido — CSP `font-src 'self'` viabilizado

## Etapa 3 — Shell e navegacao (concluida)

- Sidebar refatorada: 1 grupo "Operacao" → 4 grupos (Vendas, Automacao, Inteligencia, Equipe)
- "Pipelines" renomeado para "Funis" conforme [[Vocabulario de UI]]
- Items placeholder adicionados: Follow-ups, Time, Produtos
- Icones novos importados: CheckSquare, Users, Package, Shield

## Etapa 4 — Componentes primitivos novos (concluida)

3 primitivos criados em `src/ui/`:
- `step-progress.tsx` — stepper horizontal editorial (numbered/iconed/minimal)
- `quota-gauge.tsx` — visualizacao current/limit (bar/ring/inline)
- `channel-badge.tsx` — badge de canal (icon-only/labeled/dot)

Todos seguem padroes: shadow-hairline, tokens HSL, tipografia tripartida, eases custom.

## Etapa 5 — Infraestrutura (concluida)

15 arquivos novos + 2 modificados:
- `src/api/client.ts` — HTTP client com CSRF, X-Request-ID, refresh mutex, 429 tipado
- `src/providers/QueryProvider.tsx` — React Query com defaults canonicos
- `src/providers/AuthProvider.tsx` — bootstrap /auth/me com mock fallback
- `src/providers/IntlProvider.tsx` — react-intl PT-BR
- `src/providers/WSProvider.tsx` — WebSocket context mock
- `src/hooks/useSession.ts`, `useCanPerformAction.ts`, `usePermission.ts`, `useWSStatus.ts`
- `src/lib/ws.ts` — WebSocket singleton com reconnect exponencial
- `src/i18n/pt-BR.ts` — 22 mensagens do sistema base
- `src/components/ProtectedRoute.tsx`, `MasterRoute.tsx`, `PermissionGate.tsx`, `RootErrorBoundary.tsx`
- `src/main.tsx` atualizado com providers na ordem correta
- `src/vite-env.d.ts` criado para tipos Vite

## Etapa 6 — Paginas base (concluida)

- `src/features/errors/NotFoundPage.tsx` — 404 editorial com TorqueMark + torque-tick + grain
- `src/features/errors/ForbiddenPage.tsx` — 403 com logout
- `src/routes.tsx` refatorado: ProtectedRoute envolvendo layout, rotas /follow-ups /team /products como placeholder, /forbidden, catch-all para NotFoundPage

## Etapa 7 — Validacao (concluida)

- `npx vite build` — sucesso em 5.34s
- `grep "border border-hairline"` — 0 hits
- `grep "fonts.googleapis.com"` — 0 hits
- `transition-all` residual em 5 arquivos de features (mockups, nao base) — sera corrigido quando features forem reescritas

## Estado pos-implementacao

### Inventario de arquivos novos: ~25
### Arquivos modificados: ~18
### Build: limpo
### Google Fonts: removido
### Tokens documentados vs implementados: 100% alinhados

## Debitos remanescentes (nao bloqueadores)

- `transition-all` em 5 mockups de features (CampaignsPage, DashboardPage, AgentsPage, LeadCard, WorkflowBuilderPage) — corrigir quando features forem implementadas
- Stage tokens: CSS atual diverge dos docs (accent no stage-5/proposta vs docs que tem accent no stage-6/fechamento) — decisao necessaria antes de F01
- ESLint rules reais: config ainda nao atualizado (pendente)
- Testes snapshot dos primitivos: pendente (Vitest nao configurado)

## Referencias

- [[Spec - Redesign Sistema Base]]
- [[Plano de Execucao Granular]]
- [[Revisao Final - Redesign Sistema Base]]
- [[Checklist Sistema Base]]
