---
tags:
  - moc
  - indice
  - raiz
created: 2026-04-15
last_updated: 2026-04-22 (S52 — FASE G CONCLUÍDA, roadmap S30-S52 completo, gate go-to-prod aberto)
status: vivo
---

# 00 — Indice

**Visao.** Tornar o Torque o CRM brasileiro de referencia para times comerciais que vendem via WhatsApp, com IA embarcada e operacao multi-canal unificada.
**Missao.** Construir um SaaS B2B multi-tenant world-class, auditavel linha a linha, onde cada decisao tecnica, de produto e de design sustenta escala de decadas.

---

## Navegacao

### 01 - Produto
- [[Visao do Produto]]
- [[Personas e ICP]]
- [[Glossario]]
- [[Principios do Sistema]]
- [[Regras de Negocio Globais]]
- [[Validacoes e Invariantes]]
- [[Gotchas Conhecidos]]
- [[Decisoes de Design]]

### 02 - Arquitetura
- [[Arquitetura do Sistema]]
- [[Visao Geral Conceitual]]
- [[Camadas do Sistema]]
- [[Autenticacao e Autorizacao]]
- [[Autenticacao - Mecanismos]]
- [[Multi-tenancy]]
- [[Seguranca Web]]
- [[Seguranca - Estrategia]]
- [[Contratos e Boundaries]]
- [[Realtime e Jobs]]
- [[Requisitos Nao-Funcionais]]
- [[Padroes Transversais]]
- Permissoes/
  - [[Modelo de Permissoes]]
  - [[Papeis]]
  - [[Permissoes por Feature]]
  - [[Isolamento Multi-tenant]]

### 03 - Modelo de Dominio
- [[Entidades Principais]]
- [[Lead]], [[Organizacao]], [[Pipeline]], [[Conversa e Mensagem]]
- [[Workflow]], [[Campanha]], [[Agente IA]], [[Produto]]
- [[Time de Vendas]], [[Tag]], [[Auditoria]]

### 04 - Design
- [[Principios de Identidade Visual]]
- [[Design System Base]]
- [[Tipografia]]
- [[Motion e Animacao]]
- [[Componentes Primitivos]]
- [[Vocabulario de UI]]
- [[Criterios de Reprovacao]]
- [[Direcao Visual Frontend]]
- [[Exploracao Visual Alternativa]]

### 05 - Sistema Base
- [[Escopo do Sistema Base]]
- [[Checklist Sistema Base]]
- [[Estrutura de Pastas]]
- [[Plano de Execucao]]
- [[Plano de Execucao Granular]]
- [[Spec - Redesign Sistema Base]]
- [[Analise Pratica]]
- [[Revisao Final - Redesign Sistema Base]]
- [[Pendencias e Lacunas]]
- [[Log Implementacao Sistema Base]]
- [[UI Modes - Vendedor e Gerente]] — extensao (ADR-007): toggle Vendedor/Gerente, rota /cockpit, entidade Task, claymorphism escopado

### 06 - Funcionalidades
- Vendas/ — [[Funis Hub]], [[Gestao de Leads]], [[Pipeline WhatsApp (Qualificacao)]], [[Pipeline Confirmacao]], [[Pipeline Propostas]], [[Follow-ups]], [[Pipelines Customizados]], [[Produtos]], [[Upsell]]
- Comunicacao/ — [[Chat Multi-canal]], [[Templates de Mensagem]], [[Mensagens Agendadas]], [[Notas Internas]]
- Automacao/ — [[Workflow Builder]], [[Campanhas]], [[Regras de Pipe]]
- IA/ — [[Copilot (Agentes IA)]], [[Lead Score]], [[Oraculo Comercial]]
- Equipe/ — [[Gestao de Time]], [[Comissoes]], [[Metas]], [[Premiacoes]]
- Analytics/ — [[Dashboard Principal]], [[Analytics Comercial]], [[Analytics de UTMs]], [[Dashboard Outbound]], [[Performance Individual]], [[Ranking]], [[TV Dashboard]]
- Admin/ — [[Configuracoes]], [[Checkout e Planos]], [[Onboarding de Organizacao]], [[Master Admin]], [[Permissoes do Sistema]], [[Webhooks]], [[API Docs]]
- Fluxos End-to-End/ — [[Lifecycle de um Lead]], [[Fluxo de Qualificacao]], [[Agendamento e Confirmacao]], [[Proposta → Fechamento]], [[Campanha Outbound]], [[Atendimento via Copilot]], [[Ingestao de Leads (Webhook)]]

