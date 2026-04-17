---
tags: [sistema-base, plano, execucao, granular, frontend]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
type: plan
---

# Plano de Execucao Granular — Redesign Sistema Base

Derivado de [[Spec - Redesign Sistema Base]]. Cada etapa lista arquivos afetados, agente executor, dependencias e criterio de aceite. Etapas independentes podem rodar em paralelo; etapas com dependencia rodam em sequencia.

## Visao geral das etapas

```
Etapa 1: Preparacao (estrutura + deps)          [PARALELO com Etapa 2]
Etapa 2: Fundacao visual (tokens + fontes)       [PARALELO com Etapa 1]
Etapa 3: Layout base (shell + navegacao)         [DEPENDE de 1 e 2]
Etapa 4: Componentes base (3 primitivos novos)   [DEPENDE de 2]
Etapa 5: Infraestrutura (providers + hooks)      [DEPENDE de 1]
Etapa 6: Paginas base (login + 404 + 403)        [DEPENDE de 3, 4, 5]
Etapa 7: Validacao (testes + lint + review)       [DEPENDE de todas]
Etapa 8: Documentacao final                       [DEPENDE de 7]
```

---

## Etapa 1 — Preparacao (estrutura + dependencias)

**Paralelo com Etapa 2. Sem dependencias.**

### 1.1 Criar estrutura de pastas
```
mkdir src/api src/hooks src/contracts src/providers src/lib/domain src/i18n src/components
```
- Criterio: `ls src/` mostra todas as pastas

### 1.2 Instalar dependencias
```
pnpm add @tanstack/react-query react-intl @sentry/react
pnpm add @fontsource-variable/fraunces @fontsource/instrument-sans @fontsource/jetbrains-mono
pnpm add -D @tanstack/eslint-plugin-query
```
- Criterio: `pnpm ls` confirma todas instaladas

### 1.3 Configurar openapi-typescript
- Adicionar script em package.json: `"generate:types": "openapi-typescript http://localhost:8080/openapi.json -o src/contracts/api.gen.ts"`
- Criar `src/contracts/api.gen.ts` vazio (placeholder)
- Criterio: script existe no package.json

### 1.4 Criar contracts base
- `src/contracts/manual.ts` — rascunho tipado das 16 entidades (Lead, Organization, TeamMember, PipeWhatsApp, PipeConfirmacao, PipeProposta, Conversation, ChannelMessage, Workflow, WorkflowExecution, Campaign, CopilotAgent, FollowUp, Product, Commission, OrgQuota)
- `src/contracts/transformer.ts` — funcoes `toClient(snakeObj)` e `toServer(camelObj)`
- Criterio: types compilam sem erro

### 1.5 Criar vocabulary.ts
- `src/lib/vocabulary.ts` — constantes com termos canonicos do produto
- Criterio: arquivo existe com exports nomeados

**Arquivos afetados:** package.json, tsconfig.json (paths se necessario), pastas novas, 4 arquivos novos.
**Executor:** agente general-purpose.
**Revisao:** nao necessaria (estrutura pura).

---

## Etapa 2 — Fundacao visual (tokens + fontes)

**Paralelo com Etapa 1. Sem dependencias.**

### 2.1 Adicionar tokens faltantes em globals.css
Tokens a adicionar (valores exatos de [[Design System Base]]):
- `--heat-1` a `--heat-5`
- `--channel-whatsapp`, `--channel-messenger`, `--channel-instagram`, `--channel-sz`
- `--countdown-safe`, `--countdown-warn`, `--countdown-urgent`
- `--job-pending`, `--job-running`, `--job-completed`, `--job-failed`
- `--radius-sm`, `--radius`, `--radius-lg`, `--radius-xl`, `--radius-full` (como CSS custom properties)
- Criterio: todos os tokens existem em `:root`

### 2.2 Adicionar utility classes
- `.grain` — pseudo-element com noise texture, opacity 0.035, mix-blend overlay
- `.vignette` — pseudo-element com radial-gradient transparente→preto, pointer-events none
- `.bg-grain` e `.bg-vignette` como Tailwind plugins ou classes em globals.css
- Criterio: `<div class="grain">` mostra efeito visual, `<div class="vignette">` mostra escurecimento radial

### 2.3 Migrar Google Fonts para @fontsource
- Remover `<link>` de Google Fonts em `index.html`
- Adicionar imports no entry point (main.tsx ou globals.css):
  ```
  @import "@fontsource-variable/fraunces";
  @import "@fontsource/instrument-sans/400.css";
  @import "@fontsource/instrument-sans/500.css";
  @import "@fontsource/jetbrains-mono/400.css";
  ```
