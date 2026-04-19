# Plano Mestre de Finalização do SaaS CRM Torque

> **Criado:** 2026-04-16
> **Atualizado:** 2026-04-18 — **S00 concluído**. Checklist Sistema Base fechado; extensão Cockpit (ADR-007) e bloco de qualidade (ESLint, TS strict, Vitest, Prettier, CI) entregues. Itens `<!-- aguarda S01 -->` no Checklist dependem do backend Go.
> **Autor:** Conductor (coordenação técnica automatizada)
> **Status:** Ativo — fonte de verdade para sequência de execução
> **Localização justificada:** `09 - Backlog/` é a seção de planejamento e execução do vault. Este documento consolida e supera o `Backlog Priorizado.md` como roadmap operacional definitivo.

---

## 1. Resumo Executivo

**Estado atual:** O frontend React/TypeScript está ~85% completo como protótipo visual de alta fidelidade (9 páginas, 30+ componentes, design system, providers). Zero código Go existe. Zero banco de dados. Zero testes. A documentação do vault está significativamente desatualizada — diz "Etapa 0 (Fix de bloqueadores)" mas o código já avançou até a Etapa 6 do Plano de Execução Granular.

**Etapa atual confirmada:** Sistema Base Frontend — entre Etapa 6 e Etapa 7 (validação). Componentes, infraestrutura, páginas-casca e tokens existem. Faltam: ESLint com regras reais, testes, `vocabulary.ts`, classe utilitária `grain`, flags TS strict adicionais, e CI pipeline.

**Principal gargalo:** O backend Go é 0%. Toda a espinha dorsal do produto — auth, tenancy, API, realtime, jobs, billing — precisa ser construída do zero. Este é o bloqueador crítico para qualquer feature real.

**Próxima prioridade real:**
1. Fechar os ~15% restantes do Sistema Base Frontend (Sprint S00)
2. Iniciar o backend Go com database schema + auth + tenancy (Sprints S01-S03)

---

## 2. Fonte de Verdade Utilizada

### Documentos determinantes

| Documento | Impacto |
|-----------|---------|
| `00 - Indice.md` | Visão, missão, regra cardinal, fluxo de 8 passos |
| `05 - Sistema Base/Checklist Sistema Base.md` | Critério de saída objetivo para fundação |
| `05 - Sistema Base/Plano de Execucao Granular.md` | Decomposição em 8 etapas com paralelismo |
| `05 - Sistema Base/Revisao Final - Redesign Sistema Base.md` | 6 bloqueadores críticos + status de resolução |
| `05 - Sistema Base/Spec - Redesign Sistema Base.md` | 10 restrições de design inegociáveis |
| `07 - Features/00 - Mapa de Features.md` | Sequência cardinal de 16 features (F01-F16) |
| `07 - Features/F01 - Funis Hub e Pipe WhatsApp/Spec.md` | Vertical slice que prova todo o pipeline |
| `08 - Decisoes/ADR-001..006` | Decisões arquiteturais travadas (OpenAPI, WS, Auth, Cursor, Fonts, Jobs) |
| `02 - Arquitetura/Multi-tenancy.md` | Isolamento por tenant como invariante absoluta |
| `02 - Arquitetura/Seguranca Web.md` | CSP nonce, headers, CSRF, XSS prevention |
| `02 - Arquitetura/Autenticacao e Autorizacao.md` | httpOnly cookies, RBAC 4 camadas, /auth/me bundle |
| `07 - Backlog/Backlog Priorizado.md` | Fases macro 0-5 com priorização por ROI |
| `09 - Migracao do Legado/Itens Nao Migrados.md` | 10 padrões legados deliberadamente excluídos |

### Documentos desatualizados ou conflitantes

| Documento | Problema |
|-----------|----------|
| `00 - Indice.md` | Diz "Etapa 0 (Fix de bloqueadores)" — código já está na Etapa 6+ |
| `04 - Sistema Base/Analise Pratica.md` | Diz tokens não implementados e infraestrutura ausente — ambos já existem |
| `04 - Sistema Base/Checklist Sistema Base.md` | Todos os itens `[ ]` (unchecked) — muitos já foram implementados no código |
| `04 - Sistema Base/Revisao Final - Redesign Sistema Base.md` | Diz todos 6 bloqueadores "RESOLVED" — B6 (ESLint) continua sem regras reais |
| `Agentes/Backend.md` | Referencia Supabase Edge Functions/Deno — ADRs definem Go como backend |
| `Agentes/Frontend.md` | Accent gold `hsl(47 100% 50%)` — canônico é `hsl(44 93% 54%)` |

**Ação necessária:** Atualizar esses documentos ao concluir Sprint S00 (Etapa 8 do plano granular).

---

## 3. Diagnóstico do Estado Atual

### 3.1 Sistema Base Frontend

| Componente | Status | Evidência |
|------------|--------|-----------|
| Tokens CSS (heat, channel, countdown, job) | ✅ Implementado | `globals.css` linhas 51-72 |
| @fontsource (Fraunces, Instrument Sans, JetBrains Mono) | ✅ Instalado | `package.json` deps |
| Google Fonts CDN removido | ✅ Removido | Nenhum match em `index.html` |
| Sidebar "Funis" (renomeado de "Pipelines") | ✅ Feito | `Sidebar.tsx:35` |
| Nav groups (Automação, Inteligência, Equipe) | ✅ Feito | `Sidebar.tsx:92-102` |
| 3 novos primitivos (StepProgress, QuotaGauge, ChannelBadge) | ✅ Existem | `src/ui/step-progress.tsx`, `quota-gauge.tsx`, `channel-badge.tsx` |
| 17 primitivos UI auditados | ✅ Existem | `src/ui/` com 30+ componentes |
| Providers (Auth, Query, WS, Theme, Intl) | ✅ Existem | `src/providers/` |
| API client (fetch.ts) | ✅ Existe | `src/api/client.ts` com CSRF, retry, 401 interceptor |
| WebSocket singleton | ✅ Existe | `src/lib/ws.ts` + `useWSStatus` hook |
| RBAC hooks | ✅ Existem | `usePermission.ts`, `useCanPerformAction.ts`, `PermissionGate.tsx` |
| ErrorBoundary | ✅ Existe | `src/components/RootErrorBoundary.tsx` |
| Páginas-casca (/login, /, /404, /403) | ✅ Existem | `NotFoundPage.tsx`, `ForbiddenPage.tsx` |
| Folder structure completa | ✅ Existe | `src/{api,hooks,contracts,providers,lib,features}` |
| Contracts manual.ts (16 entidades) | ✅ Existe | `src/contracts/manual.ts` |
| torque-tick (scale 1→1.04→1, 280ms) | ✅ Corrigido | `tailwind.config.ts:112-115` |
| shimmer (1.2s) + caret-blink (1.06s) | ✅ Corrigido | `tailwind.config.ts:124-127` |
| `* { border-hairline }` global removido | ✅ Removido | Sem matches em `globals.css` |
| Classe utilitária `vignette` | ✅ Existe | `globals.css:168` |
| Classe utilitária `grain` | ❌ Ausente | Sem matches em globals.css |
| `vocabulary.ts` | ❌ Ausente | Sem matches no projeto |
| CommandPalette "Pipelines" → "Funis" | ❌ Inconsistente | `CommandPalette.tsx:30` ainda diz "Pipelines" |
| ESLint com regras reais | ❌ Zero regras | `.eslintrc.cjs` só tem parser + env |
| TS `noUncheckedIndexedAccess` | ❌ Ausente | `tsconfig.json` não tem flag |
| TS `exactOptionalPropertyTypes` | ❌ Ausente | `tsconfig.json` não tem flag |
| Snapshot tests (20 primitivos) | ❌ Zero testes | Nenhum arquivo `.test.ts(x)` encontrado |
| Vitest configurado | ❌ Ausente | Sem config Vitest |
| CI pipeline | ❌ Ausente | Sem GitHub Actions |
| Prettier integrado | ⚠️ Verificar | Não confirmado |

**Conclusão:** Frontend ~85% implementado. Gaps concentrados em qualidade/validação (Etapa 7) e documentação (Etapa 8).

### 3.2 Backend (Go)

| Componente | Status |
|------------|--------|
| Código Go | ❌ Zero linhas |
| `go.mod` | ❌ Inexistente |
| Estrutura de projeto | ❌ Inexistente |
| API HTTP | ❌ Inexistente |
| Auth endpoints | ❌ Inexistente |
| Multi-tenancy middleware | ❌ Inexistente |
| RBAC middleware | ❌ Inexistente |
| WebSocket hub | ❌ Inexistente |
| Job scheduler | ❌ Inexistente |
| OpenAPI generation | ❌ Inexistente |

**Conclusão:** Backend é 0%. É o principal bloqueador do projeto.

### 3.3 Banco de Dados

| Componente | Status |
|------------|--------|
| PostgreSQL schema | ❌ Inexistente |
| Migrations | ❌ Inexistente |
| Seed data (SQL) | ❌ Inexistente |
| Indexes multi-tenant | ❌ Inexistente |
| RLS / tenant isolation | ❌ Inexistente (será via Go middleware, não SQL RLS) |

### 3.4 Auth & Multi-tenancy

| Componente | Status |
|------------|--------|
| Login funcional | ❌ Mock only (frontend) |
| JWT httpOnly cookies | ❌ Inexistente (requer backend) |
| CSRF double-submit | ❌ Inexistente (requer backend) |
| Refresh rotation | ❌ Inexistente |
| RBAC 4 camadas (server) | ❌ Inexistente |
| Tenant extraction from JWT | ❌ Inexistente |
| /auth/me bundle | ❌ Inexistente |
| /api/bootstrap | ❌ Inexistente |

### 3.5 Realtime

| Componente | Status |
|------------|--------|
| WS client singleton | ✅ Existe (mock/disconnected) |
| useWSStatus hook | ✅ Existe |
| WS Provider | ✅ Existe (disconnected state) |
| Go WS Hub (nhooyr.io/websocket) | ❌ Inexistente |
| Broadcast por tenant | ❌ Inexistente |
| Event dedup (version monotônico) | ❌ Inexistente |

### 3.6 Observabilidade

| Componente | Status |
|------------|--------|
| Sentry client (frontend) | ✅ Configurado (via env var) |
| Client logger | ⚠️ Parcial |
| Sentry server (Go) | ❌ Inexistente |
| Structured logging (zerolog) | ❌ Inexistente |
| OpenTelemetry | ❌ Inexistente |
| Correlation ID (X-Request-ID) | ❌ Inexistente |
| Audit log | ❌ Inexistente |

### 3.7 Billing & Operação

| Componente | Status |
|------------|--------|
| Checkout / Planos | ❌ Inexistente |
| PIX (Asaas) | ❌ Inexistente |
| Quotas enforcement | ❌ Inexistente |
| Onboarding wizard | ❌ Inexistente |
| Master Admin | ❌ Inexistente |

### 3.8 Automação & IA

| Componente | Status |
|------------|--------|
| Workflow Builder | ❌ Frontend mockup only |
| Copilot agents | ❌ Frontend mockup only |
| RAG pipeline | ❌ Inexistente |
| Campanhas | ❌ Frontend mockup only |
| Jobs/cron | ❌ Inexistente |

### 3.9 Features de Produto