### 07 - Features
- [[00 - Mapa de Features]]
- [[F01 - Funis Hub e Pipe WhatsApp]] (vertical slice)
- [[F17 - Modo Vendedor (Task Cockpit)/Spec|F17 - Modo Vendedor (Task Cockpit)]] — apontador; documento operacional e [[UI Modes - Vendedor e Gerente]]
- [[Template - Feature Spec]]

### 08 - Decisoes
- [[ADR-001-contratos-openapi-snake-camel]]
- [[ADR-002-realtime-websocket-go-hub]]
- [[ADR-003-auth-httponly-cookies-samesite]]
- [[ADR-004-paginacao-cursor-based]]
- [[ADR-005-tipografia-self-hosted]]
- [[ADR-006-jobs-assincronos-202-poll]]
- [[ADR-007-modo-vendedor-gerente]] — modos de UI Vendedor/Gerente; Task unifica Follow-up; rota /cockpit; claymorphism escopado

### 09 - Backlog
- [[Plano Mestre de Finalizacao do SaaS CRM]]
- [[Backlog Priorizado]]
- [[Quick Wins]]
- [[Riscos e Duvidas]]
- [[Analise Comparativa v8 vs Torque-v2]] — gap analysis completa (features, qualidade, testes, escalabilidade) com score composto e backlog P0-P3
- [[Plano de Acao Paridade v8 - Sprints S30-S52]] — 23 sprints em 7 fases (A-G) para fechar o gap funcional; ~28 semanas; mantém ADRs e topologia linear cumulativa
- [[Auditoria Pre-Fase-C - Scored 2026-04-20]] — score composto 84/100 após Fase B (segurança 92 / testes 58 / arquitetura 91 / multi-tenancy 97 / eficiência 80 / DX 94); veredito: apto a F06 Copilot

### 10 - Referencias
- Integracoes/ — [[WhatsApp (Evolution API)]], [[Meta (Facebook Ads e Messenger)]], [[SZ.Chat]], [[Google Calendar]], [[TinyERP]], [[Asaas (Provedor de Pagamento)]], [[n8n (Orquestrador Externo)]], [[Modelo LLM Generativo]], [[Embeddings Vetoriais]], [[Text-to-Speech]], [[Observabilidade (Sentry)]]
- Processos Assincronos/ — [[Distribuicao de Leads]], [[Execucao de Workflows]], [[Fila de Webhooks]], [[Jobs Recorrentes (cron)]], [[Processamento de Mensagens Outbound]]
- [[Catalogo de APIs Externas]], [[Catalogo de Eventos]], [[Estados e Maquinas de Estado]], [[Quotas e Limites]]

### 11 - Operacional
- [[Fluxo de Trabalho]]
- [[Uso de Agentes]]
- [[Observabilidade e Logs]]

### 12 - Migracao do Legado
- [[Mapa do Vault Legado]]
- [[Top 15 Docs Prioritarios]]
- [[Itens Nao Migrados]]

### Agentes
- [[Conductor]], [[Architect]], [[Backend]], [[Frontend]], [[DBA]], [[QA]], [[Infra]], [[Automation]], [[AI]]

---

## Status atual

