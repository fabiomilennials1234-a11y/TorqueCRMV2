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

**Etapa:** S00 ✅. S01 ✅ scaffold. S02 ✅ auth+tenancy+RBAC. S03 ✅ security+observabilidade+audit+ratelimit. S04 ✅ OpenAPI+WS hub+async jobs (2026-04-19). Runtime (go mod tidy/build/test -race, migrate up) pendente no host do usuario.

**S00** — ESLint flat (a11y + hooks + ban `dangerouslySetInnerHTML`), TS strict, Prettier, Vitest (51 testes), CI, vocabulario canonico, `--clay-*` → `--card-*`/`.tactile-*`, pos-login navigation por `ui_mode`.

**S01** — `torque-api/` chi + pgx v5 + zerolog + probes + middleware stack com `StripOrganizationID`, migrations 0001–0003, Docker distroless nonroot, teste de concorrencia do partial index de Task.

**S02** — Migration 0004 `refresh_tokens`. Services password/jwt/token/permission. Repositories user+refresh (SERIALIZABLE + RevokeChain via recursive CTE). Middleware Authenticator/RequireAuth/TenantScope/CSRF/RequireFeature/RequireRole/RequireMaster. Handlers `/api/v1/auth/{login,refresh,logout,me}`, `/api/v1/me/preferences`, `/api/bootstrap`.

**S03** — Security headers parametrizados (CSP tight, HSTS 2yr preload em non-dev, COOP/CORP). Sentry SDK com PII scrubbing. Audit service/repo usando `audit_log` existente. Rate limiter in-memory. Bootstrap enriquecido.

**S04** — Migration 0005 `operations` (ENUM + SKIP LOCKED claim + retry chain + progress/result/error jsonb). Event bus in-process (DropOldest/DropNewest). WS hub `ws.Hub` tenant-scoped via `coder/websocket` com ping/pong/PongTimeout. Worker pool goroutine + semaphore. Handlers `POST/GET/DELETE /api/v1/operations` + `GET /api/v1/ws`. `/openapi.{yaml,json}` publicos com spec expandida cobrindo todos os endpoints.

Proxima sprint: **S05** — Frontend CRUD Foundation (useInfiniteList/useMutation, optimistic updates, AppError mapping).

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