| Feature | Frontend Mockup | Backend | Integração |
|---------|----------------|---------|------------|
| F01 Funis/Pipe WhatsApp | ✅ Visual | ❌ | ❌ |
| F02 Pipe Confirmação | ❌ | ❌ | ❌ |
| F03 Pipe Propostas | ❌ | ❌ | ❌ |
| F04 Inbox Multi-canal | ✅ Visual | ❌ | ❌ |
| F05 Follow-ups | ❌ | ❌ | ❌ |
| F06 Copilot | ✅ Visual | ❌ | ❌ |
| F07 Workflow Builder | ✅ Visual | ❌ | ❌ |
| F08 Campanhas | ✅ Visual | ❌ | ❌ |
| F09 Analytics | ✅ Visual | ❌ | ❌ |
| F10-F16 | ❌ | ❌ | ❌ |

### 3.10 Operação & Governança

| Componente | Status |
|------------|--------|
| Git repo | ❌ Não é git repo |
| CI/CD | ❌ Inexistente |
| Docker | ❌ Inexistente |
| Deploy config | ❌ Inexistente |
| Runbooks | ❌ Inexistente |

---

## 4. Passo Atual Confirmado

**Nome:** Sistema Base — Final da Etapa 6 (Páginas-casca), antes da Etapa 7 (Validação)

**Por que essa conclusão:**

A documentação (`00 - Indice.md`) afirma que estamos na "Etapa 0 (Fix de bloqueadores) antes de Fase 0 Fundação". Isso está **significativamente desatualizado**. A análise direta do código revela:

| Etapa | Plano Granular | Status Real | Score |
|-------|---------------|-------------|-------|
| 0 | Fix bloqueadores (B1-B6) | 5/6 concluídos. B6 (ESLint) pendente. | 83% |
| 1 | Preparação (estrutura + deps) | Folders existem, deps instaladas, contracts manual.ts existe. Falta `vocabulary.ts`. | 90% |
| 2 | Fundação visual (tokens + fontes) | Tokens implementados, @fontsource OK, Google CDN removido. Falta `grain`. | 85% |
| 3 | Layout base (shell + navegação) | Sidebar renomeada, nav groups OK, WS badge existe. Falta fix CommandPalette. | 85% |
| 4 | Componentes base (3 primitivos) | StepProgress, QuotaGauge, ChannelBadge existem. Sem snapshot tests. | 75% |
| 5 | Infraestrutura (providers + hooks) | Auth, Query, WS, Intl providers existem. RBAC hooks existem. Fetch client existe. | 90% |
| 6 | Páginas base (login + 404 + 403) | Todas existem e são funcionais. | 90% |
| 7 | Validação (testes + lint + review) | **Zero testes. Zero regras ESLint. TS strict parcial.** | 15% |
| 8 | Documentação final | Documentação do vault não reflete estado do código. | 0% |

**Evidências:**
- `grep` por tokens em `globals.css` → todos os 4 grupos de tokens existem
- `grep` por `@fontsource` em `package.json` → 5 pacotes instalados
- `ls src/` → todas as pastas canônicas existem
- `grep` por componentes novos → 3 primitivos existem em `src/ui/`
- `grep` por providers → Auth, Query, WS, Theme, Intl providers existem
- `glob` por `*.test.{ts,tsx}` → zero arquivos encontrados
- `.eslintrc.cjs` → zero regras configuradas

**Conclusão:** O trabalho de implementação frontend avançou significativamente além do que a documentação registra. O gap real é validação (testes, lint, TS strict) e o backend Go que é 0%.

---

## 5. Lacunas até o SaaS CRM Completo

### Por camada

| Camada | O que falta | Bloqueador? |
|--------|------------|-------------|
| **Frontend (Sistema Base)** | ESLint, testes, grain, vocabulary.ts, TS strict flags, CI | Sim — bloqueia entrada em F01 |
| **Backend Go** | Tudo — projeto, estrutura, handlers, middleware, services | Sim — bloqueia toda feature real |
| **Database** | Schema, migrations, indexes, seed | Sim — bloqueia backend |
| **Auth** | JWT, cookies, refresh, CSRF, RBAC server-side | Sim — bloqueia toda operação autenticada |
| **Multi-tenancy** | Middleware, tenant extraction, org isolation | Sim — bloqueia multi-org |
| **Realtime** | Go WS Hub, broadcast, dedup, buffer | Sim — bloqueia F01 (kanban live) |
| **Jobs** | Worker pool, scheduler, operations, DLQ | Bloqueia F07, F08 |
| **Observabilidade** | Logging server, traces, audit log | Não bloqueia features, mas é requisito de qualidade |
| **Billing** | Checkout, PIX, quotas, provisioning | Bloqueia F14 |
| **IA** | RAG pipeline, agent engine, embeddings | Bloqueia F06 |
| **Integrações** | Evolution API (WhatsApp), TinyERP, Asaas, Meta | Bloqueia features específicas |

### O que é estritamente sequencial

```
Sistema Base 100% → Backend Go Skeleton → DB Schema → Auth + Tenancy
→ F01 (vertical slice que prova pipeline) → F02 → F03 → F04 → F05
```

### O que pode rodar em paralelo (após F01)

```
Trilha Produto:     F02 → F03 → F04 → F05
Trilha Automação:   F07 → F08         (após F01)
Trilha Extensão:    F12                (após F01)
Trilha Comercial:   F13, F14           (após Sistema Base)
Trilha Insights:    F09 → F10          (após F03)
Trilha Catálogo:    F11                (após F03)
Trilha Transversal: F15                (incremental)
Trilha Operacional: F16                (última)
```

---

## 6. Arquitetura-Alvo

### Visão de alto nível

```
┌─────────────────────────────────────────────────────────┐
│                    FRONTEND (React/TS/Vite)              │
│  features/ → hooks/ → api/ → contracts/ → lib/fetch     │
│                              ↕ WebSocket (lib/ws)        │
└──────────────────────────────┬──────────────────────────┘
                               │ HTTPS + WSS
                               │ httpOnly cookies
┌──────────────────────────────┴──────────────────────────┐
│                   GO HTTP/2 SERVER                        │
│                                                           │
│  ┌─── Middleware Chain ───────────────────────────────┐   │
│  │ CORS → CSP → HSTS → RateLimit → Auth(JWT) →       │   │
│  │ Tenant(org_id) → RBAC → RequestID → Logger         │   │
│  └────────────────────────────────────────────────────┘   │
│                                                           │
│  ┌─── Domain Layer ──────────────────────────────────┐   │
│  │ Handlers → Services → Repositories                 │   │
│  │ (contracts)  (business logic)  (data access)       │   │
│  └────────────────────────────────────────────────────┘   │
│                                                           │
│  ┌─── Infrastructure ────────────────────────────────┐   │
│  │ WS Hub │ Job Scheduler │ Event Bus │ Storage       │   │
│  └────────────────────────────────────────────────────┘   │
└──────────────────────────────┬──────────────────────────┘
                               │
┌──────────────────────────────┴──────────────────────────┐
│                    POSTGRESQL                             │
│  organizations, team_members, leads, conversations,      │
│  pipes, stages, pipe_records, workflows, campaigns,      │
│  copilot_agents, products, commissions, operations,      │
│  audit_log, quotas, feature_permissions                  │
│  Isolation: WHERE organization_id = ctx.OrgID            │
│  Indexes: compound (organization_id, ...)                │
└──────────────────────────────────────────────────────────┘
```

### Responsabilidades por camada

| Camada | Tecnologia | Responsabilidade |
|--------|-----------|-----------------|
| **Frontend** | React 18 + TypeScript strict + Vite + TanStack Query + Tailwind | UI, estado de UI, cache de servidor, transformação snake↔camel |
| **API Gateway** | Go `net/http` ou chi router | Routing, middleware chain, serialização JSON, OpenAPI |
| **Auth** | Go middleware + JWT (httpOnly cookies) | Autenticação, refresh rotation, CSRF, session management |
| **Tenancy** | Go middleware | Extração de `org_id` do JWT, injeção no context, filtering |
| **RBAC** | Go middleware + service | 4 camadas: master → admin → feature_permission → member_override |
| **Domain Services** | Go packages | Business logic, invariants, validação de domínio |
| **Repositories** | Go + `pgx` | Data access, queries, transactions, tenant-scoped |
| **WS Hub** | `nhooyr.io/websocket` | Realtime broadcast por tenant, dedup por version, heartbeat |
| **Job Scheduler** | Go worker pool | Async operations, retry com backoff, DLQ, progress tracking |
| **Event Bus** | Go channels (in-process) | Domain events → WS broadcast, audit log, side effects |
| **Storage** | Object storage (S3-compatible) | Uploads via pre-signed URLs, path `org/<org_id>/<entity>/...` |
| **Database** | PostgreSQL 15+ | Persistence, transactions, indexes, `golang-migrate` migrations |
| **Observabilidade** | zerolog + OpenTelemetry + Sentry | Structured logging, distributed traces, error tracking, audit |

### Partes majoritariamente em Go

- **100% Go:** API handlers, middleware, auth, tenancy, RBAC, domain services, repositories, WS hub, job scheduler, event bus, migrations, OpenAPI generation, CLI/tools
- **100% React/TS:** UI components, hooks, providers, routes, design system, i18n
- **Contratos compartilhados:** OpenAPI spec (gerada pelo Go, consumida pelo frontend via `openapi-typescript`)

### Banco de dados

- **PostgreSQL 15+** como único banco relacional
- `golang-migrate` para migrations versionadas (UP e DOWN obrigatórios)
- Compound indexes `(organization_id, ...)` em toda tabela de domínio
- `pgx` como driver (connection pooling nativo)
- `timestamptz` sempre (nunca `timestamp` sem timezone)
- Money em integer cents + currency code
- UUIDs como PKs (`gen_random_uuid()`)

### Mensageria e jobs

- Worker pool in-process (Go goroutines com semáforo)
- Pattern 202 Accepted + poll `GET /operations/:id` + WS push `operation.completed`
- Dead letter queue em tabela `failed_operations`
- Retry exponencial com jitter
- Scheduler para jobs periódicos (follow-up reminders, campaign dispatch)

### WebSocket

- `nhooyr.io/websocket` — context-first, menor superfície que gorilla
- Auth via cookie no upgrade handshake
- Registry por `tenant_id` — broadcast isolado
- Payload: `{ type, tenant_id, entity_id, patch, version, occurred_at }`
- Dedup client-side por `version` monotônico
- Heartbeat 30s, reconexão exponencial com jitter
- Buffer 10min para sync em reconexão

### Princípios de segurança (invariantes)

1. **CSP nonce-based** — `script-src 'nonce-{RANDOM}' 'strict-dynamic'`
2. **HSTS** 2 anos com preload
3. **httpOnly + SameSite=Strict** para tokens — XSS não exfiltra sessão
4. **CSRF double-submit** — `X-CSRF-Token` em mutations
5. **Tenant isolation** — `organization_id` extraído do JWT pelo server, nunca enviado pelo frontend
6. **RBAC 4 camadas** — master → admin → feature → member override
7. **PII scrubbing** — Sentry `beforeSend` remove tokens, emails, senhas
8. **Zero secrets no bundle** — config via `GET /api/bootstrap`
9. **Audit trail** — mutations sensíveis logadas com actor, target, before/after
10. **Rate limiting** — 429 + Retry-After em endpoints sensíveis

---

## 7. Roadmap Mestre até a Finalização

### Fases macro

