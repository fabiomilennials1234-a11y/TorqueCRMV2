---
tipo: feature
dominio: vendas
pipeline: structural:propostas
---

# Pipeline Propostas

## Propósito

Terceiro pipeline estrutural. Onde o **valor é definido, a proposta é montada, a negociação acontece, e a venda é ganha ou perdida**. Integra com o catálogo de produtos para montar composição e calcular total. Gera comissão ao fechar.

## Stages Default

| Ordem | Nome | Descrição |
|---|---|---|
| 1 | `preparando_proposta` | Closer preparando. Proposta ainda não enviada ao lead. |
| 2 | `proposta_enviada` | Proposta apresentada ao lead (documento, áudio, reunião). |
| 3 | `negociando` | Lead fez contraproposta ou pediu ajustes. |
| 4 | `aguardando_decisao` | Lead pediu tempo; follow-up programado. |
| 5 | `vendido` | Fechado. Final positivo. |
| 6 | `perdido` | Recusado ou abandonado. Final negativo. |

Admin pode editar. As finais (`vendido`, `perdido`) são críticas — sistema dispara eventos de negócio (comissão, notificações).

## Atores e Permissões

- **Closer**: primário.
- **Admin**: tudo.
- **Copilot**: em configurações avançadas, agente pode operar, com cuidados extras (ver AI Actions em [[04 - Funcionalidades/IA/Copilot (Agentes IA)]]).
- **Workflow**: automações de lembrete/follow-up.

Ações: `pipeline.view:propostas`, `pipeline.move_entry:propostas`, `proposal.edit_items`, `proposal.apply_discount`, `proposal.mark_won`, `proposal.mark_lost`.

## Dados Envolvidos

Pipeline Entry de Propostas carrega `meta` rica:

```
{
  "items": [
    {
      "product_id": "uuid",
      "name_snapshot": "Produto X",
      "sku_snapshot": "SKU123",
      "quantity": 2,
      "unit_price": 1500.00,
      "discount": 0.10,   # 10% desconto unitário
      "total": 2700.00
    }
  ],
  "subtotal": 2700.00,
  "discount_total_percent": 0.10,
  "discount_total_value": 300.00,
  "total": 2700.00,
  "currency": "BRL",
  "valid_until": "2026-05-15",
  "notes": "texto livre",
  "payment_terms": "À vista",
  "won_at": null,
  "won_by": null,
  "lost_reason": null,
  "lost_at": null,
  "proposal_document_url": "https://..."
}
```

## Regras de Negócio

1. Entry é tipicamente criada ao lead chegar em `compareceu` de Confirmação (via regra de pipe).
2. Pode ser criada manualmente em casos específicos (ex.: venda sem reunião prévia).
3. Para `vendido`: `meta.total` obrigatório > 0, `meta.items` não-vazio.
4. Marcar `vendido` dispara cálculo de comissão (ver [[04 - Funcionalidades/Equipe/Comissões]]) e evento `LeadSold`.
5. Marcar `perdido`: `lost_reason` obrigatório (enum + texto livre).
6. `valid_until` passou sem decisão → pode disparar workflow de follow-up automático.
7. Desconto acima de X% (configurável por plano) requer aprovação de admin.
8. Snapshot de nome/SKU: proposta preserva valores do momento da criação; alteração no produto não retroage.
9. Lead pode ter múltiplas propostas históricas (repeat customer); entry ativa é uma por vez por default.

## Fluxos do Usuário

### Criar Proposta (via Confirmação)
1. Lead marcado como `compareceu` em Confirmação.
2. Regra cria entry em Propostas stage `preparando_proposta` automaticamente.
3. Closer abre entry.

### Criar Proposta (manual)
1. Drawer do lead → botão "Nova Proposta".
2. Entry criada em `preparando_proposta` vazia.

