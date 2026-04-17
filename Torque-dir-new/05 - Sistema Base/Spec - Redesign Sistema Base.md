---
tags: [spec, sistema-base, redesign, frontend, sdd]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
type: spec
scope: large
---

# Spec — Redesign do Sistema Base Frontend

## Objetivo

Implementar o Sistema Base do frontend Torque CRM como fundacao solida, visualmente fiel a identidade documentada, estruturalmente preparada para receber qualquer feature sem retrabalho, e segura desde o primeiro commit.

O front atual (`torque-web/`) e um mockup editorial de alta fidelidade com 92% de aderencia a identidade. Esta spec define o que precisa mudar para chegar a 100% e adicionar a infraestrutura ausente.

## Escopo

### Entra nesta spec
- Correcao dos debitos visuais (hardcoded HSL, border redundante, transition-all)
- Migracao de Google Fonts CDN para @fontsource (ADR-005)
- Adicao dos tokens faltantes (heat, channel, countdown, job) em globals.css
- Adicao de tokens de radius no globals.css (documentados mas nao como CSS vars)
- Rename sidebar "Pipelines" para "Funis"
- Grupos de nav "Automacao" e "Equipe" com placeholders
- Estrutura de pastas canonica (`src/{api,hooks,contracts,providers,lib/domain,i18n}`)
- Camada de contratos (openapi-typescript script + manual.ts rascunho)
- Cliente HTTP base (`src/lib/fetch.ts`)
- WebSocket singleton (`src/lib/ws.ts`)
- AuthProvider + useSession + ProtectedRoute + MasterRoute (skeleton sem backend real)
- RBAC hooks (useCanPerformAction, usePermission, PermissionGate)
- React Query com defaults canonicos
- react-intl estruturado (apenas PT-BR)
- `src/lib/vocabulary.ts` com termos canonicos
- ESLint rules reais (no-restricted-imports, dangerouslySetInnerHTML ban, import order)
- Sentry estrutura (DSN via bootstrap, scrubbing PII)
- Primitivos novos: StepProgress, QuotaGauge, ChannelBadge
- Paginas-casca: refinar `/login`, `/` (dashboard vazio), criar `/404` e `/401-403`
- Grain e vignette como utility classes reutilizaveis
- Documentacao continua no Obsidian

### NAO entra nesta spec (explicitamente)
- Qualquer pipe, kanban, DnD, inbox, Copilot, Workflow Builder, campanhas
- Analytics, Master Admin, Onboarding, Checkout
- Componentes de dominio (CountdownBadge, HeatSlider, FunnelChart, etc.)
- Backend Go real (skeleton apenas para auth)
- Testes e2e (apenas testes unitarios/snapshot dos primitivos)
- CI pipeline completo (apenas script local)

## Definicao pratica do Sistema Base

O Sistema Base e **tudo o que precisa existir para que a primeira feature (F01 — Funis Hub + Pipe WhatsApp) possa ser construida sem nenhuma preocupacao de infraestrutura, identidade visual, seguranca ou arquitetura de dados.**

A regua: sem o Sistema Base, nenhuma feature nasce. Com o Sistema Base, qualquer feature pode nascer.

## Restricoes de design (inegociaveis)

1. **Dark-first** — `<html class="dark">` hardcoded. Zero CSS de light mode. Zero toggle.
2. **Hairline-first** — toda borda via `shadow-hairline`. `border` Tailwind padrao e violacao.
3. **Tipografia tripartida** — Fraunces (display), Instrument Sans (body), JetBrains Mono (metric). Toda tela com dados exibe as 3.
4. **Accent restraint** — ouro `hsl(44 93% 54%)` so em CTAs, active states, highlights. Nunca em fundos grandes.
5. **Motion restraint** — eases custom (`ease-out-soft`, `ease-in-out-precise`). 150-300ms. Nunca `ease` padrao, nunca >350ms.
6. **Compacto** — px-2.5, py-1.5, gap-1.5 como base. Densidade informacional e identidade.
7. **HSL puro** — tokens sem wrapper. `hsl(var(--accent) / 0.5)` para alpha. Nunca hex, nunca RGB.
8. **Grain permanente** — opacity 0.035, mix-blend overlay. Vignette apenas em telas signature.
9. **Self-hosted fonts** — @fontsource. Nenhum Google Fonts CDN. CSP `font-src 'self'`.
10. **Skeleton, nunca spinner** — loading de conteudo sempre com skeleton na forma exata do conteudo.

## Diretrizes visuais obrigatorias

Extraidas de [[Direcao Visual Frontend]] e [[Principios de Identidade Visual]]:

### AppShell
- Sidebar: w-232px, bg-surface, hairline inset a direita
- TopBar: h-14, sticky top-0, bg-bg/80, backdrop-blur-xl, shadow-hairline-b
- Main: bg-bg puro, overflow-y-auto
- Grain overlay como pseudo-element no main container

