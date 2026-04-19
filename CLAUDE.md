# CLAUDE.md — Torque CRM v2

## Projeto

Torque e um SaaS B2B multi-tenant CRM para times comerciais brasileiros que vendem via WhatsApp. IA embarcada, operacao multi-canal, workflow builder, analytics.

**Dono:** Fundador/CTO (nao escreve codigo, dirige arquitetura, decide tudo). Padrao exigido: world-class em todas as camadas.

---

## Protocolo de git — sprints (INVARIANTE)

**Regra cardinal travada:** Toda sprint (S00, S01, S02, ...) segue a topologia LINEAR CUMULATIVA. Nada de fan-out de S00.

```
main
  ↑ merge via PR (release/tag)
develop  ← trunk cumulativo. Cada sprint mergeada vira parte dele.
  ↑ merge via PR ao fim de cada sprint
sprint/S0X  ← nasce de `develop` atualizada (que ja tem S0X-1)
```

**Invariantes absolutas:**

1. `sprint/S0X` SEMPRE nasce de `develop` com `git pull --ff-only` primeiro. Nunca de `main`, nunca de outra `sprint/*` em andamento (exceto dependencia explicita declarada no Plano Mestre).
2. Nenhum trabalho de sprint vai direto em `develop` — sempre via branch propria + PR.
3. Merge da sprint em `develop` usa `merge commit` (nao squash) para preservar a granularidade dos commits por dominio no historico.
4. A branch `sprint/S0X` permanece no remoto como ancora de auditoria apos o merge — nao deletar.
5. Historico linear e cumulativo: S02 assume S01 entregue, S03 assume S02 entregue, etc. Paralelismo entre sprints esta PROIBIDO (regra cardinal "uma feature por vez").

**Passos obrigatorios:**

```bash
# ANTES do primeiro edit
git checkout develop
git pull --ff-only origin develop
git checkout -b sprint/S0X

# DURANTE — commits logicos por dominio, nesta ordem:
# 1. DBA       → feat(db):       migrations, seeds, schema
# 2. Backend   → feat(backend):  services, repos, middlewares, handlers, config
# 3. QA        → test(backend):  unit + integration
# 4. Frontend  → feat(frontend): componentes, hooks, i18n
# 5. Docs      → docs(vault):    STATE.md (D0xx), Indice, Plano Mestre, ADRs

# AO FIM
git push -u origin sprint/S0X
gh pr create --base develop --head sprint/S0X --title "Sprint S0X — <objetivo>" --body "..."
```

**Nao fechar sprint sem:**
- [ ] STATE.md atualizado com `D0xx` (decisao e entregaveis daquele sprint)
- [ ] `Torque-dir-new/00 - Indice.md` status line refletindo S0X ✅
- [ ] `Torque-dir-new/09 - Backlog/Plano Mestre de Finalizacao do SaaS CRM.md` §8 marcando a sprint ENTREGUE
- [ ] Branch `sprint/S0X` pushada em `origin`
- [ ] PR aberta contra `develop`

**Referencia canonica:** `Torque-dir-new/09 - Backlog/Plano Mestre de Finalizacao do SaaS CRM.md` §8 → "Protocolo de Execução (obrigatório para TODA sprint)".

**Justificativa da topologia linear (vs fan-out):** sprints de fundacao (S01–S06) tem dependencias verticais rigidas — S02 precisa do schema de S01, S04 precisa do auth de S02, S06 precisa de tudo. Fan-out forcaria cada sprint a reimplementar o que falta e explodiria em conflitos no merge. Decisao registrada em STATE.md como D012.

---

## Protocolo de agentes (OBRIGATORIO)

**Toda task, mudanca ou request segue este protocolo automaticamente:**