**Etapa:** **Fase G CONCLUÍDA — S52 ✅ (2026-04-22)** G.2 hardening produção FINAL: supply chain signing + OpenAPI refresh + security gates + load test + pentest runbook. **release.yml**: `id-token: write` + sigstore/cosign-installer@v3 v2.4.1 + `cosign sign` por digest (imutável) em ambas as imagens (api + web) após push no GHCR. **deploy.yml**: novo job `verify` com `cosign verify --certificate-identity-regexp` pinado em `release.yml@refs/tags/<tag>` + `--certificate-oidc-issuer token.actions.githubusercontent.com`; migrate + deploy `needs: [verify]` — fork compromised não produz signature que passa no gate. **Makefile**: novo target `make security` (go vet + gosec -severity=high + govulncheck). **k6**: `/quotas` + `/integrations` no dashboard batch + 10% Meta Ads insights (cached 200 ou 412) + 5% POST /leads medindo admission middleware cost + trends separados por recurso. **Pentest runbook** (`.specs/security/pentest-runbook-S52.md`, ~350 linhas): 10 checks executáveis com receitas curl+openssl — rate limiter burst (A05), cross-tenant leak → 404 não 403 (A01), JWT tampering 3 variantes (A02/A07), CSRF missing+mismatch (A07), webhook replay idempotente (A08), **lead webhook HMAC S50** (sig válida → 201 → replay 200 → tamper 401 → prefix downgrade 401), **quota enforcement S51** (402 QUOTA_EXCEEDED com X-Quota-* headers), master impersonation **audit-first invariant D062** (audit write fail → 500 AUDIT_FAILED atomically), WS origin reject, frontend XSS bundle scan. Pentest findings template pra sign-off dual reviewer+QA. **OpenAPI spec refresh**: version 0.1.0 → **0.52.0**, 341 → **1709 linhas**, ~13 → **~80 paths** cobrindo todas as famílias S05-S51 (leads + pipes + confirmations + meetings + tasks + inbox + templates + campaigns + workflows + agents + billing + quotas + integrations + webhooks + analytics + performance + products + members + proposals + settings + onboarding + master). Hot-path schemas completos (Lead/LeadPage/LeadCreateRequest/LeadPatchRequest/Subscription/CheckoutRequest/Quota/IntegrationCredential/MetaAdAccountInsights/LeadWebhookPayload) + responses QuotaExceeded 402 canônico com headers + Unauthenticated/InvalidBody/NotFound/InvalidCredentials refs. SSE + audio/mpeg + OAuth 302 callback + CSRF-exempt subgroup documentados. **Tests**: zero regressão frontend 296/296 em 86 files; typecheck + eslint verdes. Backend tests não tocados (S52 é infra/spec/ops). **FASE G CONCLUÍDA** (S51→S52). **ROADMAP S30-S52 COMPLETO** — 23 sprints em 7 fases (A Foundation + B UX + C IA + D Automação + E Produto + F Integrações + G Hardening). **Gate Go-to-Production aberto** conforme `Plano Paridade v8 §10`: paridade ≥95% vs v8, cobertura vitest ≥70% (atingido), cobertura Go ≥75% (pending runtime), zero HIGH gosec 7d, k6 p95<500ms staging, pentest clean dual sign, dual-review money-flow, runbook dry-run. Deferreds pós-prod: impersonation cookie swap atômico, openapi-typescript regen, deep schemas nas paths minimal, k6 + pentest execution contra staging real. Commits: `85f2df9` infra · `30d9aa5` openapi. Próxima: **pós-roadmap — operação contínua + release cadence**. **Fase G em andamento — S51 ✅ (2026-04-22)** G.1 Asaas real provider + dual-review tripwire + quota enforcement runtime. **Migration 0026**: `plan_quotas` catalog (free/growth/enterprise × leads/team_members/workflows/agents) + backfill transacional de `org_quotas` pra tenants existentes (live subscription → terminal mais recente → free fallback) + permission seed `quotas.view`. **AsaasProvider real** implementa `billing.Provider` contra Asaas v3 REST (PIX MVP; boleto+cartão follow-up). `NewAsaas` **tripwire dual-review**: retorna error em APIKey vazia, main.go cai no mock mesmo com BILLING_PROVIDER=asaas. CreateCharge sequencia customer (dedup por externalReference=org_id na Asaas side) → payment (billingType=PIX, dueDate=+30min) → pixQrCode. CancelCharge com 404-as-success idempotente. Taxonomy 401/429/5xx/404. `NormalizeAsaasWebhook` converte payload Asaas → shape genérica (PAYMENT_CONFIRMED→charge.paid, OVERDUE→overdue, DELETED+REFUNDED→cancelled, unknown→asaas.<event> prefixado pra audit log); event_id composto EVENT:payment.id pra dedup correto entre retries. **Quota infra** nova: (1) `repository/quota` com Get/List/IncrementUsage (GREATEST clamp em decrement)/SeedPlanDefaults/SetAdminAdjustment/SetPurchasedAddons + constants Resource{Leads,TeamMembers,Workflows,Agents}. (2) `middleware/quota.RequireQuota(reader, resource)` factory panic em resource empty; runtime 402 QUOTA_EXCEEDED com body JSON canônico + headers X-Quota-{Resource,Limit,Usage,Remaining} em admit+deny pro upsell banner; master bypass; fail-closed em ErrNotFound (missing row=402, força provisioning explícito). (3) `handler/quotas`: GET /quotas + GET /quotas/:resource member + PATCH /quotas/:resource master (admin_adjustment + purchased_addons). **Wiring leads**: `leadshandler.WithQuota(quotaRepo)` encadeável; Routes() condicionalmente aplica RequireQuota só no POST; create chama IncrementUsage(+1) pós-success, softDelete chama IncrementUsage(-1). Pattern drop-in pra team_members/workflows/agents em sprint futura. **Billing webhook upgrade**: `WithQuotaRepo(quotaRepo)` seeda plan_quotas→org_quotas em charge.paid pra tenant upgrading do plano aplicar novo effective_limit imediatamente; novo POST /webhooks/billing/asaas normaliza payload nativo Asaas antes de forwardar pra state machine genérica. **Config**: ASAAS_API_KEY, ASAAS_BASE_URL (default sandbox). **main.go**: quotaRepo shared entre leads/quotas/billing-webhook; AsaasProvider construído quando APIKey setada com fallback mock em erro. **Frontend**: useQuotas + useQuota hooks; QuotaMeter 3-tone progress (≤80 neutral / 80-99 warning / 100 danger) com ARIA meter role + clamp 100; FunisHubPage header surfaca leads quota. api/errors: QUOTA_EXCEEDED → "Atualize em Configurações → Plano e faturamento". **Tests backend 20 funcs**: asaas_test 13 (dual-review tripwire, CreateCharge multi-hop, non-BRL+zero rejects, Cancel 404/5xx, 401/429 taxonomy, Normalize mapping + unknown pass-through + empty-id + garbage-json + event_id composed). quota_test 7 (admits+headers, 402 at cap, fail-closed, master bypass, unauth, missing-tenant, factory panic). **Tests frontend 9**: useQuotas 3 (list + detail + URL-encoding) + QuotaMeter 6 (undefined + unconfigured + pct+copy + clamp + ARIA + custom label). **Full suite 296/296 em 86 files** (+9 cases +2 files vs S50); typecheck + lint --max-warnings 0 verde. Deferreds: team_members/workflows/agents wiring (pattern pronto), boleto+cartão Asaas. Commits: `8209b44` db · `e7be776` backend · `3156135` tests · `cddaae7` frontend. Próxima: **S52 — OpenAPI refresh + cosign keyless + gosec high + pentest staging + k6 load test (FASE G FINAL)**. **Fase F CONCLUÍDA — S50 ✅ (2026-04-22)** F.2 Meta Ads Insights + Lead Webhook + SZ.Chat + TinyERP sync + F02→GCal. **Migration 0025**: `lead_webhook_events` (UNIQUE org+external_id absorve retries), `meta_insights_cache` (TTL 15min at query time), provider CHECK + `szchat`. **Meta provider** (InsightsProvider novo): Graph API 2 GETs (account + campaign), appsecret_proof HMAC opcional, códigos 190/102→ErrAuthFailed, 17/4/613→ErrRateLimited, decimalToCents BR+US, sumLeadActions case-insensitive, cpLead=-1 pra leads=0. **SZ.Chat** (MessagingProvider): Bearer + POST /messages + normalizeKind. **TinyERP SyncProducts**: paginado /produtos.pesquisa.php + sink UpsertBySKU narrow interface; productrepo.UpsertBySKU ON CONFLICT (org,sku) WHERE sku IS NOT NULL + xmax=0 pra distinguir insert vs update; result `{fetched, inserted, updated, skipped}`. **Lead webhook público** POST /webhooks/lead sem Authenticator/CSRF; HMAC-SHA256 em X-Torque-Lead-Signature: sha256=<hex> (constant-time, reject unprefixed downgrade), X-Torque-Tenant: <uuid>, body cap 64KiB, empty secret=503 secure default, UNIQUE(org,external_id) dedup→200 idempotente, success→leadrepo.Create + publish(lead.created)→workflow BusSubscriber dispara lead_created trigger→201 {lead_id, external_id}. **F02→GCal** (deferred S49 closure): confirmations.WithIntegrations(gcal, store, logger); snapshot pre-MarkConfirmed + goroutine 20s cria evento "Reuniao confirmada — <channel>" + default 30min. **IntegrationsSection rich** (extraído de SettingsPage para features/settings/IntegrationsSection.tsx): cards Google/TinyERP/Meta/SZ.Chat com last_success_at relative time pt-BR (agora/min/h/d/absolute) + last_error warning badge; Sheet connect modals (TinyERP api_key / Meta access_token+account_id / SZ.Chat api_key+channel_id); TinyERP sync action inline; a11y htmlFor+useId; exactOptionalPropertyTypes-safe. **Hooks novos**: useConnectMeta/Disconnect, useConnectSZChat/Disconnect, useSyncTinyERPProducts, useMetaAdsInsights. **Config**: META_GRAPH_BASE_URL, META_APP_SECRET, SZCHAT_BASE_URL, LEAD_WEBHOOK_SECRET. **main.go**: providers nil-guarded; productSyncSink adapter bridgeia productrepo↔tinyerp sem cyclic import; /webhooks/lead mounted outside /api/v1; confirmations wired. **Tests backend 31 funcs novas** (meta 10 + szchat 7 + tinyerp sync 7 + leadwebhook 6 + MockInsights 1). **Tests frontend 287/287 em 84 files** (+13 cases, +1 file vs S49); tsc --noEmit + eslint --max-warnings 0 verde. Deferreds: SZ.Chat inbound webhook (outbound-first release); Meta Lead Ads relay-to-webhook orchestration (pull-based insights no release inicial). Commits: `092e397` db · `1f40812` backend · `3a77357` tests · `191d25a` frontend. **FASE F CONCLUÍDA** (S49→S50). Próxima fase: **G — Hardening produção (S51-S52)** Asaas real + dual-review + quota enforcement + OpenAPI refresh + cosign + gosec + pentest staging + k6 load test. **Fase F em andamento — S49 ✅ (2026-04-20)** F.1 Google Calendar real + TinyERP foundation. Migration 0024 `integration_credentials` com AES-256-GCM encrypted tokens (chave em INTEGRATION_ENCRYPTION_KEY env) + observability fields (last_success_at/last_error_*, external_account_id, scopes) + provider CHECK IN ('google','tinyerp','meta'). Crypto package minimal com ErrDecrypt opaco + nonce random por chamada. Credential store com WHERE organization_id = $1 invariante + encrypt-on-upsert + decrypt-on-get. GCal adapter real com DefaultEndpoints production + WithEndpoints escape-hatch pra httptest; OAuth 2.0 AuthURL access_type=offline+prompt=consent; ExchangeCode/RefreshToken com 401→ErrAuthFailed/5xx→ErrUnreachable; CreateMeetingForOrg com refresh auto se expires<now+2min + POST events Bearer + htmlLink; Disconnect revoke best-effort. TinyERP adapter com token-param POST pedido.incluir.php + envelope parsing codigo_erro 6/30. Integrations handler state HMAC TTL 10min ConstantTimeCompare rotas member list + admin OAuth connect/callback/disconnect + tinyerp connect/disconnect/push-order; callback CSRF-exempt subgrupo (OAuth 302 não carrega X-CSRF-Token). domain.WithOrgID/OrgIDFrom novo carrier pra goroutine background (middleware key unexported). Meetings handler WithIntegrations opcional: goroutine 20s ctx bg cria evento GCal real em POST /meetings + UPDATE external_provider/external_id; DELETE espelha com CancelMeetingForOrg. Config: INTEGRATION_ENCRYPTION_KEY (fatal non-dev, dev fallback sync.Once WARN) + GOOGLE_OAUTH_* + TINYERP_BASE_URL + INTEGRATION_STATE_SECRET fallback JWT. Frontend useIntegrations hook + SettingsPage Integrações tab wired real: card Google Conectar/Desconectar (full-page nav pra /api/v1/integrations/google/connect), card TinyERP "em breve" até S50, ?tab= query param pra preselecionar seção no callback. Tests backend 36 funcs (crypto 7 + gcal 10 + tinyerp 8 + credential 2 gated + handler 9). Tests frontend 274/274 em 83 files (+5 cases +1 file vs D068) coverage 78.48/75.30/74.75/62.22 acima ratchet 70/70/70/60. Deferred: sync-products, confirmation F02→gcal, Meta Ads, SZ.Chat, Lead Webhook, IntegrationsSection rich UI → S50. Próxima: **S50 — F.2 Meta Ads Insights + Lead Webhook + SZ.Chat + IntegrationsSection rich**. **Remediation-C ✅ (2026-04-20)** lift de cobertura pré-Fase F. 15 test files novos (5 pages: CampaignDetail/AgentMetrics/CustomPipe/TVDashboard/UpsellPage + 6 hooks extras: useAnalytics/useCampaigns/useInbox/useMeetings/usePerformance/useWorkflows + 2 lib: ws/utils). Vitest ratchet step 4: 55/55/50/48 → **70/70/70/60**. Medido (live): lines **78.21** / stmts **75.02** / funcs **74.20** / branches **62.22**. **269/269 em 82 files** (+74 cases, +14 files vs S48). Gitignore ajustado (*.tsbuildinfo + .claude/scheduled_tasks.lock). Score testes 72→88/100 (+16pp); composto 87→89/100 (+2pp). Próxima fase: **F — Integrações externas (S49-S50)** Google Calendar real + TinyERP + Meta Ads + Lead Webhook + SZ.Chat. **Fase E concluída — S48 ✅ (2026-04-20)** F12 Pipes custom + Upsell + F13 Agenda + TV Dashboard. Migration 0023 meetings + seeds meetings.view/manage. Hooks useMeetings + AgendaPage /agenda (lista agrupada pt-BR + create modal). CustomPipePage /pipe/:id reusa hooks de pipe com board horizontal. UpsellPage /upsell com grid de leads recentes. TVDashboardPage /tv fora do AppShell com setInterval 20s + 3 widgets rotating reusando Analytics+Performance. Routes /agenda, /upsell, /pipe/:id no AppShell; /tv top-level. Tests AgendaPage 2 cenários. 195/195 em 68 files. **Fase E completa** (S46→S47→S48): F08 Campanhas wizard → F09 Performance 4 tabs → F12+F13 Pipes/Upsell/Agenda/TV. Próxima fase: **F — Integrações externas (S49-S50)** Google Calendar + TinyERP + Meta Ads + Lead Webhook. **S47 ✅** F09 Performance. **S46 ✅** F08 Campanhas UI. Migration 0022: goals + commissions + awards + seeds performance.view/manage. Repo com Ranking LEFT JOIN proposals won (drift-free) + CRUD + SetCommissionStatus com COALESCE stamps. Handler split Routes (member) / AdminRoutes (admin). Frontend usePerformance hooks + PerformancePage 4 tabs (Ranking picker 7/30/90d + BRL bars; Metas período; Comissões totalizadores + lista status badges; Premiações). Tests 2 cenários. Deferred: progresso % metas + comissão automática. Próxima: S48 (F12 Pipes custom + Upsell + Agenda + TV Dashboard). **S46 ✅** F08.1 Campanhas UI. CampaignsPage com 2 tabs (Em andamento/Arquivadas partition por status) + grid cards com stats counters. CreateCampaignModal wizard 3 steps (identidade/audiência DSL reusando TriggerFilter S40/schedule). CampaignDetailPage /campaigns/:id com status+stats+recipients+ações contextuais (pause/resume/cancel). Routes /campaigns/:id lazy. Tests 2 cenários. Próxima: S47 Performance. **Fase D concluída — S45 ✅** F07.3 Workflow actions extras + Execuções UI + debug run. Backend: 3 novos handlers (CreateTask, CallAgent, HTTPRequest com validateHTTPURL https-only) + NewDispatcherS45 factory registra 7 kinds; eventTriggerMap estendido (lead.stage_changed, message.received); GET /runs/:id/steps novo. Frontend: useWorkflowRunSteps (polling 2s), WorkflowExecutionsPage em /workflows/:id/executions com 2-column list+timeline + "Rodar em debug" enfileira run debug; link Execuções no canvas. 191/191 em 66 files, coverage lines 64.2%. **Fase D completa** (S43→S44→S45): canvas xyflow + executor + actions + execuções UI. Deferreds: handlers side-effect reais, retry/DLQ, schedule trigger (cron), canvas debug inline. Próxima fase: **E — F08 Campanhas + F09 Performance + F12 Pipes custom (S46-S48)**. **S44 ✅** F07.2 executor. **S43 ✅** F07.1 canvas. Repo executor_queries.go com ClaimPendingRun (SKIP LOCKED), AppendRunStep/CompleteRunStep, ListActiveWorkflowsByTrigger. Service workflow com Dispatcher (registry + 4 handlers stubs) + Executor (DAG walk com MaxStepsPerRun=100) + Runner goroutine + BusSubscriber (lead.created → lead_created). Branch evaluator minimal com ==/!= + lead.*/input.*/prev.<stepId>.* dotted paths. Wired em main.go. Tests dispatcher_test.go 11 cenários. **Deferred honesto**: handlers side-effects reais (Evolution/lead repo) + retry/DLQ + triggers stage_changed/message.received/schedule para S45. Próxima: S45. **S43 ✅** F07.1 canvas xyflow + list. Dep @xyflow/react v12.10.2. WorkflowListPage substitui seed com useWorkflows live. WorkflowCanvasPage /workflows/:id com palette draggable (6 node types), ReactFlow screenToFlowPosition, onConnect merge next_step_ids, onNodeDragStop PUT position, inspector JSON config. Routes /workflows list + /workflows/:id canvas. 189/189 em 65 files, coverage lines 63.33%. Próxima: S44 executor worker. **Fase C concluída — S42 ✅** F06.6 Copilot Metrics page. Backend GetAgentMetrics com state breakdown + message aggregates (tokens in/out/avg latency filtered assistant). Handler GET /agents/:id/metrics com RFC3339 window + 400 codes (INVALID_SINCE/UNTIL/WINDOW, WINDOW_TOO_LARGE > 365d). Frontend: AgentMetricsPage em /copilot/:id/metrics com picker 7d/30d/90d + 4 KPI cards (pt-BR) + stacked sessions-by-state bar. 187/187 em 64 files, coverage lines 63.21%. **Fase C completa** (S37→S38→S39→S40→S41→S42): OpenRouter+SSE / Playground UI / RAG pgvector+Gemini / Triggers matcher / TTS ElevenLabs / Metrics. Frontend +36 cases, +13 files desde início da fase; coverage 59.17 → 63.21 (+4.04pp). Próxima fase: **D — F07 Workflow Builder (S43-S45)**. **S41 ✅** F06.5 TTS. Migration 0021 agents.tts_enabled + tts_voice_id. Service ai/tts.go com ElevenLabsTTS (POST /v1/text-to-speech/:voice_id + xi-api-key + audio/mpeg, 5000-char/5MB caps, taxonomia 401/429/5xx/4xx) + MockTTS deterministic fallback. Config ELEVENLABS_BASE_URL|API_KEY|MODEL_ID (default eleven_multilingual_v2). Handler: novo POST /agents/:id/tts/preview (sem storage, mp3 inline). Frontend: TTSPanel no editor com checkbox + voice_id + "Ouvir preview" + inline player (objectURL cleanup no unmount). 185/185 em 63 files, coverage lines 63.08%. Outbound pipeline (worker message.outbound → S3 → kind=audio + org_quotas) adiado para sprint futura. Próxima: S42 Metrics. **S40 ✅** F06.4 Copilot triggers. Migration 0020 agent_triggers + conversations.assigned_agent_id + seeds triggers.view|manage. Matcher puro (ops eq/neq/contains/in/present/absent + all/any + kill-switch/status guards + unknown-op fail-closed). Repo CRUD + ListActiveTriggers (JOIN agents) + AssignAgent. Handlers GET/POST /agents/:id/triggers + PATCH/DELETE /triggers/:tid + WS events agent_trigger.{created,updated,deleted}. Frontend: hooks + TriggersPanel no editor (form inline com field/op/value/priority, rule list com toggle/trash). Coverage ratchet 60/60/55/50, medido lines 62.71/stmts 60.22/funcs 57.23/branches 52.01. 183/183 em 62 files. Worker dispatch diferido para Fase D (reusa workflow executor). Próxima: S41 TTS ElevenLabs. **S39 ✅** F06.3 Copilot RAG. Migration 0019 pgvector + knowledge_chunks embedding vector(768) + HNSW cosine + agents.knowledge_collection_id FK + seeds knowledge.view|manage. Docker-compose dev switched to pgvector/pgvector:pg15. Backend: Embedder interface (Gemini real + Mock fallback), chunker paragraph-aware 500/50, repo ListCollections/InsertChunks/SimilaritySearch com encodeVector, Ingest service goroutine detached com status transitions. Playground handler embeda último user turn + topK=5 + prependContext delimitado. Frontend: hooks knowledge + KnowledgePanel no editor (select + create inline + sources list + ingest form). 178/178 em 61 files, coverage lines 62.23%. Próxima: S40. **S38 ✅** F06.2 Copilot Playground UI. AgentListPage real substitui mockup seed; AgentPlaygroundPage (`/copilot/:id`) layout Vercel — editor lateral (system_prompt, model, temp, max_tokens) com dirty detection + chat SSE com Parar/Regenerar. Hook `useAgentStream` consome `fetch` + `ReadableStream.getReader()` (EventSource não suporta CSRF); parser `\n\n` com frames delta/done/error; AbortController limpo no unmount. Backend: `agentView` agora expõe `system_prompt` via `toAgentView` helper unificado. Tests: useAgentStream 4 cenários + AgentListPage 2 cenários. **173/173 em 60 files**, coverage lines **61.56%** (+2.39pp). Próxima: S39 (Agent Wizard 20+ steps + session persistence). S37 ✅ F06.1 OpenRouter + Playground SSE backend. Remediation-B ✅ (84→87/100). Fase B concluída. S00-S29 ✅ — Roadmap original de 30 sprints COMPLETO. S29 entregou security audit OWASP Top 10, k6 load test, runbooks operacionais e security-scan.yml.

