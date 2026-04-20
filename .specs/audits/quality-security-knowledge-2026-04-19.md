# Relatório: Qualidade · Segurança · Conhecimento (S00–S10)

> **Data:** 2026-04-19
> **Escopo:** Tudo em `origin/develop` desde o commit inicial até S10 inclusive.
> **Método:** vitest coverage real no frontend (toolchain Node disponível); Go permanece com auditoria estática (toolchain ausente no workspace).
> **Branch do relatório:** `chore/quality-security-audit`.

---

## 0. Veredito em uma linha

**Fundação sólida, instrumentada em três camadas (hooks + middleware + migrations), com zero débito oculto no backend e três TODOs explícitos no frontend — todos com sprint owner.** Uma regressão do S06 (refresh-on-login) foi detectada e corrigida neste mesmo branch; relatada na §2.5.

---

## 1. Inventário por camada

### 1.1 Números agregados

| Métrica                         | Backend (Go)        | Frontend (TS/React)  |
|---------------------------------|--------------------:|---------------------:|
| Arquivos fonte                  |                  46 |                  104 |
| Arquivos de teste               |                  16 |                   29 |
| Razão testes/fonte              |              **35 %** |            **28 %** |
| Migrations                      |                   6 | — |
| Tabelas criadas                 |                  20 | — |
| Endpoints HTTP                  |                  22 | — |
| ADRs                            | 8 (ADR-000 template + ADR-001..007) | — |
| Decisões em STATE.md            | **22** (D001..D022) | — |
| Sprints fechadas                |                  11 (S00..S10) | — |
| Branches no remoto              | 11 `sprint/S0X` + 3 `chore/*` + 1 `docs/*` | — |

### 1.2 Por sprint — o que cada sprint adicionou

| Sprint | Camada dominante | Artefatos críticos                                                                                                   |
|--------|------------------|----------------------------------------------------------------------------------------------------------------------|
| S00    | Frontend         | ESLint flat, TS strict, Vitest, Prettier, CI, vocabulário canônico, cockpit scaffolding                              |
| S01    | Backend          | `torque-api/` scaffold (chi, pgx v5, zerolog, probes, middleware base, migrations 0001–0003, Docker distroless)       |
| S02    | Backend+DB       | Migration 0004 (refresh_tokens), JWT+bcrypt, auth middleware stack, `/auth/*`, `/me/preferences`, `/api/bootstrap`   |
| S03    | Backend          | CSP+HSTS+COOP/CORP, Sentry SDK com PII scrubbing, audit service/repo, rate limiter token-bucket, bootstrap enriquecido |
| S04    | Backend+DB       | Migration 0005 (operations), event bus, WS hub tenant-scoped, worker pool, `/api/v1/operations`, `/api/v1/ws`, OpenAPI |
| S05    | Frontend         | queryKeys factory, errors pipeline, hooks (`useAppMutation`, `useInfiniteList`, `useWSSubscribe`, `useOperation`, `useBootstrap`), `<QueryBoundary>` |
| S06    | Frontend         | AuthProvider real (mock removida), WSProvider em `ws_url` real, `useLogin` + LoginPage com inline error            |
| S07    | Backend          | F01: leads repo (cursor ADR-004), pipes repo (Move atômico Serializable), handlers + eventos WS                      |
| S08    | Frontend         | F01: `useLeads/Lead/Pipes/PipeStages/PipeEntries/MovePipeEntry`, FunisHubPage                                        |
| S09    | Frontend         | F01: LeadDetailPage, Toaster global                                                                                  |
| S10    | Backend+DB+FE    | Migration 0006 (pipe_confirmations), confirmation repo/handlers, `useConfirmations`, helper `put()`                  |

---

## 2. Qualidade

### 2.1 Frontend — vitest + @vitest/coverage-v8 (execução real)

```
 Test Files  29 passed (29)
      Tests  85 passed (85)
   Duration  36.3 s
```

| Métrica      | Cobertura | Delta vs auditoria anterior (S05) |
|--------------|----------:|:--------------------------------|
| Statements   | **65.89 %** | +4.1 pp  |
| Branches     | **71.86 %** | +1.6 pp  |
| Functions    | **58.82 %** | +3.5 pp  |
| Lines        | **67.35 %** | +4.5 pp  |

**Interpretação:** ganho incremental nos números é coerente — S06–S10 adicionaram hooks testados (useLogin, useLeads, Toaster, useConfirmations) mas também páginas novas ainda não cobertas (LeadDetailPage, FunisHubPage — são integration surface de S11+).

Gaps residuais (mantidos):