- Adicionar `<link rel="preload">` para variantes criticas no index.html
- Criterio: DevTools Network mostra fontes servidas de localhost, zero requests para fonts.googleapis.com

### 2.4 Mapear tokens novos no tailwind.config.ts
- `colors.heat.1..5`, `colors.channel.whatsapp/messenger/instagram/sz`, `colors.countdown.safe/warn/urgent`, `colors.job.pending/running/completed/failed`
- `borderRadius` usando custom properties
- Criterio: `bg-heat-3`, `text-channel-whatsapp`, `bg-countdown-urgent`, `bg-job-running` funcionam como classes

### 2.5 Fix debitos visuais
- `KanbanPage.tsx:102` — trocar `bg-[hsl(222_20%_8%/0.3)]` por `bg-bg/30`
- `LoginPage.tsx` — remover `border border-hairline` onde shadow-hairline ja existe
- Padronizar `transition-all` em LeadCard e DashboardPage para `transition-[color,background-color,box-shadow,transform,opacity]`
- Criterio: grep por `border-white`, `border-hairline` (sem shadow), `transition-all`, `ease` (sem var) retorna zero hits

**Arquivos afetados:** globals.css, tailwind.config.ts, index.html, main.tsx, KanbanPage.tsx, LoginPage.tsx, LeadCard.tsx, DashboardPage.tsx.
**Executor:** agente frontend (com skills frontend-design + ui-ux-pro-max carregadas).
**Revisao:** obrigatoria — agente reviewer valida contra [[Criterios de Reprovacao]].

---

## Etapa 3 — Layout base (shell + navegacao)

**Depende de Etapa 1 e Etapa 2.**

### 3.1 Refatorar Sidebar
- Rename "Pipelines" para "Funis"
- Criar 4 grupos de nav: Vendas, Automacao, Inteligencia, Equipe
- Items placeholder para rotas futuras: Follow-ups, Time, Produtos
- Footer: Configuracoes, Ajuda
- Area Master (condicional, visivel apenas para is_master — hardcoded true ate AuthProvider existir)
- Criterio: sidebar mostra 4 grupos com labels corretos

### 3.2 Adicionar badge WS no TopBar
- Importar `useWSStatus()` (sera mockado ate ws.ts existir)
- Dot indicator: connected (success sutil), connecting (warning pulse), disconnected (danger discreto)
- Criterio: badge visivel no TopBar

### 3.3 Adicionar grain no main container
- AppShell main area recebe classe `grain` como overlay
- Criterio: grain sutil visivel sobre o conteudo principal

**Arquivos afetados:** Sidebar.tsx, TopBar.tsx, AppShell.tsx.
**Executor:** agente frontend.
**Revisao:** visual — captura de tela comparada com [[Direcao Visual Frontend]].

---

## Etapa 4 — Componentes base (3 primitivos novos)

**Depende de Etapa 2 (tokens precisam existir).**

### 4.1 StepProgress
- `src/ui/step-progress.tsx`
- Variantes: numbered, iconed, minimal
- Consome tokens: accent (ativo), success (completo), ink-dim (futuro), hairline (conector)
- Shadow-hairline nos circulos
- Animacao: ease-in-out-precise no avancar de step
- Criterio: renderiza N steps, estado ativo/completo/pendente, acessivel (aria-current)

### 4.2 QuotaGauge
- `src/ui/quota-gauge.tsx`
- Variantes: bar, ring, inline
- Consome tokens: success (verde ate 70%), warning (70-90%), danger (>90%)
- Prop `can_add` controla estado visual
- Criterio: renderiza current/limit, muda cor por banda, estado bloqueado quando can_add=false

### 4.3 ChannelBadge
- `src/ui/channel-badge.tsx`
- Variantes: icon-only, labeled, dot
- Consome tokens: --channel-whatsapp/messenger/instagram/sz
- Icones: Lucide (MessageCircle/Facebook/Instagram/Headphones) ou SVG custom
- Criterio: renderiza 4 canais, 3 variantes, cores corretas

**Arquivos afetados:** 3 arquivos novos em src/ui/.
**Executor:** agente frontend.
**Revisao:** snapshot test + revisao contra [[Componentes Primitivos]].

---

## Etapa 5 — Infraestrutura (providers + hooks)

**Depende de Etapa 1 (pastas e deps). Paralelo com Etapa 3 e 4.**

### 5.1 QueryProvider
- `src/providers/QueryProvider.tsx`
- QueryClient com staleTime 5min, gcTime 10min, refetchOnWindowFocus false
- Criterio: provider wraps app, devtools visivel em dev

