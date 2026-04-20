---
tags:
  - backlog
  - analise
  - comparativa
  - v8
  - torque-v2
created: 2026-04-20
last_updated: 2026-04-20
status: vivo
---

# Análise Comparativa v8milennialsb2bv2 vs Torque-v2

> Gerado por agent-conductor em 2026-04-20 com 3 sub-agentes Explore em modo very thorough. Evidências citadas são `arquivo:linha` lidos diretamente em ambos os projetos.

---

## Sumário Executivo

1. **v8 é produto maduro com 47 páginas + 142 edge functions + 325 migrations + ~73.5% cobertura**; Torque-v2 é rebuild greenfield com 15 páginas reais + 17 migrations + 24 handlers Go + ~22% cobertura por arquivo.
2. **Torque-v2 vence em arquitetura e segurança** (7 ADRs cumpridos: httpOnly cookies, cursor-based pagination, CSP tight, StripOrganizationID, WebSocket hub tenant-scoped, PII scrubbing). v8 armazena sessão em `localStorage` (XSS-vulnerável) e tem 7 god components (>800 linhas).
3. **v8 vence em completude de features** por larga margem: tem Chat WhatsApp em tempo real, Copilot com wizard+playground+RAG+TTS, Automações visuais via @xyflow com 30 action types, TinyERP/Google Calendar/Meta Ads integrados, TV Dashboard, Ranking/Metas/Comissões completos.
4. **Torque-v2 tem infraestrutura superior** (CI/CD completo com 4 workflows, security-scan diário, k6 load test, runbooks operacionais) e documentação viva (44 decisões em STATE.md, vault Obsidian sincronizado).
5. **Veredito: Torque-v2 tem melhor fundação, pior completude.** Para atingir paridade funcional com v8 são ~26-34 semanas de desenvolvimento dedicado. Nada do v8 precisa ser portado antes de continuar — a fundação do Torque-v2 é superior e migrar código v8 violaria ADRs centrais (ex.: localStorage auth, offset pagination).

---

## Relatório 1 — Gap de Funcionalidades

### 1.A Mapa de rotas (v8 completo vs Torque-v2 atual)

| # | Rota v8 | Torque-v2 | Status |
|---|---------|-----------|--------|
| 1 | `/` (RootRedirect conditional) | `/` (Dashboard) | 🔄 redesenhada |
| 2 | `/landing` | ❌ nenhuma | ❌ **AUSENTE** |
| 3 | `/auth`, `/signup`, `/reset-password` | `/login` | ⚠️ **PARCIAL** — sem signup público nem reset |
| 4 | `/privacidade`, `/docs` | ❌ nenhuma | ❌ **AUSENTE** |
| 5 | `/checkout`, `/checkout/success` | `/billing` | ⚠️ **PARCIAL** — mock provider, sem success page |
| 6 | `/onboarding` | `/onboarding` + OnboardingGate | ✅ **PRESENTE** |
| 7 | `/dashboard` | `/` (DashboardPage) | ✅ **PRESENTE** |
| 8 | `/funis` (FunisHub + permission) | `/pipeline` (KanbanPage) | ⚠️ **PARCIAL** — sem hub multi-funil no frontend |
| 9 | `/campanhas/:id` | `/campaigns` | ⚠️ **PARCIAL** — sem rota de detalhe |
| 10 | `/pipe-confirmacao` | backend `/api/v1/confirmations` | ⚠️ **PARCIAL** — backend existe, UI não |
| 11 | `/pipe-propostas` | backend `/api/v1/proposals` | ⚠️ **PARCIAL** — backend existe, UI não |
| 12 | `/pipe-whatsapp` | `/pipeline` (KanbanPage genérica) | 🔄 **REDESENHADA** — Torque-v2 fez pipe genérico |
| 13 | `/follow-ups` (Revisao.tsx) | `/follow-ups` → ComingSoonPage | ❌ **AUSENTE** (placeholder) |
| 14 | `/checklists` | ❌ nenhuma | ❌ **AUSENTE** |
| 15 | `/leads` | dentro de `/pipeline` (drawer) | 🔄 **REDESENHADA** — sem rota dedicada |
| 16 | `/performance` (1.443 LOC consolidado) | `/analytics` (331 LOC parcial) | ⚠️ **PARCIAL** — sem ranking/metas/premiações |
| 17 | `/comissoes` | ❌ nenhuma | ❌ **AUSENTE** |
| 18 | `/equipe` | `/settings` (aba Team) | 🔄 **REDESENHADA** — integrado em Settings |
| 19 | `/configuracoes` | `/settings` | ✅ **PRESENTE** (577 linhas, 8 tabs) |
| 20 | `/tv` (TVDashboard) | ❌ nenhuma | ❌ **AUSENTE** |
| 21 | `/produtos` | `/products` | ✅ **PRESENTE** (ProductsPage 297 linhas com cursor) |
| 22 | `/copilot` (Copilot.tsx) | `/copilot` (AgentsPage) | ⚠️ **PARCIAL** — sem chat UI, sem playground |
| 23 | `/copilot/metricas` | ❌ nenhuma | ❌ **AUSENTE** |
| 24 | `/copilot/novo` (Playground) | ❌ nenhuma | ❌ **AUSENTE** |
| 25 | `/copilot/:id/editar` | ❌ nenhuma | ❌ **AUSENTE** |
| 26 | `/copilot/novo-wizard` (deprecated D008 no v8) | ❌ nenhuma | ❌ **AUSENTE** (desnecessário) |
| 27 | `/copilot/teste-wizard` (dev) | ❌ nenhuma | ❌ **AUSENTE** (dev) |
| 28 | `/chat`, `/chat-whatsapp` | `/inbox` (InboxPage 370 linhas) | ⚠️ **PARCIAL** — InboxPage é lista, sem chat real-time |
| 29 | `/upsell` | ❌ nenhuma | ❌ **AUSENTE** |
| 30 | `/pipe/custom/:slug` | ❌ nenhuma | ❌ **AUSENTE** (F12 bloqueado) |
| 31 | `/agenda` (1.385 LOC) | ❌ nenhuma | ❌ **AUSENTE** |
| 32 | `/campanhas` → `/funis` redirect | `/campaigns` | ⚠️ **PARCIAL** |
| 33 | `/automacoes`, `/novo`, `/:id`, `/execucoes` | `/workflows` | ⚠️ **PARCIAL** — 1 rota vs 4 |
| 34 | `/templates` (MessageTemplates) | ❌ nenhuma | ❌ **AUSENTE** |
| 35 | `/master` (layout + 6 sub-rotas) | `/master` (MasterPage única) | ⚠️ **PARCIAL** — só Organizations + Health |
| 36 | `/master/organizations` | ✅ em MasterPage | ✅ **PRESENTE** |
| 37 | `/master/users` | ❌ | ❌ **AUSENTE** |
| 38 | `/master/plans` | ❌ | ❌ **AUSENTE** |
| 39 | `/master/features` | ❌ | ❌ **AUSENTE** |
| 40 | `/master/audit-logs` | ❌ | ❌ **AUSENTE** |
| 41 | `/master/operations` | ❌ (backend tem `/api/v1/operations`) | ⚠️ **PARCIAL** — sem UI |
| 42 | `/cockpit` | `/cockpit` (CockpitShell + CockpitView) | 🔄 **REDESENHADA** — exclusivo Torque-v2 (ADR-007) |