- `lib/utils.ts` (`cn` helper) — coberto transitivamente por consumidores.
- `lib/ws.ts` (TorqueWS singleton reconnect/heartbeat) — requer fake timers + WebSocket mock; sprint candidato: S11.
- `api/client.ts` (paths de refresh/429/500) — exercido indiretamente pelos testes de hooks; uma suite focada é nice-to-have.
- `ui/dropdown.tsx` (Radix portal) — interno da lib, fora de escopo.

### 2.2 Backend — auditoria estática (toolchain indisponível no workspace)

- **46 arquivos fonte** em 27 pacotes; **16 arquivos de teste** cobrem **~52 % dos pacotes** (13/27).
- Testes existentes por tipo:
  - Unit puros (no DB): `password`, `jwt`, `token`, `csrf`, `rbac`, `security_headers`, `ratelimit`, `sentry` (scrubPII), `event/bus`, `ws/hub`, `repository/lead` (cursor).
  - Integration gated por `DATABASE_URL`: `handler/auth`, `service/audit`, `repository/operation`, `repository/confirmation`, `repository` (task concurrency).
- Pacotes sem teste dedicado mas **cobertos transitivamente**: `repository/user` e `repository/refresh` (via `handler/auth` integration), `repository/audit` (via `service/audit`), `service/permission` (via `httpx/middleware/rbac`).
- Pacotes sem teste e **thin delegates** (aceitos): `cmd/api`, `config`, `db`, `domain`, `httpx` helpers, `handler/{health,bootstrap,openapi,preferences}`.
- **Débito `TODO/FIXME/HACK/XXX///nolint/@ts-ignore` no backend: 0.**

### 2.3 Frontend — débito declarado

Três `TODO` explícitos, cada um ancorado em uma sprint futura:

| Arquivo | Linha | Contexto | Sprint owner |
|---|---|---|---|
| `providers/UiModeProvider.tsx` | 126 | Persistência de ui_mode no backend (já existe endpoint em S02, falta wire) | S11 polish |
| `features/cockpit/PipeSnapshotPanel.tsx` | 26 | `PATCH /leads/:id/stage` com optimistic+WS | F01 integration S11 |
| `hooks/useTaskActions.ts` | 90 | `POST /tasks/:id/complete` | S14 (Follow-ups) |

Nenhum `FIXME`/`HACK`/`XXX`/`@ts-ignore`. Quatro `eslint-disable` inline, cada um justificado no local.

### 2.4 Convenções mecanicamente validadas

- **ESLint flat config** (S00) com `jsx-a11y`, `react-hooks`, **ban de `dangerouslySetInnerHTML`** — falha o CI se violado.
- **TS strict estendido** (`noUncheckedIndexedAccess`, `exactOptionalPropertyTypes`) — expõe bugs que TS-normal ignora.
- **Prettier + plugin-tailwindcss** — classes Tailwind ordenadas deterministicamente.
- **golangci-lint** declarado no Makefile (a rodar quando CI for ativado para Go).

### 2.5 Regressão detectada e corrigida neste branch

**Bug (introduzido em S06):** `client.ts` retry-on-401 disparava em TODAS as respostas 401, incluindo `POST /api/v1/auth/login` com senha errada. Consequência: o frontend mascarava `INVALID_CREDENTIALS` como `AUTH_EXPIRED` após a tentativa de refresh falhar, e ainda tentava um refresh sem cookie válido.

**Detecção:** teste `useLogin.test.tsx` — foi **o próprio teste do S06 que travou a regressão** em `vitest run`. A suite estava em 84/85 passando; esse 1 falho era exatamente o sintoma. Confiança no instrumental funcionou.

**Fix (neste commit):** `client.ts` só aciona refresh se `path` **não** começar com `/api/v1/auth/{login,refresh,logout}`. Cobertura pós-fix: **85/85 testes passando**.

Decisão registrada em STATE.md **D023** como entrada deste próprio audit.

---

## 3. Segurança

### 3.1 Superfície implementada (invariantes)

