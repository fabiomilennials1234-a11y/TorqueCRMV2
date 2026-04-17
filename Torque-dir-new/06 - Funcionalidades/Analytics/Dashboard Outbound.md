---
tipo: feature
dominio: analytics
---

# Dashboard Outbound

## Propósito

Métricas dedicadas à **prospecção ativa** (outbound): mensagens enviadas, taxa de resposta, performance de sequências em campanhas, tempo para primeira resposta, leads convertidos via outbound. Foco do prospectador e do gestor de growth.

## Atores e Permissões

- **Admin**: acesso total.
- **Prospectador**: acesso ao seu (e se admin permite, ao time).

Ações: `analytics.view:outbound`.

## Widgets

### KPIs Principais
- **Mensagens enviadas** no período.
- **Taxa de resposta** média.
- **Leads engajados** (responderam ao menos 1x).
- **Leads qualificados via outbound** (responderam + avançaram no funil).
- **Ticket médio de vendas-outbound**.
- **Mensagens por resposta** (eficiência — menos é melhor).

### Sequência de Campanhas
- Por campanha ativa: taxa de resposta por stage da sequência.
- Identifica stage onde lead responde mais (otimizar).

### Performance por Prospectador
- Tabela: mensagens enviadas, respostas, qualificados, conversões.
- Ranking.

### Bloqueios e Reports
- Taxa de lead que bloqueou número.
- Taxa de lead que reportou spam.
- Alertas de risco de reputação do canal.

### Horário Ideal
- Heatmap: hora do dia × dia da semana com taxa de resposta.
- Base para ajustar dispatch window.

### Comparação de Templates
- Top 5 templates com maior taxa de resposta.
- Bottom 5 (candidatos a revisar).

### Funnel Outbound
- Enviados → Respondidos → Qualificados → Reunião agendada → Venda.

## Métricas Calculadas

### Taxa de resposta
```
response_rate = leads_que_responderam / leads_que_receberam × 100
```

### Taxa de resposta do template
```
template_response_rate = respostas_iniciadas_em_template_X / envios_do_template_X × 100
```

### Eficiência (mensagens por resposta)
```
msg_per_response = mensagens_enviadas / leads_que_responderam
```

### Taxa de bloqueio
```
block_rate = leads_que_bloquearam / leads_que_receberam × 100
```

### Lead time outbound (envio → resposta)
```
lead_time = média(first_response_at - first_outbound_at)
```

## Filtros

- Período.
- Campanha específica.
- Canal.
- Prospectador.
- Template.

## Fontes

- Conversas + mensagens outbound.
- Entries de campanhas.
- Leads com origem `outbound` ou enrolados em campanha.

## Edge Cases

- **Canal com muitos bloqueios**: alerta vermelho, sugere pausar.
- **Taxa de resposta 0% em campanha**: indica problema (template, público, timing).
- **Lead respondeu após period**: ainda contabilizado via `first_response_at`.
- **Lead responde a mensagem antiga (fora de campanha ativa)**: atribuição é contextualizada.

## Performance

- Views materializadas diárias.
- Métricas instantâneas para período curto (hoje, 7d) em cache.

## Validações

- Período válido.
- Permissão.

## UX

- Alertas destacados (taxa de bloqueio alta, template com < 5% resposta).
- Sugestões inline ("Template X tem taxa de resposta 20% acima da média — considere aumentar uso").
- Dark-first.
