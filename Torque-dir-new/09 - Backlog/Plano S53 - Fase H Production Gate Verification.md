---
tags:
  - sprint
  - s53
  - fase-h
  - go-to-prod
  - gate-verification
created: 2026-04-24
status: ativo
parent: "[[Plano de Acao Paridade v8 - Sprints S30-S52]]"
depends_on:
  - "[[Auditoria Pre-Producao S52 - 2026-04-24]]"
---

# Sprint S53 — Fase H Production Gate Verification

> Sprint que sucede o roadmap Paridade v8 (S30-S52). Não é sprint de feature — é sprint de EVIDÊNCIA EMPÍRICA. Transforma "entregue em código" em "rodando em realidade". Critério inegociável: todos os 8 itens do Gate §10 com evidência executada.

## Contexto

Auditoria pos-S52 de 2026-04-24 revelou que o gate §10 não esta fechado: **1 done, 1 partial, 4 pending, 2 unknown**. Alem disso, tres bugs bloqueadores foram descobertos — dois ja corrigidos nesta sessao (migration 0022 FK quebrada, `busSub` redeclaracao em main.go), um nao corrigido (workflow handlers sao stubs 7/7).

**Sprint S53 vive pra destravar o go-to-prod**, não pra adicionar escopo. Todo trabalho aqui endereca evidencia ou gaps de runtime critico descobertos na auditoria.

## Topologia de branch (invariante linear)

```
develop (com S52 mergeada)
  ↑ merge via PR ao fim de S53
sprint/S53  ← nasce de develop atualizada
```

```bash
git checkout develop
git pull --ff-only origin develop
git checkout -b sprint/S53
```

## Escopo (3 semanas)

### Semana 1 — Quick wins + bugs bloqueadores

Alguns ja executados no conductor-session de 2026-04-24 (sinalizados com ✅):

- [x] Bug #1 — migration 0022 FK `proposals` → `pipe_proposals(pipe_entry_id)` (preserva nome `proposal_id` pra API JSON)
- [x] Bug #2 — `cmd/api/main.go:207` renomear `busSub` → `wfSub` (fixa `go build`)
- [x] `.env.example` com 15+ vars S39-S52 (Asaas/OpenRouter/Gemini/ElevenLabs/Google OAuth/TinyERP/Meta/SZ.Chat/Lead webhook/Integration encryption)
- [x] `FunisHubPage.tsx:58` — link `/funil/:id` → `/pipe/:id`
- [x] Sidebar — badges hardcoded removidos (24 leads, 9 conversas, "Plano Growth 64%")
- [x] `.github/CODEOWNERS` criado com placeholders pra billing/quotas/auth/master/jwt/crypto/permission/migrations/runbooks
- [ ] **Branch protection GitHub** em `main` + `develop` exigindo review de CODEOWNERS (ação humana no GitHub UI)
- [ ] `api.gen.ts` regen via `npm run generate:types` apontando pro backend com OpenAPI 0.52.0 ativo — resolver drift com `manual.ts` + ajustes de tipo que cascateiam
- [ ] `knowledge_chunks.embedding_text` cleanup — migration `0027_cleanup_legacy.up.sql` DROP COLUMN
- [ ] Meetings UNIQUE include org_id — migration `0028_meetings_org_unique.up.sql`
- [ ] Trigram GIN em `leads.name|email|phone` + `messages.body` — migration `0029_search_trgm.up.sql`
- [ ] Impersonation cookie-swap atomico em `handler/master/master.go:150-199` — mintar JWT short-lived, `http.SetCookie` HttpOnly/Secure/SameSite=Strict path `/api` TTL 30min

### Semana 2 — Críticos de runtime (handlers, quotas, AI safety)

Sem estes o produto nao executa automacao real nem suporta carga financeira.