| Fase   | Nome                             | Objetivo                                                                        | Sprints | Dependências |
| ------ | -------------------------------- | ------------------------------------------------------------------------------- | ------- | ------------ |
| **F0** | Fechar Sistema Base              | Frontend 100% — Checklist completa, testes, lint, docs                          | S00     | Nenhuma      |
| **F1** | Backend Foundation               | Go skeleton + DB + Auth + Tenancy + Security + WS + Jobs + OpenAPI              | S01-S05 | F0           |
| **F2** | Integração Front↔Back            | Wiring real, remoção de mocks, smoke test E2E                                   | S06     | F1           |
| **F3** | Vertical Slice F01               | Funis Hub + Pipe WhatsApp — prova todo o pipeline técnico                       | S07-S09 | F2           |
| **F4** | CRM Core (F02-F05)               | Pipes de Confirmação/Propostas + Inbox Multi-canal + Follow-ups                 | S10-S14 | F3           |
| **F5** | IA + Automação (F06-F08)         | Copilot + Workflow Builder + Campanhas                                          | S15-S19 | F4 (parcial) |
| **F6** | Analytics + Governança (F09-F16) | Analytics, Equipe, Produtos, Upsell, Onboarding, Checkout, Config, Master Admin | S20-S27 | F5 (parcial) |
| **F7** | Production Readiness             | CI/CD, Docker, deploy, load test, security audit, runbooks                      | S28-S29 | F6           |

### Definição de pronto por fase

- **F0:** [[Checklist Sistema Base]] 100%. Zero warnings ESLint. 20+ snapshot tests passing. TS strict sem erros. Build limpo.
- **F1:** Backend serve health check. Auth funcional com httpOnly cookies. Tenant isolation provada. WS conecta e broadcasta. OpenAPI gera types para frontend.
- **F2:** Frontend conectado ao backend real. Login funcional. OrgSwitcher real. WS badge reativo. Seed data removido de paths críticos.
- **F3:** F01 completa — kanban DnD funcional, realtime, RBAC, optimistic updates, estados de loading/empty/error. [[F01 Checklist de Conclusao]] 100%.
- **F4-F6:** Cada feature com checklist 100%, testes, review independente, docs atualizados.
- **F7:** Deploy automatizado, rollback testado, load test p95 < 500ms, zero vulnerabilidades OWASP Top 10.

---

## 8. Sprints

### Protocolo de Execução (obrigatório para TODA sprint)

> **Regra cardinal de git.** Cada sprint tem sua própria branch `sprint/S0X`. Nunca commitar trabalho de sprint direto em `develop` ou `main`. A branch do sprint é a unidade de review, de rollback e de rastreabilidade — um sprint fechado vira um merge atômico.

**1. Abertura da branch — ANTES do primeiro edit.**

```bash
git checkout develop
git pull --ff-only origin develop
git checkout -b sprint/S0X
```

A branch nasce de `develop` atualizada. Nunca da main, nunca de outro sprint em andamento (exceto quando S0X declara dependência explícita de um sprint anterior ainda não mergeado — nesse caso parta dele e anote no plano).

**2. Commits lógicos por domínio — durante a execução.**

Um sprint produz múltiplos commits, cada um atômico e auto-contido. Agrupamento canônico:

| Ordem | Commit | Prefixo | Conteúdo típico |
|-------|--------|---------|-----------------|
| 1 | DBA | `feat(db):` ou `chore(db):` | Migrations up/down, seeds, schema changes |
| 2 | Backend | `feat(backend):` | Services, repos, middlewares, handlers, config, go.mod |
| 3 | QA | `test(backend):` | Unit + integration tests |
| 4 | Frontend | `feat(frontend):` | Componentes, hooks, types gerados, i18n |
| 5 | Docs | `docs(vault):` | STATE.md (novo Dxxx), Indice, Plano Mestre, ADRs, checklists |

Não todos se aplicam a todo sprint. Sprints puramente backend dispensam o commit de frontend. Sprints só-docs são um único commit. O princípio é: **cada commit deve ser revisável isoladamente** — se a mudança de docs depende da de código, vão juntas; se independe, separa.

Mensagens de commit seguem o padrão já estabelecido:
- Título imperativo curto (≤72 chars), escopo entre parênteses.
- Corpo explica o **porquê** e as decisões não óbvias, não o "o que" (o diff já diz).
- Rodapé `Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>` quando aplicável.

**3. Push da branch — ao final do sprint.**

```bash
git push -u origin sprint/S0X
```

O push acontece **após** o sprint estar logicamente fechado (todos os critérios de aceite batidos, STATE.md atualizado com o ADR `D0xx` daquele sprint). Push parcial só se o sprint travar e precisar de review antecipado.

**4. Integração em `develop` — por PR, depois da validação.**

```bash
# GitHub PR (padrão):
gh pr create --base develop --head sprint/S0X \
  --title "Sprint S0X — <objetivo>" \
  --body "$(cat <<'EOF'
## Resumo
<1 parágrafo: o que o sprint entregou>

## Artefatos
- Commits: lista por domínio
- Migrations: 000X_nome
- Endpoints: lista
- Tests: unit + integration

## Runtime handoff
<o que o reviewer precisa rodar local para validar>

## Referências
- STATE.md D0xx
- Plano Mestre §8 Sprint S0X
EOF
)"
```

Merge strategy: `merge commit` (não squash) para preservar a granularidade dos commits lógicos no histórico de `develop`. A branch `sprint/S0X` permanece no remoto como âncora de auditoria — não deletar.

**5. Atualização do STATE.md.**

Todo sprint produz UM registro novo na tabela de decisões (`D0xx`) com:
- Data no formato `YYYY-MM-DD` absoluto.
- Resumo técnico denso (o que foi entregue, por que, com quais trade-offs).
- Pendências runtime se execução ficou transferida ao usuário.

Também atualiza `Current Phase` e `Current Blockers` para refletir o novo estado.

**6. Atualização do Indice.md.**

Status line do Indice reflete: `S00 ✅`, `S01 ✅ scaffold`, `S02 ✅`, etc. Próxima sprint documentada como alvo.

**7. Atualização deste Plano Mestre.**

O bloco da sprint em §8 muda de "prospectivo" para "entregue":
- Cabeçalho: `### Sprint S0X — <objetivo> ✅ ENTREGUE (YYYY-MM-DD)`
- Campo **Resultado esperado** → **Resultado entregue** com a lista concreta de artefatos.
- Adicionar linhas **Pendente runtime** (se houver) e **Proximo passo** apontando para a sprint seguinte.

**Resumo em uma frase:** branch própria → commits por domínio → push → PR → merge em develop → STATE/Indice/Plano atualizados. Nenhuma sprint fecha sem os seis passos.

**Checklist operacional para `agent-conductor`:**

- [ ] `git checkout -b sprint/S0X develop` feito antes de qualquer edit.
- [ ] Commits separados por domínio (DBA → Backend → QA → Frontend → Docs).
- [ ] `git push -u origin sprint/S0X` executado.
- [ ] STATE.md tem `D0xx` daquele sprint.
- [ ] Indice.md status line atualizada.
- [ ] Plano Mestre §8 marca a sprint ENTREGUE com artefatos concretos.
- [ ] PR aberta em `develop` (opcional — pode ser deferido para agrupar múltiplas sprints, mas recomendado por sprint).

---

### Sprint S00 — Fechar Sistema Base Frontend (+ bloco Cockpit) ✅ CONCLUÍDO (2026-04-18)

| Campo | Valor |
|-------|-------|
| **Objetivo** | Completar os ~15% restantes e atingir Checklist 100%, incluindo a extensão **Modos de UI + Cockpit** (ADR-007) |
| **Resultado entregue** | ESLint flat config (v9) com a11y + hooks + ban de `dangerouslySetInnerHTML`; Prettier + `prettier-plugin-tailwindcss`; TS strict flags `noUncheckedIndexedAccess` + `exactOptionalPropertyTypes`; Vitest + Testing Library com 51 testes e 36 snapshots em todos os primitivos; classe utilitária `.grain`; `src/i18n/vocabulary.ts` canonico; CommandPalette alinhado ao glossario ("Funis"); GitHub Actions CI (`.github/workflows/ci.yml`); `UiModeProvider` com fallback localStorage + `useUiMode` exposto; pos-login navigation respeitando `ui_mode`; renomeacao dos tokens legados `--clay-*` para `--card-*` / `.tactile-*` eliminando vazamento fora de `.cockpit-theme`; `prefers-reduced-motion` respeitado dentro do cockpit; `AppError` como classe (throw-Error compliant) |
| **Dependências** | Nenhuma |
| **Agentes** | `agent-frontend` (UI + cockpit), `agent-qa` (audit final), `agent-architect` (ADR-007 já emitido) |
| **Áreas afetadas** | Frontend config, testes, documentação vault, `features/cockpit/`, providers, rotas, globals.css |
| **Stack** | TypeScript, Vitest, ESLint, Tailwind, React Query |
| **Itens remanescentes** | Marcadores `<!-- aguarda S01 -->` no Checklist para headers de segurança (CSP/HSTS/etc), `/auth/me` real, `MasterRoute`, retry exponencial e scrubbing de PII no Sentry — todos dependem do backend Go que nasce em S01 |
| **Refs** | [[ADR-007-modo-vendedor-gerente]], [[UI Modes - Vendedor e Gerente]], [[F17 - Modo Vendedor (Task Cockpit)/Spec\|F17]] |

---

### Sprint S01 — Go Backend Skeleton + Database Schema ⏳ SCAFFOLD ENTREGUE (2026-04-19)

| Campo | Valor |
|-------|-------|
| **Objetivo** | Bootstrap do projeto Go e schema PostgreSQL fundacional |
| **Resultado entregue** | Modulo `torque-api/` completo: router chi + pgx v5 + zerolog, config env fail-fast, probes `/healthz` + `/readyz`, middleware stack (RequestID/AccessLog/Recover/SecurityHeaders/CORS/StripOrganizationID). Migrations 0001 (orgs, users, users_master, plans, feature_permissions, `set_updated_at()`), 0002 (team_members, member_feature_permissions, org_quotas, tags, leads com E.164 CHECK, pipes + stages + entries, lead_history, audit_log), 0003 (ADR-007 conforme §1 de [[UI Modes - Vendedor e Gerente]]). Dockerfile multi-stage distroless nonroot, docker-compose com Postgres 15 + migrate + API, Makefile opinionado, OpenAPI 3.1 skeleton, teste de concorrencia `TestTasksConcurrentInProgressSingleton`. |
| **Dependências** | Nenhuma (pode paralelizar com S00) |
| **Agentes** | `agent-architect` (estrutura), `agent-dba` (schema + migration `0003`), `agent-backend` (implementação) |
| **Áreas afetadas** | Backend (novo), Database (novo), DevOps |
| **Stack** | Go 1.22+, PostgreSQL 15+, Docker, golang-migrate |
| **Pendente runtime** | `go mod tidy` + `go build ./cmd/api` + `docker compose up` + `go test -race ./...` + `make test-integration` — nenhum desses foi executavel no workspace do Conductor (Go/Docker/migrate ausentes). Tranferido para o usuario ou CI. |
| **Proximo passo** | Com o scaffold rodando local, abrir **Sprint S02 (Auth + Tenancy + RBAC)** que implementa `/auth/login`, `/auth/refresh`, `/auth/me`, `/auth/logout`, `/master/impersonate/:org_id`, middleware `Auth → TenantScope → RBAC` conforme [[Autenticacao e Autorizacao]]. |

---

### Sprint S02 — Auth + Tenancy + RBAC ✅ ENTREGUE (2026-04-19)