1. **Toda task** → Invoque a skill `agent-conductor` via Skill tool para triagem e roteamento
2. **Conductor classifica** → Identifica dominio(s), seleciona agente(s), define escopo
3. **Conductor ativa agente(s)** → Invoca skill do especialista via Skill tool. Cada agente carrega contexto do vault + .specs/
4. **Agente executa** → Com sua persona, regras e skills integradas (TDD, debugging, review)
5. **Documentacao** → Vault Obsidian atualizado com mudancas (features, backlog, decisoes)

**Nenhuma task chega a um especialista sem passar pelo Conductor.** Isso garante triagem correta, ordem de dependencias, e documentacao.

### Como invocar

```
# O Conductor e invocado PRIMEIRO para triagem:
Skill tool → agent-conductor

# O Conductor entao invoca o(s) especialista(s):
Skill tool → agent-backend    (para Go API, middleware, services)
Skill tool → agent-frontend   (para React, UI, design)
Skill tool → agent-dba        (para SQL, migrations, schemas)
Skill tool → agent-qa         (para testes, verificacao)
Skill tool → agent-infra      (para deploy, CI/CD)
Skill tool → agent-automation (para jobs, workflows, cron)
Skill tool → agent-ai         (para Copilot, RAG, embeddings)
Skill tool → agent-architect  (para decisoes de sistema)
```

### Contexto obrigatorio antes de agir

Antes de qualquer execucao, o agente ativado DEVE ler:
- `.specs/project/STATE.md` — decisoes e estado atual
- `Torque-dir-new/00 - Indice.md` — visao geral do vault
- Docs especificos do dominio no vault (listados em cada skill)

---

## Stack

### Backend (Go — dominante)
- **Linguagem:** Go 1.22+
- **Router:** chi
- **Database:** PostgreSQL 15+ via pgx (connection pooling nativo)
- **Migrations:** golang-migrate (UP + DOWN obrigatorios)
- **Auth:** JWT em httpOnly cookies (SameSite=Strict), refresh rotation
- **WebSocket:** nhooyr.io/websocket (hub por tenant)
- **OpenAPI:** kin-openapi (gera spec, frontend consome via openapi-typescript)
- **Logging:** zerolog (structured JSON)
- **Tracing:** OpenTelemetry
- **Errors:** Sentry Go SDK
- **Jobs:** Worker pool in-process (goroutines + semaforo)

### Frontend (React/TypeScript)
- **Framework:** React 18.3.1 + TypeScript strict
- **Build:** Vite 6
- **State:** TanStack Query v5 (server), React Context (auth/theme)
- **Styling:** Tailwind 3.4 + CSS variables HSL
- **Components:** shadcn/ui (Radix) + Lucide icons
- **Fonts:** @fontsource (Fraunces, Instrument Sans, JetBrains Mono)
- **i18n:** react-intl (PT-BR only)
- **DnD:** @dnd-kit/core
- **Charts:** Visx (signature), Recharts (where speed > visual)
- **Tests:** Vitest + Testing Library
- **Design:** Dark-first, editorial, accent gold hsl(44 93% 54%)

### Infraestrutura
- **Deploy:** Docker Compose (dev), Hostinger VPS + EasyPanel (prod)
- **CI/CD:** GitHub Actions (lint → test → build → deploy)
- **Storage:** Object storage S3-compatible (pre-signed URLs)
- **Observabilidade:** Sentry + zerolog + OpenTelemetry

---

## Estrutura do projeto