**Total:** v8 tem 47 rotas × Torque-v2 tem 15 rotas reais + 2 placeholders. **Cobertura funcional de rotas ≈ 32%.**

### 1.B Backend / Lógica de negócio

| Feature | v8 | Torque-v2 | Status |
|---------|-----|-----------|--------|
| Copilot wizard 20+ steps | `CopilotWizard.tsx` 1.188 LOC (deprecated D008) + Playground | `/api/v1/agents` (9 endpoints + RBAC admin-only) | ⚠️ **PARCIAL** — backend só; sem UI |
| Copilot playground | `CopilotPlayground.tsx` | ❌ | ❌ **AUSENTE** |
| Copilot métricas | `CopilotMetrics.tsx` | ❌ | ❌ **AUSENTE** |
| Copilot RAG + embeddings | `_shared/embeddings.ts` (100% coverage D027), `generate-faq-embeddings` (Gemini 2.0, 1536d + pgvector) | ❌ nenhum | ❌ **AUSENTE** |
| Copilot TTS | `tts-elevenlabs.ts` + `audio-sender.ts` (100% coverage D030/D022) | ❌ | ❌ **AUSENTE** |
| Copilot triggers ativação | `ActivationTriggersStep.tsx` | ❌ | ❌ **AUSENTE** |
| Automações visual editor | `AutomacoesEditor.tsx` + @xyflow/react + 30 action types | `/workflows` (WorkflowBuilderPage 357 linhas, sem @xyflow) | ⚠️ **PARCIAL** — backend DAG OK; UI sem canvas |
| Automações DAG executor | `workflow-executor.ts` (92.6% coverage D012) + `process-workflow-executions` | schema `workflows + workflow_nodes + workflow_edges` | ⚠️ **PARCIAL** — schema OK, executor não construído |
| Automações histórico | `AutomacoesExecucoes.tsx` | ❌ | ❌ **AUSENTE** |
| Pipelines customizados CRUD | `CustomPipeline.tsx` + `useCustomPipelines` 995 LOC | F12 marcado como **bloqueado** no vault | ❌ **AUSENTE** |
| Pipelines dispatch rules | `PipeDispatchRulesSection.tsx` 836 LOC + `pipe-rule-dispatch` edge fn | ❌ | ❌ **AUSENTE** |
| WhatsApp real-time chat | `ChatWhatsApp.tsx` 2.443 LOC + `useWhatsAppChat.ts` 1.099 LOC + Evolution API + Supabase Realtime | InboxPage 370 linhas + backend `/api/v1/conversations` | ⚠️ **PARCIAL** — lista + backend, sem chat real-time |
| WhatsApp templates | `MessageTemplates.tsx` + `/templates` | ❌ | ❌ **AUSENTE** |
| WhatsApp sender | `_shared/outbound-sender.ts` (96.77% coverage) + `send-meta-message` + TTS | `service/integration/` (mock only, S27) | ⚠️ **PARCIAL** — adapter interface OK, implementação real pendente |
| Campanhas criação | `CreateCampanhaModal.tsx` 1.163 LOC | backend `/api/v1/campaigns` (CRUD) | ⚠️ **PARCIAL** — backend OK, UI modal não existe |
| Campanhas detalhes | `CampanhaDetail.tsx` em `/campanhas/:id` | ❌ | ❌ **AUSENTE** |
| Campanhas distribuição SDR/Closer | `CampanhaDispatchRulesSection.tsx` 833 LOC + `campaign-rule-dispatch` | schema `campaign_dispatch_rules` | ⚠️ **PARCIAL** — schema OK, lógica não |
| Performance ranking | `Performance.tsx` 1.443 LOC (ranking + metas + premiações consolidados) | AnalyticsPage 331 linhas (parcial) | ❌ **AUSENTE** (ranking, metas, premiações) |
| Performance comissões | `Comissoes.tsx` dedicada | ❌ | ❌ **AUSENTE** |
| Propostas + TinyERP | `PipePropostas.tsx` 1.436 LOC + 8 edge functions TinyERP (tinyerp-connect, fetch-nfe, push-order, etc.) | backend `/api/v1/proposals` | ⚠️ **PARCIAL** — backend CRUD OK, TinyERP zero |
| Propostas confirmação de pedidos | `tinyerp-push-order` + ConfirmacaoDetailModal 890 LOC | backend `/api/v1/confirmations` (S10) | ⚠️ **PARCIAL** |
| Analytics receita | Performance.tsx + useDashboardMetrics | AnalyticsPage placeholder | ⚠️ **PARCIAL** |
| Analytics pipeline | dashboard funnel viz | AnalyticsPage + repo analytics | ⚠️ **PARCIAL** |
| Analytics equipe | Performance.tsx | ❌ | ❌ **AUSENTE** |
| Analytics UTM | schema `lead_utm_source/_medium/_campaign` | schema leads tem UTM fields (migration 0002) | ⚠️ **PARCIAL** — stored, sem UI |
| Analytics Win/Loss | useDashboardMetrics + conversion logic | ❌ | ❌ **AUSENTE** |
| Analytics forecast | `oraculo-comercial` edge fn (AI-driven) | ❌ | ❌ **AUSENTE** |
| Confirmação reagendamento | `ConfirmacaoDetailModal.tsx` 890 LOC + Google Calendar sync | ❌ | ❌ **AUSENTE** |
| Confirmação filtros | `PipeConfirmacao.tsx` | backend existe, UI não | ⚠️ **PARCIAL** |
| Checklists | `ChecklistPage.tsx` + permissão dedicada | ❌ | ❌ **AUSENTE** |
| Upsell | `Upsell.tsx` 357 LOC + `ImportUpsellClientsContent.tsx` 970 LOC + `tinyerp-push-upsell-order` | ❌ (parte de F12) | ❌ **AUSENTE** |
| Agenda | `Agenda.tsx` 1.385 LOC + react-big-calendar + 5 edge functions Google Calendar | ❌ | ❌ **AUSENTE** |
| TV Dashboard | `TVDashboard.tsx` `/tv` (sem LayoutWrapper) | ❌ | ❌ **AUSENTE** |

