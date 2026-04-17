---
tipo: feature
dominio: equipe
---

# Comissões

## Propósito

Cálculo e gestão de comissões sobre vendas. Automatiza o que tradicionalmente é feito em planilhas: quando lead é marcado como vendido, sistema calcula comissão conforme regras configuradas, cria registro auditável, permite aprovação e exportação.

## Atores e Permissões

- **Admin**: configura regras, aprova, exporta.
- **Membro**: vê sua própria comissão.
- **Sistema**: calcula automaticamente em `ProposalWon`.

Ações: `commission.view:own`, `commission.view:all`, `commission.edit_rules`, `commission.approve`, `commission.export`.

## Dados Envolvidos

### Commission Rule
- `id`, `organization_id`.
- `name`.
- `scope`: `global` | `product` | `member` | `specialty` | `campaign`.
- `scope_ref`: referência (product_id, member_id, specialty, campaign_id).
- `calculation_type`: `percent_of_total` | `percent_of_margin` | `fixed_per_sale` | `tiered`.
- `value`: JSON conforme type (ex.: `{percent: 0.10}` ou `{tiers: [{min: 0, max: 10000, percent: 0.05}, {min: 10000, percent: 0.08}]}`).
- `role_split`: opcional (ex.: SDR recebe 30%, Closer 70%).
- `is_active`.
- `priority`: ordem de resolução quando múltiplas rules match.
- `effective_from`, `effective_until`.

### Commission Entry (registro histórico)
- `id`, `organization_id`.
- `member_id`.
- `lead_id`, `pipeline_entry_id` (venda origem).
- `rule_id` aplicada.
- `base_value` (valor da venda).
- `commission_value` (valor calculado).
- `percent` (se aplicável).
- `role_in_sale`: `sdr` | `closer` | `responsible` | `sole`.
- `status`: `pending` | `approved` | `paid` | `cancelled`.
- `calculated_at`, `approved_at`, `paid_at`.
- `approved_by`, `paid_by`.
- `notes`.

## Regras de Negócio

1. Cálculo automático ao `LeadSold` / `ProposalWon`.
2. Sistema resolve qual rule aplicar:
   - Prioridade: member-specific > product-specific > specialty > global.
   - Se múltiplas mesmo escopo: `priority` ASC.
3. Split por papel: se rule tem `role_split`, cria múltiplas Commission Entries (uma por role envolvido).
   - Ex.: venda de R$ 10k, rule 10% com split SDR 30% / Closer 70%:
     - SDR recebe R$ 300.
     - Closer recebe R$ 700.
4. Se lead **sem** SDR assignment no momento da venda: split do SDR é descartado ou redirecionado (config).
5. Commission em `pending` pode ser ajustada ou rejeitada pelo admin.
6. `approved`: pronta para pagamento, ainda não paga.
7. `paid`: registro imutável (alteração via nova entrada de estorno).
8. Cancelamento de venda (reabrir proposta perdida): estorna comissão criando entry negativa.
9. Moeda: entry sempre em moeda da venda; relatório converte.

## Fluxos do Usuário

### Configurar Regras
1. `Configurações → Comissões`.
2. Lista de rules ativas/inativas.
3. Criar nova: form com scope, calculation_type, value, role_split.
4. Preview: simulação com venda exemplo.
5. Salvar.

### Ver Comissões (membro)
1. Menu → `Comissões`.
2. Lista de entries do próprio user.
3. Filtros: período, status.
4. Resumo: total pending, approved, paid no mês.
5. Detalhe de cada entry: qual venda, valor base, % aplicada.

### Aprovar Comissões (admin)
1. Menu → `Comissões → Revisão`.
2. Lista entries `pending` agrupadas por membro.
3. Admin pode:
   - Aprovar individual.
   - Aprovar em bulk (selecionar todas).
   - Ajustar valor (com motivo).
   - Rejeitar (com motivo).
4. Em aprovação: status = `approved`, timestamp, approved_by.

### Marcar como Pago
1. Admin exporta relatório, paga externamente, volta para marcar.
2. Ou integração futura com folha de pagamento.
3. `paid_at`, `paid_by`, `status=paid`.

### Estornar
1. Lead "vendido" reaberto como "perdido" (raro, exige permissão).
2. Sistema cria entry negativa compensando comissão paga.
3. Admin decide recuperar (desconto em comissão futura) ou absorver.

### Exportar
- CSV/Excel com filtros aplicados.
- Uso: folha de pagamento, contabilidade.

## Cálculo — Exemplos

### Rule: 10% do total
```
venda = R$ 5.000
comissão = 5.000 × 0.10 = R$ 500
```

### Rule: tiered
```
venda = R$ 15.000
tier 1 (0-10k): 5% → 500
tier 2 (10-20k): 8% → 400 (sobre 5k acima de 10k)
comissão total = 900
```

### Rule: fixo por venda
```
venda de qualquer valor → R$ 300 por venda
```

### Rule: split por role
```
venda = R$ 10.000
rule: 10% com split SDR 30% / Closer 70%
total comissão = R$ 1.000
SDR entry: R$ 300
Closer entry: R$ 700
```

### Rule por produto
```
venda tem produto A (R$ 3k) + produto B (R$ 2k) = R$ 5k total
produto A tem rule 10%, produto B tem rule 15%
comissão = 3000×0.10 + 2000×0.15 = 300 + 300 = R$ 600
```

## Automações e Eventos

### Emite
- `CommissionCalculated(entry_id, member, value)`.
- `CommissionApproved`, `CommissionPaid`, `CommissionCancelled`, `CommissionReversed`.

### Reage
- `LeadSold` / `ProposalWon` → cálculo.
- `ProposalReopened` → estorno.
- `MemberRemoved` → comissões pending atribuídas a membro removido: admin decide.

## Integrações

- **Pipeline Propostas**: origem das vendas.
- **Produto**: commission_config override.
- **Membro**: commission_config override.
- **Analytics**: dashboards financeiros.
- **Audit**: cada ação registrada.
- **Futuro**: integração com folha de pagamento / ERP financeiro.

## Edge Cases

- **Venda sem SDR**: split descarta SDR ou redireciona a admin (config).
- **Venda com múltiplos closers** (split manual): admin configura manualmente no campo `meta.splits`.
- **Mudança de rule após venda**: entries existentes mantêm rule aplicada (snapshot).
- **Membro saiu da empresa com comissão pending**: admin decide pagar ou não (em conformidade com contrato).
- **Moeda estrangeira**: entry em moeda original; relatório agrega em moeda padrão.
- **Proposta com desconto grande**: comissão calculada sobre total final (pós-desconto) por default; config pode calcular sobre bruto.
- **Split inválido** (soma % != 100): valida ao criar rule.

## Validações

- `value`: schema conforme type.
- Tiers: ranges não se sobrepõem, cobrem faixas monotonicamente.
- Split: percentuais não-negativos, soma ≤ 1.0.
- Effective dates: start < end.

## Métricas

- Total comissões do período (pending, approved, paid).
- Comissão média por venda.
- Top earners.
- Commission-to-revenue ratio (% da receita indo a comissão).
- Tempo médio de aprovação.
- Estornos (sinal de churn ou problema).

## Segurança

- Dados financeiros sensíveis: visibilidade restrita.
- Membro vê só suas; admin vê todas.
- Export em audit log com quem/quando.
- Valor nunca editável após `paid` (estornos criam entry nova).
