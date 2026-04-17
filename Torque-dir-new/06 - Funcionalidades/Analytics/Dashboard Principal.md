---
tipo: feature
dominio: analytics
---

# Dashboard Principal

## Propósito

Visão de abertura do admin: **métricas top-level** da operação em um golpe de vista. Objetivo é responder, em < 5 segundos, "como está meu negócio hoje?".

## Atores e Permissões

- **Admin**: vê org completa.
- **Membros**: versão restrita (próprias métricas).

Ações: `analytics.view:own`, `analytics.view:team`, `analytics.view:company`.

## Filtros

- **Período**: hoje, 7d, 30d, mês, trimestre, ano, custom.
- **Responsável**: todos / eu / membro específico.
- **Origem**: todos / filtro.
- **Comparação**: vs período anterior (toggle).

## Widgets Principais

### KPIs de topo (cards)
- **Leads novos**: contagem no período + delta% vs comparativo.
- **Leads qualificados** (atingiram stage X): contagem + delta.
- **Reuniões realizadas**: contagem + delta.
- **Vendas fechadas**: contagem + delta.
- **Receita**: soma de `meta.total` em propostas vendidas + delta.
- **Ticket médio**: `receita / vendas`.
- **Pipeline value**: soma de `meta.total` em entries não-finais + delta.
- **Taxa de conversão (lead → venda)**: `vendas / leads` no período.
- **Tempo médio de fechamento (lead time)**: média de `won_at - lead.created_at`.

### Funil
- Representação visual do funnel:
  - Leads criados → qualificados → reunião agendada → reunião realizada → proposta → venda.
- Cada stage com contagem e conversão para o próximo.

### Série Temporal
- Gráfico de linha: leads/dia (ou semana, conforme período).
- Múltiplas linhas: leads criados, qualificados, vendidos.

### Breakdown por Origem
- Gráfico de barras horizontal: leads por origem.
- Opcional: conversão por origem (overlay).

### Ranking (top membros)
- Top 5 SDR por leads qualificados.
- Top 5 Closer por receita.

### Atividade em Tempo Real
- Feed: últimas N ações do time (novo lead, reunião marcada, venda).
- Útil para "pulso" da operação.

### Alertas
- Lead aguardando atendimento há > Y min.
- Follow-up em atraso.
- Meta em risco.

## Métricas Calculadas (Fórmulas)

### Taxa de conversão (lead → venda)
```
conversao = vendas_no_periodo / leads_no_periodo × 100
```
Observação: denominador pode ser "leads criados no período" OU "leads que passaram pelo funil no período" — escolher e documentar.

### Ticket médio
```
ticket_medio = receita_total / qtd_vendas
```

### Lead time de venda
```
lead_time = média(won_at - lead.created_at) para vendas no período
```

### Velocity
```
velocity = leads_qualificados_por_dia × ticket_medio × taxa_fechamento
```

## Fontes dos Dados

- Consultas agregadas sobre:
  - Tabela de leads (com filtros de data).
  - Tabela de pipeline entries (stage transitions, status).
  - Tabela de mensagens/conversas (para tempo de resposta).
  - Tabela de propostas (items, valores).

Views materializadas recomendadas para períodos longos (mês/ano).

## Regras de Agregação

- Períodos em timezone da organização.
- "Leads novos no período" = `created_at BETWEEN start AND end`.
- "Vendas no período" = `won_at BETWEEN start AND end` (não `entered_at` da proposta).
- Comparativo: mesmo número de dias do período anterior.
- NULLs tratados (lead sem score → exclui de avg de score).

## Fluxos do Usuário

### Visualizar
1. Usuário loga → cai em dashboard.
2. Filtros aplicam default "últimos 30d, todos".
3. Widgets carregam paralelos (skeleton durante).
4. Interação: clicar em widget → deep-link para tela de análise.

### Comparar
- Toggle "vs período anterior" muda exibição para delta%.
- Cor verde = melhor, vermelho = pior.

### Export
- Botão "Export dashboard" → PDF ou imagem.

### Customizar (futuro)
- Admin pode arrastar widgets, esconder, adicionar custom.

## Automações

### Sem emissão própria
- Dashboard é leitura pura.

### Consumo
- Reage a filtros em URL (compartilhar deep link).
- Atualização realtime opcional (widgets atualizam com debounce em eventos).

## Edge Cases

- **Sem dados no período**: estado vazio com CTA ("Crie seu primeiro lead").
- **Período muito longo** (365 dias): usar view materializada, caso demore: skeleton + "Calculando...".
- **Múltiplas moedas** em vendas: converte tudo para moeda padrão da org usando taxa fixa (configurável) ou do dia.
- **Timezone**: converte datas para fuso da org.
- **Métrica indefinida** (divisão por zero): mostra "—" em vez de erro.

## Performance

- Widgets paralelos.
- Cache de 60s em métricas pesadas (invalidação em eventos).
- Views materializadas refreshed em background (cron).
- Queries com indexes adequados.
- Lazy-load de widgets fora da viewport.

## Validações

- Filtro de período válido (start < end, não muito distante).
- Usuário tem permissão para ver escopo.

## Acessibilidade

- Gráficos com descrição textual alternativa.
- Cores não únicas para encoding (também formas/labels).
- Navegação por teclado.

## UX

- Dark-first.
- Grid responsivo (reorganiza em mobile).
- Atalhos: `G D` para voltar ao dashboard.
- Pin/unpin de widgets.