### 1.C Infraestrutura e sistema

| Item | v8 | Torque-v2 | Veredito |
|------|-----|-----------|----------|
| Auth login/signup/reset | Supabase Auth + `/auth` `/signup` `/reset-password` | `/api/v1/auth/login`+`/refresh`+`/logout` + LoginPage | 🔄 **REDESENHADA** — Torque-v2 usa httpOnly cookies (ADR-003), v8 usa localStorage (vulnerável a XSS) |
| Billing Asaas/PIX | `asaas.ts` 100% coverage + `checkout-create-payment` + `asaas-webhook` | `service/billing/provider.go` + MockProvider (AsaasProvider real deferred D039) | ⚠️ **PARCIAL** — v8 wins, mas Torque-v2 tem interface pronta e dual-review obrigatório pending |
| Onboarding wizard | `Onboarding.tsx` + `OnboardingGate` | `/onboarding` + OnboardingPage + OnboardingGate component | ✅ **PRESENTE** (6 steps canonical, estado persistido, gate redirecionador) |
| RBAC 4 camadas | `permission_engine.ts` 95.31% coverage + `save-member-permissions` + `app_role` enum | `middleware/rbac.go` cascade master > master_only > admin_only > member_override > default | ✅ **PRESENTE** (ambas as abordagens equivalentes; Torque-v2 tem testes unitários do cascade) |
| Multi-tenancy | 120 `CREATE POLICY` RLS policies via JWT claim `organization_id` | `StripOrganizationID` middleware + `OrgIDFrom(ctx)` + toda query WHERE `organization_id = $1` | ✅ **PRESENTE** — abordagens diferentes mas ambas válidas; Torque-v2 mais defensivo (body field rejected com 400) |
| WebSocket real-time | Supabase Realtime (módulo-scoped, não globalmente auditado) | `internal/ws/hub.go` nhooyr.io/websocket tenant-scoped hub, patches only (ADR-002) | ✅ **PRESENTE** — Torque-v2 tem implementação mais explícita e testada |
| Observabilidade Sentry | `@sentry/react` + `@sentry/vite-plugin` + source maps + `_shared/sentry.ts` | `observability/sentry/` + PII scrubbing (truncate IPv4/24, IPv6/48) + beforeSend redacts 10+ keys | ✅ **PRESENTE** — Torque-v2 tem scrubbing mais rigoroso |
| Observabilidade logs estruturados | `_shared/logger.ts` JSON + breadcrumbs | zerolog structured + request_id propagation | ✅ **PRESENTE** |
| Observabilidade OpenTelemetry | ❌ | planejado (CLAUDE.md) ainda não wired | ❌ **AUSENTE nos dois** |
| Rate limiting | não explícito no v8 | `middleware/ratelimit.go` token-bucket 5rps anon / 30rps user + Retry-After | ✅ **PRESENTE apenas em Torque-v2** |
| CSP + HSTS | headers via Vite plugin (`vite.config.ts:14-19` básicos) | `security_headers.go` CSP tight `default-src 'none'` + HSTS 2yr preload em non-dev + COOP/CORP | ✅ **PRESENTE apenas em Torque-v2** (muito mais rigoroso) |
| CSRF protection | ❌ (Supabase JWT em Authorization header, vulnerável a token exfil via XSS) | double-submit token em cookie não-httpOnly + header `X-CSRF-Token` + ConstantTimeEqual | ✅ **PRESENTE apenas em Torque-v2** |
| Refresh token rotation | Supabase autoRefreshToken embedded | `repository/refresh` com SERIALIZABLE tx + reuse detection revoga chain via recursive CTE | ✅ **PRESENTE** — Torque-v2 explícito, v8 opaco |
| Landing page | `Landing.tsx` `/landing` | ❌ | ❌ **AUSENTE no Torque-v2** |
| Quota enforcement | `org-quota-enforcement` spec (D005) + `checkout-provision-org` aplicando | schema `org_quotas` delta model (migration 0002), sem enforcement runtime | ⚠️ **PARCIAL** (Torque-v2 só schema) |