**S00** — ESLint flat (a11y + hooks + ban `dangerouslySetInnerHTML`), TS strict, Prettier, Vitest (51 testes), CI, vocabulario canonico, `--clay-*` → `--card-*`/`.tactile-*`, pos-login navigation por `ui_mode`.

**S01** — `torque-api/` chi + pgx v5 + zerolog + probes + middleware stack com `StripOrganizationID`, migrations 0001–0003, Docker distroless nonroot, teste de concorrencia do partial index de Task.

**S02** — Migration 0004 `refresh_tokens`. Services password/jwt/token/permission. Repositories user+refresh (SERIALIZABLE + RevokeChain via recursive CTE). Middleware Authenticator/RequireAuth/TenantScope/CSRF/RequireFeature/RequireRole/RequireMaster. Handlers `/api/v1/auth/{login,refresh,logout,me}`, `/api/v1/me/preferences`, `/api/bootstrap`.

**S03** — Security headers parametrizados (CSP tight, HSTS 2yr preload em non-dev, COOP/CORP). Sentry SDK com PII scrubbing. Audit service/repo usando `audit_log` existente. Rate limiter in-memory. Bootstrap enriquecido.

**S04** — Migration 0005 `operations` (ENUM + SKIP LOCKED claim + retry chain). Event bus in-process. WS hub `ws.Hub` tenant-scoped via `coder/websocket`. Worker pool goroutine+semaphore. Handlers `POST/GET/DELETE /api/v1/operations` + `GET /api/v1/ws`. `/openapi.{yaml,json}` publicos.

