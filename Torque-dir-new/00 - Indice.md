---
tags:
  - moc
  - indice
  - raiz
created: 2026-04-15
last_updated: 2026-04-16
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
- [[Template - Feature Spec]]

### 08 - Decisoes
- [[ADR-001-contratos-openapi-snake-camel]]
- [[ADR-002-realtime-websocket-go-hub]]
- [[ADR-003-auth-httponly-cookies-samesite]]
- [[ADR-004-paginacao-cursor-based]]
- [[ADR-005-tipografia-self-hosted]]
- [[ADR-006-jobs-assincronos-202-poll]]

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

**Etapa:** Sistema Base — Final da Etapa 6 (Paginas-casca), antes da Etapa 7 (Validacao).

O frontend esta ~85% completo como prototipo visual de alta fidelidade. Zero codigo Go existe. Zero testes. A fonte de verdade para execucao e o [[Plano Mestre de Finalizacao do SaaS CRM]] em `09 - Backlog/`.

Proximas sprints: **S00** (fechar Sistema Base frontend) + **S01** (Go backend skeleton + DB schema) em paralelo.

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