### 1.D Design System e UX

| Componente | v8 | Torque-v2 | Veredito |
|------------|-----|-----------|----------|
| shadcn/ui primitives | 54 primitivos | 19 primitivos em `src/ui/` (avatar, badge, button, card, channel-badge, dropdown, empty-state, input, kbd, page-header, pill, quota-gauge, score-meter, separator, sheet, skeleton, spark, step-progress, tabs, tooltip) | ❌ **MENOS** — Torque-v2 ainda não tem dialog/alert-dialog/checkbox/slider/form/select/switch |
| Command Palette | não explicitamente identificado | `shell/CommandPalette.tsx` presente | ✅ **PRESENTE apenas em Torque-v2** |
| Cockpit (modo vendedor) | ❌ (v8 não tem dual mode) | `/cockpit` com CockpitShell + CockpitView + UiModeToggle (ADR-007) | ✅ **PRESENTE apenas em Torque-v2** (exclusivo) |
| TV Dashboard | `TVDashboard.tsx` fullscreen KPI display | ❌ | ❌ **AUSENTE** |
| Tipografia | Google Fonts (CDN) | self-hosted @fontsource-variable/fraunces + dm-serif + instrument-sans + jetbrains-mono (ADR-005) | ✅ **Torque-v2 supera** |

---

## Relatório 2 — Qualidade de Código

### v8milennialsb2bv2

| Dimensão | Nota | Evidência |
|----------|------|-----------|
| Estrutura e organização | **7/10** | Boa separação pages/components/hooks/services; porém 7 god components >800 linhas (`ChatWhatsApp.tsx:2443`, `ActionPanel.tsx:1654`, `LeadDetailDrawer.tsx:1490`, `useCampanhas.ts:1489`, `Performance.tsx:1443`, `PipePropostas.tsx:1436`, `useImportLeads.ts:1411`, `Agenda.tsx:1385`, `CopilotWizard.tsx:1188`) |
| Padrões e convenções | **7/10** | TypeScript strict; mas 9 componentes deprecated ainda importados; types.ts 8.721 linhas auto-gerado. 24+ ocorrências de `organization_id:` em payloads (echo do JWT, não violação real). |
| Segurança | **5/10** | ❌ localStorage auth (XSS-vuln, documentado em CONCERN-S1 do STATE.md); ❌ sem CSRF explícito; ❌ sem rate limiter; ✅ 120 RLS policies; ✅ CSP básico em vite.config.ts:14-19; ✅ 1 uso `dangerouslySetInnerHTML` safe (Recharts); 40 markers TODO/FIXME (non-critical) |
| Performance | **8/10** | ✅ 47 lazy routes + `lazyRetry` exponential backoff + manual chunks (vendor/supabase/charts/motion/query/dnd) + 518 chunks no dist/; ⚠️ sem memoização sistemática |
| Maintainability | **6/10** | ✅ LeadDetailDrawer consolida 5 modais deprecated (bom refactor); ❌ 9 deprecated components ainda navegáveis; ❌ pages de 1.400+ linhas são pesadas para modificar |
| **Score global v8** | **6.6/10** | |

### Torque-v2

