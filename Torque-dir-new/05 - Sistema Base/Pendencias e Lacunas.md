---
tags:
  - sistema-base
  - pendencias
  - decisoes
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Pendências e Lacunas

Itens conhecidos que **não bloqueiam o Sistema Base** mas precisam ser resolvidos antes de features específicas que dependem deles. Registrados aqui para não se perderem.

## Microcopy e tom de voz

**Estado:** referências validadas (Linear, Stripe), falta documento de princípios com exemplos concretos.

**Decisão:** seguir tom **direto, seco, confiante, sem exclamações, sem emojis**. Detalhar conforme features chegam — cada feature nova contribui com exemplos reais de seus copies para um catálogo incremental.

**Quando resolver:** na primeira feature que precisar de copy denso (provavelmente F04 Inbox ou F06 Copilot).

## Charts

**Estado:** decisão tomada.

**Decisão:**
- **Visx** para telas signature: Dashboard, Analytics, TV
- **Primitivos custom** para sparklines e gauges (`ScoreMeter` e `Sparkline` já existem)
- **Recharts** apenas onde velocidade de entrega importa mais que refino visual

**Quando resolver:** documentar oficialmente na primeira feature de analytics (F09).

## i18n

**Estado:** decisão tomada.

**Decisão:** estruturar `react-intl` desde a base. Apenas **PT-BR ativo**. Chaves organizadas por feature (`features/<feature>/i18n/pt-BR.json`).

**Quando resolver:** já no Sistema Base — entra como item da fundação.

## Skeleton patterns por tela

**Estado:** primitivo `Skeleton` existe, variantes por tela ainda não.

**Decisão:** definir **ao entrar em cada feature**. Cada feature descreve seus próprios skeleton variants no seu `Design.md`. Não faz parte do Sistema Base.

## Tom de empty states

**Estado:** primitivo `EmptyState` existe, catálogo de copy específico não.

**Decisão:** construir **feature-a-feature**. Cada feature define seus estados vazios no `Spec.md` / `Design.md`. Base fornece apenas o componente.

---

## Referências

- [[Escopo do Sistema Base]]
- [[Checklist Sistema Base]]
- [[Plano de Execucao]]
- [[00 - Mapa de Features]]
