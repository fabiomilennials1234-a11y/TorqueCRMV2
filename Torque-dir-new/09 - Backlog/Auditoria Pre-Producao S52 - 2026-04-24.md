---
tags:
  - auditoria
  - go-to-prod
  - gate
  - s52
created: 2026-04-24
status: vivo
autoria: Conductor + 8 especialistas (architect, backend, frontend, dba, qa, infra, automation, ai)
---

# Auditoria Pre-Producao S52 (2026-04-24)

> Executada no fechamento de S52 ("FASE G CONCLUIDA + ROADMAP S30-S52 COMPLETO + Gate go-to-prod aberto") para responder: **o sistema esta pronto pra ativar producao?**. Veredito curto: **NAO. Gaps bloqueadores. Sprint S53 (Fase H — Production Gate Verification) e inegociavel.**

## 1. Veredito

**NOT READY.** Gate §10 do Plano Paridade v8 tem **1 done, 1 partial, 4 pending, 2 unknown** (de 8 criterios). Mais grave: 3 bugs duros descobertos nesta auditoria que o STATE.md nao tinha detectado:

1. **DB clean-boot FALHA** — migration `0022_performance.up.sql:67` referenciava `proposals(id)` (tabela inexistente; existe `pipe_proposals`). Qualquer ambiente novo explode no `migrate up`. **Corrigido nesta sessao** (FK aponta pra `pipe_proposals(pipe_entry_id)`; coluna preservou nome `proposal_id` pra alinhar com API JSON).
2. **Backend nao compilava** — `cmd/api/main.go:207` redeclarava `busSub :=` ja declarado em `:162`. `go build ./...` falhava. **Corrigido nesta sessao** (renomeado para `wfSub`).
3. **Workflow engine eh teatro** — 7/7 action handlers sao stubs literais (`dispatcher.go:149`, `actions_s45.go:41,73,109`). UI marca run como "succeeded", mundo real recebe zero. **NAO corrigido — e' 1 sprint inteira de trabalho (S53).**

## 2. Placar Gate §10

| # | Criterio | Status | Evidencia real |
|---|----------|--------|----------------|
| 1 | Paridade funcional ≥95% vs v8 | ⚠️ pending | D045 mediu 32% ha 2 meses; S30-S52 endereçou muito, nao revalidou o numero |
| 2 | Cobertura vitest ≥70% lines | ✅ done | 78.48 / 75.30 / 74.75 / 62.22 em 86 files / 296 tests. CI ratchet ativo. |
| 3 | Cobertura Go ≥75% packages criticos | ❓ unknown | `go test -coverprofile` nunca rodou; 19/55 test files gated em DATABASE_URL |
| 4 | Zero HIGH gosec+govulncheck+Trivy 7d | ❓ unknown | security-scan.yml wired, historico de 7d verde nao auditado |
| 5 | k6 p95<500ms staging | ❌ pending | Script `.specs/loadtest/api-baseline.k6.js` existe, thresholds codificados, **nunca executado** |
| 6 | Pentest passou | ❌ pending | Runbook `.specs/security/pentest-runbook-S52.md` pronto (10 checks), **zero execucao** |
| 7 | Dual-review money-flow | ⚠️ partial | Protocolo doc; **CODEOWNERS ausente** — **corrigido nesta sessao** (`.github/CODEOWNERS` criado). Branch protection pendente. |
| 8 | Runbook game day dry-run | ❌ pending | `incident-response.md` e `oncall-basics.md` completos; **zero exercicio** |

## 3. Gaps criticos consolidados (8 agentes)

### Bloqueadores absolutos (must-fix antes de deploy)
1. **Workflow handlers stubs** (Automation/Backend) — 7/7 actions no-op. Produto "automatiza" nada. Effort: **20-30h**.
2. **Workflow sem retry/DLQ** (Automation) — qualquer transient failure mata run. Effort: **12-16h**.
3. **Agent trigger dispatcher ausente** (AI) — matcher S40 existe mas nada chama. Triggers sao UI-only. Effort: **4h**.
4. **Budget cap AI ausente** (AI) — sem `org_quotas.ai_tokens_per_month`, prod pode quebrar caixa com loop de agent. Effort: **4-8h**.
5. **PII scrub pre-LLM ausente** (AI) — lead PII (CPF/telefone/email) indo cru pra OpenRouter/Gemini. **Risco LGPD.** Effort: **4-6h**.
6. **Quota enforcement parcial** (Backend) — so POST /leads gated; `team_members`, `workflows`, `agents` pattern-ready nao wired. Effort: **6h**.
7. **`api.gen.ts` stub** (Frontend) — arquivo tem 7 linhas apesar de OpenAPI 0.52.0 ter 1709. Contract safety desligada. Effort: **2-4h**.
8. **Backup/DR ZERO** (Infra) — sem script pg_dump, sem runbook restore, sem retention. **Data loss = game over.** Effort: **1-2d**.
9. **Zero E2E** (QA) — produto multi-canal WebSocket sem nenhum teste ponta-a-ponta. Effort: **2-3d** (Playwright minimo).
10. **Rate-limiter in-memory** (Infra) — single-pod only. 2 replicas = limite dobra. Effort: **1d** (Redis/valkey).