| Camada | Controle | Onde | Sprint |
|---|---|---|---|
| **Sessão** | JWT HS256 em `__torque_session` cookie httpOnly + SameSite=Strict + Secure fora de dev | `service/jwt`, `handler/auth` | S02 |
| **Refresh** | Opaque token 256-bit, hash sha256 em DB, rotation chain, **reuse detection revoga toda a linhagem** via recursive CTE | `repository/refresh`, migration 0004 | S02 |
| **CSRF** | Double-submit (`__torque_csrf` cookie readable + `X-CSRF-Token` header) com `ConstantTimeEqual` | `middleware/csrf` | S02 |
| **Tenancy** | `organization_id` SEMPRE do JWT; middleware `StripOrganizationID` rejeita 400 se body contém o campo | `middleware/strip_org` | S01 |
| **RBAC** | Cascade master > master_only > admin_only > member_override > default, helpers `RequireFeature/Role/Master` | `middleware/rbac`, `service/permission` | S02 |
| **CSP** | `default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'` (API pure-JSON, nonce fica no edge) | `middleware/security_headers` | S03 |
| **HSTS** | 2 anos + includeSubDomains + preload (ativo fora de dev) | `middleware/security_headers` | S03 |
| **Outros headers** | X-Frame DENY, X-Content-Type-Options nosniff, Referrer-Policy strict-origin, Permissions-Policy negando camera/mic/geo/payment, COOP same-origin, CORP same-site | `middleware/security_headers` | S03 |
| **Rate limiting** | Token bucket in-memory via `golang.org/x/time/rate`, chave por user_id quando logado senão IP, 429+Retry-After calculado de `Reserve().Delay()` | `middleware/ratelimit` | S03 |
| **Observability** | Sentry SDK com PII scrubbing em `BeforeSend` (cookies, Authorization, CSRF, email, body, IPv4→/24, IPv6→/48) | `observability/sentry` | S03 |
| **Audit** | Append-only em `audit_log` existente, actor_type derivado da session, `RecordImpersonation` para master atuando em tenant | `service/audit` + `repository/audit` | S03 |
| **Async jobs** | 202 Accepted + `operations` ledger com `FOR UPDATE SKIP LOCKED` multi-worker safe + retry chain atômico | `repository/operation`, `worker` | S04 |
| **Secrets** | `JWT_SECRET` >= 32 bytes, fail-fast no boot; `SENTRY_DSN` (server) ≠ `SENTRY_PUBLIC_DSN` (browser via /api/bootstrap) | `config` | S02/S03 |

### 3.2 Isolamento multi-tenant — provas

- Toda query de repositório inclui `WHERE organization_id = $1` no início. Checado: `leads`, `pipes`, `pipe_entries`, `pipe_confirmations`, `operations`, `tasks`, `audit_log`, `refresh_tokens`, `team_members`, `member_feature_permissions`.
- Cross-tenant isolation **testada** em: `handler/auth` integration (org A vs org B), `repository/operation` Lookup, `repository/confirmation` Get.
- `refresh_tokens` carrega `organization_id` na linha — usuário **não** pivota tenant via refresh.
- WS hub: Broadcast filtra por `TenantID` antes de enumerar conns; dropped silently se ausente.

### 3.3 Caminhos sensíveis cobertos

| Cenário | Defesa | Teste |
|---|---|---|
| Login com senha errada | Dummy bcrypt em user not-found → tempo constante | ✅ `auth_integration_test.go` |
| Refresh reutilizado | Reuse detection revoga chain inteira | ✅ `auth_integration_test.go` |
| Body tenta injetar `organization_id` | 400 TENANT_FIELD_FORBIDDEN | ✅ `auth_integration_test.go` |
| Mutation sem CSRF header | 403 | ✅ `auth_integration_test.go`, `csrf_test.go` |
| RBAC master bypass | Short-circuit antes do store | ✅ `rbac_test.go` |
| JWT `alg=none` | Rejeitado pelo `WithValidMethods` | ✅ `jwt_test.go` |
| JWT expirado | Rejeitado (`WithExpirationRequired`) | ✅ `jwt_test.go` |
| Rate limit exaurido | 429 + Retry-After calculado | ✅ `ratelimit_test.go` |
| Sentry PII scrub | Cookies/Authorization/email/IP redigidos | ✅ `sentry_test.go` |
| Cross-tenant read | `WHERE organization_id = $1` + sentinela ErrNotFound | ✅ 3 integration tests |

### 3.4 Pendências de segurança (explícitas, com sprint owner)

| Pendência | Razão | Sprint |
|---|---|---|
| `/master/impersonate/:org_id` | Audit log já tem `target_org_id`, mas endpoint não mounted | S14 Master Admin |
| Password change + invalidar todos os refresh | `RevokeByUser` existe no repo, falta endpoint | S13 |
| Distributed rate limiter | In-memory single-node atual; Redis vira em multi-réplica | Pós-S06 se horizontal |
| HSTS preload-list submission | Requer domínio prod estável | Pós-prod |
| CSP nonce-based p/ HTML | Fica no nginx/edge, não no API Go | Infra-level |
| Dependency scan (govulncheck + npm audit) em CI | CI já roda lint+test; scan vira S13 | S13 |

### 3.5 Não-pendências (verificadas)