```
Torque-v2/
├── torque-web/                  # Frontend React/TypeScript
│   ├── src/
│   │   ├── api/                 # HTTP clients (snake↔camel transform)
│   │   ├── contracts/           # api.gen.ts (gerado) + manual.ts
│   │   ├── features/            # UI por dominio (pipes, inbox, copilot...)
│   │   ├── hooks/               # React Query hooks
│   │   ├── lib/                 # fetch.ts, ws.ts, domain utilities
│   │   ├── providers/           # Auth, Query, WS, Theme, Intl
│   │   ├── shell/               # AppShell, Sidebar, TopBar
│   │   ├── ui/                  # Primitivos UI (button, input, badge...)
│   │   ├── styles/              # globals.css, tokens
│   │   └── i18n/                # Messages PT-BR
│   └── package.json
│
├── torque-api/                  # Backend Go (a ser criado)
│   ├── cmd/api/                 # Entrypoint
│   ├── internal/
│   │   ├── config/              # Env vars
│   │   ├── handler/             # HTTP handlers
│   │   ├── middleware/          # Auth, tenant, RBAC, CSRF, CSP, CORS, rate limit
│   │   ├── service/             # Business logic
│   │   ├── repository/          # Data access (pgx)
│   │   ├── domain/              # Entities, value objects
│   │   ├── ws/                  # WebSocket hub
│   │   ├── worker/              # Async job pool
│   │   └── event/               # Event bus (domain → WS)
│   ├── migrations/              # SQL (golang-migrate)
│   ├── go.mod
│   └── Makefile
│
├── Torque-dir-new/              # Obsidian vault (fonte de verdade)
├── .specs/                      # Specs tecnicas do projeto
├── .claude/                     # Skills e settings do Claude Code
└── CLAUDE.md                    # Este arquivo
```

---

## Vault Obsidian (Fonte de verdade)

O vault em `Torque-dir-new/` e a fonte principal de verdade para decisoes, specs e estado. Estrutura:

```
00 - Indice.md          Navegacao central e status
01 - Produto/           Visao, personas, glossario, principios, regras de negocio
02 - Arquitetura/       Arq tecnica + conceitual + permissoes
03 - Modelo de Dominio/ Entidades: Lead, Pipeline, Workflow (13 files)
04 - Design/            Design system, tipografia, motion, componentes
05 - Sistema Base/      Fundacao frontend: spec, checklist, plano
06 - Funcionalidades/   Specs de features por dominio (7 subfolders)
07 - Features/          Mapa F01-F16 + specs de execucao
08 - Decisoes/          ADRs (ADR-001 a ADR-006)
09 - Backlog/           Plano mestre, backlog priorizado, riscos
10 - Referencias/       Integracoes, processos assincronos, catalogos
11 - Operacional/       Fluxo de trabalho, agentes, observabilidade
12 - Migracao do Legado Vault legado, top 15 docs, exclusoes
Agentes/                9 perfis: Conductor, Architect, Backend, Frontend, DBA, QA, Infra, Automation, AI
```

**Antes de trabalhar em qualquer feature:** ler a spec correspondente no vault.
**Apos concluir:** atualizar docs relevantes.

---

## Time de agentes

### Protocolo obrigatorio

Toda tarefa passa pelo Conductor antes de chegar a um agente especialista.

```
Tarefa → Conductor (triagem) → Agente(s) especialista(s) → Review → Documentacao
```

### Roteamento por sinal

| Sinal | Agente(s) |
|-------|----------|
| cmd/api, internal/, handler, middleware, service, repository | Backend |
| src/components, src/features, src/pages, UI, visual, CSS, design | Frontend |
| migrations/, table, index, SQL, schema | DBA |
| test, coverage, verification, QA, flaky | QA |
| deploy, Docker, CI/CD, env vars, monitoring | Infra |
| cron, automation, workflow trigger, jobs | Automation |
| copilot, agent IA, RAG, embeddings, prompt | AI |
| architecture, cross-cutting, trade-off, boundaries | Architect |

### Sequencias comuns

| Tarefa | Sequencia |
|--------|-----------|
| Feature nova completa | Architect → DBA → Backend → Frontend → QA |
| Automacao nova | Architect (se decisao) → Automation → Backend → QA |
| Mudanca de IA | AI → Backend → Frontend (se UI) → QA |
| Bug de UI | Frontend → QA |
| Bug de API | Backend → QA |
| Query lenta | DBA → Backend → QA |
| Deploy/config | Infra |