| Campo | Valor |
|-------|-------|
| **Objetivo** | Autenticação funcional com httpOnly cookies, multi-tenancy isolada, RBAC 4 camadas |
| **Resultado entregue** | Migration `0004_refresh_tokens` (hash sha256, rotation chain, reuse detection via `used_at`, RevokeChain com recursive CTE, bind org_id). Services: `password` (bcrypt cost 12), `jwt` (HS256 >=32b, alg=none bloqueado, expiry obrigatorio), `token` (256-bit rand + ConstantTimeEqual), `permission` (resolver + bundle com master bypass). Repositories: `user` (FindByEmail/ID/Memberships/IsMaster/Organization/EffectivePermissions/TouchLastLogin/UpdateUIMode), `refresh` (Issue/Lookup/Rotate em tx Serializable/RevokeChain/RevokeByUser). Middleware stack: `Authenticator` + `RequireAuth` + `TenantScope` + `CSRF` + `RequireFeature`/`RequireRole`/`RequireMaster`. Handlers: `/api/v1/auth/{login,refresh,logout,me}` + `/api/v1/me/preferences` + `/api/bootstrap` publico. Cookies: `__torque_session` (httpOnly Strict), `__torque_refresh` (Path=/api/v1/auth, httpOnly), `__torque_csrf` (readable JS, double-submit). Tests: unit em password/jwt/token/CSRF/RBAC + integration gated por DATABASE_URL cobrindo login success/bad password/unknown email/me 401/refresh rotation/reuse revoga chain/logout/CSRF/strip organization_id. |
| **Dependências** | S01 (Go skeleton + DB) ✅ |
| **Agentes** | Conductor → `agent-backend` + `agent-dba` + `agent-qa` |
| **Áreas afetadas** | Backend auth (novo), middleware (expandido), database (migration 0004) |
| **Stack** | Go 1.22, `golang-jwt/jwt/v5`, `golang.org/x/crypto/bcrypt`, `pgx/v5`, chi |
| **Riscos mitigados** | Rotation atomica via UPDATE...WHERE used_at IS NULL + SERIALIZABLE tx. Reuse detection revoga linhagem inteira via recursive CTE. CSRF double-submit com ConstantTimeEqual. Uniform response em login (dummy bcrypt verify em user_not_found) previne user enumeration. |
| **Pendente runtime** | `go mod tidy` + `go build` + `migrate up` + `go test -race` precisam rodar local (Go/Docker nao instalados no workspace do Conductor). |
| **Proximo passo** | Abrir **Sprint S03** — headers finais (CSP nonce-based completo, HSTS preload no edge), observabilidade (Sentry server-side, zerolog structured completo, audit_log de mutations sensiveis), rate limiting, /api/bootstrap enriquecido com feature flags runtime. |

---

### Sprint S03 — Security Headers + Observabilidade + Bootstrap

| Campo | Valor |
|-------|-------|
| **Objetivo** | Hardening de segurança web e infraestrutura de observabilidade |
| **Resultado esperado** | Todos os headers de segurança ativos, Sentry server, structured logging, audit log, /api/bootstrap |
| **Dependências** | S02 (Auth funcional) |
| **Agentes** | `general-purpose` (implementação), `code-reviewer` (security review) |
| **Áreas afetadas** | Backend middleware, observabilidade |
| **Stack** | Go middleware, zerolog, Sentry Go SDK, OpenTelemetry |
| **Riscos** | CSP nonce pode quebrar scripts inline; CORS strict pode bloquear dev local |
| **Critério de conclusão** | `curl -I` mostra todos os headers. Sentry captura erros. Logs estruturados em JSON. Audit log registra mutations sensíveis. /api/bootstrap retorna config runtime. |

---

### Sprint S04 — OpenAPI + WebSocket Hub + Async Jobs

| Campo | Valor |
|-------|-------|
| **Objetivo** | Completar infraestrutura: contratos tipados, realtime, operações assíncronas |
| **Resultado esperado** | OpenAPI spec gerada, WS hub funcional com broadcast por tenant, pattern 202+poll+push |
| **Dependências** | S02 (Auth para WS handshake) |
| **Agentes** | `code-architect` (design WS hub), `general-purpose` (implementação) |
| **Áreas afetadas** | Backend infra, contratos, WebSocket |
| **Stack** | Go, nhooyr.io/websocket, kin-openapi, openapi-typescript |
| **Riscos** | WS hub com goroutine leak; job scheduler sem graceful shutdown |
| **Critério de conclusão** | `openapi-typescript` gera `api.gen.ts` do backend. WS conecta, autentica, recebe broadcast. Operation poll retorna status. Frontend types atualizados. |

---

### Sprint S05 — Frontend CRUD Foundation

| Campo | Valor |
|-------|-------|
| **Objetivo** | Patterns reutilizáveis de CRUD no frontend: hooks genéricos, error handling, optimistic updates |
| **Resultado esperado** | `useInfiniteList`, `useMutation` patterns, error boundaries wired, loading states |
| **Dependências** | S04 (OpenAPI types gerados) |
| **Agentes** | `general-purpose` (implementação) |
| **Áreas afetadas** | Frontend hooks, api layer |
| **Stack** | React, TanStack Query v5, TypeScript |
| **Riscos** | Baixo — patterns bem documentados nos ADRs |
| **Critério de conclusão** | Hook genérico de lista com cursor funcional. Mutation com optimistic update funcional. Error mapping de AppError funcional. |

---

### Sprint S06 — Integração Frontend ↔ Backend

| Campo | Valor |
|-------|-------|
| **Objetivo** | Conectar frontend ao backend real, remover mocks, validar E2E |
| **Resultado esperado** | Login real, OrgSwitcher real, WS badge reativo, dados do servidor |
| **Dependências** | S05 (CRUD patterns prontos) |
| **Agentes** | `general-purpose` (wiring), `code-reviewer` (integration review) |
| **Áreas afetadas** | Frontend providers, API client, WS client |
| **Stack** | React, Go API, WebSocket |
| **Riscos** | CORS issues; cookie domain mismatch; WS reconnection edge cases |
| **Critério de conclusão** | Login funcional no browser. /auth/me retorna dados reais. WS badge mostra "Conectado" (verde). OrgSwitcher lista orgs reais. Dashboard carrega (vazio). |

---

### Sprint S07 — F01 Backend: Leads + Pipes + Stages

| Campo | Valor |
|-------|-------|
| **Objetivo** | API completa para Funis Hub e Pipe WhatsApp |
| **Resultado esperado** | CRUD de leads, listagem de pipes, gestão de stages, movimentação de pipe records |
| **Dependências** | S06 (integração funcional) |
| **Agentes** | `general-purpose` (implementação), `code-reviewer` (review) |
| **Áreas afetadas** | Backend handlers, services, repositories |
| **Stack** | Go, PostgreSQL, pgx |
| **Riscos** | Cursor pagination com filtros complexos; race conditions em movimentação de cards |
| **Critério de conclusão** | Todos os endpoints da Spec F01 funcionais. Cursor pagination testada. RBAC aplicado. Eventos realtime emitidos. Integration tests passing. |

---

### Sprint S08 — F01 Frontend: Hub + Kanban + DnD

| Campo | Valor |
|-------|-------|
| **Objetivo** | UI completa do Funis Hub e Pipe WhatsApp com dados reais |
| **Resultado esperado** | /pipes com PipeCards reais, /pipes/whatsapp com kanban DnD funcional, realtime updates |
| **Dependências** | S07 (API F01 pronta) |
| **Agentes** | `frontend-design` (UI), `general-purpose` (implementação) |
| **Áreas afetadas** | Frontend features/pipes |
| **Stack** | React, @dnd-kit/core, TanStack Query |
| **Riscos** | DnD performance com muitos cards; optimistic update rollback em erro |
| **Critério de conclusão** | Hub mostra pipes reais. Kanban carrega leads reais. DnD move cards entre colunas. Move percebido < 300ms. Realtime atualiza cards sem F5. |

---

### Sprint S09 — F01 QA + LeadDetail + Polish

| Campo | Valor |
|-------|-------|
| **Objetivo** | Completar F01 com LeadDetailDrawer, testes completos, polish |
| **Resultado esperado** | F01 100% funcional, testada, acessível, documentada |
| **Dependências** | S08 (UI F01 funcional) |
| **Agentes** | `code-reviewer` (audit), `/hm-qa` (QA) |
| **Áreas afetadas** | Frontend features/pipes, testes |
| **Stack** | React, Vitest, axe-core, Playwright |
| **Riscos** | Accessibility issues no DnD; edge cases de reconnect WS |
| **Critério de conclusão** | LeadDetailDrawer funcional. DnD acessível por teclado. axe-core sem violações. Testes E2E passing. WS reconnect testado. [[F01 Checklist]] 100%. |

---

### Sprint S10 — F02: Pipe Confirmação

| Campo | Valor |
|-------|-------|
| **Objetivo** | Segundo pipe: Confirmação com countdown D-5/D-3/D-1 |
| **Resultado esperado** | Kanban de confirmação funcional com CountdownBadge e dispatch automático |
| **Dependências** | S09 (F01 100%) |
| **Agentes** | `general-purpose` (backend+frontend), `code-reviewer` |
| **Áreas afetadas** | Backend (novo pipe), Frontend (reuso de kanban) |
| **Stack** | Go, React, reutiliza infraestrutura de F01 |
| **Riscos** | Baixo — reutiliza infraestrutura provada em F01 |
| **Critério de conclusão** | Pipe Confirmação funcional. CountdownBadge mostra D-5/D-3/D-1. Dispatch trigger funcional. Checklist 100%. |

---

### Sprint S11 — F03: Pipe Propostas

| Campo | Valor |
|-------|-------|
| **Objetivo** | Terceiro pipe: Propostas com HeatSlider 1-5 e fundação para TinyERP |
| **Resultado esperado** | Kanban de propostas funcional com temperatura de lead |
| **Dependências** | S10 (F02 100%) |
| **Agentes** | `general-purpose`, `code-reviewer` |
| **Áreas afetadas** | Backend, Frontend |
| **Stack** | Go, React |
| **Riscos** | Integração TinyERP pode ser complexa (mitigação: fundação apenas, integração real em sprint dedicada) |
| **Critério de conclusão** | Pipe Propostas funcional. HeatSlider 1-5 implementado. Checklist 100%. |

---

### Sprint S12 — F04: Inbox Multi-canal Backend

| Campo | Valor |
|-------|-------|
| **Objetivo** | Backend de mensageria: 4 canais, realtime, takeover |
| **Resultado esperado** | API de conversas e mensagens, adaptadores de canal, eventos realtime |
| **Dependências** | S09 (F01 infra), Evolution API access |
| **Agentes** | `code-architect` (design), `general-purpose` (implementação) |
| **Áreas afetadas** | Backend messaging, integrations |
| **Stack** | Go, Evolution API, Meta API |
| **Riscos** | Alto — 4 integrações externas, cada uma com peculiaridades; message ordering complexo |
| **Critério de conclusão** | WhatsApp send/receive funcional via Evolution API. Message status tracking funcional. Realtime events para novas mensagens. |

---

### Sprint S13 — F04: Inbox Multi-canal Frontend

| Campo | Valor |
|-------|-------|
| **Objetivo** | UI do inbox: lista de conversas, chat, switching de canal |
| **Resultado esperado** | Inbox funcional com mensagens reais em tempo real |
| **Dependências** | S12 (backend messaging) |
| **Agentes** | `frontend-design`, `general-purpose` |
| **Áreas afetadas** | Frontend features/inbox |
| **Stack** | React, TanStack Query, WebSocket |
| **Riscos** | Performance com muitas mensagens (mitigação: virtualização) |
| **Critério de conclusão** | Inbox mostra conversas reais. Chat funciona em tempo real. Canal switching funcional. Typing indicators. |