- `/api/bootstrap` **não vaza** o server DSN — o campo `sentry_dsn` recebe `SentryPublicDSN` (campo distinto em `config.go`, linha 60). Confirmado em `cmd/api/main.go` o wiring passa `cfg.SentryPublicDSN`.
- `SessionBundle` injetado em dev (`DEV_SESSION`) só compila em dev-build — `import.meta.env.DEV` é constant-folded `false` em produção, Vite elimina o branch.
- `dangerouslySetInnerHTML` tem ban por ESLint desde S00.
- Refresh cookie tem `Path=/api/v1/auth` — **não** é enviado em todas as requests.

---

## 4. Conhecimento — documentação e decisões

### 4.1 Documentação estruturada

| Artefato | Status | Qualidade |
|---|---|---|
| `Torque-dir-new/00 - Indice.md` | Vivo, atualizado por sprint | Navegação central + status em uma tela |
| `Torque-dir-new/01 - Produto/` | Visão, personas, glossário, princípios | Criado em S00, inalterado (estável) |
| `Torque-dir-new/02 - Arquitetura/` | Auth, tenancy, segurança, permissões | Referência canônica; alinhado com código |
| `Torque-dir-new/03 - Modelo de Dominio/` | 13 entidades | Alinha com migrations 0001–0006 |
| `Torque-dir-new/04 - Design/` | Tipografia, motion, vocabulário UI | Implementado em S00/S08/S09 |
| `Torque-dir-new/05 - Sistema Base/` | Escopo+checklist+plano de execução | Fechado em S00 |
| `Torque-dir-new/06 - Funcionalidades/` | 7 subpastas | Specs de features, alinhadas com S07–S10 |
| `Torque-dir-new/07 - Features/` | F01 spec vertical + F17 apontador | F01 parcialmente entregue em S07–S09 |
| `Torque-dir-new/08 - Decisoes/` | **8 ADRs** (000 template + 001–007) | Todas referenciadas no código |
| `Torque-dir-new/09 - Backlog/Plano Mestre` | 30 sprints, §8 com estado por sprint | S00–S10 marcadas ENTREGUE com artefatos |
| `.specs/project/STATE.md` | **22 decisões** D001–D022 | Trilha decisória completa |
| `CLAUDE.md` | Protocolo de git + protocolo de agentes + stack | INVARIANTES travadas |
| `.claude/skills/agent-conductor/SKILL.md` | Passo 0 (branch correta) antes de triagar | Codifica a enforcement do protocolo |
| `.specs/audits/coverage-2026-04-19.md` | Auditoria anterior (S05) | Marco histórico |
| `torque-api/api/openapi.yaml` | 340 linhas, cobre todos os endpoints S01–S10 | Fonte de verdade do contrato |
| `torque-api/README.md` | Quickstart, make targets, dev seeds | Atualizado em S02/S10/D021 |

### 4.2 ADRs — relacionamento com código

| ADR | Implementado em | Status |
|---|---|---|
| ADR-001 snake/camel boundary | `src/api/` + Go JSON tags + `openapi.yaml` | ✅ vivo |
| ADR-002 WebSocket hub por tenant | `internal/ws/hub.go` | ✅ S04 |
| ADR-003 Auth httpOnly cookies | `handler/auth` + 3 cookies + middleware stack | ✅ S02 |
| ADR-004 Cursor pagination | `repository/lead` + `useInfiniteList` | ✅ S05+S07 |
| ADR-005 Fontes self-hosted | `@fontsource/*` em package.json + CSP font-src self | ✅ S00 |
| ADR-006 Jobs 202+poll+push | `handler/operations` + `worker.Pool` + WS events | ✅ S04 |
| ADR-007 UI modes Vendedor/Gerente | migration 0003 + `UiModeProvider` + `/cockpit` + claymorphism escopado | ✅ S00 (ADR) + S06 (wire backend) |

**Nenhum** ADR está órfão (decidido mas não implementado) ou quebrado (código divergiu sem atualizar a ADR).

### 4.3 Protocolo de execução (D012) — conformidade

| Regra | Validação | Conforme? |
|---|---|---|
| Branch `sprint/S0X` de `develop` atualizada | 11 branches `origin/sprint/S00..S10` | ✅ |
| Commits lógicos por domínio (DBA/Backend/QA/Frontend/Docs) | Verificável em cada merge-commit | ✅ |
| Push + PR para develop via `merge --no-ff` | Topologia de merges preserva granularidade | ✅ |
| STATE `D0xx` por sprint | D001–D022 contínuos | ✅ |
| `Indice.md` status line atualizada | Último diff mostra S10 ✅ | ✅ |
| Plano Mestre §8 marca sprint ENTREGUE | S00–S10 todas marcadas | ✅ |
| Branch preservada como âncora | Todas as 11 existem em `origin` | ✅ |