### Perfis completos

Perfis detalhados de cada agente em `Torque-dir-new/Agentes/`. Skills invocaveis via `/agent-conductor`, `/agent-backend`, etc.

---

## Arquitetura (decisoes travadas)

| ADR | Decisao |
|-----|---------|
| ADR-001 | OpenAPI gerado pelo Go, consumido via openapi-typescript. Wire snake_case, frontend camelCase |
| ADR-002 | WebSocket via nhooyr.io/websocket. Hub Go por tenant. Patches only (nunca rows completas) |
| ADR-003 | Auth via httpOnly cookies (SameSite=Strict). CSRF double-submit. Refresh rotation |
| ADR-004 | Paginacao cursor-based em todas as listas. Offset proibido |
| ADR-005 | Fontes self-hosted via @fontsource. CSP font-src 'self'. Zero CDN externo |
| ADR-006 | Jobs assincronos via 202 Accepted + poll GET /operations/:id + WS push |

Detalhes em `Torque-dir-new/08 - Decisoes/`.

---

## Multi-tenancy (invariante absoluta)

- `organization_id` extraido EXCLUSIVAMENTE do JWT pelo backend. Nunca enviado pelo frontend.
- Middleware strip preventivo remove `organization_id` de bodies de mutation.
- Toda tabela de dominio tem `organization_id NOT NULL` + compound index.
- Repository sempre aplica `WHERE organization_id = ctx.OrgID`.
- Master Admin: `/master/impersonate/:org_id` emite novo JWT. Audit log marca `actor_type='master'`.

---

## Seguranca (invariantes)

- CSP nonce-based (`script-src 'nonce-{RANDOM}' 'strict-dynamic'`)
- HSTS 2 anos com preload
- httpOnly + SameSite=Strict para tokens (XSS nao exfiltra sessao)
- CSRF double-submit (X-CSRF-Token em mutations)
- PII scrubbing no Sentry (beforeSend remove token, password, email, cookie)
- Zero secrets no bundle (config via GET /api/bootstrap)
- Audit trail para mutations sensiveis
- Rate limiting com 429 + Retry-After
- `dangerouslySetInnerHTML` banido por ESLint

---

## Convencoes de codigo

### Go (Backend)

- **Packages:** lowercase, singular (handler, service, repository, domain)
- **Files:** snake_case.go (lead_handler.go, auth_middleware.go)
- **Functions/Methods:** PascalCase exportado, camelCase interno
- **Errors:** Wrap com contexto (`fmt.Errorf("create lead: %w", err)`)
- **Context:** Sempre primeiro parametro. Propaga org_id, user_id, request_id
- **Transactions:** Para operacoes atomicas. Sempre com defer rollback
- **Tests:** _test.go no mesmo package. Table-driven tests
- **Logs:** zerolog com campos estruturados (org_id, user_id, request_id sempre)
- **Linting:** golangci-lint

### TypeScript (Frontend)

- **Components:** PascalCase.tsx (LeadCard.tsx)
- **Hooks:** camelCase com `use` prefix (useLeads.ts)
- **Imports:** @/ alias para src/. Ordem: React → third-party → ui → features → hooks → lib → types
- **State:** TanStack Query para server state. useState para UI state. Context para auth/theme
- **Error handling:** try/catch em mutations, toast.error para user, Sentry para tracking
- **Styling:** Tailwind + CSS variables HSL. Dark-first. Zero hex/RGB
- **Types:** api.gen.ts (gerado, nunca editar). manual.ts para tipos adicionais
- **Security:** Nunca enviar organization_id em mutations. Credentials: 'include'

### Serialization boundary

- **Wire (JSON):** snake_case (Go idiom + Postgres alignment)
- **Frontend (TS):** camelCase (JS idiom)
- **Transform:** src/api/ faz a conversao bidirecional
- **Dates:** ISO 8601 UTC (Z suffix)
- **Money:** Integer cents + currency code
- **IDs:** UUID (gen_random_uuid())