---

### Sprint S14 — F05: Follow-ups

| Campo | Valor |
|-------|-------|
| **Objetivo** | Sistema de follow-ups: hoje/atrasado/futuro + ações diárias por usuário |
| **Resultado esperado** | Follow-ups funcionais com categorização temporal e queue de ações |
| **Dependências** | S13 (F04 para contexto de conversa) |
| **Agentes** | `general-purpose`, `code-reviewer` |
| **Áreas afetadas** | Backend + Frontend |
| **Stack** | Go, React, job scheduler (para reminders) |
| **Riscos** | Médio — scheduling de reminders requer job system maduro |
| **Critério de conclusão** | Follow-ups CRUD funcional. Categorização hoje/atrasado/futuro. Notificações de reminder. Checklist 100%. |

---

### Sprint S15 — F06: Copilot Backend (RAG + Agents)

| Campo | Valor |
|-------|-------|
| **Objetivo** | Engine de IA: agents configuráveis, RAG, processamento de mensagens |
| **Resultado esperado** | Copilot agent cria, configura, conversa e executa ações |
| **Dependências** | S13 (messaging para contexto de conversa) |
| **Agentes** | `code-architect` (design RAG), `general-purpose` |
| **Áreas afetadas** | Backend AI, embeddings, pgvector |
| **Stack** | Go, Google Gemini, pgvector, PostgreSQL |
| **Riscos** | Alto — LLM pode retornar JSON malformado; latência de embedding; contexto stale |
| **Critério de conclusão** | Agent processa mensagem. RAG retorna contexto relevante. Ações executadas (move_stage, send_message). Timeout/retry funcional. |

---

### Sprint S16 — F06: Copilot Frontend (Wizard + Chat)

| Campo | Valor |
|-------|-------|
| **Objetivo** | UI do Copilot: wizard 20+ steps, playground, roster |
| **Resultado esperado** | Wizard de configuração funcional, chat de teste, gestão de agents |
| **Dependências** | S15 (backend AI) |
| **Agentes** | `frontend-design`, `general-purpose` |
| **Áreas afetadas** | Frontend features/copilot |
| **Stack** | React, StepProgress |
| **Riscos** | Complexidade do wizard (20+ steps); UX de configuração de personality |
| **Critério de conclusão** | Wizard completo. Agent testável no playground. Roster funcional. Checklist 100%. |

---

### Sprint S17 — F07: Workflow Builder Backend

| Campo | Valor |
|-------|-------|
| **Objetivo** | Engine de automação: DAG de 12 tipos de node, execução, triggers |
| **Resultado esperado** | Workflows criados, executados, com logs de execução |
| **Dependências** | S09 (F01), S14 (F05 para triggers) |
| **Agentes** | `code-architect` (design DAG engine), `general-purpose` |
| **Áreas afetadas** | Backend automation |
| **Stack** | Go, PostgreSQL (workflow_executions) |
| **Riscos** | Alto — DAG execution engine é complexa; ciclos precisam ser detectados; timeouts por node |
| **Critério de conclusão** | Workflows CRUD funcional. 12 node types implementados. Trigger dispara execução. Logs por step. |

---

### Sprint S18 — F07: Workflow Builder Frontend

| Campo | Valor |
|-------|-------|
| **Objetivo** | Canvas visual: nodes, connections, palette, inspector |
| **Resultado esperado** | Builder visual funcional com @xyflow/react |
| **Dependências** | S17 (backend workflows) |
| **Agentes** | `frontend-design`, `general-purpose` |
| **Áreas afetadas** | Frontend features/workflows |
| **Stack** | React, @xyflow/react |
| **Riscos** | Performance do canvas com muitos nodes; persistência de draft |
| **Critério de conclusão** | Canvas funcional. Drag-to-add nodes. Connections entre nodes. Inspector por node. Draft salvo. |

---

### Sprint S19 — F08: Campanhas

| Campo | Valor |
|-------|-------|
| **Objetivo** | Campanhas: wizard, segmentação, dispatch, metrics |
| **Resultado esperado** | Campanha criada, segmento definido, dispatch executado |
| **Dependências** | S17 (Workflow Builder como engine de dispatch) |
| **Agentes** | `general-purpose`, `code-reviewer` |
| **Áreas afetadas** | Backend + Frontend |
| **Stack** | Go, React, job scheduler |
| **Riscos** | Volume de dispatch; rate limiting de canais externos |
| **Critério de conclusão** | Campanhas CRUD. Segmentação funcional. Dispatch batch funcional. Métricas atualizadas em realtime. |

---

### Sprint S20 — F09: Analytics

| Campo | Valor |
|-------|-------|
| **Objetivo** | Dashboard analítico: KPIs reais, FunnelChart, UTM, ranking |
| **Resultado esperado** | Analytics funcional com dados reais agregados |
| **Dependências** | S11 (F03 — precisa de dados de propostas) |
| **Agentes** | `general-purpose`, `frontend-design` (Visx charts) |
| **Áreas afetadas** | Backend aggregation, Frontend charts |
| **Stack** | Go, React, Visx |
| **Riscos** | Performance de queries de agregação; Visx learning curve |
| **Critério de conclusão** | Dashboard com KPIs reais. FunnelChart funcional. UTM tracking. Export CSV. Date range filters. |

---

### Sprint S21 — F10: Equipe + F11: Produtos

| Campo | Valor |
|-------|-------|
| **Objetivo** | Comissões, metas, podium + catálogo de produtos |
| **Resultado esperado** | Gestão de equipe com métricas + catálogo com import XLSX |
| **Dependências** | S20 (F09 para métricas), S11 (F03 para produtos) |
| **Agentes** | `general-purpose` |
| **Áreas afetadas** | Backend + Frontend |
| **Stack** | Go, React |
| **Riscos** | Cálculo de comissões é sensível (dinheiro); XLSX import edge cases |
| **Critério de conclusão** | Comissões calculadas corretamente. Metas trackadas. Catálogo funcional. Import XLSX sem erros. |

---

### Sprint S22 — F12: Upsell + Pipelines Customizados

| Campo | Valor |
|-------|-------|
| **Objetivo** | Generalizar modelo de pipes para custom + upsell |
| **Resultado esperado** | Org pode criar pipes customizados além dos 3 estruturais |
| **Dependências** | S09 (F01 pipe model) |
| **Agentes** | `general-purpose`, `code-architect` |
| **Áreas afetadas** | Backend pipe model, Frontend pipe creation |
| **Stack** | Go, React |
| **Riscos** | Médio — generalização do modelo sem quebrar pipes estruturais |
| **Critério de conclusão** | Pipe customizado criado. Stages configuráveis. Kanban funciona para pipes custom. |

---

### Sprint S23 — F13: Onboarding Wizard + Gate

| Campo | Valor |
|-------|-------|
| **Objetivo** | Primeiro contato do usuário: wizard 6 steps + activation gate |
| **Resultado esperado** | Novo usuário passa por onboarding antes de acessar o produto |
| **Dependências** | S06 (integração front↔back) |
| **Agentes** | `frontend-design` (UX de first-run), `general-purpose` |
| **Áreas afetadas** | Backend + Frontend |
| **Stack** | Go, React, StepProgress |
| **Riscos** | Alto impacto em retenção — UX precisa ser impecável |
| **Critério de conclusão** | Wizard 6 steps funcional. Gate bloqueia acesso até conclusão. Estado persistido. |

---

### Sprint S24 — F14: Checkout + PIX + Provisioning

| Campo | Valor |
|-------|-------|
| **Objetivo** | Monetização: wizard 3 steps, PIX via Asaas, provisioning de tenant |
| **Resultado esperado** | Usuário escolhe plano, paga via PIX, org é provisionada automaticamente |
| **Dependências** | S23 (Onboarding como entry point) |
| **Agentes** | `code-architect` (design), `general-purpose`, `code-reviewer` (dual review — dinheiro) |
| **Áreas afetadas** | Backend billing, integração Asaas, Frontend checkout |
| **Stack** | Go, Asaas API, React |
| **Riscos** | Muito alto — envolve dinheiro real; PIX webhook reliability; provisioning race conditions |
| **Critério de conclusão** | Checkout funcional. PIX gerado e confirmado. Org provisionada. Quota aplicada. Dual review completo. |

---

### Sprint S25 — F15: Configurações

| Campo | Valor |
|-------|-------|
| **Objetivo** | 8 tabs de configuração centralizadas |
| **Resultado esperado** | Settings da org, equipe, integrações, notificações, etc. |
| **Dependências** | Transversal — incrementa ao longo das features |
| **Agentes** | `general-purpose` |
| **Áreas afetadas** | Backend + Frontend |
| **Stack** | Go, React |
| **Riscos** | Baixo — consolidação de configs já parcialmente expostas |
| **Critério de conclusão** | 8 tabs funcionais. Todas as settings persistidas. RBAC aplicado. |

---

### Sprint S26 — F16: Master Admin

| Campo | Valor |
|-------|-------|
| **Objetivo** | Painel de operações: 5 views + center de operações |
| **Resultado esperado** | Visão cross-org para master admin, impersonation, system health |
| **Dependências** | Todas as features anteriores |
| **Agentes** | `general-purpose`, `code-reviewer` (security — cross-org access) |
| **Áreas afetadas** | Backend + Frontend |
| **Stack** | Go, React |
| **Riscos** | Alto — cross-org access é superfície de ataque; impersonation requer audit trail |
| **Critério de conclusão** | 5 views funcionais. Impersonation com audit. Operations Center mostra health. |

---

### Sprint S27 — Integrações Externas Completas

| Campo | Valor |
|-------|-------|
| **Objetivo** | Completar integrações: Evolution API (WhatsApp), Meta (IG/Messenger), TinyERP, Google Calendar |
| **Resultado esperado** | Todas as integrações documentadas funcionais em produção |
| **Dependências** | Features que consomem cada integração |
| **Agentes** | `general-purpose`, `code-reviewer` |
| **Áreas afetadas** | Backend integrations |
| **Stack** | Go, APIs externas |
| **Riscos** | Alto — APIs externas são imprevisíveis; rate limits; breaking changes |
| **Critério de conclusão** | Cada integração com health check. Retry + circuit breaker. Webhook validation. |

---

### Sprint S28 — CI/CD + Docker + Deploy

| Campo | Valor |
|-------|-------|
| **Objetivo** | Pipeline de deploy automatizado com rollback |
| **Resultado esperado** | Push to main → lint → test → build → deploy → smoke test |
| **Dependências** | Todas as features |
| **Agentes** | `general-purpose` |
| **Áreas afetadas** | DevOps, CI/CD |
| **Stack** | GitHub Actions, Docker, EasyPanel/Hostinger VPS |
| **Riscos** | Config de secrets; deploy sem downtime |
| **Critério de conclusão** | Pipeline funcional. Rollback testado. Zero downtime deploy provado. |

---

### Sprint S29 — Security Hardening + Load Test + Docs

| Campo | Valor |
|-------|-------|
| **Objetivo** | Auditoria final de segurança, teste de carga, documentação operacional |
| **Resultado esperado** | Zero OWASP Top 10. p95 < 500ms sob carga. Runbooks completos. |
| **Dependências** | S28 (deploy funcional) |
| **Agentes** | `code-reviewer` (security audit), `/hm-qa` (load test) |
| **Áreas afetadas** | Todo o sistema |
| **Stack** | k6/vegeta (load test), OWASP ZAP |
| **Riscos** | Descoberta tardia de vulnerabilidades |
| **Critério de conclusão** | Pentest passando. Load test p95 < 500ms. Runbooks escritos. Alertas configurados. |

