# STATE.md — Torque CRM v2

> Last updated: 2026-04-19

## Current Phase
Sistema Base Frontend **concluido (S00 ✅, 2026-04-18)**. Sprint **S01 iniciada (2026-04-19)** e scaffold entregue: modulo Go `torque-api/`, migrations 0001–0003, servidor chi em `:8080` com probes, Docker Compose, Makefile, teste de concorrencia do unique partial index. Execucao runtime (go build, docker compose up, go test -race) pendente pois Go/Docker nao estao instalados no workspace do Conductor — transferida para o usuario. Sprint S02 (auth + tenancy + RBAC) comeca quando o scaffold rodar local.

## Key Decisions

| ID | Date | Decision | Rationale |
|----|------|----------|-----------|
| D001 | 2026-04-15 | Vault Obsidian criado como fonte de verdade | Substituir vault legado, comecar limpo |
| D002 | 2026-04-15 | 6 ADRs formalizados (OpenAPI, WS, Auth, Cursor, Fonts, Jobs) | Decisoes arquiteturais travadas antes de implementar |
| D003 | 2026-04-15 | 9 agent profiles documentados no vault | Conductor, Architect, Backend, Frontend, DBA, QA, Infra, Automation, AI |
| D004 | 2026-04-15 | Etapa 0 blockers (B1-B6) identificados e 5/6 resolvidos | B6 (ESLint) continua sem regras reais |
| D005 | 2026-04-16 | Plano Mestre de Finalizacao criado (30 sprints) | Roadmap operacional completo ate producao |
| D006 | 2026-04-16 | Vault reorganizado (12 secoes, zero dualidades) | Duas taxonomias paralelas mergeadas em uma |
| D007 | 2026-04-17 | CLAUDE.md + .specs/ + agent skills criados | Protocolos de agentes e skills extraidos do legado e adaptados para Go |
| D008 | 2026-04-17 | ADR-007: Modos de UI Vendedor/Gerente + Task unifica Follow-up + rota /cockpit + claymorphism escopado | (1) Vendedor e Gerente sao modos de UI (preferencia pessoal), nao roles RBAC — invariante `admin/membro/master` preservada. (2) `Task` absorve `Follow-up` (rename sem custo pois backend e 0%) e ganha estado `in_progress` unico por assignee. (3) Rota separada `/cockpit` com shell proprio — evita condicionar AppShell por modo. (4) Preferencia persistida em `users.ui_mode_preference` via `PATCH /me/preferences`. (5) Claymorphism vive apenas sob `.cockpit-theme`, accent gold canonico preservado. (6) Classificacao: extensao do Sistema Base, nao feature F17 vertical — F17 fica como apontador. |
| D009 | 2026-04-18 | Sprint S00 concluido | Sistema Base Frontend fechado. ESLint flat config com regras reais (a11y + hooks + ban `dangerouslySetInnerHTML`), TS strict estendido (`noUncheckedIndexedAccess` + `exactOptionalPropertyTypes`), Prettier + plugin-tailwindcss, Vitest + Testing Library (51 testes, 36 snapshots), GitHub Actions CI, vocabulario canonico em `src/i18n/vocabulary.ts`, `.grain` utility, renomeacao legacy `--clay-*` → `--card-*`/`.tactile-*` eliminando vazamento, pos-login navigation + `useUiMode` + fallback localStorage, `AppError` como classe. Legacy `src/lib/vocabulary.ts` removido (nao importado). |
| D010 | 2026-04-19 | Sprint S01 scaffold Go Backend | Modulo `github.com/milennials/torque-api` com chi + pgx v5 + zerolog + google/uuid. Bootstrap com graceful shutdown, config via env (fail-fast em `DATABASE_URL`), probes `/healthz` + `/readyz`. Middleware stack: RequestID → AccessLog → Recover → SecurityHeaders → CORS → StripOrganizationID (rejeita 400 se body traz `organization_id`). Migrations 0001 (orgs, users, users_master, plans, feature_permissions, `set_updated_at()`, extensions pgcrypto+citext), 0002 (team_members com unique (org,user), member_feature_permissions, org_quotas no delta model, tags, leads com CHECK E.164 + phone_or_email, pipes + pipe_stages + pipe_entries, lead_history, audit_log com target_org_id para impersonacao master), 0003 (ADR-007 conforme §1 de `UI Modes - Vendedor e Gerente`). Dockerfile multi-stage distroless nonroot, docker-compose com Postgres 15 + migrate + API. Makefile com run/test/test-integration/migrate-*/docker-*. OpenAPI 3.1 skeleton em `api/openapi.yaml`. Teste de concorrencia `TestTasksConcurrentInProgressSingleton` (32 goroutines contra o unique partial index `uq_tasks_one_in_progress_per_assignee`, gated por DATABASE_URL e `-short`). |
| D012 | 2026-04-19 | Protocolo de git: topologia linear cumulativa (regra travada, INVARIANTE) | **Regra:** toda sprint (S00, S01, S02, ...) nasce de `develop` atualizada (ja com S0X-1 mergeada) e volta via PR para `develop`. Historico linear, cumulativo, sem fan-out a partir de S00. Branch `sprint/S0X` permanece no remoto apos merge como ancora de auditoria. **Justificativa (vs fan-out de S00):** sprints de fundacao S01–S06 tem dependencias verticais rigidas — S02 precisa do schema de S01, S04 precisa do auth de S02, S06 precisa de tudo. Fan-out forcaria cada sprint a reimplementar a base e produziria conflitos massivos. Tambem violaria a regra cardinal "uma feature por vez" (CLAUDE.md). **Commits por sprint (ordem obrigatoria):** (1) DBA `feat(db):` migrations/seeds, (2) Backend `feat(backend):` services/repos/middlewares/handlers/config/go.mod, (3) QA `test(backend):` unit+integration, (4) Frontend `feat(frontend):` componentes/hooks/i18n, (5) Docs `docs(vault):` STATE.md novo D0xx + Indice + Plano Mestre ENTREGUE. **Enforcement:** `CLAUDE.md §"Protocolo de git — sprints"` + `.claude/skills/agent-conductor/SKILL.md` Passo 0 + `Plano Mestre §8 Protocolo de Execução`. **Checklist de fechamento inviolavel:** STATE.md `D0xx` + `00 - Indice.md` status line + Plano Mestre §8 ENTREGUE + push `sprint/S0X` + PR para `develop`. Se qualquer item falta, a sprint NAO esta fechada. |