| Dimensão | Nota | Evidência |
|----------|------|-----------|
| Estrutura e organização | **9/10** | Separação clara features/hooks/ui/shell/providers/contracts; apenas 1 arquivo >500 linhas (`SettingsPage.tsx:577`); backend Go modular (cmd/internal/handler/repository/service) |
| Padrões e convenções | **9/10** | TypeScript strict + `noUncheckedIndexedAccess` + `exactOptionalPropertyTypes` (configuração mais rigorosa que v8); imports via alias `@/`; PascalCase/camelCase/snake_case bem separados por camada; ESLint flat config com 7 regras reais incluindo ban de `dangerouslySetInnerHTML` (`eslint.config.js:55-61`), `no-explicit-any` (L66), `no-floating-promises` (L65), `react-hooks/exhaustive-deps: error` (L52) |
| Segurança | **10/10** | ✅ httpOnly + SameSite=Strict + Secure cookies (ADR-003); ✅ CSRF double-submit com ConstantTimeEqual; ✅ CSP `default-src 'none'; frame-ancestors 'none'`; ✅ HSTS 2yr preload em prod; ✅ Rate limiter token-bucket com Retry-After; ✅ StripOrganizationID middleware rejeita body.organization_id com 400 TENANT_FIELD_FORBIDDEN; ✅ PII scrubbing (10 keys, IPv4/24, IPv6/48); ✅ 0 uso `dangerouslySetInnerHTML`; ✅ MaxBytesReader(1 MiB) em todo DecodeJSON; ✅ URI scheme allowlist (https only) anti-SSRF; ✅ bcrypt cost 12 + needs-rehash; ✅ JWT HS256 WithValidMethods (rejeita alg=none); ✅ refresh rotation com reuse detection revoga chain; apenas 3 TODOs |
| Performance | **6/10** | ⚠️ **Sem lazy loading em routes.tsx** — todas as 15 páginas são static imports (`routes.tsx:2-21`); ✅ manual chunks em vite.config.ts (react-vendor/radix-vendor/motion-vendor); ✅ cursor-based pagination em todos os lists (ADR-004); ✅ TanStack Query v5 com staleTime configurado; ⚠️ memoização ad-hoc |
| Maintainability | **9/10** | ✅ 3 TODOs totais; ✅ zero código deprecated; ✅ 44 decisões arquiteturais em STATE.md; ✅ vault Obsidian sincronizado; ✅ cada sprint tem branch própria preservada como audit anchor |
| **Score global Torque-v2** | **8.6/10** | |

---

## Relatório 3 — Cobertura de Testes

### v8milennialsb2bv2

- **Baseline (D007)**: 9.33% → **73.53% lines / 60.04% branches / 67.15% funcs / 68.58% stmts** após Fase 1 (D025).
- **Thresholds vitest.config.ts** (L38-41): lines ≥72, statements ≥68, functions ≥67, branches ≥59. **Enforcement CI** ativo.
- **142 test files total** — 706 tests em D007; progressão documentada D007-D031.
- **Módulos 100%**: `permissions.ts` (98 linhas), `followup-sender`, `workflow-condition-evaluator`, `lead-service`, `asaas`, `embeddings`, `meta-api`, `audio-sender`, `ai-queue` — 9 módulos críticos blindados.
- **Módulos 95%+**: `auth.ts` (98.93%), `permission_engine` (95.31%), `user-auth` (98.48%), `tinyerp-utils` (100%), `google-calendar-utils` (100%), `workflow-executor` (92.6%), `workflow-action-handler` (95.82%), `natural-messaging` (94.31%).
- **Gaps conhecidos**: `ai-action-executor.ts` (1.323 LOC) estagnado em 94.27% — dead code em ACTION_HISTORY_MAP.
- **Qualidade**: table-driven, vi.resetModules pattern, mocks corretos, branches cobertos. Testes testam comportamento, não renderização.

### Torque-v2

- **Thresholds vitest.config.ts**: ❌ **não enforced** (reporter configurado, nenhum `thresholds:`).
- **35 frontend test files** (hooks 11 + component 1 + api 1 + providers 1 + ...). Última run local: **99/99 tests em 35 files**. Estimativa: ~22% cobertura por arquivo.
- **27 backend Go test files** — unit (jwt, password, token, csrf, provider, sentry, security_headers, ratelimit, httpx) + integration (auth_integration, operation_integration, audit_integration) + repo (lead, member, product, confirmation, onboarding, settings, proposal, master, subscription) + infra (event bus, ws hub) + pipe admin.
- **Módulos críticos SEM teste frontend**: `useWorkflows` (229 LOC), `useTaskActions` (211 LOC), `useCampaigns`, `useProposals`, `useOrgSettings`, `useBilling`, `useTasks`, `useOnboarding`, `useConfirmations`, `usePipes` admin methods.
- **Backend**: bem coberto em auth/leads/operations/repos. Gap: sem integration test de tentativa de leak cross-tenant (assertion "org A tenta GET `/api/v1/leads/<org-B-uuid>` → 404").
- **Qualidade**: mocks corretos via vi.stubGlobal('fetch'), renderHook + waitFor, assertions em URL targeting e mutation payloads. Padrão sólido mas cobertura insuficiente.

### Métricas comparativas

| Métrica | v8 | Torque-v2 | Delta |
|---------|-----|-----------|-------|
| Test files totais | 142 | 62 (35 front + 27 back) | v8: +80 files |
| Test cases estimados | 706+ (D007) → ~1.200 hoje | 99 front + ~60 back = ~160 | v8: ~7× |
| Cobertura statements | ~68.58% | ~22% estimada | v8 +47pp |
| Threshold enforcement | ✅ sim (CI) | ❌ não configurado | v8 supera |
| Módulos 100% | 9 | 1 (`ws/hub`) | v8 supera |
| Qualidade assertions | alta (table-driven) | média-alta (hooks cobertos, components não) | v8 supera ligeiramente |

---

## Relatório 4 — Eficiência e Escalabilidade

### 4.A Arquitetura de dados

