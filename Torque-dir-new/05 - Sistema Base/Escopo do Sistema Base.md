---
tags:
  - sistema-base
  - escopo
  - fundacao
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Escopo do Sistema Base

## Princípio

O Sistema Base contém **apenas o que é primitivo, essencial e de uso básico** para viabilizar a construção inicial do produto. É a fundação sobre a qual cada feature será erguida. Tudo que não for estritamente necessário à fundação fica fora — se pode ser construído no contexto de uma feature específica, não pertence aqui.

A régua é simples: **sem isso, nenhuma feature nasce. Com isso, qualquer feature pode nascer.**

## O que entra (lista canônica)

### Shell e navegação mínima
- `AppShell` — layout raiz com regiões fixas
- `Sidebar` com grupos vazios (sem módulos de produto listados)
- `TopBar` com slots para busca, status, perfil
- `CommandPalette` estrutural (sem comandos de domínio registrados)
- `TorqueMark` (marca visual / logotipo)

### Design system base
- Tokens HSL completos:
	- Canvas (superfícies, elevações)
	- Type (hierarquia tipográfica)
	- Accent (marca e estados interativos)
	- Semânticos (success, warning, danger, info)
	- Stages (por pipe — cores canônicas de estágio)
	- Calor (temperatura de lead)
	- Canal (WhatsApp, Instagram, Email, SMS)
	- Countdown (D-5 / D-3 / D-1)
	- Job (status de processamento async)
- Tipografia tripartida self-hosted (Fraunces + Instrument Sans + JetBrains Mono)

### Primitivos UI
- 17 primitivos já existentes (auditados)
- 3 primitivos de base a adicionar:
	- `StepProgress`
	- `QuotaGauge`
	- `ChannelBadge`

### Estrutura de pastas canônica
- `src/features/`
- `src/hooks/`
- `src/api/`
- `src/contracts/`
- `src/lib/{fetch,ws,domain}/`
- `src/providers/`

Ver [[Estrutura de Pastas]].

### Camada de contratos
- `openapi-typescript` configurado via script
- `api.gen.ts` (vazio inicial — preenchido quando backend expõe OpenAPI)
- `manual.ts` — rascunho das entidades fundamentais

### Cliente HTTP base
`src/lib/fetch.ts`:
- `credentials: 'include'`
- Interceptor 401 → refresh
- Strip de `organization_id` em mutations (tenancy enforçado no servidor)
- Error mapping uniforme
- Retry exponencial em erros transientes

### Autenticação mínima
- `/login` (página real, não placeholder)
- `AuthContext` + `AuthProvider`
- `useSession()`
- `ProtectedRoute` / `MasterRoute`
- `GET /auth/me` bundle (sessão + permissões + org em uma chamada)
- Cookies httpOnly, SameSite=Strict

### RBAC primitivo
- `useCanPerformAction(action, resource?)`
- `usePermission(permission)`
- `<PermissionGate>` (esqueleto sem regras de domínio)

### Estado global mínimo
- React Query defaults:
	- `staleTime: 5min`
	- `gcTime: 10min`
	- `refetchOnWindowFocus: false`
- `AuthContext`
- `OrgContext`
- `ThemeContext` (dark-only inicialmente)

### Realtime base
- `src/lib/ws.ts` — singleton com reconexão exponencial
- `useWSStatus()` — hook para status da conexão
- Sem handlers de domínio (cada feature registra os seus)

### Tratamento de erros e estados
- `<RootErrorBoundary>`
- `<RouteErrorBoundary>`
- Padrões `Skeleton` e `EmptyState`

### Segurança de produção
- CSP estrita nonce-based
- HSTS
- X-Frame-Options DENY
- X-Content-Type-Options nosniff
- Referrer-Policy
- Permissions-Policy
- CORS com whitelist
- `GET /api/bootstrap` (elimina necessidade de `VITE_*` com secrets)

### Observabilidade base
- Sentry com scrubbing de PII
- Logger cliente estruturado
- Badge de WS status no `TopBar`

### Páginas-casca
- `/login`
- `/` (dashboard vazio com layout correto)
- `/404`
- `/401-403`

### i18n
- `react-intl` estruturado
- Apenas PT-BR ativo (decisão validada — ver [[Pendencias e Lacunas]])

### Qualidade
- ESLint rules reais (não apenas `recommended`)
- Prettier
- TS strict reforçado

### Vocabulário
- `src/lib/vocabulary.ts` com termos canônicos (Funis, Leads, Estágio, etc.)

## O que NÃO entra (explicitamente)

Nada relacionado a produto específico. Lista enumerada para eliminar ambiguidade:

- Qualquer pipe específico
- Kanban com DnD
- Inbox multi-canal
- Copilot
- Workflow Builder
- Campanhas
- Analytics
- Master Admin
- Onboarding
- Checkout
- Quotas visuais
- Comissões
- Premiações
- Integração TinyERP
- PIX
- `FunnelChart`
- `CountdownBadge`
- `HeatSlider`
- `AchievementBadge`
- `OperationsCenter`
- `WorkflowCanvas`
- Wizards de qualquer tipo

**Tudo isso é [[00 - Mapa de Features]].**

## Critério de saída

O Sistema Base está pronto quando [[Checklist Sistema Base]] estiver 100% marcado. Só então F01 em [[00 - Mapa de Features]] pode começar.

## Referências

- [[Checklist Sistema Base]]
- [[Estrutura de Pastas]]
- [[Plano de Execucao]]
- [[Pendencias e Lacunas]]
- [[00 - Mapa de Features]]
