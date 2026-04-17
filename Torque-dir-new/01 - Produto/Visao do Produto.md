---
tags:
  - produto
  - visao
  - estrategia
created: 2026-04-15
last_updated: 2026-04-15
status: vivo
---

# Visão do Produto — Torque CRM

## O que é

**Torque CRM** é um SaaS B2B multi-tenant brasileiro. Um CRM comercial com **IA conversacional embarcada**, **automações visuais** e **comunicação multi-canal unificada** em um único espaço de trabalho.

Não é um CRM tradicional com "chat integrado". É um CRM construído desde a fundação para times que vendem em conversa — onde a inbox de WhatsApp, o pipeline kanban, o [[Copilot]] e o analytics são a mesma superfície de trabalho, não módulos costurados.

## Para quem

Times comerciais brasileiros B2B — PMEs e médias empresas — que fazem **outbound e inbound via WhatsApp** como canal principal de vendas. Operações com estrutura comercial clara (SDRs qualificando, closers fechando, gestor acompanhando ranking e metas), que já perderam dinheiro em ferramentas genéricas estrangeiras mal traduzidas ou em CRMs nacionais que não entendem de IA.

O cliente típico hoje: ~30 organizações ativas, operação comercial profissionalizada, ticket médio relevante, ciclo de venda de dias a semanas, alto volume de conversas por SDR.

## Proposta de valor

**O único CRM brasileiro com agente de IA conversacional integrado ao WhatsApp, pipelines visuais e analytics de funil completo — da aquisição em Meta Ads até a emissão de NF-e no TinyERP.**

Três coisas que ninguém entrega junto no Brasil:
1. Um **[[Copilot]]** que responde, qualifica e agenda pelo SDR dentro do WhatsApp oficial, com tom configurável por org e guardrails.
2. **Pipelines visuais** com `Calor`, `Agendado`, `Compareceu`, `Vendido`, `Perdido` como primeira classe — vocabulário de venda real, não CRM genérico.
3. **Analytics ponta-a-ponta** que amarra gasto de Meta Ads a receita faturada no ERP, por SDR, por [[Pipe]], por campanha.

Ver vocabulário oficial em [[Glossario e Vocabulario]].

## Três pilares

### 1. Pipelines
Kanban como superfície principal da operação. Multi-funil por org. `Stage` é dado de primeira classe. `Calor` (1–5) é atributo do lead. Transições de `Stage` disparam automações. Visual world-class, denso, responsivo, sem firulas.

### 2. Comunicação Unificada
Inbox única para **WhatsApp** (oficial via Meta Cloud API e via SZ.Chat como provider alternativo), **Messenger**, **Instagram Direct**. Threading por `Lead`, não por número. Histórico completo, anexos, áudio, templates HSM, janela de 24h tratada como primitivo.

### 3. IA Embarcada
- **[[Copilot]]** — agente IA que conversa pelo SDR no WhatsApp, qualifica, agenda, passa para humano no momento certo.
- **[[Oraculo Comercial]]** — coach IA no dashboard que lê os números e aponta onde o time está sangrando pipeline.
- **Dispatch Rules** — sequências automáticas de mensagens em follow-up, escaláveis por `Stage` e por `Calor`.

## Modelo de negócio

SaaS recorrente. Precificação por **plano + seats + addons**.

- **Planos** — camadas de capacidade (limites de orgs-filhas, mensagens/mês, automações, seats base).
- **Seats** — usuários pagos por org (admins e membros).
- **Addons** — recursos opcionais ligados por feature flag (ex.: número extra de WhatsApp, tokens extras de IA, canal Instagram).
- **Pagamento** — integração **Asaas** com **PIX** e **cartão**. Cobrança recorrente gerida pelo Sistema Base.
- **Tamanho atual** — ~30 organizações ativas. A arquitetura precisa levar a milhares sem reescrita.

## Stack alvo

- **Backend** — **Go** novo, escrito do zero, substituindo o legado. Foco em domínio explícito, boundaries de contexto claras, contratos versionados, observabilidade desde o primeiro commit.
- **Frontend** — **React** moderno, dark-first, tipografia editorial, sistema de design próprio inspirado em Apple, Linear, Stripe, Vercel e Airbnb. Nada de template. Nada de "dashboard genérico".
- **Multi-tenancy** — isolamento lógico por `org_id` em todo o domínio, enforcement no gateway e no repositório. Master Admin opera cross-tenant via escopo explícito.

Detalhamento em `02 - Arquitetura` e `03 - Design`. Migração do CRM antigo em `09 - Migração do Legado`.

## Padrão exigido

**World-class, inegociável.** Em todas as camadas — segurança, arquitetura, performance, qualidade de código, design de interface, experiência de uso.

Se um auditor técnico de um fundo de M&A olhasse esse codebase amanhã, não haveria nada para esconder. Se um designer da Linear ou da Stripe abrisse a interface, não reconheceria um template. Esse é o piso. Não o teto.
