---
tags: [migracao, legado, vault, referencia]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Mapa do Vault Legado

Vault anterior do projeto Torque está em:
`C:\Users\torch\Desktop\milennials\v8milennialsb2bv2\Obsidian\Segundo Cerebro\Torque-wiki`

Este documento mapeia a estrutura top-level do legado para que, durante a construção do Torque-v2, seja possível consultar pontos específicos sem se perder. Complementa [[Top 15 Docs Prioritarios]] e [[Itens Nao Migrados]].

## Estrutura principal

### `01 - Identidade/`
Comportamentos esperados do agente, permissões operacionais, estilo de trabalho. Útil para comparar com a nova identidade documentada no Torque-v2.

### `02 - Arquitetura/`
Visão geral do sistema, inventário de módulos, specs detalhadas:
- `ARCHITECTURE` — padrões gerais (hooks, React Query, multi-tenancy, edge functions)
- `CONCERNS` — débitos técnicos (S1 a S6, D1 a D5) que NÃO devem reaparecer
- `CONVENTIONS` — naming, estrutura de pastas, query keys, padrões de import
- `INTEGRATIONS` — integrações externas (WhatsApp, pagamentos, IA)
- `STACK` — stack técnica completa do legado
- `STRUCTURE` — organização detalhada do repositório
- `TESTING` — estratégia de testes vigente
- `Modulos.md` — inventário completo de páginas, hooks, edge functions

### `03 - Operacional/`
Fluxos operacionais, limitações conhecidas, gotchas, runbooks. Fonte de "coisas que dão errado" para evitar no Torque-v2.

### `04 - Decisoes/`
ADRs do legado. Principal: `ADR-2026-04-12-arquitetura-inicial.md` com decisões D1 a D9. Nota: **D5 está obsoleta** (era Supabase BaaS; foi substituída pela decisão de backend Go próprio). Demais decisões seguem úteis como referência.

### `05 - Log de Contexto/`
Registros de sessão por data. Mais recente: `2026-04-12`. Serve como memória de trabalho das últimas decisões antes da virada para o Torque-v2.

### `06 - Features/`
Documentação por domínio, organizada em subpastas:
- `Admin/` — permissões, master admin, checkout, onboarding
- `Analytics/` — performance, podium, badges
- `Automacao/` — workflow builder
- `Comunicacao/` — chat WhatsApp, multi-canal, follow-ups
- `Equipe/` — gestão de usuários e papéis
- `IA/` — Copilot (wizard, RAG, TTS)
- `Integracoes/` — catálogo de integrações
- `Vendas/` — Pipe WhatsApp, Pipe Confirmação, Pipe Propostas

### `07 - Feature Specs/`
Specs no formato SDD (Spec-Driven Development): cada feature com `design`, `tasks`, `spec`. Granularidade de implementação. Úteis para ter paralelo direto na construção de features equivalentes no Torque-v2.

## Estratégia de migração

**Consultar o legado como referência, NÃO copiar.**

O vault `Torque-dir-new` é a nova fonte de verdade deste ciclo. Quando uma feature do Torque-v2 tem correspondente no legado, o fluxo é:

1. Identificar a doc do legado via [[Top 15 Docs Prioritarios]].
2. Ler para entender decisões, edge cases, aprendizados acumulados.
3. Decidir ativamente o que vai para o Torque-v2 e o que fica fora (ver [[Itens Nao Migrados]]).
4. Escrever a nota do Torque-v2 do zero, referenciando aprendizados do legado mas sem copiar estrutura nem texto.
5. Wikilinks apontam **exclusivamente** para notas do `Torque-dir-new`, nunca para o legado.

### Por que não copiar

- O legado tem débitos técnicos (CONCERN-S1 a S6, CONCERN-D1 a D5) que não devem se reproduzir.
- O backend muda de Supabase para Go, o que invalida parcelas significativas de decisões do legado.
- A reescrita obriga revisão crítica de cada conceito, em vez de aceitar heranças implícitas.
- Dois vaults com referências cruzadas viram um emaranhado imantendo em 3 meses.

### Quando o legado é intocável referência

- Modelagem de domínio (entidades, relações, campos) — geralmente boa e revalidada.
- Edge cases de produto encontrados em produção — aprendizado caro, preservar.
- Especificações detalhadas de wizards, fluxos, permissões — podem ser replicadas em estrutura mas não em texto.

### Quando ignorar

- Tudo relacionado a Supabase, Deno edge functions, pg_cron, pg_net, RLS via SQL.
- Decisões arquiteturais associadas ao modelo BaaS.
- Ver [[Itens Nao Migrados]] para lista explícita.