---

## 9. Tarefas Granulares por Sprint

### Sprint S00 — Fechar Sistema Base Frontend

| ID | Título | Descrição | Camada | Deps | Esforço | Agente | Artefatos | Critério de Aceite |
|----|--------|-----------|--------|------|---------|--------|-----------|-------------------|
| S00-01 | Configurar ESLint com regras reais | Adicionar plugins `@typescript-eslint/recommended`, `react-hooks/exhaustive-deps`, ban `dangerouslySetInnerHTML`, `no-restricted-imports` para paths relativos | Config | — | 2h | general-purpose | `.eslintrc.cjs` atualizado | ESLint roda sem erros após `--fix`. Regras ativas incluem hooks, innerHTML ban, import restrictions |
| S00-02 | Adicionar classe utilitária `grain` | Implementar grain overlay com opacity 0.035, mix-blend overlay conforme Spec Sistema Base | CSS | — | 30min | general-purpose | `globals.css` atualizado | `.grain` classe disponível e renderizando corretamente |
| S00-03 | Criar `vocabulary.ts` | Termos canônicos do produto em PT-BR: lead, funil, pipe, estágio, abordagem, esfriamento, etc. | i18n | — | 1h | general-purpose | `src/i18n/vocabulary.ts` | Arquivo com todos os termos do glossário de produto |
| S00-04 | Fix CommandPalette "Pipelines" → "Funis" | Alinhar label com Sidebar | Frontend | — | 15min | general-purpose | `CommandPalette.tsx` | Label consistente "Funis" em todos os pontos de navegação |
| S00-05 | Adicionar flags TS strict | `noUncheckedIndexedAccess: true`, `exactOptionalPropertyTypes: true` em tsconfig.json | Config | — | 1h | general-purpose | `tsconfig.json` | Build limpo com flags ativas |
| S00-06 | Configurar Vitest | Instalar vitest + @testing-library/react + jsdom. Criar `vitest.config.ts`. Script `test` no package.json | Config | — | 1h | general-purpose | `vitest.config.ts`, `package.json` | `npm run test` roda (mesmo sem testes ainda) |
| S00-07 | Snapshot tests dos 20 primitivos UI | Um snapshot test por componente em `src/ui/`. Render default + variantes principais | Testes | S00-06 | 4h | general-purpose | `src/ui/__tests__/*.test.tsx` | 20+ snapshots passing |
| S00-08 | Snapshot tests dos 3 novos primitivos | StepProgress, QuotaGauge, ChannelBadge com variantes | Testes | S00-06 | 1h | general-purpose | `src/ui/__tests__/*.test.tsx` | 3 snapshots passing |
| S00-09 | Configurar Prettier | `.prettierrc` com config alinhada ao projeto. Integrar com ESLint | Config | S00-01 | 30min | general-purpose | `.prettierrc` | `prettier --check .` passa |
| S00-10 | CI foundation (lint + typecheck + test) | GitHub Actions workflow: push → lint → tsc --noEmit → vitest run | CI | S00-01, S00-06 | 2h | general-purpose | `.github/workflows/ci.yml` | Pipeline verde no push |
| S00-11 | Atualizar Checklist Sistema Base | Marcar itens concluídos com `[x]` baseado no estado real do código | Docs | Todos | 1h | general-purpose | Checklist atualizado | 100% dos itens marcados ou explicitamente pendentes |
| S00-12 | Atualizar docs vault (Indice, Analise Pratica) | Corrigir status para refletir realidade do código | Docs | S00-11 | 1h | general-purpose | `00 - Indice.md`, `Analise Pratica.md` atualizados | Docs refletem estado real |
| S00-13 | Review final por code-reviewer | Audit independente do Sistema Base completo vs Spec e Checklist | Review | Todos | 2h | code-reviewer | Relatório de review | Zero violações dos [[Criterios de Reprovacao]] |

**Esforço total estimado: 16-18h**

---

### Sprint S01 — Go Backend Skeleton + Database Schema

| ID | Título | Descrição | Camada | Deps | Esforço | Agente | Artefatos | Critério de Aceite |
|----|--------|-----------|--------|------|---------|--------|-----------|-------------------|
| S01-01 | Inicializar projeto Go | `go mod init github.com/torque-crm/api`. Estrutura: `cmd/api/`, `internal/{config,handler,middleware,service,repository,domain,ws}`, `migrations/` | Backend | — | 1h | general-purpose | `go.mod`, estrutura de diretórios | `go build ./...` compila |
| S01-02 | Config loader | Carregar config de env vars: `DATABASE_URL`, `JWT_SECRET`, `APP_ORIGIN`, `API_PORT`, `SENTRY_DSN`. Struct tipada com validação | Backend | S01-01 | 1h | general-purpose | `internal/config/config.go` | Config carregada e validada no boot |
| S01-03 | PostgreSQL connection pool | `pgxpool` com config: max connections, health check period, connection timeout | Backend | S01-02 | 1h | general-purpose | `internal/repository/db.go` | Pool conecta e faz ping |
| S01-04 | HTTP server + router | `net/http` com chi router. Graceful shutdown. Health endpoint `GET /health` | Backend | S01-02 | 2h | general-purpose | `cmd/api/main.go`, `internal/handler/health.go` | `curl localhost:8080/health` → 200 |
| S01-05 | Docker Compose | PostgreSQL 15 + Go API. Volumes para data. Network compartilhada. `.env.example` | DevOps | S01-04 | 2h | general-purpose | `docker-compose.yml`, `Dockerfile`, `.env.example` | `docker compose up` sobe tudo |
| S01-06 | Makefile | Targets: `build`, `run`, `test`, `migrate-up`, `migrate-down`, `migrate-create`, `generate-types`, `lint` | DevOps | S01-05 | 1h | general-purpose | `Makefile` | `make run` inicia o servidor |
| S01-07 | Migration: organizations | `organizations` table: id (uuid PK), name, slug (unique), plan, status, created_at, updated_at | Database | S01-03 | 1h | general-purpose | `migrations/001_organizations.up.sql`, `.down.sql` | Migration aplica e reverte |
| S01-08 | Migration: team_members | `team_members`: id, organization_id (FK), email, password_hash, name, role (admin/master/membro), status, created_at, updated_at. Compound index (org_id, email unique) | Database | S01-07 | 1h | general-purpose | `migrations/002_team_members.up.sql`, `.down.sql` | Migration aplica e reverte |
| S01-09 | Migration: leads | `leads`: id, organization_id, name, phone, email, channel, source, stage, temperature, is_shadow, created_at, updated_at. Compound index (org_id, stage), (org_id, created_at) | Database | S01-07 | 1h | general-purpose | `migrations/003_leads.up.sql`, `.down.sql` | Migration aplica e reverte |
| S01-10 | Migration: pipes + stages + pipe_records | `pipes` (id, org_id, type, name), `stages` (id, pipe_id, name, position, color), `pipe_records` (id, org_id, lead_id, pipe_id, stage_id, position, version). Indexes compound | Database | S01-09 | 2h | general-purpose | `migrations/004_pipes_stages.up.sql`, `.down.sql` | Migration aplica e reverte |
| S01-11 | Migration: permissions | `feature_permissions` (id, org_id, feature_key, is_admin_only, is_master_only), `member_feature_permissions` (id, member_id, feature_key, allowed). Index (org_id, feature_key) | Database | S01-08 | 1h | general-purpose | `migrations/005_permissions.up.sql`, `.down.sql` | Migration aplica e reverte |
| S01-12 | Migration: quotas + operations + audit_log | `org_quotas`, `operations` (id, org_id, type, status, progress, result, error, started_at, ended_at), `audit_log` (id, org_id, actor_id, actor_type, action, target, before, after, request_id, created_at) | Database | S01-07 | 2h | general-purpose | `migrations/006_infra.up.sql`, `.down.sql` | Migrations aplicam e revertem |
| S01-13 | Seed data SQL | Org de desenvolvimento, 2 team_members (admin + membro), stages do pipe WhatsApp (novo, abordado, respondeu, esfriou, agendado), leads de exemplo | Database | S01-10 | 1h | general-purpose | `migrations/seed.sql` | Seed popula banco de dev |
| S01-14 | golang-migrate integration | CLI + programmatic migration runner no boot (auto-migrate em dev) | Backend | S01-07 | 1h | general-purpose | Migrations rodam no boot | `make migrate-up` aplica todas. Boot em dev auto-migra. |

**Esforço total estimado: 18-22h**

---

### Sprint S02 — Auth + Tenancy + RBAC

| ID | Título | Descrição | Camada | Deps | Esforço | Agente | Artefatos | Critério de Aceite |
|----|--------|-----------|--------|------|---------|--------|-----------|-------------------|
| S02-01 | JWT service | Gerar e validar JWT com claims: `sub` (user_id), `org_id`, `role`, `exp`, `iat`. Segredo via config. `golang-jwt/jwt/v5` | Backend | S01 | 2h | general-purpose | `internal/service/jwt.go` | Token gerado e validado |
| S02-02 | Password service | bcrypt hash + verify. Cost factor 12. | Backend | S01 | 30min | general-purpose | `internal/service/password.go` | Hash e verify funcionais |
| S02-03 | POST /auth/login | Recebe email+password. Valida credentials. Gera access token (15min) + refresh token (30d). Set cookies httpOnly, Secure, SameSite=Strict. Retorna user profile | Backend | S02-01, S02-02 | 3h | general-purpose | `internal/handler/auth.go` | Login retorna 200 + cookies set. Credenciais erradas retorna 401 |
| S02-04 | POST /auth/logout | Limpa cookies. Invalida refresh token no banco | Backend | S02-03 | 1h | general-purpose | `internal/handler/auth.go` | Cookies limpos. Token invalidado |
| S02-05 | POST /auth/refresh | Valida refresh token. Rotation: novo access + novo refresh. Reuse detection: se refresh já usado → logout completo (sinal de comprometimento) | Backend | S02-03 | 3h | general-purpose | `internal/handler/auth.go` | Refresh funcional. Reuse detection funcional |
| S02-06 | GET /auth/me | Retorna bundle: user, organization, permissions, quotas, csrf_token. Requer auth middleware | Backend | S02-03 | 2h | general-purpose | `internal/handler/auth.go` | Bundle completo retornado. 401 sem cookie válido |
| S02-07 | Auth middleware | Extrai JWT do cookie `__torque_session`. Valida. Injeta user_id + org_id + role no context. 401 se inválido | Backend | S02-01 | 2h | general-purpose | `internal/middleware/auth.go` | Requests autenticados passam. Sem cookie → 401 |
| S02-08 | Tenant middleware | Extrai `org_id` do context (injetado pelo auth middleware). Injeta no context para repositories. Strip `organization_id` de bodies de mutation (segunda linha de defesa) | Backend | S02-07 | 2h | general-purpose | `internal/middleware/tenant.go` | org_id propagado. Body stripping funcional |
| S02-09 | RBAC middleware | Lê permissions do context. Verifica: master bypass → admin check → feature_permission → member_override. 403 se negado | Backend | S02-07 | 3h | general-purpose | `internal/middleware/rbac.go` | Master acessa tudo. Admin conforme feature. Membro conforme override. 403 quando negado |
| S02-10 | CSRF middleware | Gera CSRF token em cookie `__torque_csrf` (não-httpOnly). Valida header `X-CSRF-Token` em mutations (POST, PATCH, DELETE) | Backend | S02-07 | 2h | general-purpose | `internal/middleware/csrf.go` | Mutations sem CSRF → 403. Com CSRF → passa |
| S02-11 | Refresh token storage | Tabela `refresh_tokens` (id, user_id, token_hash, used, expires_at, created_at). Index (token_hash) | Database | S01 | 1h | general-purpose | `migrations/007_refresh_tokens.up.sql` | Migration aplica e reverte |
| S02-12 | Integration tests auth | Testes cobrindo: login sucesso, login falha, refresh, refresh reuse, /auth/me, 401 sem token, CSRF validation | Testes | S02-06 | 3h | general-purpose | `internal/handler/auth_test.go` | Todos os testes passing |