| Aspecto | v8 | Torque-v2 | Veredito |
|---------|-----|-----------|----------|
| Modelo | Supabase direto do frontend com RLS | Go API em 3 camadas (handler → service → repo) | Torque-v2 vence em separação; v8 vence em velocidade de iteração |
| Paginação | Offset via `.range(from, to)` (v8 `useLeads`, `useConversationHistory`, `useLeadTimeline`) | Cursor base64(timestamp\|id) em todas as lists (ADR-004) | **Torque-v2 supera** — cursor-based é escalável >10k records |
| N+1 risk | RLS policy complexas podem gerar seq scan | WHERE organization_id + compound index em toda tabela | Torque-v2 mais defensivo |
| Caching | TanStack Query v5 com staleTime ad-hoc | TanStack Query v5 + queryKeys factory + invalidate systematic | **Torque-v2 supera** — padrão unificado |

### 4.B Real-time

| Aspecto | v8 | Torque-v2 |
|---------|-----|-----------|
| Engine | Supabase Realtime (postgres_changes filtered by org_id, 2s debounce) | WebSocket hub Go (nhooyr.io/websocket) tenant-scoped `internal/ws/hub.go` |
| Protocolo | full row diffs (Supabase) | patches only `{type, entity_id, patch, version}` (ADR-002) |
| Escalabilidade | bound by Supabase connection limits (managed, não auditado) | in-process hub per-instance + swappable para Redis Pub/Sub |
| **Veredito** | Ambos funcionam. Torque-v2 mais explícito e testado; v8 mais simples |

### 4.C Jobs assíncronos

| Aspecto | v8 | Torque-v2 |
|---------|-----|-----------|
| Padrão | Edge Functions + `pg_cron` 10+ jobs | 202 Accepted + poll GET `/operations/:id` + WS push (ADR-006) |
| Ledger | spread em múltiplas tabelas (workflow_executions, dispatch_rules, etc.) | single `operations` table com ENUM status + SKIP LOCKED claim |
| Retry | dead_letter table + `retry-dead-letter-jobs` edge fn | in-process worker pool com exponential backoff + retry budget |
| **Veredito** | Torque-v2 mais elegante e padronizado; v8 mais pragmático com cron scheduler nativo |

### 4.D Bundle e carregamento

| Aspecto | v8 | Torque-v2 |
|---------|-----|-----------|
| Lazy loading | 47 páginas lazy via `lazyRetry()` com retry exponential após deploy | ❌ **0 páginas lazy** — todas static imports em `routes.tsx:2-21` |
| Manual chunks | 6 chunks (vendor/supabase/charts/motion/query/dnd) | 3 chunks (react-vendor/radix-vendor/motion-vendor) |
| Chunks dist | 518 arquivos em `dist/assets/` | não buildado nesta sessão |
| **Veredito** | **v8 supera dramaticamente** em bundle strategy. Torque-v2 tem deficit crítico: app inicial carrega todas as páginas de uma vez |

### 4.E Multi-tenancy e risco de data leak

| Aspecto | v8 | Torque-v2 |
|---------|-----|-----------|
| Isolation | 120 RLS policies (WHERE org_id = JWT claim) | `StripOrganizationID` rejeita body field + toda query `WHERE organization_id = $1` + `OrgIDFrom(ctx)` middleware |
| Frontend | 24+ ocorrências de `organization_id:` em mutations (echo do JWT, não violação mas **superfície de erro** se RLS policy for loose) | ❌ **NENHUMA** ocorrência de `organization_id` em mutation (1 exception em `useMaster.ts` — impersonation, path param explícito) |
| JWT source of truth | cliente envia JWT que carrega `user_metadata.organization_id` | backend extrai `OrgID` do JWT claim, nunca do body |
| Failure mode | RLS policy mal escrita = leak silencioso | middleware rejeita com 400 TENANT_FIELD_FORBIDDEN |
| **Veredito** | **Torque-v2 supera em defense-in-depth.** v8 depende 100% de RLS correta; Torque-v2 tem 2 camadas (middleware + per-query filter) |

---

## Relatório 5 — Score Final Comparativo

| Dimensão | v8 | Torque-v2 | Δ | Veredito |
|----------|------|-----------|------|----------|
| Completude de features | **85/100** | **35/100** | -50pp | **v8** |
| Qualidade de código | **66/100** | **86/100** | +20pp | **Torque-v2** |
| Cobertura de testes | **73/100** | **25/100** | -48pp | **v8** |
| Segurança | **55/100** | **95/100** | +40pp | **Torque-v2** |
| Performance/Eficiência | **75/100** | **60/100** | -15pp | **v8** (bundle/lazy) |
| Escalabilidade | **65/100** | **80/100** | +15pp | **Torque-v2** |
| Arquitetura | **60/100** | **90/100** | +30pp | **Torque-v2** |
| DX (Developer Experience) | **60/100** | **85/100** | +25pp | **Torque-v2** (ADRs, STATE.md, vault, CI/CD completo) |
| **SCORE COMPOSTO (média simples)** | **67.4/100** | **69.5/100** | **+2.1pp** | **EMPATE TÉCNICO (Torque-v2 levemente à frente)** |

### 5.1 Torque-v2 é X% melhor/pior que o v8?

**Por score composto: Torque-v2 é ~3% melhor que v8 em média ponderada simples.** Mas isso esconde a verdade: são sistemas em fases diferentes. v8 é produto em produção (~30 orgs ativas); Torque-v2 é fundação pronta mas sem produto rodando. Se ponderar por completude de features (ship-readiness), **v8 está ~145% mais pronto para produção**. Se ponderar por qualidade da fundação, **Torque-v2 está ~40% mais sólido** (segurança + arquitetura + DX).

### 5.2 As 5 maiores vantagens do Torque-v2 sobre o v8