### 5.2 AuthProvider (mock)
- `src/providers/AuthProvider.tsx`
- AuthContext com user, org, role, is_master, permissions
- useSession() hook
- Mock: retorna usuario hardcoded ate backend Go existir
- Criterio: useSession() retorna dados do mock

### 5.3 IntlProvider
- `src/providers/IntlProvider.tsx`
- react-intl com locale pt-BR
- `src/i18n/pt-BR.ts` com mensagens base (labels do shell, empty states genericos, erros)
- Criterio: `<FormattedMessage>` funciona no app

### 5.4 Fetch client
- `src/api/client.ts`
- Wrapper fetch com credentials:'include', CSRF header injection, 401 retry com dedup
- Error mapping para AppError discriminado
- Criterio: chamada a /auth/me funciona (mock server ou MSW)

### 5.5 WebSocket singleton
- `src/lib/ws.ts`
- Conexao singleton, reconexao exponencial com jitter (1s-30s), heartbeat 30s
- useWSStatus() hook com estados: connecting, connected, disconnected, reconnecting
- Criterio: hook retorna estado, reconexao funciona apos desconexao simulada

### 5.6 RBAC hooks
- `src/hooks/useCanPerformAction.ts`
- `src/hooks/usePermission.ts`
- `src/components/PermissionGate.tsx`
- Cascata: master bypass → admin → feature_permissions → member_permissions
- Mock ate /auth/me real
- Criterio: PermissionGate renderiza/esconde children corretamente

### 5.7 Route guards
- `src/components/ProtectedRoute.tsx` — redireciona para /login se nao autenticado
- `src/components/MasterRoute.tsx` — redireciona para /401-403 se nao master
- Criterio: rotas protegidas redirecionam corretamente

### 5.8 Error boundaries
- `src/components/RootErrorBoundary.tsx`
- `src/components/RouteErrorBoundary.tsx`
- Criterio: erro em componente filho e capturado e mostra UI de fallback

### 5.9 Sentry
- Inicializar no main.tsx com DSN mock
- beforeSend scrubbing token/password/authorization/cookie/email
- Criterio: Sentry.init() roda sem erro; beforeSend filtra campos sensiveis

### 5.10 Wiring no main.tsx
- Ordem: SentryProvider > QueryProvider > AuthProvider > IntlProvider > WSProvider > RouterProvider
- Criterio: app renderiza sem erro com todos os providers

**Arquivos afetados:** ~15 arquivos novos + main.tsx modificado.
**Executor:** agente general-purpose (infraestrutura, nao visual).
**Revisao:** obrigatoria — agente reviewer valida contra [[Escopo do Sistema Base]] e [[ADR-003]].

---

## Etapa 6 — Paginas base (login + 404 + 403)

**Depende de Etapa 3 (shell), 4 (primitivos), 5 (providers).**

### 6.1 Refinar /login
- Split horizontal: form (50%) + painel editorial (50%)
- Painel editorial: Fraunces text-display-2xl, TorqueMark grande, vignette, grain, bg-grid
- Form: eyebrow "Bem-vindo de volta" em accent, Input email + password, Button primary, SSO slot
- Estados: default, loading (button spinner), error (inline abaixo do campo), success (torque-tick)
- Criterio: screenshot comparavel com [[Direcao Visual Frontend]] secao 7

### 6.2 Criar /404
- `src/features/errors/NotFoundPage.tsx`
- Layout centralizado, TorqueMark com rotacao sutil (torque-tick ease), titulo "404" em font-display, descricao em font-sans ink-muted, CTA "Voltar ao inicio"
- Criterio: acessar rota inexistente mostra pagina 404

### 6.3 Criar /401-403
- `src/features/errors/ForbiddenPage.tsx`
- Similar ao 404 mas com mensagem "Acesso restrito" e CTA de logout
- Criterio: MasterRoute redireciona para esta pagina

### 6.4 Refinar / (dashboard vazio)
- Layout com PageHeader (font-display, "Bom dia, {nome}")
- Grid de cards placeholder (skeleton-like, sem dados)
- Grain overlay visivel
- Criterio: shell completo, sem dados, visualmente coerente

**Arquivos afetados:** LoginPage.tsx (refatorar), 2 arquivos novos, DashboardPage.tsx (simplificar).
**Executor:** agente frontend (com skills frontend-design + ui-ux-pro-max).
**Revisao:** obrigatoria — agente reviewer + checklist [[Criterios de Reprovacao]].

