---
tipo: feature
dominio: analytics
---

# Analytics Comercial

## Propósito

Visão **detalhada do funil comercial**: conversão stage-a-stage, tempo por stage, gargalos, propostas ganhas vs perdidas, motivos de perda, ticket médio. Complementa o Dashboard Principal com profundidade.

## Atores e Permissões

- **Admin**: acesso total.
- **Membros** (SDR/Closer): versão com próprios dados.

Ações: `analytics.view:company`, `analytics.view:own`.

## Widgets

### Funil Completo (ilustrado)

Gráfico de funil com as 7+ etapas canônicas:
1. Lead criado (100%).
2. Lead abordado.
3. Lead respondeu.
4. Reunião agendada.
5. Reunião realizada.
6. Proposta enviada.
7. Proposta ganha.

Para cada transição: conversão (%) + volume absoluto.

### Tempo por Stage

Tabela ou boxplot:
- Stage.
- Tempo médio, mediano, p90.
- Total de entries que passaram.

Sinaliza **gargalos**: stages com tempo excessivo comparado a baseline.

### Conversão por Origem

Tabela: origem × métricas (taxa por stage, conversão final, ticket médio).
Identifica canais mais qualificados.

### Win Rate por Motivo de Perda

Pizza/donut: distribuição de `lost_reason` em propostas perdidas.
Ajuda a priorizar objeções (ex.: 40% perde por preço → revisar pricing).

### Cohort Analysis

Leads criados em semana X → % ainda ativos em semana X+1, +2, +3, +4.
Mostra retenção e velocidade de decisão.

### Heatmap de Dia/Hora

Quando leads entram? Respondem? Convertem?
Útil para dimensionar plantão e campanhas.

### Top/Bottom Performers

- Top 5 closers por receita.
- Top 5 por win rate.
- Members com conversão abaixo de X% (investigação).

## Métricas Calculadas

### Conversão stage A → B
```
conv_A_to_B = count(entries_que_chegaram_em_B) / count(entries_que_passaram_por_A)
no período filtrado
```

### Tempo médio em stage
```
tempo_stage = média(moved_to_current_at - entered_stage_X_at)
```

### Win rate
```
win_rate = propostas_ganhas / (ganhas + perdidas) em período
```

### Lost rate por motivo
```
para cada reason:
  ratio = count(perdidas com reason=X) / total_perdidas
```

### Velocidade (velocity)
```
velocity = leads_qualificados × taxa_fechamento × ticket_medio / dias_periodo
# unidade: R$/dia
```

## Fluxos do Usuário

### Navegar
1. Menu → `Analytics → Comercial`.
2. Seletor de período (default 30d).
3. Seletores secundários: pipeline (se múltiplos), responsável (se admin).
4. Cada widget é interativo:
   - Click em barra do funil → lista de leads que estão ali.
   - Click em motivo de perda → leads específicos perdidos por essa razão.
   - Hover → detalhes.

### Drill-down
- De conversão → lista de leads.
- De lead → drawer.
- Ciclo completo: métrica → individual → ação.

### Comparar Períodos
- Toggle "comparar com período anterior".
- Exibição dual ou delta.

### Exportar
- CSV/PDF com filtros aplicados.

## Dimensões e Filtros

- **Período** (obrigatório).
- **Pipeline** (se múltiplos, especialmente custom).
- **Responsável**.
- **Origem**.
- **Tag**.
- **Valor de venda range** (mínimo, máximo).
- **Produto** (em quais vendas aparece).

## Fontes de Dados

- Lead + Pipeline Entries + Lead History (para transições datadas).
- Propostas (pipe Propostas meta + items).
- Times (para breakdown por membro).
- Tags (para segmentação).

## Regras de Agregação

- Transições contadas 1x mesmo se lead revisitar stage.
- Tempo em stage: usa último período contínuo na stage (caso lead entrou e saiu múltiplas vezes).
- Lead criado pré-período mas convertido dentro: conta na conversão do período mas não em "leads criados".
- Definir convenção e manter consistente.

## Edge Cases

- **Org pequena com poucos dados**: métricas ruidosas; UI mostra aviso "amostra pequena" em < 30 eventos.
- **Lead com múltiplos entries históricos**: analytics escolhe entry mais recente por default.
- **Mudanças de pipeline histórico**: stages renomeadas mantêm continuidade via id.
- **Lead cross-pipe** (em múltiplos pipes): funil agrega presença em qualquer sem duplicar.

## Performance

- Views materializadas atualizadas em cron (ex.: diariamente 3h).
- Caching de 5 min em queries de usuário ativo.
- Particionamento de tabelas grandes por mês.
- Queries paralelas.

## Validações

- Período razoável (≤ 2 anos).
- Filtros consistentes.

## Métricas (meta-metrics)

- Tempo de carga do dashboard (p95).
- Queries mais lentas (para otimização).
- Uso de export.

## UX

- Gráficos responsivos.
- Tooltips ricos.
- Dark-first.
- Legendas claras.
- Cores indicando bom/ruim contextual (verde/vermelho relativo a benchmark).