### Corrigidos nesta sessao (2026-04-24)
- ✅ Build fail `busSub` em main.go:207 (renomeado para `wfSub`)
- ✅ FK quebrada migration 0022 (`proposals` → `pipe_proposals`)
- ✅ `.env.example` defasado: adicionadas 15+ vars S39-S52 (Asaas, OpenRouter, Gemini, ElevenLabs, Google OAuth, TinyERP, Meta, SZ.Chat, Lead webhook, Integration encryption)
- ✅ Link quebrado FunisHubPage (`/funil/:id` → `/pipe/:id`)
- ✅ Sidebar: badges hardcoded `24` / `9` / "Plano Growth 64%" removidos
- ✅ CODEOWNERS criado (`.github/CODEOWNERS`) cobrindo billing, quotas, auth, master, jwt, crypto, permission, migrations, runbooks

### Nao-bloqueadores mas dividas altas
- Impersonation cookie swap atomico (S52 deferred) — 3-4h
- `knowledge_chunks.embedding_text` cleanup — 10min
- GIN trigram em messages.body + leads.name/email/phone — 50min (alto risco sob >10k msgs/tenant)
- `meetings` UNIQUE sem org_id — 10min (risco medio — colisao cross-tenant em Google Calendar IDs compartilhados)
- 22 handlers sem test HTTP-level — M effort
- 24 repos sem tenant isolation test — padrao existe, precisa aplicar
- OpenTelemetry prometido mas nao wired — 4h ou atualizar spec
- Release draft manual (`release.yml:101`) precisa documentar no runbook
- Schedule trigger cron ausente — 8-10h
- TTS outbound pipeline ausente — 10-14h

## 4. Wins reais (manter padrao)

- **Frontend acima de "85% prototipo"**: coverage 78%, typecheck/lint zero-warning, AgentPlaygroundPage 890 LOC SSE streaming, claymorphism scoped (ADR-007), dual-theme HSL tokens, `lazyRetry` com backoff, code-split AppShell.
- **Infra S-tier**: cosign keyless + verify + migrate gating + distroless nonroot + read-only FS + no-new-privileges em prod compose. Padrao de empresa 9-digitos.
- **Schema solido**: 40 tabelas, tenancy invariant universal, 100% FKs com ON DELETE explicito, pgvector HNSW, migrations 100% reversiveis (apos fix da 0022).
- **Security invariantes exemplares**: StripOrganizationID strict reject 400, audit-fail-closed em master impersonate, HMAC constant-time em lead webhook, CSRF double-submit + OAuth state HMAC com TTL.
- **27 handlers + 27 repos + 11 services + ~145 endpoints** — cobertura de superficie muito mais madura que STATE sugere.
- **Tests: 296/296 frontend, 55 `_test.go` backend, dual-review tripwire em Asaas, pentest runbook 10 checks executaveis, k6 script bem desenhado**. Especificacao boa; execucao pendente.

## 5. Referencias

- Architect veredito: secao 1-6 em `.` (este documento)
- Backend audit: 26 migrations, 145 endpoints, bugs concretos em `cmd/api/main.go:207` e `migrations/0022_performance.up.sql:67`
- Frontend audit: `api.gen.ts` stub, FunisHubPage link broken, Sidebar badges, F02/F03 UI missing, fontes divergentes (DM Serif vs Fraunces no CLAUDE.md)
- DBA audit: FK 0022 quebrada (corrigida), trigram missing em leads/messages, meetings UNIQUE sem org_id
- QA audit: gate §10 score real (1✅/2⚠️/5❌), CODEOWNERS ausente (corrigido), E2E inexistente
- Infra audit: backup/DR zero, .env.example defasado (corrigido), OTel nao wired, rate-limiter in-memory
- Automation audit: 7/7 workflow handlers stubs, trigger dispatcher ausente, cron inexistente
- AI audit: tokens nao persistidos playground, budget cap ausente, PII scrub ausente, prompt injection via RAG

## 6. Proximos passos

Ver [[Plano S53 - Fase H Production Gate Verification]] (criado nesta sessao).

**Soft-launch recommendado pos-S53**: 3 tenants piloto handpicked × 30 dias antes de tenant pagante externo.

---

*Auditoria executada em 2026-04-24 pelo Conductor com 8 agentes especialistas em paralelo. Decisao: D073 no STATE.md.*