---

## Etapa 7 — Validacao (testes + lint + review)

**Depende de todas as etapas anteriores.**

### 7.1 ESLint config
- `.eslintrc.cjs` com rules:
  - `no-restricted-imports` (banir @supabase/*, nao-allowed imports)
  - `react/no-danger` (banir dangerouslySetInnerHTML)
  - Import order (grupos: external, internal @/, types)
  - @tanstack/query eslint plugin
- Criterio: `pnpm lint` passa sem erros

### 7.2 Testes snapshot dos primitivos
- Vitest + @testing-library/react
- Snapshot para cada um dos 20 primitivos (17 existentes + 3 novos)
- Criterio: `pnpm test` passa

### 7.3 TypeScript strict check
- `pnpm tsc --noEmit` sem erros
- Criterio: zero erros de tipo

### 7.4 Build limpo
- `pnpm build` sem warnings criticos
- Criterio: bundle produzido com sucesso

### 7.5 Review final por agente
- Agente code-reviewer audita codebase inteiro contra:
  - [[Criterios de Reprovacao]] (10 items)
  - [[Spec - Redesign Sistema Base]] (criterios de aceitacao)
  - [[Checklist Sistema Base]]
- Criterio: veredicto "aprovado" ou "aprovado com ressalvas menores"

**Arquivos afetados:** .eslintrc.cjs, vitest.config.ts (se necessario), arquivos de teste.
**Executor:** agente general-purpose (config) + agente reviewer (auditoria).

---

## Etapa 8 — Documentacao final

**Depende de Etapa 7.**

### 8.1 Atualizar Checklist Sistema Base
- Marcar items completados
- Documentar items pendentes (que dependem de backend Go)

### 8.2 Atualizar Analise Pratica
- Refletir debitos corrigidos
- Atualizar "alinhamento com identidade" para refletir estado pos-redesign

### 8.3 Registrar decisoes tomadas
- Se decisoes novas surgiram durante execucao, criar ADRs

### 8.4 Atualizar 00 - Indice.md
- Adicionar notas novas criadas neste ciclo

**Arquivos afetados:** 4+ notas no Obsidian.
**Executor:** agente general-purpose.

---

## Resumo de paralelismo

```
[Etapa 1: Preparacao]  ─────────────────────────┐
                                                  ├──→ [Etapa 3: Shell]  ──┐
[Etapa 2: Fundacao visual]  ─────────────────────┤                         │
                                                  ├──→ [Etapa 4: Prims]  ──┤
                                                  │                         ├──→ [Etapa 6: Pages] ──→ [Etapa 7: Validacao] ──→ [Etapa 8: Docs]
[Etapa 5: Infraestrutura]  ─────────────────────────────────────────────────┘
  (paralelo com 3 e 4, depende de 1)
```

Etapas 1, 2, 5 podem comecar simultaneamente (5 depende de 1 mas e rapido).
Etapas 3 e 4 dependem de 2.
Etapa 6 depende de 3, 4 e 5.
Etapas 7 e 8 sao sequenciais no final.

## Agentes para cada etapa

| Etapa | Agente | Skills | Motivo |
|-------|--------|--------|--------|
| 1 | general-purpose | — | Estrutura de pastas e tipos, nao visual |
| 2 | frontend executor | frontend-design, ui-ux-pro-max | Tokens e identidade visual |
| 3 | frontend executor | frontend-design | Shell e navegacao |
| 4 | frontend executor | frontend-design, ui-ux-pro-max | Componentes novos |
| 5 | general-purpose | — | Infraestrutura pura, nao visual |
| 6 | frontend executor | frontend-design, ui-ux-pro-max | Paginas signature |
| 7 | code-reviewer | — | Validacao independente |
| 8 | general-purpose | — | Documentacao |

## Estimativa

| Etapa | Esforco | Bloqueador |
|-------|---------|-----------|
| 1 | 2-3h | Nenhum |
| 2 | 3-4h | Nenhum |
| 3 | 2-3h | Etapas 1+2 |
| 4 | 4-5h | Etapa 2 |
| 5 | 4-6h | Etapa 1 |
| 6 | 4-6h | Etapas 3+4+5 |
| 7 | 2-3h | Tudo anterior |
| 8 | 1-2h | Etapa 7 |
| **Total** | **22-32h** | |

## Referencias

- [[Spec - Redesign Sistema Base]]
- [[Escopo do Sistema Base]]
- [[Checklist Sistema Base]]
- [[Direcao Visual Frontend]]
- [[Analise Pratica]]
- [[Criterios de Reprovacao]]