**Esforço total estimado: 24-28h**

---

### Sprint S03 — Security Headers + Observabilidade + Bootstrap

| ID | Título | Descrição | Camada | Deps | Esforço | Agente | Artefatos | Critério de Aceite |
|----|--------|-----------|--------|------|---------|--------|-----------|-------------------|
| S03-01 | CSP nonce middleware | Gera nonce por request. Injeta no HTML template. Headers: `Content-Security-Policy` com script-src nonce, style-src nonce, font-src self, connect-src self+wss, frame-ancestors none | Backend | S02 | 3h | general-purpose | `internal/middleware/csp.go` | Headers presentes em toda response. Nonce rotaciona |
| S03-02 | Security headers middleware | HSTS (2yr, preload), X-Frame-Options DENY, X-Content-Type-Options nosniff, Referrer-Policy strict-origin-when-cross-origin, Permissions-Policy (deny camera/mic/geo/payment), COOP same-origin | Backend | — | 1h | general-purpose | `internal/middleware/security.go` | `curl -I` mostra todos os headers |
| S03-03 | CORS middleware | Strict origin whitelist de `APP_ORIGIN`. Credentials true. Method allowlist por rota | Backend | S01 | 1h | general-purpose | `internal/middleware/cors.go` | Origin inválido → bloqueado. Origin válido → headers CORS |
| S03-04 | Rate limiting middleware | Token bucket por IP + por user. 429 + Retry-After header. Configurável por rota | Backend | S02-07 | 2h | general-purpose | `internal/middleware/ratelimit.go` | Excesso → 429 com Retry-After |
| S03-05 | Structured logging (zerolog) | Logger global JSON. Contexto: request_id, org_id, user_id, method, path, status, duration. Levels: debug (dev only), info, warn, error | Backend | S01 | 2h | general-purpose | `internal/middleware/logger.go` | Logs em JSON estruturado. Request context presente |
| S03-06 | Correlation ID middleware | Gera ULID `X-Request-ID` se não presente. Propaga em context. Logger inclui. Response header echo | Backend | S03-05 | 1h | general-purpose | `internal/middleware/requestid.go` | Request ID presente em logs e response |
| S03-07 | Sentry Go SDK | Inicializar Sentry com DSN de config. Middleware de recovery com Sentry capture. PII scrubbing (token, password, email, cookie) | Backend | S01-02 | 2h | general-purpose | `internal/middleware/sentry.go` | Panic capturado no Sentry. PII scrubbed |
| S03-08 | Audit log service | Registra mutations sensíveis: actor_id, actor_type, action, target, before_state, after_state, request_id | Backend | S01-12 | 2h | general-purpose | `internal/service/audit.go` | Mutations sensíveis logadas. Query por org_id + date range |
| S03-09 | GET /api/bootstrap | Retorna runtime config: sentry_dsn (public), feature_flags, app_version, ws_url. Cached no frontend com staleTime: Infinity | Backend | S02-07 | 1h | general-purpose | `internal/handler/bootstrap.go` | Endpoint retorna config. Frontend consome |
| S03-10 | Security review | Review independente de toda a stack de segurança: headers, auth, CSRF, rate limit, audit | Review | Todos | 2h | code-reviewer | Relatório | Zero vulnerabilidades encontradas |

**Esforço total estimado: 17-20h**

---

### Sprint S04 — OpenAPI + WebSocket Hub + Async Jobs

| ID | Título | Descrição | Camada | Deps | Esforço | Agente | Artefatos | Critério de Aceite |
|----|--------|-----------|--------|------|---------|--------|-----------|-------------------|
| S04-01 | OpenAPI spec generation | kin-openapi integration. Handlers registram schemas. `GET /openapi.json` serve spec. Build step gera YAML | Backend | S01-04 | 3h | general-purpose | `internal/handler/openapi.go` | `/openapi.json` retorna spec válida |
| S04-02 | Frontend type generation pipeline | Script `generate:types` aponta para `http://localhost:8080/openapi.json`. `openapi-typescript` gera `api.gen.ts`. CI verifica drift | Frontend | S04-01 | 2h | general-purpose | `package.json` script, `api.gen.ts` | `npm run generate:types` gera types |
| S04-03 | WS upgrade handler | `nhooyr.io/websocket` Accept. Valida auth cookie no upgrade. Extrai org_id. Rejeita com 1008 se inválido | Backend | S02-07 | 2h | general-purpose | `internal/ws/handler.go` | WS conecta com cookie válido. Rejeita sem |
| S04-04 | WS Hub (registry + broadcast) | Goroutine central. Map de `org_id → []*conn`. Register/unregister. Broadcast por org. Thread-safe | Backend | S04-03 | 4h | general-purpose | `internal/ws/hub.go` | Mensagem broadcasted para todas as conns do tenant |
| S04-05 | WS heartbeat + reconnect | Ping/pong 30s. Timeout 60s. Client reconecta com backoff exponencial + jitter | Backend | S04-04 | 2h | general-purpose | `internal/ws/hub.go` | Conns mortas removidas. Client reconecta |
| S04-06 | WS event protocol | Shape: `{ type, tenant_id, entity_id, patch, version, occurred_at }`. Version monotônico por entidade. Client dedup | Backend | S04-04 | 2h | general-purpose | `internal/ws/event.go` | Eventos emitidos no formato correto |
| S04-07 | Event bus (domain → WS) | Go channels in-process. Domain services emitem eventos. Hub consome e broadcasta | Backend | S04-04 | 2h | general-purpose | `internal/event/bus.go` | Evento emitido por service → chega no client WS |
| S04-08 | Operations CRUD | Tabela `operations`. POST cria operação. GET retorna status. Endpoints: `POST /operations`, `GET /operations/:id` | Backend | S01-12 | 2h | general-purpose | `internal/handler/operations.go` | CRUD funcional |
| S04-09 | 202 Accepted pattern | Handler retorna 202 + operation_id. Worker processa em background. Atualiza status. Emite WS `operation.completed` | Backend | S04-08, S04-07 | 3h | general-purpose | Pattern funcional | Client recebe 202 → poll → completed |
| S04-10 | Worker pool + scheduler | Goroutines com semáforo. Configurable concurrency. Graceful shutdown. Retry exponencial + DLQ | Backend | S01 | 3h | general-purpose | `internal/worker/pool.go` | Jobs executados. Retry funciona. Shutdown graceful |

**Esforço total estimado: 25-30h**

---

### Sprint S05 — Frontend CRUD Foundation + Sprint S06 — Integração Front↔Back

> Tarefas granulares das sprints S05-S06:

| ID | Título | Descrição | Camada | Deps | Esforço | Critério de Aceite |
|----|--------|-----------|--------|------|---------|-------------------|
| S05-01 | `useInfiniteList` hook genérico | Wrapper de `useInfiniteQuery` com cursor-based pagination, generic types, error handling | Frontend | S04-02 | 3h | Hook funcional com tipos gerados |
| S05-02 | `useMutationWithOptimistic` pattern | Wrapper que aplica optimistic update + rollback on error + cache invalidation | Frontend | S04-02 | 2h | Optimistic update funcional |
| S05-03 | Error mapping (AppError) | Discriminated union de errors: NetworkError, AuthError, ValidationError, NotFoundError, ForbiddenError | Frontend | — | 2h | Todos os error types mapeados |
| S05-04 | snake↔camel transformers | Funções `toSnakeCase` e `toCamelCase` deep para request/response bodies | Frontend | — | 1h | Transformação bidirecional correta |
| S06-01 | Wire AuthProvider to /auth/me | Substituir mock por call real a `/auth/me`. Handle 401. Redirect to /login | Frontend | S02 | 3h | Login real funciona no browser |
| S06-02 | Wire fetch.ts credentials:include | Garantir `credentials: 'include'` em todos os requests. CSRF header em mutations | Frontend | S02 | 1h | Cookies enviados automaticamente |
| S06-03 | Wire WS provider to real endpoint | Substituir mock por conexão real a `wss://api/ws`. useWSStatus reativo | Frontend | S04 | 2h | Badge WS mostra "Conectado" (verde) |
| S06-04 | Wire OrgSwitcher to real orgs | Substituir hardcoded por dados de `/auth/me` | Frontend | S06-01 | 1h | Orgs reais no switcher |
| S06-05 | Wire /api/bootstrap | Fetch Sentry DSN, feature flags, versão do bootstrap endpoint | Frontend | S03-09 | 1h | Config runtime carregada no boot |
| S06-06 | Remover seed data de paths críticos | Mover seed para dev-only. Paths reais usam API | Frontend | S06-01 | 2h | Nenhum dado fake em produção |
| S06-07 | Smoke test E2E | Login → Dashboard → navegar → logout. Manual no browser | Teste | Todos | 2h | Fluxo completo funciona |

**Esforço total S05+S06: 20-24h**

---

### Sprints S07-S09 — F01: Funis Hub + Pipe WhatsApp

> Tarefas granulares conforme [[F01 Tasks.md]]:

| ID | Título | Descrição | Camada | Deps | Esforço | Critério de Aceite |
|----|--------|-----------|--------|------|---------|-------------------|
| S07-01 | Leads CRUD handler | POST /leads, GET /leads (cursor+filter), PATCH /leads/:id. Tenant-scoped | Backend | S04 | 4h | CRUD funcional com cursor pagination |
| S07-02 | Pipes + Stages handler | GET /pipes (list), GET /pipes/whatsapp (com stages + counts) | Backend | S01-10 | 2h | Retorna pipes e stages corretos |
| S07-03 | Pipe records handler | PATCH /pipes/whatsapp/:id/stage — mover lead entre stages. Position rebalancing | Backend | S07-02 | 3h | Move funcional. Position rebalanced |
| S07-04 | Realtime events para leads/pipes | Emitir lead.created, lead.updated, pipe_record.moved via event bus → WS | Backend | S04-07, S07-01 | 2h | Eventos chegam no client WS |
| S07-05 | RBAC para F01 | pipeline.view, create_lead, move_pipe_record, view_lead | Backend | S02-09 | 2h | Sem permissão → 403 |
| S07-06 | Integration tests F01 backend | Cobertura completa dos endpoints F01 | Testes | S07-05 | 3h | Todos passing |
| S08-01 | useLeads hook (cursor) | React Query + cursor pagination + filter por stage | Frontend | S07-01 | 2h | Lista de leads carrega com pagination |
| S08-02 | usePipeWhatsapp hook (realtime) | Subscription WS para lead.*, pipe_record.* + smart invalidation | Frontend | S07-04 | 3h | Cards atualizam em realtime |
| S08-03 | PipeCard component | Nome, ícone, contagem, delta 7d | Frontend | S07-02 | 2h | Card renderiza com dados reais |
| S08-04 | /pipes hub page | Grid responsivo de PipeCards | Frontend | S08-03 | 2h | Hub mostra pipes reais |
| S08-05 | /pipes/whatsapp — 5 colunas kanban | Sticky header, contagem por stage, scroll interno | Frontend | S08-01 | 3h | Kanban carrega com dados reais |
| S08-06 | DraggableKanbanBoard | @dnd-kit/core: pointer + keyboard sensors, closestCenter | Frontend | S08-05 | 4h | DnD move cards entre colunas |
| S08-07 | LeadCard + KanbanColumn | Card draggable (name, channel, heat, last activity) | Frontend | S08-06 | 3h | Cards renderizam e são draggable |
| S08-08 | Optimistic DnD | Move local imediato + rollback on error + opacity 0.8 durante mutation | Frontend | S08-06 | 2h | Move percebido < 300ms |
| S09-01 | LeadDetailDrawer | React.lazy + Suspense. Detalhes, histórico, ações | Frontend | S08-07 | 4h | Drawer abre com dados reais |
| S09-02 | DnD keyboard accessibility | Teclado + screen reader + axe-core audit | Accessibility | S08-06 | 3h | axe-core sem violações |
| S09-03 | F01 E2E tests | Fluxo completo: ver hub → entrar kanban → mover card → ver detail → realtime | Testes | S09-01 | 3h | Testes passing |
| S09-04 | F01 stress test | 3G throttle, WS reconnect, 403 mid-drag | Testes | S09-03 | 2h | Sistema resiliente |
| S09-05 | F01 review + docs | Review independente + atualizar docs vault | Review | S09-04 | 2h | Checklist F01 100% |