**S05** — queryKeys + errors pipeline + hooks (useAppMutation, useInfiniteList, useWSSubscribe, useOperation*, useBootstrap) + QueryBoundary + vitest coverage.

**S06** — AuthProvider real via `/api/v1/auth/me`, WSProvider conectando `ws_url`, `useLogin` + LoginPage inline error.

**S07** — Repositorios `lead` (cursor ADR-004 + search + soft-delete) e `pipe` (Move atomico Serializable). Handlers `/api/v1/leads` CRUD + `/api/v1/pipes` list/stages/entries/move. Eventos `lead.{created,updated,deleted}` e `pipe_entry.moved` no bus.

**S16–S20** — F06 Copilot frontend hooks, F07 Workflow backend+frontend, F08 Campanhas backend+frontend, F09 Analytics read-only summaries. Detalhes em STATE.md D030–D034.

**S21** — F10 Equipe + F11 Produtos. Migration 0014 (products + seeds members/products perms). Backend split Read/Admin (RequireRole). Self-demote/self-deactivate refused. Frontend: SettingsPage Equipe com dados reais, ProductsPage nova. Detalhes em STATE.md D036.

Próxima sprint: **S22** — F12 Propostas integradas com produtos + Follow-up UI.

---

## Fluxo obrigatorio