### Sidebar
- 4 grupos: Vendas (Dashboard, Funis, Conversas, Follow-ups), Automacao (Fluxos, Campanhas), Inteligencia (Agentes IA, Analytics), Equipe (Time, Produtos)
- Footer: Configuracoes, Ajuda
- Master area (visivel apenas para is_master): Master Panel
- Item inativo: text-ink-muted, font-sans text-sm
- Item hover: bg-elevated/50
- Item ativo: left bar 2px accent, bg-elevated, text-ink

### TopBar
- Search trigger: bg-surface, shadow-hairline, Cmd+K hint
- Notificacoes: icone Bell ghost com dot accent quando ha itens
- User menu: Avatar sm + Dropdown com secoes Conta/Org/Sair

### Login
- Split horizontal: form (esquerda) + painel editorial (direita)
- Painel editorial: Fraunces display-2xl, vignette, grain, grid bg, TorqueMark grande
- Form: eyebrow em accent, campos com hairline focus, CTA primary
- Sucesso: torque-tick animation

### 404 / 401-403
- Layout centralizado sobre bg
- TorqueMark com rotacao sutil
- Titulo em font-display
- Descricao em font-sans text-ink-muted
- CTA para voltar

## Componentes/layouts afetados

### Modificar (existentes)
| Arquivo | O que muda |
|---------|-----------|
| `src/styles/globals.css` | +tokens heat/channel/countdown/job, +radius como CSS vars, +utility classes grain/vignette |
| `index.html` | Remover Google Fonts link, adicionar preload @fontsource |
| `package.json` | +@fontsource-variable/fraunces, +@fontsource/instrument-sans, +@fontsource/jetbrains-mono, +@tanstack/react-query, +react-intl, +@sentry/react, -google fonts link |
| `tailwind.config.ts` | +mapeamento tokens novos (heat, channel, countdown, job, radius) |
| `src/shell/Sidebar.tsx` | Rename "Pipelines" para "Funis", adicionar grupos Automacao/Equipe, items placeholder |
| `src/features/pipeline/KanbanPage.tsx` | Fix hardcoded HSL em Kbd (linha 102) |
| `src/features/auth/LoginPage.tsx` | Remover border redundante, refinar layout editorial |
| `src/main.tsx` | Adicionar providers (QueryClient, AuthProvider, IntlProvider, SentryProvider) |
| `src/App.tsx` | Adicionar ErrorBoundary root |
| `src/routes.tsx` | Adicionar rotas /404, /401-403, wraps com ProtectedRoute |

### Criar (novos)
| Arquivo | Proposito |
|---------|-----------|
| `src/providers/QueryProvider.tsx` | React Query com defaults canonicos |
| `src/providers/AuthProvider.tsx` | AuthContext + useSession + bootstrap /auth/me |
| `src/providers/IntlProvider.tsx` | react-intl com pt-BR |
| `src/providers/WSProvider.tsx` | WebSocket context |
| `src/hooks/useSession.ts` | Sessao do usuario autenticado |
| `src/hooks/useCanPerformAction.ts` | RBAC cascata 4 camadas |
| `src/hooks/usePermission.ts` | Feature permission check |
| `src/hooks/useWSStatus.ts` | Status da conexao WS |
| `src/api/client.ts` | Fetch wrapper com credentials + CSRF + 401 retry |
| `src/contracts/manual.ts` | Rascunho das 16 entidades canonicas |
| `src/contracts/api.gen.ts` | Placeholder (gerado por openapi-typescript) |
| `src/contracts/transformer.ts` | snake_case para camelCase e vice-versa |
| `src/lib/ws.ts` | WebSocket singleton com reconnect + dedup |
| `src/lib/vocabulary.ts` | Termos canonicos do produto |
| `src/lib/domain/` | Pasta para regras puras de dominio (vazia por ora) |
| `src/i18n/pt-BR.ts` | Catalogo de mensagens PT-BR |
| `src/ui/step-progress.tsx` | Stepper horizontal editorial |
| `src/ui/quota-gauge.tsx` | Visualizacao current/limit |
| `src/ui/channel-badge.tsx` | Badge de canal com cor canonica |
| `src/components/ProtectedRoute.tsx` | Guard de rota autenticada |
| `src/components/MasterRoute.tsx` | Guard de rota master |
| `src/components/PermissionGate.tsx` | Conditional render por permissao |
| `src/components/RootErrorBoundary.tsx` | Error boundary raiz |
| `src/components/RouteErrorBoundary.tsx` | Error boundary por rota |
| `src/features/errors/NotFoundPage.tsx` | Pagina 404 |
| `src/features/errors/ForbiddenPage.tsx` | Pagina 401/403 |

## Riscos