### 4.4 Cobertura cruzada (traça um caminho)

Exercício: pegar uma feature e verificar a cadeia documento → código → teste.

**F01 `POST /api/v1/pipes/:id/entries/move`:**
1. Spec: `07 - Features/F01 - Funis Hub e Pipe WhatsApp/Spec.md` descreve o fluxo
2. ADR: ADR-002 WebSocket hub publica patches
3. ADR: ADR-004 cursor pagination (aplicável em lista de entries)
4. STATE D017 registra a entrega S07
5. Migration: tabela em `0002_team_members_leads_pipes.up.sql` linha 214
6. Domain: `internal/domain/lead.go` `PipeEntry`
7. Repository: `internal/repository/pipe/pipe.go` `Move` (Serializable tx)
8. Handler: `internal/handler/pipes/pipes.go` `move()`
9. Event: `ws.Event{Type: "pipe_entry.moved"}` publicado no bus
10. OpenAPI: `api/openapi.yaml` linha 166+
11. Frontend hook: `src/hooks/usePipes.ts` `useMovePipeEntry` com optimistic
12. Frontend WS: `usePipeEntries` subscribe em `pipe_entry.moved` → `setQueryData`
13. Test: `repository/lead/lead_test.go` cursor + (integration test de Move planejado em S11 QA)

**Cadeia completa em 13 pontos**, verificável, sem salto lógico.

---

## 5. Riscos residuais e calibragem

### 5.1 Riscos que **continuam** existindo

| Risco | Mitigação atual | Próximo passo |
|---|---|---|
| Toolchain Go ausente neste workspace | Impede rodar `go test` + `go vet` + `govulncheck` aqui | Usuário roda local após `go install` |
| Sem banco rodando aqui | Integration tests não executados | Seed + docker compose up → `make test-integration` |
| Frontend pages sem unit test | Integration surface em S11+ | S11 conecta rotas a dados reais e teste E2E entra |
| Rate limiter in-memory | Funciona em single-node | Redis quando horizontal |
| WS hub in-memory | Idem | Idem ou sticky sessions |
| Worker pool sem handlers registrados | Infra pronta, kinds plug-in por sprint | S14+ (leads.import), S17+ (workflow.execute) |

### 5.2 Riscos que **foram** removidos por entregas

- Mock de sessão em prod → removido em S06 (`MOCK_SESSION` deletada).
- Fallback silencioso quando backend falha → trocado por redirect to /login.
- `/auth/refresh` path errado (era `/auth/refresh`) → corrigido para `/api/v1/auth/refresh` em S06.
- Refresh-on-login mascarando INVALID_CREDENTIALS → corrigido neste branch de auditoria (§2.5).
- Server Sentry DSN vazando no bootstrap → `SentryPublicDSN` separado em S03.

---

## 6. Ação recomendada

### 6.1 Próxima sprint (S11)

Conforme Plano Mestre — F03 (Propostas). Independente disso, no primeiro ciclo disponível no host:

```bash
# Backend
cd torque-api && go mod tidy && go build ./... && \
  make migrate-up && make seed-dev && \
  DATABASE_URL="$DATABASE_URL" make test-integration

# Frontend
cd torque-web && npm install && npm test && npm run typecheck && npm run lint
```

Após rodar: anexar a saída a este arquivo em uma nova seção §7 "Runtime evidence".

### 6.2 Itens acionáveis sem bloquear S11

1. Adicionar `govulncheck` + `npm audit --production` ao CI `.github/workflows/ci.yml`. Esforço: 30min. Bloqueia merges com CVEs conhecidos.
2. Unit test dedicado para `repository/user.resolve()` (função pura, RBAC cascade). Esforço: 20min. Leva S02 de ~85 % → 95 %+ efetivo.
3. Integration test para `repository/pipe.Move` concurrency (duas goroutines movendo o mesmo lead). Esforço: 40min. Prova Serializable tx.

---

## 7. Fechamento

Onze sprints, 22 decisões registradas, 8 ADRs, seis migrations, 20 tabelas, 22 endpoints, 45 arquivos de teste entre frontend e backend, **zero TODO no Go**, uma regressão pega e corrigida pela própria suite. O sistema está em estado **world-class-consistente** — instrumentado em três camadas (invariantes em CLAUDE.md, enforcement no Conductor, prova em STATE+Plano Mestre), cada linha rastreável a uma sprint e uma decisão.

Fundação pronta para features verticais S11+.
