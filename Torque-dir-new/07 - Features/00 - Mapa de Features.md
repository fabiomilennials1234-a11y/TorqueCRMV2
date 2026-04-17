---
tags:
  - features
  - mapa
  - roadmap
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Mapa de Features

## Regra cardinal

**Nunca avance para a próxima feature antes da anterior estar pronta, validada, documentada e estável.**

"Pronta" não é "a demo funcionou". É o `Checklist de Conclusao.md` da feature 100%, e o produto aguentando uso real sem regressão. Qualquer atalho aqui vira dívida que vai cobrar juros compostos.

## Pré-requisito global

Todas as features dependem de [[Escopo do Sistema Base]] com [[Checklist Sistema Base]] em 100%. Nada começa antes disso.

## Legenda

- **Status:** `not started` / `in progress` / `blocked` / `done`
- **Esforço:** S (dias) / M (semanas) / L (mês+)

## Tabela

| ID  | Nome                                                                     | Status      | Dependências          | Esforço | Observação |
| --- | ------------------------------------------------------------------------ | ----------- | --------------------- | ------- | ---------- |
| F01 | [[F01 - Funis Hub e Pipe WhatsApp/Spec\|Funis Hub + Pipe WhatsApp]]      | not started | Sistema Base 100%     | L       | Vertical slice que prova o pipeline técnico inteiro (contratos, realtime, DnD, permissões, estados) |
| F02 | Pipe Confirmação (Countdown D-5/D-3/D-1)                                 | not started | F01                   | M       | Reutiliza kanban de F01 + introduz `CountdownBadge` |
| F03 | Pipe Propostas (HeatSlider 1-5 + TinyERP)                                | not started | F01                   | M       | Introduz `HeatSlider` e integração TinyERP |
| F04 | Inbox Multi-canal (4 canais + realtime + takeover)                       | not started | F01                   | L       | WhatsApp, Instagram, Messenger, SZ.Chat + takeover humano |
| F05 | Follow-ups (hoje/atrasado/futuro + Ações do Dia)                         | not started | F01                   | M       | Fila de ações diárias por usuário |
| F06 | Copilot (wizard 20+ steps, RAG, TTS, playground)                         | not started | F04                   | L       | Depende de Inbox para contexto conversacional |
| F07 | Workflow Builder (@xyflow/react, 12 node types)                          | not started | F01                   | L       | Canvas visual de automações |
| F08 | Campanhas (wizard + kanban + dispatch rules)                             | not started | F07                   | L       | Consome Workflow Builder |
| F09 | Analytics (Dashboard, FunnelChart, UTMs, Ranking, Performance, TV, Outbound) | not started | F01..F03          | L       | Introduz stack Visx oficialmente |
| F10 | Equipe (comissões, metas, premiações + confetti)                         | not started | F09                   | M       | Depende de métricas de performance |
| F11 | Produtos (catálogo, import XLSX, materiais)                              | not started | F03                   | S       | Alimenta Propostas |
| F12 | Upsell + Pipelines Customizados                                          | not started | F01                   | M       | Generaliza modelo de pipe |
| F13 | Onboarding Wizard + Gate                                                 | not started | Sistema Base          | M       | Pode rodar em paralelo cedo |
| F14 | Checkout + PIX + Provisioning                                            | not started | Sistema Base          | L       | Entrada comercial, pode paralelizar |
| F15 | Configurações completas (8 tabs)                                         | not started | Transversal           | M       | Consolidada ao longo do roadmap |
| F16 | Master Admin (Orgs/Users/Audit/Operations/Features)                      | not started | Várias                | L       | Última — requer visão completa do produto |

## Fluxo sugerido (ordem de execução)

1. **Trilha Produto Core:** F01 → F02 → F03 → F04 → F05
2. **Trilha IA:** F06 (após F04)
3. **Trilha Automação:** F07 → F08 (após F01)
4. **Trilha Insights:** F09 → F10 (após F03)
5. **Trilha Catálogo:** F11 (após F03)
6. **Trilha Extensão:** F12 (após F01)
7. **Trilha Comercial (paralela):** F13, F14 (podem iniciar cedo)
8. **Trilha Transversal:** F15 (incremental)
9. **Trilha Operacional:** F16 (último)

## Referências

- [[Escopo do Sistema Base]]
- [[Checklist Sistema Base]]
- [[Template - Feature Spec]]