1. **Segurança radicalmente superior** — httpOnly cookies (v8 usa localStorage, XSS-vulnerável); CSP `default-src 'none'`; CSRF double-submit; HSTS 2yr preload; PII scrubbing rigoroso; MaxBytesReader; bcrypt cost 12 + needs-rehash; refresh rotation com reuse detection revoga chain via recursive CTE; rate limiter token-bucket. v8 não tem nenhum desses salvo CSP básico.
2. **Arquitetura 7 ADRs formalizados e cumpridos** — OpenAPI snake↔camel transform; WebSocket hub tenant-scoped com patches only; cursor pagination uniforme; self-hosted fonts; 202+poll async jobs; modo Vendedor/Gerente. v8 tem decisões implícitas espalhadas no código.
3. **Multi-tenancy defense-in-depth** — `StripOrganizationID` middleware rejeita body field antes de chegar no handler; toda query filtra `organization_id`. v8 depende 100% de RLS policy correta (superfície de erro maior).
4. **CI/CD completo e documentado** — 4 workflows (ci.yml backend+front+lint, release.yml Docker GHCR, deploy.yml migrate→deploy→smoke, security-scan.yml gosec+govulncheck+npm audit+Trivy nightly). v8 tem pipeline mais básico.
5. **Observabilidade operacional** — zerolog structured + OpenTelemetry planejado + Sentry PII scrubbing de 10+ keys + truncate IPv4/24 IPv6/48; runbooks `.specs/runbooks/incident-response.md` + `oncall-basics.md`; k6 load test `.specs/loadtest/api-baseline.k6.js` com p95<500ms gate. v8 tem Sentry + logs, sem runbooks nem load test committed.

### 5.3 As 5 maiores lacunas do Torque-v2 que precisam ser fechadas

1. **Ausência total de lazy loading no frontend** — `routes.tsx:2-21` importa todas as 15 páginas estaticamente. v8 tem 47 lazy routes com `lazyRetry` exponential. **Impacto: bundle inicial 3-5× maior do que necessário; FCP/LCP regredidos.** Fix: 2-3 dias.
2. **Copilot/IA ausente integralmente** — zero código de wizard, playground, RAG, embeddings, TTS, triggers. v8 tem 10 edge functions + 13 componentes + embeddings Gemini 2.0 + ElevenLabs TTS. **Impacto: F06 é diferencial competitivo do Torque e está 0% implementado.** Fix: 6-10 semanas.
3. **Chat WhatsApp em tempo real inexistente** — InboxPage é lista; não há chat UI. v8 tem `ChatWhatsApp.tsx` 2.443 LOC + `useWhatsAppChat.ts` 1.099 LOC + Evolution API integrada. **Impacto: F04 core do produto.** Fix: 3-4 semanas.
4. **Cobertura de testes insuficiente** — sem thresholds no vitest.config.ts; 9+ hooks críticos sem teste (useWorkflows 229 LOC, useTaskActions 211 LOC, useCampaigns, useProposals, useOrgSettings, useBilling, useTasks, useOnboarding, useConfirmations). v8 tem 73.5% lines / 68% stmts com enforcement. **Impacto: regressões não detectadas; risco ao ir a prod.** Fix: 2-3 semanas para paridade.
5. **Automações sem executor e sem UI canvas** — schema `workflows` existe mas `workflow-executor.ts` equivalente ao v8 (92.6% coverage, 30 action types) não foi construído; `WorkflowBuilderPage.tsx:357` renderiza placeholder sem @xyflow. **Impacto: F07 backbone de automação comercial.** Fix: 4-6 semanas.

### 5.4 Esforço estimado para paridade funcional

**Total: 26-34 semanas** (≈ 6-8 meses) de 1 dev full-stack dedicado, considerando:

| Bloco | Features | Semanas |
|-------|----------|---------|
| F06 Copilot completo (wizard ou playground + RAG + TTS + embeddings + triggers + edge functions equivalentes a 10 do v8) | 6-10 |
| F04 Chat WhatsApp real-time (ChatUI + useChat hook + Evolution API adapter + Supabase Realtime-equivalent no WS hub) | 3-4 |
| F07 Workflow executor backend + @xyflow canvas frontend + 30 action types + execuções UI | 4-6 |
| F08 Campanhas UI completa (CreateModal + Detail + DispatchRulesSection) | 2 |
| F09 Performance completo (Ranking + Metas + Premiações + Comissões + Win/Loss + UTM) | 3 |
| F12 Pipelines customizados (generalização pipe + CRUD + dispatch rules + UI drag-drop) | 3-4 |
| F13 Upsell + Agenda + TV Dashboard + Checklists + Templates | 3-4 |
| F16 Master Admin completo (Users + Plans + Features + AuditLogs + Operations) | 2 |
| Lazy loading + cobertura testes até 70% + thresholds enforcement | 2-3 |
| AsaasProvider real + dual-review + quota enforcement runtime | 1-2 |

### 5.5 Recomendação final

**NÃO porte nada do v8 ao Torque-v2 antes de continuar desenvolvimento.**

