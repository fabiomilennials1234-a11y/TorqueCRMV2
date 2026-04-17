---
tipo: feature
dominio: analytics
---

# Analytics de UTMs

## Propósito

Atribuição de leads às campanhas de marketing via **parâmetros UTM** (source, medium, campaign, term, content). Responde: quais campanhas pagas e orgânicas geram leads, qual converte, qual custa menos, qual tem maior LTV.

## Atores e Permissões

- **Admin**: acesso total.
- **Prospectador**: acesso à sua campanha (se integração associa UTMs a campanhas).

Ações: `analytics.view:company`, `analytics.view:outbound`.

## Dados Envolvidos

Cada lead pode ter UTMs:
- `utm_source` (ex.: `facebook`, `google`, `newsletter`).
- `utm_medium` (ex.: `cpc`, `organic`, `email`).
- `utm_campaign` (ex.: `launch-2026-q2`).
- `utm_term` (keyword).
- `utm_content` (variante A/B).

Entrada: via webhook de ingestão, form público, ou integração (n8n passa os valores).

## Métricas Calculadas

### CPL (Custo Por Lead)
```
CPL = gasto_em_campanha / leads_gerados
```
Requer que o custo seja informado manualmente ou via integração com plataforma de ads.

### CAC (Customer Acquisition Cost)
```
CAC = gasto_em_campanha / clientes_fechados
```

### ROAS (Return On Ad Spend)
```
ROAS = receita_gerada_pelos_leads_dessa_campanha / gasto_em_campanha
```

### Taxa de Qualificação por UTM
```
qualified_rate = leads_qualificados_dessa_campanha / total_leads_dessa_campanha
```

### LTV por Origem
```
ltv = soma_receita_total (incluindo upsells) / qtd_clientes_dessa_origem
```

## Widgets

### Tabela Principal
Colunas:
- Source.
- Medium.
- Campaign.
- Leads gerados.
- Taxa de resposta.
- Leads qualificados.
- Taxa de qualificação.
- Leads convertidos.
- Taxa de conversão.
- Receita.
- Gasto (se informado).
- CPL.
- CAC.
- ROAS.

Ordenação e filtragem.

### Gráfico de Barras: Top 10 Campanhas por Leads
Visual comparativo.

### Gráfico de Barras: Top 10 Campanhas por Receita
Distinto do acima — volume de leads ≠ receita.

### Gráfico de Pizza: Distribuição de Origens
`utm_source` breakdown no período.

### Funnel por UTM Source
Funnel para cada top source — permite comparar comportamento do lead por canal.

### Tendência Temporal
Linha por campanha selecionada: leads/dia ao longo do tempo.

## Filtros

- Período.
- Source (multi-select).
- Medium.
- Campaign (autocomplete com todos os utm_campaign conhecidos).
- Gasto > 0 apenas (filtra campanhas sem tracking de custo).

## Entrada de Custo

### Manual
- Admin entra `utm_costs` na UI:
  - Campaign id (string).
  - Período.
  - Gasto em moeda.

### Integração (futuro)
- Conector com Meta Ads, Google Ads: pull automático de gasto por campaign.

## Regras de Negócio

1. Leads sem UTMs são categorizados como `organic_or_direct` por default.
2. Atribuição é **first-touch**: UTM na criação do lead é preservada mesmo se lead interage com outras campanhas depois.
3. Multi-touch (futuro): considera sequência de interações — requer feature de tracking adicional.
4. Receita atribuída ao lead que converteu; incluindo upsells (extensível).

## Fluxos do Usuário

### Análise de ROAS
1. Abre Analytics UTMs.
2. Seleciona período (ex.: mês passado).
3. Filtra campanhas com gasto > 0.
4. Ordena por ROAS DESC.
5. Identifica campanhas mais eficientes → replica.
6. Identifica campanhas com ROAS baixo → corta ou otimiza.

### Comparar Variantes A/B
1. Filtra `utm_campaign` = "launch-2026-q2".
2. Agrupa por `utm_content` (variant_a vs variant_b).
3. Compara leads, qualificação, conversão.

### Drill Down
- Click em campanha → lista de leads atribuídos.
- Click em lead → drawer.

## Automações

### Sem emissão
- Read-only.

### Consumo
- Reage a dados dos leads. Recalcula em eventos.

## Integrações

- **Lead**: fonte de UTMs.
- **Pipeline**: conversão.
- **Propostas**: receita.
- **Future: Meta Ads, Google Ads**: pull de gasto.
- **Campanhas internas**: não confundir — são coisas diferentes (campanhas externas de ads vs. campanhas outbound do Torque).

## Edge Cases

- **Lead sem UTMs**: `(direct)` / `(not_set)` como valor.
- **UTMs inválidos** (string com caractere estranho): sanitiza ao armazenar.
- **Campanha sem gasto informado**: mostra "—" em CPL/CAC/ROAS; não calcula.
- **Atribuição duplicada** (lead criado 2x): dedupe deve impedir; se falha, escolhe o primeiro para atribuição.
- **Moedas diferentes**: converte para padrão da org.

## Validações

- UTMs: ≤ 255 chars cada.
- Custos: ≥ 0.

## Métricas

- Coverage: % de leads com UTMs preenchidas.
- Top campanhas.
- Evolução mês a mês.

## UX

- Tabela densa mas legível.
- Ordem padrão por "leads" DESC.
- Cor (verde/vermelho) indicando ROAS bom/ruim relativo.
- Deep link para lista de leads filtrada por UTMs.
