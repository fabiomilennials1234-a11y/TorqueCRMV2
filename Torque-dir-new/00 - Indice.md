---
tags:
  - moc
  - indice
  - raiz
created: 2026-04-15
last_updated: 2026-04-17
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

**Etapa:** **Fase D em andamento — S43 ✅ (2026-04-20)** F07.1 Workflow Builder canvas xyflow + list page. Dep @xyflow/react v12.10.2. WorkflowListPage substitui seed com useWorkflows live. WorkflowCanvasPage /workflows/:id com palette draggable (6 node types), ReactFlow screenToFlowPosition, onConnect merge next_step_ids, onNodeDragStop PUT position, inspector JSON config. Routes /workflows list + /workflows/:id canvas. 189/189 em 65 files, coverage lines 63.33%. Próxima: S44 executor worker. **Fase C concluída — S42 ✅** F06.6 Copilot Metrics page. Backend GetAgentMetrics com state breakdown + message aggregates (tokens in/out/avg latency filtered assistant). Handler GET /agents/:id/metrics com RFC3339 window + 400 codes (INVALID_SINCE/UNTIL/WINDOW, WINDOW_TOO_LARGE > 365d). Frontend: AgentMetricsPage em /copilot/:id/metrics com picker 7d/30d/90d + 4 KPI cards (pt-BR) + stacked sessions-by-state bar. 187/187 em 64 files, coverage lines 63.21%. **Fase C completa** (S37→S38→S39→S40→S41→S42): OpenRouter+SSE / Playground UI / RAG pgvector+Gemini / Triggers matcher / TTS ElevenLabs / Metrics. Frontend +36 cases, +13 files desde início da fase; coverage 59.17 → 63.21 (+4.04pp). Próxima fase: **D — F07 Workflow Builder (S43-S45)**. **S41 ✅** F06.5 TTS. Migration 0021 agents.tts_enabled + tts_voice_id. Service ai/tts.go com ElevenLabsTTS (POST /v1/text-to-speech/:voice_id + xi-api-key + audio/mpeg, 5000-char/5MB caps, taxonomia 401/429/5xx/4xx) + MockTTS deterministic fallback. Config ELEVENLABS_BASE_URL|API_KEY|MODEL_ID (default eleven_multilingual_v2). Handler: novo POST /agents/:id/tts/preview (sem storage, mp3 inline). Frontend: TTSPanel no editor com checkbox + voice_id + "Ouvir preview" + inline player (objectURL cleanup no unmount). 185/185 em 63 files, coverage lines 63.08%. Outbound pipeline (worker message.outbound → S3 → kind=audio + org_quotas) adiado para sprint futura. Próxima: S42 Metrics. **S40 ✅** F06.4 Copilot triggers. Migration 0020 agent_triggers + conversations.assigned_agent_id + seeds triggers.view|manage. Matcher puro (ops eq/neq/contains/in/present/absent + all/any + kill-switch/status guards + unknown-op fail-closed). Repo CRUD + ListActiveTriggers (JOIN agents) + AssignAgent. Handlers GET/POST /agents/:id/triggers + PATCH/DELETE /triggers/:tid + WS events agent_trigger.{created,updated,deleted}. Frontend: hooks + TriggersPanel no editor (form inline com field/op/value/priority, rule list com toggle/trash). Coverage ratchet 60/60/55/50, medido lines 62.71/stmts 60.22/funcs 57.23/branches 52.01. 183/183 em 62 files. Worker dispatch diferido para Fase D (reusa workflow executor). Próxima: S41 TTS ElevenLabs. **S39 ✅** F06.3 Copilot RAG. Migration 0019 pgvector + knowledge_chunks embedding vector(768) + HNSW cosine + agents.knowledge_collection_id FK + seeds knowledge.view|manage. Docker-compose dev switched to pgvector/pgvector:pg15. Backend: Embedder interface (Gemini real + Mock fallback), chunker paragraph-aware 500/50, repo ListCollections/InsertChunks/SimilaritySearch com encodeVector, Ingest service goroutine detached com status transitions. Playground handler embeda último user turn + topK=5 + prependContext delimitado. Frontend: hooks knowledge + KnowledgePanel no editor (select + create inline + sources list + ingest form). 178/178 em 61 files, coverage lines 62.23%. Próxima: S40. **S38 ✅** F06.2 Copilot Playground UI. AgentListPage real substitui mockup seed; AgentPlaygroundPage (`/copilot/:id`) layout Vercel — editor lateral (system_prompt, model, temp, max_tokens) com dirty detection + chat SSE com Parar/Regenerar. Hook `useAgentStream` consome `fetch` + `ReadableStream.getReader()` (EventSource não suporta CSRF); parser `\n\n` com frames delta/done/error; AbortController limpo no unmount. Backend: `agentView` agora expõe `system_prompt` via `toAgentView` helper unificado. Tests: useAgentStream 4 cenários + AgentListPage 2 cenários. **173/173 em 60 files**, coverage lines **61.56%** (+2.39pp). Próxima: S39 (Agent Wizard 20+ steps + session persistence). S37 ✅ F06.1 OpenRouter + Playground SSE backend. Remediation-B ✅ (84→87/100). Fase B concluída. S00-S29 ✅ — Roadmap original de 30 sprints COMPLETO. S29 entregou security audit OWASP Top 10, k6 load test, runbooks operacionais e security-scan.yml.

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