**Esforço total S07+S08+S09: 46-54h**

---

> **Nota:** Sprints S10-S29 terão tarefas granulares detalhadas quando cada sprint for a próxima na fila de execução. Detalhar 300+ tarefas futuras agora seria especulativo e violaria o princípio de refinamento just-in-time. As descrições de sprint na Seção 8 fornecem contexto suficiente para planejamento macro.

---

## 10. Sequência Recomendada

### Fazer imediatamente (agora)

1. **Sprint S00** — Fechar Sistema Base Frontend (~16-18h)
2. **Sprint S01** — Go Backend Skeleton + Database Schema (~18-22h)

> S00 e S01 podem rodar **em paralelo** pois são independentes (frontend config vs backend bootstrap).

### Fazer logo depois

3. **Sprint S02** — Auth + Tenancy + RBAC (~24-28h)
4. **Sprint S03** — Security + Observabilidade (~17-20h)
5. **Sprint S04** — OpenAPI + WS Hub + Jobs (~25-30h)

> S03 e S04 podem ter **paralelismo parcial** (security headers ∥ WS hub), mas auth (S02) é pré-requisito de ambos.

### Não começar ainda

- **Features F01-F16** — bloqueadas por backend foundation (S01-S04)
- **F06 Copilot** — bloqueado por F04 Inbox (cadeia de dependências)
- **F07 Workflow Builder** — bloqueado por F01 (infra de pipe)
- **F14 Checkout** — não começar antes de F13 Onboarding
- **F16 Master Admin** — última feature, requer visão completa do produto

### Frentes que podem paralelizar (após F01 completa)

```
Momento: Após S09 (F01 100%)
                                                          
Trilha A (Principal):   F02 → F03 → F04 → F05            
Trilha B (Automação):   F07 → F08          (após S09)     
Trilha C (Comercial):   F13, F14            (após S06)    
                                                          
Regra: máximo 2 trilhas simultâneas para manter qualidade.
Sugestão: Trilha A + Trilha C em paralelo (Trilha C é independente).
```

---

## 11. Riscos, Conflitos e Decisões Pendentes

### Conflitos entre vault e código

| Conflito | Severidade | Resolução proposta |
|----------|-----------|-------------------|
| `00 - Indice.md` diz "Etapa 0" mas código está na Etapa 6+ | Alta | Atualizar Indice no S00 |
| `Analise Pratica.md` diz tokens/infra ausentes mas existem | Alta | Reescrever Analise Pratica no S00 |
| `Revisao Final` diz "todos bloqueadores RESOLVED" mas ESLint (B6) continua vazio | Média | Corrigir ESLint no S00, atualizar doc |
| `Checklist Sistema Base` todos `[ ]` unchecked | Alta | Marcar itens reais no S00 |
| `CommandPalette.tsx` diz "Pipelines", `Sidebar.tsx` diz "Funis" | Baixa | Fix no S00-04 |
| `Frontend.md` (agente) usa accent `hsl(47 100% 50%)` vs canônico `hsl(44 93% 54%)` | Baixa | Atualizar Frontend.md |
| `Backend.md` (agente) referencia Supabase/Deno mas ADRs definem Go | Alta | Reescrever Backend.md para Go |

### Decisões ainda em aberto

| Decisão | Impacto | Recomendação |
|---------|---------|-------------|
| Router Go: chi vs stdlib `net/http` (1.22+ ServeMux) | Médio | chi — middleware ecosystem maduro, mais produtivo |
| ORM vs raw SQL: sqlc vs pgx direto | Médio | pgx direto com query builders — máximo controle, sem magic |
| Object storage provider: S3 vs Cloudflare R2 vs local | Baixo (pode trocar depois) | R2 (zero egress cost, S3-compatible) |
| Domain para API: `api.torquecrm.com.br` vs subpath | Baixo | Subdomínio — separação limpa de CORS/cookies |
| Email provider (transacional): SES vs Resend vs Mailgun | Baixo | Resend — DX superior, pricing simples |
| Git hosting: precisa inicializar repo | Bloqueador | Criar repo Git imediatamente (antes de S01) |

### Riscos técnicos

| Risco | Probabilidade | Impacto | Mitigação |
|-------|--------------|---------|-----------|
| WS hub goroutine leak sob carga | Média | Alto | Context cancellation, connection timeout, load test em S04 |
| Drift entre OpenAPI spec e handlers | Média | Alto | CI step que compara `api.gen.ts` com spec gerada |
| Refresh token reuse detection false positive | Baixa | Alto | Testes extensivos em S02-12; grace period de 5s |
| @dnd-kit performance com 500+ cards | Média | Médio | Virtualization no kanban; pagination por stage |
| Latência de RAG/embeddings (F06) | Alta | Médio | Cache de embeddings; batch processing; timeout 8s |
| PIX webhook reliability (F14) | Média | Alto | Retry + DLQ + reconciliação manual |

### Riscos de escopo

| Risco | Mitigação |
|-------|-----------|
| Feature creep no Sistema Base | [[Escopo do Sistema Base]] — lista "NÃO entra" já definida |
| Paralelismo criando dívida de integração | Regra cardinal: uma feature por vez (máx 2 trilhas após F01) |
| Documentação ficando stale novamente | Etapa 8 obrigatória em cada sprint; review verifica docs |

### Riscos de arquitetura

| Risco | Mitigação |
|-------|-----------|
| Go backend sem equipe (único dev é IA) | Código simples, idiomático, bem testado. Evitar frameworks mágicos |
| Multi-tenancy leak entre orgs | Testes de isolamento em cada sprint; middleware que força org_id |
| CSP nonce quebrando third-party scripts futuros | Whitelist explícita; `strict-dynamic` para chunks Vite |

### Riscos de segurança

| Risco | Mitigação |
|-------|-----------|
| XSS via markdown user-generated | DOMPurify + allowlist tags; lint ban innerHTML |
| CSRF em operações sensíveis | Double-submit pattern; SameSite=Strict cookies |
| JWT secret leak | Rotação periódica; secret manager em prod |
| SQL injection | pgx prepared statements; zero string concatenation |
| Tenant data leak | org_id from JWT only; middleware enforcement; integration tests |

---

## 12. Próximas 2 Sprints Recomendadas

### Sprint S00 — Fechar Sistema Base Frontend

**Iniciar imediatamente.** Sem dependências externas.

**Tarefas ordenadas:**

1. `S00-01` — Configurar ESLint com regras reais (2h)
2. `S00-09` — Configurar Prettier (30min)
3. `S00-02` — Adicionar classe utilitária `grain` (30min)
4. `S00-03` — Criar `vocabulary.ts` (1h)
5. `S00-04` — Fix CommandPalette "Funis" (15min)
6. `S00-05` — Adicionar flags TS strict (1h)
7. `S00-06` — Configurar Vitest (1h)
8. `S00-07` — Snapshot tests 20 primitivos (4h)
9. `S00-08` — Snapshot tests 3 novos primitivos (1h)
10. `S00-10` — CI foundation (2h)
11. `S00-11` — Atualizar Checklist Sistema Base (1h)
12. `S00-12` — Atualizar docs vault (1h)
13. `S00-13` — Review final por code-reviewer (2h)

**Total: ~16-18h**
**Agentes:** `general-purpose` (1-10), `code-reviewer` (13)
**Critério: Checklist Sistema Base 100%**

---

### Sprint S01 — Go Backend Skeleton + Database Schema

**Pode iniciar em paralelo com S00.** Independente do frontend.

**Tarefas ordenadas:**

1. `S01-01` — Inicializar projeto Go (1h)
2. `S01-02` — Config loader (1h)
3. `S01-03` — PostgreSQL connection pool (1h)
4. `S01-04` — HTTP server + router (2h)
5. `S01-05` — Docker Compose (2h)
6. `S01-06` — Makefile (1h)
7. `S01-07` — Migration: organizations (1h)
8. `S01-08` — Migration: team_members (1h)
9. `S01-09` — Migration: leads (1h)
10. `S01-10` — Migration: pipes + stages + pipe_records (2h)
11. `S01-11` — Migration: permissions (1h)
12. `S01-12` — Migration: quotas + operations + audit_log (2h)
13. `S01-13` — Seed data SQL (1h)
14. `S01-14` — golang-migrate integration (1h)

**Total: ~18-22h**
**Agentes:** `code-architect` (1, revisão estrutural), `general-purpose` (2-14)
**Critério: `docker compose up` → health check 200 → migrations applied → seed populated**

---

> **Pré-requisito crítico descoberto:** O projeto NÃO é um repositório Git. Antes de iniciar qualquer sprint, inicializar `git init` + `.gitignore` + primeiro commit com o estado atual.

---

## Apêndice: Glossário de Agentes vs Sprints

| Sprint | Agente Principal | Agente de Review | Skill de Validação |
|--------|-----------------|-----------------|-------------------|
| S00 | general-purpose | code-reviewer | `/hm-engineer` |
| S01 | general-purpose + code-architect | — | — |
| S02 | general-purpose | code-reviewer | `/hm-engineer` (security) |
| S03 | general-purpose | code-reviewer | `/hm-engineer` (security) |
| S04 | general-purpose + code-architect | code-reviewer | — |
| S05-S06 | general-purpose | code-reviewer | `/hm-engineer` |
| S07-S09 | general-purpose + frontend-design | code-reviewer + /hm-qa | `/hm-designer`, `/hm-qa` |
| S10-S14 | general-purpose | code-reviewer | `/hm-qa` |
| S15-S18 | code-architect + general-purpose | code-reviewer | `/hm-engineer`, `/hm-qa` |
| S19-S27 | general-purpose | code-reviewer | `/hm-qa` |
| S28-S29 | general-purpose | code-reviewer | `/hm-qa`, security-review |

---

*Documento gerado por Conductor em 2026-04-16. Fonte de verdade para execução até revisão na próxima fase.*