**Workflow engine real (agent-automation + agent-backend):**
- [ ] `SendMessageAction` — implementar chamada real ao Evolution adapter com circuit breaker S27
- [ ] `UpdateLeadAction` — chamar `leadrepo.Update` com patch parcial + publish `lead.updated`
- [ ] `CreateTaskAction` — chamar `taskrepo.Create` + assign
- [ ] `CallAgentAction` — abrir sessao em `agent_sessions`, enfileirar user message + chamar provider stream
- [ ] `HTTPRequestAction` — fetch real com timeout 30s + SSRF allowlist (block 127./10./172./192.168./169.254./metadata.google.internal)
- [ ] `WaitAction` — implementar suspensao real (marcar run `suspended`, enfileirar `workflow_wake` em `operations` com `scheduled_at`)
- [ ] Migration `0030_workflow_attempts.up.sql` — adicionar `attempts int NOT NULL DEFAULT 0`, `max_attempts int NOT NULL DEFAULT 3`, `next_retry_at timestamptz NULL` em `workflow_runs`
- [ ] Retry exponential backoff (15s/1min/5min/30min/caps em 2h) em `executor.go` — fail transient → incrementa attempts, agenda next_retry_at
- [ ] Dead-letter queue — quando attempts >= max_attempts, marca `failed` + insere em `workflow_run_failures` (nova tabela) com snapshot
- [ ] Watchdog — query `running` com `started_at < now() - 10min` sem step append → marca `failed` com note "orphaned runner"

**Agent trigger dispatcher (agent-ai + agent-automation):**
- [ ] Novo `service/ai/trigger/subscriber.go` — `BusSubscriber` registra no `event.Bus`, filtra `message.received` + `conversation.created`, chama `Matcher.Match`, se match: emite `conversation.assign_agent` event + UPDATE `conversations.assigned_agent_id`
- [ ] Wire em `cmd/api/main.go` ao lado do `wfSub`
- [ ] Test `trigger_subscriber_test.go` — event dispara matcher; kill-switch `agents.status='disabled'` skipa

**Quota enforcement completo (agent-backend):**
- [ ] `handler/members/members.go:add` — `RequireQuota(h.quota, quotarepo.ResourceTeamMembers)`; na deactivate chama `IncrementUsage(-1)` se ativo
- [ ] `handler/workflows/workflows.go:create` — `RequireQuota(..., ResourceWorkflows)`; archive chama `IncrementUsage(-1)`
- [ ] `handler/agents/agents.go:createAgent` — `RequireQuota(..., ResourceAgents)`; status=disabled chama `IncrementUsage(-1)`
- [ ] Integration test por resource — 402 at cap, headers canonicos, master bypass

**AI budget cap (agent-ai + agent-dba):**
- [ ] Migration `0031_ai_token_quota.up.sql` — adicionar `ai_tokens` em `plan_quotas` seeds (free: 50k · growth: 500k · enterprise: 5M); `resource_key` regex ja aceita
- [ ] `handler/agents/playground.go` — middleware `RequireAITokenBudget` antes de `provider.Chat`; pos-stream incrementa `IncrementUsage(+usage.total_tokens)`
- [ ] `handler/agents/tts.go` — quota separada `tts_seconds` (via len(text)/avg_chars_per_sec) se habilitado
- [ ] Fronted QuotaMeter em `SettingsPage` tab Plano pra AI tokens

**PII scrub pre-LLM (agent-ai + agent-backend):**
- [ ] Novo `service/ai/pii/scrub.go` — regexes pra CPF (`\d{3}\.?\d{3}\.?\d{3}-?\d{2}`), telefone BR (`(\+?55\s?)?\(?\d{2}\)?\s?\d{4,5}-?\d{4}`), email (`[\w.+-]+@[\w-]+\.[\w.-]+`). Substitui por `[CPF-REDIGIDO]`, `[TELEFONE-REDIGIDO]`, `[EMAIL-REDIGIDO]`.
- [ ] Aplicar em `prependContext` (pre-RAG) + user turn (pre-chat); system prompt nao scrubado (curado).
- [ ] Log fraccao scrubada pra observability sem conteudo.
- [ ] Test com 20+ patterns reais (com espacos, com/sem mascaras).

**Kill-switch que interrompe stream em voo (agent-ai):**
- [ ] `ai/runtime_registry.go` — in-memory map `agentID → []context.CancelFunc`; registra no start do stream, cancela todos na kill-switch flip.
- [ ] `handler/agents/agents.go:disable` — chama `registry.CancelAll(agentID)` antes de retornar 204.