| Risco | Impacto | Mitigacao |
|-------|---------|----------|
| @fontsource pode ter tamanho de bundle diferente do Google Fonts CDN | FOUT visivel | Preload das variantes criticas + font-display: swap |
| AuthProvider sem backend Go real = mock perpetuo | Auth nunca testada | Criar mock server minimo (MSW) ou backend Go skeleton em paralelo |
| Tokens novos (heat/channel/countdown/job) sem uso imediato | Tokens orfaos no CSS | Documentar que sao pre-requisito de F01..F04, nao orfaos |
| ESLint rules novas quebram codigo existente | CI vermelho | Aplicar --fix e resolver em batch antes de ativar como error |
| React Query + react-intl + Sentry aumentam bundle | Tempo de load | Code splitting agressivo (ja tem manualChunks no Vite) |

## Criterios de aceitacao

### Fundacao visual
- [ ] Todos os tokens documentados existem em globals.css (canvas, type, accent, semanticos, stages, heat, channel, countdown, job, shadow, radius)
- [ ] Tailwind config mapeia todos os tokens para classes semanticas
- [ ] @fontsource instalado e funcional; Google Fonts CDN removido
- [ ] Nenhum `<link>` para fonts.googleapis.com no bundle
- [ ] Classes font-display, font-sans, font-metric produzem tipografia correta com opsz e tabular-nums
- [ ] Grain overlay visivel em 0.035 opacity
- [ ] Vignette como utility class aplicavel
- [ ] Zero uso de `border` Tailwind padrao em componentes (apenas shadow-hairline)
- [ ] Zero uso de `ease` CSS padrao (apenas variaveis custom)
- [ ] Zero hardcoded hex/RGB em componentes

### Shell e navegacao
- [ ] Sidebar com 4 grupos corretos (Vendas/Automacao/Inteligencia/Equipe)
- [ ] Label "Funis" (nao "Pipelines")
- [ ] Items placeholder para rotas futuras (Follow-ups, Time, Produtos)
- [ ] TopBar com badge WS status
- [ ] CommandPalette funcional com Cmd+K

### Infraestrutura
- [ ] Pastas src/{api,hooks,contracts,providers,lib/domain,i18n} existem
- [ ] React Query configurado com defaults canonicos
- [ ] react-intl com PT-BR ativo
- [ ] Fetch client com credentials:'include'
- [ ] WebSocket singleton com reconexao e useWSStatus
- [ ] AuthProvider com useSession (mock ate backend existir)
- [ ] ProtectedRoute/MasterRoute/PermissionGate funcionais
- [ ] manual.ts com rascunho das 16 entidades
- [ ] vocabulary.ts com termos canonicos
- [ ] ESLint com rules reais (import order, no dangerouslySetInnerHTML, no-restricted-imports)
- [ ] Sentry inicializado (DSN via mock ate bootstrap existir)

### Paginas-casca
- [ ] /login refinado com layout editorial signature
- [ ] / (dashboard) vazio com shell completo
- [ ] /404 com TorqueMark e copy editorial
- [ ] /401-403 com mensagem e logout

### Primitivos novos
- [ ] StepProgress com variantes numbered/iconed/minimal
- [ ] QuotaGauge com variantes bar/ring/inline
- [ ] ChannelBadge com 4 canais e 3 variantes
- [ ] Todos com shadow-hairline, tokens HSL, tipografia correta

### Seguranca
- [ ] Nenhum VITE_* com secret no bundle
- [ ] font-src 'self' viabilizado (sem Google Fonts CDN)
- [ ] dangerouslySetInnerHTML banido por ESLint
- [ ] Cookies httpOnly pattern documentado no AuthProvider

### Qualidade
- [ ] Zero violacao dos [[Criterios de Reprovacao]]
- [ ] Testes snapshot dos 20 primitivos (Vitest)
- [ ] TypeScript strict sem erros

## Estrategia de validacao

1. **Pre-implementacao:** spec revisada por agente reviewer (em andamento)
2. **Durante:** cada bloco e validado contra [[Criterios de Reprovacao]] antes de avancar
3. **Pos-implementacao:** agente de revisao independente audita o codebase inteiro contra esta spec
4. **Criterio de saida:** [[Checklist Sistema Base]] 100% marcado

## Documentos fonte

- [[Principios de Identidade Visual]]
- [[Design System Base]]
- [[Tipografia]]
- [[Motion e Animacao]]
- [[Componentes Primitivos]]
- [[Vocabulario de UI]]
- [[Criterios de Reprovacao]]
- [[Escopo do Sistema Base]]
- [[Checklist Sistema Base]]
- [[Plano de Execucao]]
- [[Direcao Visual Frontend]]
- [[Analise Pratica]]
- [[ADR-001-contratos-openapi-snake-camel]]
- [[ADR-002-realtime-websocket-go-hub]]
- [[ADR-003-auth-httponly-cookies-samesite]]
- [[ADR-004-paginacao-cursor-based]]
- [[ADR-005-tipografia-self-hosted]]
- [[ADR-006-jobs-assincronos-202-poll]]
