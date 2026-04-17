---
tags: [migracao, legado, referencia, priorizacao]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Top 15 Docs Prioritários

Documentos do vault legado que devem ser consultados durante a adaptação para o Torque-v2. Ordenados por impacto no ciclo de construção. Cada entrada inclui o motivo concreto da consulta.

Vault legado em: `C:\Users\torch\Desktop\milennials\v8milennialsb2bv2\Obsidian\Segundo Cerebro\Torque-wiki`

Estratégia de uso: ler como referência, nunca copiar. Ver [[Mapa do Vault Legado]] e [[Itens Nao Migrados]].

| # | Path (relativo a Torque-wiki/) | Título | Por quê |
|---|---|---|---|
| 1 | `02 - Arquitetura/Specs/ARCHITECTURE.md` | Architecture | Padrões de hooks, React Query, multi-tenancy e composição de providers. Define contratos de leitura e mutação que o Torque-v2 herda na Fase 0. |
| 2 | `02 - Arquitetura/Specs/CONCERNS.md` | Concerns | Débitos técnicos S1 a S6 e D1 a D5 que NÃO devem se repetir. Checklist de anti-padrões para revisão obrigatória. |
| 3 | `02 - Arquitetura/Specs/CONVENTIONS.md` | Conventions | Naming, estrutura de pastas, padrões de query key, imports. Base para as convenções do Torque-v2 (preservar o que funcionou, mudar o que não). |
| 4 | `02 - Arquitetura/Modulos.md` | Módulos | Inventário completo de páginas, hooks e edge functions. Mapa de superfície do legado para entender o que precisa de equivalente no Torque-v2. |
| 5 | `04 - Decisoes/ADR-2026-04-12-arquitetura-inicial.md` | ADR inicial | Decisões D1 a D9. D5 (Supabase BaaS) obsoleta; D1, D2, D3, D4, D6, D7, D8, D9 continuam relevantes como ponto de partida arquitetural. |
| 6 | `06 - Features/Admin/Permissoes Sistema.md` | Permissões | Cascata RBAC em 4 camadas (master > org_admin > manager > agent). Crítico para `usePermission`, `PermissionGate` e enforcement server-side na Fase 1. |
| 7 | `06 - Features/Vendas/Pipe WhatsApp.md` | Pipe WhatsApp | Spec do kanban principal (F01). Colunas, DnD, regras de movimentação, estados de card. Referência direta para a fatia vertical na Fase 2. |
| 8 | `06 - Features/Comunicacao/Chat WhatsApp.md` | Chat | Modelo `contact_key`, 4 canais, batch de 8s. Referência para F04 (Inbox Multi-canal) e para entender o modelo de mensagem que o realtime consome. |
| 9 | `06 - Features/IA/Copilot.md` | Copilot | Wizard de 20+ steps, RAG, TTS. Complexidade UI mais alta do roadmap (F06). Ler antecipadamente para influenciar contratos de API. |
| 10 | `06 - Features/Automacao/Workflow Builder.md` | Workflow Builder | 12 node types, modelo de execução, persistência de draft. Referência para F07. Impacta decisão de state management (autômato vs store puro). |
| 11 | `07 - Feature Specs/org-quota-enforcement - spec.md` | Quota Enforcement | Modelo delta com requisitos REQ-Q01 a Q18. Define como quotas são verificadas, consumidas e reportadas. Base do `QuotaGauge` e do enforcement server-side. |
| 12 | `06 - Features/Admin/Master Admin.md` | Master Admin | 5 views + Operations Center. Referência para F16, que é a feature interna de governança. Informa quais dados o backend precisa expor via API dedicada. |
| 13 | `06 - Features/Admin/Checkout e Planos.md` | Checkout | Wizard 3 steps + integração PIX. Referência para F14. Criticamente sensível (dinheiro) e exige dupla revisão de segurança. |
| 14 | `06 - Features/Analytics/Performance.md` | Performance | 4 tabs, podium, badges. Referência para F09 e F10. Define métricas calculadas que o backend precisa agregar. |
| 15 | `06 - Features/Admin/Onboarding.md` | Onboarding | Gate de ativação + 6 steps. Referência para F13. Primeiro contato do usuário com o produto, alto impacto em retenção. |

## Protocolo de consulta

1. Antes de começar uma feature nova, localizar o doc correspondente nesta tabela.
2. Ler na íntegra. Anotar edge cases, decisões relevantes e lições aprendidas.
3. Verificar se algo do doc é um [[Itens Nao Migrados|item deliberadamente não migrado]].
4. Escrever nota nova no `Torque-dir-new` com as decisões adaptadas ao contexto atual.
5. Nunca criar wikilink apontando para o vault legado.
