---
tipo: integracao
direcao: bidirectional
criticidade: alta
---

# Asaas (Provedor de Pagamento)

Integração com Asaas (provedor brasileiro) para cobrança recorrente das assinaturas do Torque E para cobrança de clientes finais das organizações (opcional).

## Propósito

- **Cobrança das assinaturas do próprio Torque**: cliente paga mensalidade via cartão/boleto/Pix → webhooks confirmam.
- **Opcional: cobrança dos clientes finais**: organização pode emitir cobranças aos seus próprios leads vendidos (ex.: "gerar boleto após venda") direto pelo Torque.

## Contrato

- API REST do Asaas.
- Webhooks de eventos financeiros.

## Autenticação

- API key do Asaas por organização (no contexto "emitir cobrança para cliente final").
- API key master (do Torque) para cobranças do próprio Torque.
- Vault cifrado.

## Endpoints Consumidos

### Customer
- `POST /customers` — criar cliente.
- `GET /customers/:id` — detalhe.
- `PATCH /customers/:id` — atualizar.

### Subscription (recorrência — usado para assinatura do Torque)
- `POST /subscriptions` — criar recorrência.
- `PATCH /subscriptions/:id` — atualizar.
- `DELETE /subscriptions/:id` — cancelar.

### Payment (cobrança única)
- `POST /payments` — criar cobrança (boleto, pix, cartão, crédito).
- `GET /payments/:id` — detalhe.
- `POST /payments/:id/refund` — estornar.

### Webhook
- Configurado no Asaas (URL por ambiente).

## Webhooks Recebidos

Eventos:
- `PAYMENT_CREATED`: cobrança criada.
- `PAYMENT_UPDATED`.
- `PAYMENT_CONFIRMED`: cliente confirmou (antes de efetivar).
- `PAYMENT_RECEIVED`: pagamento recebido.
- `PAYMENT_OVERDUE`: vencido.
- `PAYMENT_REFUNDED`: estornado.
- `PAYMENT_DELETED`.
- `SUBSCRIPTION_CREATED/UPDATED/DELETED`.
- `INVOICE_CREATED/UPDATED`.

Autenticação: Asaas envia header `asaas-access-token` com token pré-configurado. Torque valida.

### Payload exemplo (PAYMENT_RECEIVED)
```json
{
  "event": "PAYMENT_RECEIVED",
  "payment": {
    "id": "pay_xxx",
    "customer": "cus_yyy",
    "value": 299.00,
    "status": "RECEIVED",
    "billingType": "CREDIT_CARD",
    "dueDate": "2026-04-20",
    "paymentDate": "2026-04-18",
    "externalReference": "torque_subscription_zzz"
  }
}
```

## Fluxos

### Assinatura do Torque (checkout → cobrança recorrente)
1. Cliente faz checkout no Torque.
2. Torque cria customer e subscription no Asaas.
3. Asaas processa primeira cobrança.
4. Webhook `PAYMENT_RECEIVED` → Torque provisiona organização.
5. Mensalmente: Asaas tenta cobrar → webhook → Torque atualiza status.

### Cobrança de Cliente Final (feature da org)
1. Proposta ganha no Torque (venda).
2. Admin da org opt-in: "Gerar cobrança no Asaas ao fechar".
3. Torque cria customer + payment no Asaas (em nome da org).
4. Lead recebe boleto/link por email/WhatsApp.
5. Lead paga → webhook → Torque marca pagamento + emite evento (pode disparar workflow).

### Estorno
1. Admin clica "Estornar" no Torque.
2. Torque chama `POST /payments/:id/refund`.
3. Webhook confirma.
4. Torque reverte comissão relacionada.

## Regras de Negócio

1. API key do Torque master separada de API keys de orgs.
2. Cobrança de cliente final por org requer feature no plano + API key própria da org.
3. Em caso de chargeback: suspensão imediata + audit.
4. Webhook idempotente (dedupe por `payment.id`).
5. Estado consistente: webhook é source of truth; Torque só estado local.
6. Múltiplos boletos ativos para mesma assinatura: regra de consolidação.

## Edge Cases

- **Webhook recebido duplicado**: dedupe ok.
- **Webhook chega fora de ordem** (PAYMENT_RECEIVED antes de PAYMENT_CREATED): Torque cria registro local sob demand.
- **Chargeback semanas depois**: webhook → suspende + audit.
- **Cartão recusado repetidamente**: Asaas tenta 3x; Torque suspende assinatura após.
- **Pix expirado**: regenerar automaticamente.
- **Organização desconectou Asaas** durante cobrança em aberto: cobrança continua (Asaas tem token válido até expirar); Torque só para de criar novas.

## Rate Limits

- Asaas: limites conforme plano.
- Torque respeita e enfileira quando excede.

## Fallback

- Se Asaas indisponível na hora do checkout: erro ao cliente, retry manual.
- Admin pode marcar cobrança como paga manualmente (bypass — com audit) em caso de pagamento fora do sistema.

## Segurança

- API key em vault.
- HTTPS obrigatório.
- Webhook signature validado.
- Nunca armazenar número de cartão; tokenização no Asaas.
- PCI compliance via provider.

## Observabilidade

- Log por chamada.
- Métricas: payments created/day, receipts, overdue, refunds.
- Dashboard de saúde por org.
- Alertas em falhas consecutivas.

## LGPD

- Dados do cliente final compartilhados com Asaas: consentimento ao aceitar cobrança.
- Asaas é o operator para processamento de pagamento.

## Evolução

- Suporte a outros provedores (Stripe, Mercado Pago) via adaptador.
- Split de pagamento automatizado (quando org tem múltiplos beneficiários).