### Montar Proposta
1. Abre editor de proposta.
2. Adiciona items (busca produto por nome/SKU).
3. Define quantidade, preço unitário (default do produto, editável conforme permissão), desconto.
4. Sistema calcula subtotal, total.
5. Opcional: gerar documento PDF/link a enviar ao lead.
6. Define validade e condições de pagamento.

### Enviar Proposta
1. Editor com botão "Enviar ao lead".
2. Sistema envia mensagem no canal preferido (WhatsApp) com link/PDF.
3. Move stage para `proposta_enviada`.
4. `sent_at` registrado.

### Negociar
1. Lead pede ajuste.
2. Closer move para `negociando`.
3. Edita items (respeitando permissão de desconto).
4. Reenvia nova versão (registra versão histórica).

### Fechar (Vendido)
1. Closer clica "Marcar como Vendido".
2. Confirmação (valor, data de pagamento esperada).
3. Entry move para `vendido`.
4. Sistema:
   - Emite `LeadSold(lead_id, amount, items, closer_id)`.
   - Dispara cálculo de comissão.
   - Move lead em outros pipes como configurado (ex.: arquiva em WhatsApp).
   - Opcional: cria entry em pipe "Onboarding de Cliente".
   - Atualiza analytics.

### Perder
1. Closer clica "Marcar como Perdido".
2. Form: motivo (enum: preço, timing, concorrente, sumiu, qualificou-mal, outro) + texto.
3. Entry move para `perdido`.
4. Sistema emite `LeadLost(reason)`.
5. Workflow opcional: adiciona tag, cria follow-up para recontato em 6 meses.

## Automações e Eventos

### Emite
- `ProposalCreated`, `ProposalSent`, `ProposalUpdated`, `ProposalWon` (= `LeadSold`), `ProposalLost`.
- `LeadStageChanged(propostas, ...)`.
- `CommissionTriggered(member_id, value)`.

### Reage
- `LeadStageChanged(confirmacao → compareceu)` → cria entry em Propostas.
- `ProposalValidityExpired` (SLA) → dispara follow-up.

## Integrações

- **Produto**: catálogo para montar items.
- **Comissões**: cálculo automático em `vendido`.
- **Chat**: envio de proposta (link/PDF) ao lead.
- **Asaas / provedor de pagamento**: pode gerar cobrança ao fechar (opcional).
- **Document Generation**: gera PDF da proposta.
- **Pipeline Confirmação**: origem.
- **Workflow**: triggers e actions sobre esta pipe.
- **Analytics**: receita, ticket médio, taxa de ganho, LTV projetado.

## Edge Cases

- **Item com produto descontinuado**: usa snapshot; closer vê aviso.
- **Preço negativo**: rejeitado.
- **Múltiplas moedas num mesmo entry**: rejeita (regra: todos items em mesma moeda).
- **Reabrir proposta ganha**: exige permissão admin + auditoria; estorna comissão se já paga.
- **Split de comissão** (múltiplos closers na mesma venda): suportado via campo `meta.splits: [{member_id, percent}]`.
- **Moeda estrangeira**: suportada; valor em moeda nativa preservado, analytics converte por taxa do dia.

## Validações

- Items: ao menos 1 ao marcar vendido.
- `unit_price >= 0`, `quantity > 0`.
- `discount` em `[0, 1]`.
- `total` computado server-side (não confiar no cliente).
- `valid_until` futura na criação.
- Desconto global acima de threshold: permissão.
- `lost_reason` obrigatório ao marcar perdido.

## Métricas

- **Taxa de fechamento (win rate)**: `vendido / (vendido + perdido)`.
- **Ticket médio**: `média(total)` em `vendido`.
- **Tempo da proposta ao fechamento**: `won_at - entered_at`.
- **Taxa de perda por motivo**: distribuição dos `lost_reason`.
- **Receita projetada**: soma de `total` em stages não-finais (pipeline value).
- **Conversão stage-a-stage**.
- **Performance por closer**: ticket, win rate, tempo médio.
- **Produto mais vendido**: contagem e receita.
- **Desconto médio aplicado**.