## Current Blockers
- Scaffold S01 precisa rodar local (go mod tidy → go build → docker compose up → go test -race) para destravar S02. Go/Docker/migrate nao estao instalados no workspace do Conductor; execucao pertence ao usuario.
- Backend ainda nao expoe `/auth/*`, `/tasks/*`, `/me/preferences` (S02+ e implementacao de Task handlers em sprints posteriores). Frontend do cockpit segue consumindo fixtures mock.
- Itens `<!-- aguarda S01 -->` no Checklist Sistema Base (CSP nonce-based, HSTS, `/auth/me` real, `MasterRoute`, scrubbing de PII no Sentry, retry exponencial, `/api/bootstrap`) passam a ser entregaveis de S02/S03.

## Lessons Learned
- Documentacao do vault pode ficar significativamente desatualizada vs codigo
- Sempre verificar estado real do codigo antes de confiar em status documentados
- Duas taxonomias paralelas no vault causam confusao — cada numero deve ter UMA pasta
- Mudanca transversal que afeta AppShell/roteamento/preferencia de usuario e Sistema Base, nao feature vertical — o numero F17 e preservado apenas como apontador de backlog

## Preferences
- CTO padrao: world-class, dark-first, Go backend, seguranca desde commit 1
- Vault e fonte de verdade para decisoes e specs
- Agentes usados proativamente, com briefings densos e autossuficientes
- Uma feature por vez (regra cardinal) — extensoes de Sistema Base precedem features verticais
