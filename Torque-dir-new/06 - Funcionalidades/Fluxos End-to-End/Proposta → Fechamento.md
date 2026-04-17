---
tipo: fluxo
---

# Proposta → Fechamento

Do comparecimento em reunião até o desfecho (venda ou perdida). Atravessa Pipeline Propostas, Produtos, Comissões, Workflow de fechamento.

## Diagrama

```
Reunião realizada (compareceu)
          │
          ▼
Entry em Propostas "preparando_proposta"
          │
          ▼
Closer monta proposta:
  - Adiciona produtos
  - Define valores, descontos
  - Gera documento (PDF/link)
          │
          ▼
Envia proposta ao lead (mensagem)
Stage: "proposta_enviada"
          │
          ▼
Lead responde:
  ├── Aceita  ──────────────────────────►
  ├── Pede ajuste ─► "negociando"
  │                      │
  │                      ▼
  │                  Proposta v2 enviada
  │                      │
  │                      ▼ [loop até fechar ou perder]
  │
  ├── Silencia ─► SLA → workflow follow-up
  └── Recusa ──► "perdido" (motivo)
                         │
                         ▼
                    Workflow reciclagem
                    (follow-up +6m, campanha reengajamento)

Aceitou:
          ▼
Closer marca "Vendido"
          │
          ▼
Sistema dispara:
  ├── Emite LeadSold(amount, items, closer, sdr)
  ├── Calcula Comissão (split SDR/Closer)
  ├── Cria pedido no ERP (se integrado)
  ├── Adiciona tag "cliente"
  ├── Remove de pipes ativos (ou arquiva)
  ├── Cria entry em Onboarding Cliente (custom pipe)
  ├── Envia mensagem de agradecimento
  ├── Notifica membros envolvidos
  └── Atualiza métricas (receita, ticket, win rate)
```

## Passo a Passo

### 1. Abertura da Proposta
- Entry criado automaticamente em `preparando_proposta` após `compareceu` em Confirmação.
- Closer abre o editor.

### 2. Montagem
- Busca produtos por nome/SKU.
- Adiciona items: qty, preço unitário (default do produto), desconto.
- Sistema calcula subtotal e total.
- Define validade (`valid_until`) — padrão 15 dias.
- Condições de pagamento.
- Observações.
- Gera documento: PDF ou link hospedado.

### 3. Envio
- Closer clica "Enviar proposta ao lead".
- Sistema envia mensagem com template:
```
Olá {{lead.name}}, segue nossa proposta:

{{proposal.summary}}

Total: R$ {{proposal.total}}
Válida até: {{proposal.valid_until}}

Link completo: {{proposal_document_url}}
```
- Stage move para `proposta_enviada`.
- `sent_at` registrado.

### 4. Negociação
- Lead responde com ajustes.
- Closer move para `negociando`.
- Edita items (respeitando permissões de desconto).
- Reenvia nova versão (armazena histórico de versões).
- Volta para `proposta_enviada`.

### 5. Follow-up de Validade
- Workflow: se passar X dias sem resposta, mensagem de lembrete.
- Se `valid_until` vence: stage move para `aguardando_decisao` ou workflow específico.

### 6. Desfecho Positivo
- Closer clica "Marcar como Vendido".
- Confirmação com valor final e data de pagamento esperado.
- Sistema:
  - `meta.won_at = now`, `meta.won_by = closer_id`.
  - Stage `vendido`.
  - Emite `ProposalWon` + `LeadSold`.
  - **Comissão**: calculadora dispara.
    - Resolve rule aplicável (per-member > per-product > specialty > global).
    - Aplica split (SDR % + Closer %).
    - Cria Commission Entries status=`pending` para cada papel.
  - **ERP**: se integrado, cria pedido no ERP (TinyERP).
  - **Tags**: adiciona `cliente`.
  - **Arquiva** lead nos outros pipes ativos (ou conforme config).
  - **Onboarding Cliente**: cria entry se existe pipe customizado configurado.
  - **Mensagem de agradecimento** automática.
  - **Notificações**: SDR (se envolvido), admin, equipe.
  - **Analytics**: atualiza receita, win rate, ticket médio.

### 7. Desfecho Negativo
- Closer clica "Marcar como Perdido".
- Form obrigatório:
  - `lost_reason`: enum (preço, timing, concorrente, sumiu, qualificou-mal, outro).
  - Comentário livre.
- Stage `perdido`.
- Emite `ProposalLost` + `LeadLost`.
- Workflow opcional:
  - Tag `perdido:preço` (ou outra razão).
  - Follow-up agendado para 6 meses (recontato).
  - Entrada em campanha de reengajamento.

## Regras de Negócio

1. `vendido` exige `meta.items` não-vazio e `meta.total > 0`.
2. `perdido` exige `lost_reason`.
3. Desconto acima de X% requer permissão admin (configurável por plano).
4. Preços snapshot: alteração no produto não retroage.
5. Reabrir proposta ganha: admin-only + confirmação + estorno de comissão.
6. Dois closers na mesma venda: split manual via `meta.splits`.

## Métricas-chave

- **Win rate**: `vendidos / (vendidos + perdidos)`.
- **Ticket médio**.
- **Tempo proposta → fechamento**.
- **Receita por closer**.
- **Motivo de perda distribution**.
- **Taxa de retorno de propostas perdidas** (lead perdido volta e fecha).

## Eventos Emitidos

- `ProposalCreated`.
- `ProposalSent`.
- `ProposalUpdated` (versões).
- `ProposalWon` → `LeadSold`.
- `ProposalLost` → `LeadLost`.
- `CommissionCalculated`.

## Integrações

- **Pipeline Propostas**: core.
- **Produtos**: catálogo.
- **Comissões**: cálculo em vendido.
- **ERP**: criação de pedido.
- **Workflow**: follow-ups, validade.
- **Copilot**: pode gerenciar negociação (com cuidados).
- **Chat**: envio de proposta e comunicação.
- **Analytics**: receita, win rate, ticket.
- **Google Calendar**: possivelmente marcar próximas reuniões de negociação.

## Pitfalls Comuns

- **Proposta enviada sem valor**: bloquear no backend.
- **Closer não marca resultado** (fica em `proposta_enviada` eternamente): SLA + alert.
- **Cálculo de comissão errado**: revisar rules ativas; testar no setup.
- **Reabertura frequente** (vende, desfaz): sinal de problema no processo de closing.
- **Desconto descontrolado**: permissão + alerta.

## Otimizações

- **Template de proposta**: biblioteca de propostas-modelo por segmento.
- **Aprovação de desconto**: workflow exige admin autorizar antes de enviar.
- **Comparativo de performance**: closers viam win rate; competição saudável.
- **Saída inteligente**: agente IA lê resposta do lead e sugere próxima ação.