---

## RBAC (4 camadas)

```
if master → allow (audit trail obrigatorio)
if admin && !feature.master_only → allow
if feature.is_admin_only → deny
else → check member_feature_permissions OR feature_permissions.default
```

Roles no sistema: APENAS `admin`, `membro`, `master`. SDR/Closer sao conceitos de negocio, derivados de atributos.

---

## Glossario critico

| Termo | Em codigo | Em UI | NUNCA usar |
|-------|----------|-------|-----------|
| Lead | `lead` | Lead | prospect, contato, cliente, oportunidade |
| Pipe/Funil | `pipe` | Funil | pipeline (sobrecarregado), funnel |
| Stage | `stage` | Stage | coluna, etapa, fase, step |
| Calor | `calor` | Calor | temperatura, score, heat |
| Copilot | `copilot` | Copilot | bot, chatbot, assistente |
| Org | `org`, `organization` | Organizacao | tenant (em UI), workspace, conta |

---

## Fluxo de trabalho obrigatorio

1. **Entender** — ler vault, codigo, decisoes passadas
2. **Documentar** — registrar entendimento em nota do vault
3. **Propor** — escrever proposta tecnica
4. **Validar** — submeter a revisao antes de codigo
5. **Executar** — implementar o que foi validado
6. **Revisar** — /hm-engineer, /hm-designer, /hm-qa conforme camada
7. **Documentar resultado** — atualizar vault
8. **Avancar** — proximo item

---

## Estado atual do projeto

Frontend React/TS ~85% completo (prototipo visual). Zero codigo Go. Zero banco. Zero testes.

**Plano mestre:** `Torque-dir-new/09 - Backlog/Plano Mestre de Finalizacao do SaaS CRM.md`

**Proximas sprints:**
- S00: Fechar Sistema Base frontend (ESLint, testes, docs)
- S01: Go backend skeleton + database schema (em paralelo com S00)

---

## Areas frageis (cuidado redobrado)

- **Copilot/IA:** Parte mais fragil do sistema. 20+ steps wizard, RAG, prompt engineering. Mudancas cirurgicas
- **Permissoes:** RBAC 4 camadas com cascade. Qualquer mudanca afeta todo o sistema
- **Multi-tenancy:** Leak de dados entre tenants e falha catastrofica. Testes de isolamento obrigatorios
- **Checkout/Billing:** Envolve dinheiro real (PIX via Asaas). Dual review obrigatorio
- **WebSocket Hub:** Goroutine leak sob carga. Load test obrigatorio

---

## Skills disponiveis

- `/agent-conductor` — Triagem, roteamento, coordenacao
- `/agent-architect` — Decisoes de sistema, trade-offs
- `/agent-backend` — Go API, middleware, services
- `/agent-frontend` — React, UI/UX, design
- `/agent-dba` — PostgreSQL, migrations, queries
- `/agent-qa` — Testes, verificacao, cobertura
- `/agent-infra` — Deploy, CI/CD, monitoring
- `/agent-automation` — Jobs, cron, workflows
- `/agent-ai` — Copilot, RAG, embeddings
- `/second-brain` — Atualizacao do vault Obsidian

---

## Comandos

```bash
# Frontend
cd torque-web
npm run dev          # Dev server (porta 5173)
npm run build        # Build producao
npm run test         # Vitest
npm run lint         # ESLint
npm run generate:types  # OpenAPI → api.gen.ts

# Backend (futuro)
cd torque-api
make run             # Servidor Go (porta 8080)
make test            # go test ./...
make lint            # golangci-lint
make migrate-up      # Aplicar migrations
make migrate-down    # Reverter ultima migration
make migrate-create  # Criar nova migration

# Docker
docker compose up    # Postgres + Go API
docker compose down
```