### Semana 3 — Execução do Gate §10 contra staging real

**Provisionamento staging (agent-infra):**
- [ ] VPS Hostinger dedicada `staging.torquecrm.com.br` + managed Postgres 15 com `pgvector` toggle habilitado
- [ ] Dominio configurado (Cloudflare DNS + Let's Encrypt via EasyPanel)
- [ ] GitHub Environments `staging` com required reviewers + todas as 15+ env vars seeded (gerar valores reais: JWT_SECRET 48-byte, INTEGRATION_ENCRYPTION_KEY 32-byte, BILLING_WEBHOOK_SECRET 32-byte, LEAD_WEBHOOK_SECRET 32-byte, provider sandbox keys)
- [ ] Tag `v0.53.0-rc1` em `sprint/S53` — **primeira vez que cosign sign real roda** via `release.yml`
- [ ] Actions → Deploy → staging com `image_tag=v0.53.0-rc1`, `run_migrations=true` — primeira exec de `verify` job real
- [ ] Smoke test: `curl staging/healthz`, `curl staging/readyz`, login via frontend, Sentry recebe test event
- [ ] Validar qualquer falha → sub-issue + re-sprint imediata

**Backup / DR (agent-infra + agent-dba):**
- [ ] `.specs/runbooks/backup-restore.md` — script `pg_dump` diario → object storage versioned (S3-compat), retention 30d, cron trigger
- [ ] `scripts/backup-postgres.sh` executavel + `scripts/restore-postgres.sh`
- [ ] Restore drill em staging — dump prod, wipe staging, restore, validate smoke
- [ ] RTO target: 1h · RPO target: 24h (reavaliar com cliente pagante)
- [ ] `INTEGRATION_ENCRYPTION_KEY` rotation runbook em `.specs/runbooks/key-rotation.md` + escrow

**Security scan 7-dias green (agent-infra):**
- [ ] `gh run list --workflow=security-scan.yml --limit 30 --json conclusion,createdAt` commitar em `.specs/security/security-scan-history-2026-05-01.json`
- [ ] Triage qualquer HIGH/CRITICAL finding
- [ ] Se alguma RED: abrir issue, fixar, re-rodar. Janela de 7 dias consecutivos verde = fechado.
- [ ] Pinnar `securego/gosec` e `aquasecurity/trivy-action` por SHA (nao tag `@master`)
- [ ] Escanear `torque-web` image com Trivy (nao so API)

**k6 load test (agent-qa + agent-infra):**
- [ ] Provisionar test tenants `alpha/beta/staging-master` + seed dados representativos (~1000 leads, 500 conversations, 200 campaigns)
- [ ] `BASE_URL=https://staging.torquecrm.com.br k6 run --summary-export=.specs/loadtest/run-2026-05-01.json .specs/loadtest/api-baseline.k6.js`
- [ ] Gravar p95/p99/error-rate por endpoint group
- [ ] Falha em qualquer threshold → stop-ship, diagnosticar
- [ ] Commit `.specs/loadtest/run-2026-05-01.md` com analysis

**Pentest staging (agent-qa + agent-infra + Security):**
- [ ] Executar os 10 checks em `.specs/security/pentest-runbook-S52.md` passo a passo
- [ ] `cp .specs/security/pentest-staging-template.md .specs/security/pentest-staging-2026-05-01.md`
- [ ] Para cada check: PASS/FAIL + request/response dumps
- [ ] Zero HIGH/CRITICAL findings
- [ ] Dual sign-off duplo (Security + QA)

**Go coverage mensurada (agent-qa + agent-backend):**
- [ ] Docker compose up -d postgres em CI
- [ ] `DATABASE_URL=... go test -coverprofile=coverage.out ./...` em `ci.yml` publicando artifact
- [ ] `go tool cover -func=coverage.out` — valida ≥75% em packages críticos: `service/billing`, `service/jwt`, `service/permission`, `httpx/middleware`, `repository/lead`, `repository/quota`
- [ ] Se abaixo: adicionar testes antes de merge

**Game day dry-run (agent-infra + agent-backend + Security):**
- [ ] Simular 4 cenários em staging com timer:
  1. DB kill 30s → medir detecção, failover, recovery; validar runbook §incident-response
  2. Asaas 5xx sustentado 5min → circuit breaker + degraded mode + webhook retry handling
  3. Worker pool goroutine leak (forcar via test flag) → OOM kill simulation
  4. Evolution 503 em lead webhook dispatch → HMAC reject com audit
- [ ] Registrar timeline em `.specs/runbooks/gameday-2026-05-02.md`
- [ ] Findings → issues → fix antes de abrir prod

**Feature flags minimos (agent-backend):**
- [ ] Env-var-based kill switches `COPILOT_ENABLED`, `WORKFLOWS_ENABLED`, `CAMPAIGNS_ENABLED`, `ASAAS_ENABLED`
- [ ] `/api/bootstrap` expõe flags por tenant pra frontend
- [ ] Handler guarda `if !flags.CopilotEnabled { 503 FEATURE_DISABLED }`
- [ ] Toggles documentados em runbook (como desligar Copilot para todos os tenants em 5min)

## Checklist de fechamento inviolavel

Sprint S53 NAO esta fechada sem:

- [ ] Branch `sprint/S53` pushada em origin
- [ ] PR aberta contra `develop` via `gh pr create`
- [ ] STATE.md com decisao `D074` descrevendo escopo entregue + escopo deferido honestamente
- [ ] `00 - Indice.md` status line atualizada
- [ ] Plano Mestre §8 marcando S53 ENTREGUE + artifatos
- [ ] Commits organizados por dominio (DBA → Backend → QA → Frontend → Infra → Docs)
- [ ] typecheck + lint zero warnings
- [ ] vitest verde (regressoes zero)
- [ ] integration tests Go verdes (com DATABASE_URL)
- [ ] Cobertura nao regrediu (ratchet)
- [ ] **Gate §10 com 8/8 evidencias concretas e sign-off duplo onde aplicavel**

## Pos-S53: Soft launch

Nao abrir tenants externos pagantes imediatamente. Recomendacao:

1. **Semana 1-4 pos-S53**: 3 tenants piloto handpicked (equipe interna + 2 conhecidos). Monitorar Sentry error rate, p95 latency, AI token spend, webhook dedup rate, WS conn stability.
2. **Criterios de abertura pra tenant pagante externo**:
   - Zero P0 incidentes em 30 dias consecutivos
   - p95 stable < 500ms em janela de 7d
   - AI cost/tenant previsivel (modelo de pricing validado)
   - Runbook incident-response exercitado em 1 incidente real (nao simulacao)
   - Suporte humano (CS) treinado nos cenarios do gameday
3. **Open beta**: max 20 tenants. Preco de early-adopter. 60 dias.
4. **GA**: apos 90 dias soft launch + open beta com dados positivos.

**Sprint S54+**: paralelo, features que ficaram pra pos-prod (SZ.Chat inbound, Meta Lead Ads relay, Asaas boleto+cartao, schedule trigger cron, TTS outbound pipeline, F02 Countdown UI, F03 HeatSlider UI, workflow side-effects restantes, matterialized views upsell, E2E Playwright, Redis rate-limiter, OpenTelemetry wiring, dispatch SDR/Closer, materialized upsell_opportunities).

## Referencias

- [[Auditoria Pre-Producao S52 - 2026-04-24]] — auditoria completa
- [[Plano de Acao Paridade v8 - Sprints S30-S52]] §Gate Go-to-Production — critérios §10
- `.specs/security/pentest-runbook-S52.md` — 10 checks
- `.specs/loadtest/api-baseline.k6.js` — cenarios k6
- `.specs/runbooks/incident-response.md` — runbook oncall
- `.github/CODEOWNERS` — dual-review money-flow

---

*Plano S53 inaugurado em 2026-04-24 apos auditoria de 8 agentes especialistas. Owner: Conductor. Decisao: D073 no STATE.md.*