Toda mudanca relevante segue, sem excecao, os 8 passos:

1. **Entender** — ler o contexto, o codigo existente, as decisoes passadas.
2. **Documentar** — registrar o entendimento atual em nota do vault.
3. **Propor** — escrever a proposta tecnica (arquitetura, contratos, impactos).
4. **Validar** — submeter a revisao do fundador antes de qualquer codigo.
5. **Executar** — implementar exatamente o que foi validado.
6. **Revisar** — passar por `/hm-engineer`, `/hm-designer`, `/hm-qa` conforme a camada.
7. **Documentar resultado** — atualizar a nota com o que foi entregue de fato, divergencias e consequencias.
8. **Avancar** — so entao pegar o proximo item.

Pular passos nao e velocidade. E divida.

---

## Regra de ouro

**Sistema Base vs Features.** O Sistema Base e a fundacao multi-tenant, invariante e compartilhada (auth, orgs, roles, billing, seats, planos, flags, audit, observabilidade). Features sao verticais de produto que rodam sobre essa fundacao.

**Uma feature por vez.** Nenhuma feature comeca antes de o Sistema Base correspondente estar pronto, testado e documentado. Nenhuma segunda feature comeca antes de a primeira estar world-class — nao "funcionando", world-class. Paralelismo em feature e o caminho mais curto pra um produto mediano em tudo.

---

## Estrutura do vault

```
00 - Indice.md              Navegacao central e status
01 - Produto/               Visao, personas, glossario, principios, regras de negocio
02 - Arquitetura/            Arquitetura tecnica, auth, tenancy, seguranca, permissoes
03 - Modelo de Dominio/      Entidades de dominio (Lead, Pipeline, Workflow, etc.)
04 - Design/                 Design system, tipografia, motion, componentes, criterios
05 - Sistema Base/           Fundacao frontend: spec, checklist, plano, auditoria
06 - Funcionalidades/        Specs de features por dominio (Vendas, IA, Equipe, etc.)
07 - Features/               Mapa de features e specs de execucao (F01, F02, etc.)
08 - Decisoes/               ADRs (Architecture Decision Records)
09 - Backlog/                Plano mestre, backlog priorizado, riscos
10 - Referencias/            Integracoes externas, processos assincronos, catalogos
11 - Operacional/            Fluxo de trabalho, agentes, observabilidade
12 - Migracao do Legado/     Referencia ao vault legado, docs prioritarios, exclusoes
Agentes/                     Perfis dos 9 agentes especializados
```

Cada numero tem exatamente UMA pasta. Sem dualidades. Sem ambiguidades.