Razões:
1. **Código v8 viola ADRs centrais do Torque-v2**: localStorage auth (vs ADR-003 httpOnly), offset pagination (vs ADR-004 cursor), Supabase direto do frontend (vs Go API 3-camadas).
2. **God components do v8 seriam dívida técnica imediata**: `ChatWhatsApp.tsx` 2.443 LOC, `ActionPanel.tsx` 1.654 LOC, `LeadDetailDrawer.tsx` 1.490 LOC. Torque-v2 hoje não tem nenhum arquivo acima de 577 linhas.
3. **Testes do v8 são acoplados a Supabase**, não reutilizáveis no stack Go+pgx.
4. **Vault Obsidian do Torque-v2 é canônico hoje** — o v8 CLAUDE.md descreve um sistema diferente (supabase-centric, roles sdr/closer em enum, not using httpOnly cookies). Misturar causaria drift documental grave.

**Portável é conhecimento**, não código:
- **Mapa de entities/tabelas**: v8 tem 120 tabelas em 325 migrations; Torque-v2 tem ~36 tabelas em 17 migrations. Auditar quais entities faltam (ex.: `webhook_deliveries`, `message_templates`, `commission_*`) e listar como backlog.
- **Lista de 30 action types do workflow-action-handler.ts**: portar conceitualmente para o DAG do Torque-v2.
- **Shape das edge functions de AI (embeddings, prompt synthesis, audio-sender)**: usar como spec para F06 Copilot.
- **Copy em PT-BR de UIs críticas**: CampanhaKanban, Performance, Agenda — copy já refinado pode ser reutilizado.
- **Bundle strategy (lazyRetry + manualChunks configuration)**: adaptar para Torque-v2 imediatamente.

**Ordem recomendada para executar o backlog de paridade:**

1. **Semana 1-2**: Lazy loading em todas as 15+ rotas do Torque-v2 + vitest thresholds enforcement + 9 hooks faltantes cobertos.
2. **Semana 3-6**: F04 Chat WhatsApp real-time (maior gap do produto, bloqueia F06 copilot).
3. **Semana 7-16**: F06 Copilot completo (maior diferencial competitivo).
4. **Semana 17-22**: F07 Workflow Builder com @xyflow + executor.
5. **Semana 23-26**: F08 Campanhas UI + F09 Performance completo + F16 Master Admin completo.
6. **Semana 27-30**: F12 Pipelines custom + F13 Upsell/Agenda/TV.
7. **Semana 31-34**: AsaasProvider real + quota enforcement + dual-review money flow + pentest real em staging + load test em staging.

Ao fim disso o Torque-v2 alcança paridade e supera v8 em qualidade (ver Score Final).

---

## Recomendações priorizadas (backlog de ações)

### P0 — Dívida técnica imediata (antes de qualquer nova feature)

1. **Lazy loading em routes.tsx** — portar padrão `lazyRetry` do v8 adaptado, reduzir bundle inicial. Arquivo: `torque-web/src/routes.tsx:2-21`.
2. **Vitest coverage thresholds enforcement** — adicionar `test.coverage.thresholds: { lines: 60, statements: 60, functions: 50, branches: 50 }` em `torque-web/vitest.config.ts`.
3. **Backend integration test de tenant isolation** — org A tenta GET `/api/v1/leads/<uuid de org-B>` deve retornar 404 (não 403). Adicionar em `torque-api/internal/repository/lead/lead_test.go`.
4. **Dependabot/Renovate ativação** — para `torque-api/go.mod` e `torque-web/package.json` (follow-up documentado em D044).

### P1 — Paridade funcional crítica

5. **F04 Chat WhatsApp real-time** — ChatPage + useChat hook + WS subscribe em `conversation.message`; adapter Evolution API em `service/integration/messaging_evolution.go`.
6. **F06 Copilot base** — AgentWizard (simplificado vs v8) + Playground + embeddings via pgvector + OpenRouter client.
7. **F07 Workflow canvas + executor** — instalar @xyflow/react, 12 node types prioritários (não 30 de v8), executor em worker pool.

### P2 — Completude de produto

8. **F08-F09 UI** — CampanhaDetail + CampanhaKanban; PerformanceRanking + PerformanceMetas consolidado.
9. **F12 Pipelines customizados** — generalização do schema de pipes + CRUD admin + dispatch rules.
10. **F14 AsaasProvider real** — dual-review de money-flow + integração Asaas + webhook real.

### P3 — Ops e hardening

11. **OpenAPI spec refresh** post-S04 (~20 handlers drifted, documentado em D035).
12. **GHCR keyless signing via cosign** em `release.yml` (add `id-token: write`).
13. **gosec dentro do ci.yml main job** (não só security-scan.yml).
14. **Impersonation cookie swap atomic com audit row** (S26 follow-up).

---

## Evidências gerais

- **Torque-v2 Backend**: 17 migrations, 24 handler packages, 21 repos, ~6.077 LOC Go em handlers + ~7.758 LOC em repos. Commit atual: `562d1b9` (S29 merged).
- **Torque-v2 Frontend**: 157 arquivos TS, 15 páginas reais + 2 ComingSoonPage, 35 test files com 99/99 passing.
- **v8 Backend**: 142 edge functions + 33 shared libs, 325 migrations com 120 RLS policies, 142 test files.
- **v8 Frontend**: 51 páginas (44 + 7 master), 47 lazy routes, 7 god components >800 LOC.
- **Cobertura**: v8 73.53%/68.58% enforced; Torque-v2 ~22% sem enforcement.
- **Decisões formalizadas**: v8 tem D007-D031 focadas em cobertura; Torque-v2 tem D001-D044 cobrindo arquitetura + sprints + segurança + hardening.

> Última atualização: 2026-04-20 por agent-conductor. Próxima revisão sugerida após entregar P0+P1 (estimado 8-12 semanas).
