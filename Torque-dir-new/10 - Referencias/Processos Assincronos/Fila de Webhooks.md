---
tipo: async
---

# Fila de Webhooks

Fila de **entrega de webhooks de saída** para endpoints externos configurados por organizações. Garante entrega eventual com retry, isolamento de falhas, e visibilidade.

## Fluxo

```
Evento de domínio emitido
        │
        ▼
Fan-out: para cada Webhook Endpoint subscrito + filter match
        │
        ▼
Cria Webhook Delivery (status=pending)
        │
        ▼
Worker (job process-webhook-deliveries, cada minuto, batch 100):
  - Pega pending com next_retry_at <= now
  - Prepara payload JSON
  - Calcula signature HMAC com secret do endpoint
  - POST ao URL do endpoint
  - Timeout 10s
        │
        ▼
Resposta:
  ├── 2xx → status=success, grava resposta
  ├── 4xx → status=failed (erro permanente, não retry)
  ├── 5xx ou timeout → increment attempts
  │                     if attempts < 5: status=retrying, next_retry_at=now + backoff
  │                     if attempts >= 5: status=failed (dead letter)
```

## Backoff

Tentativas: `30s, 2min, 10min, 1h, 6h`.

## Payload Formato

```json
POST https://endpoint.cliente.com/torque-webhook

Headers:
  Content-Type: application/json
  X-Torque-Event: LeadCreated
  X-Torque-Delivery-Id: <uuid>
  X-Torque-Timestamp: 2026-04-15T12:34:56Z
  X-Torque-Signature: sha256=<hmac do body com secret>

Body:
{
  "event_name": "LeadCreated",
  "event_id": "...",
  "organization_id": "...",
  "timestamp": "...",
  "data": {
    "lead": { /* payload */ }
  }
}
```

## Idempotência (lado do receiver)

- Cliente pode rejeitar duplicatas via `X-Torque-Delivery-Id`.
- Torque não faz no receiver — é compromisso do cliente.

## Estados

- `pending`: aguardando primeira tentativa.
- `retrying`: em retry entre tentativas.
- `success`: entrega confirmada (2xx).
- `failed`: dead letter (4xx permanente ou esgotou retries).

## Dead Letter

- Delivery em `failed` fica armazenada.
- UI do admin: "Deliveries falhadas" com filtro.
- Admin pode inspecionar payload, response, erro.
- "Replay": cria nova delivery (nova sequência de retry).
- Retention: deliveries failed retidas por 30 dias.

## Observabilidade

- Cada delivery loga: status, attempts, duration, response code.
- Métricas:
  - Deliveries/dia por endpoint.
  - Taxa de sucesso.
  - Latência p95.
  - Dead letter size.
- Alertas:
  - Endpoint com > X% falha em janela → notifica admin.
  - Dead letter cresce → alerta.

## Health Degradation

- Se endpoint tem taxa de falha > 50% nas últimas 100 tentativas:
  - Marca `health_degraded`.
  - UI avisa admin.
  - Pode opcionalmente pausar deliveries ao endpoint até admin investigar.

## Rate Limiting

- Por endpoint: max N deliveries/segundo (config).
- Protege o destino de ser derrubado.

## Fan-out Eficiente

- Evento pode resultar em múltiplas deliveries (vários endpoints subscritos).
- Criação das deliveries é rápida (insert em lote).
- Worker paraleliza envios.

## Segurança

- HMAC signature permite cliente validar autenticidade.
- Secret rotacionável pelo admin.
- URL validada contra SSRF (não localhost, ranges privados em prod).
- HTTPS obrigatório.

## Filters

- Endpoint pode definir filters para reduzir fan-out (ex.: só leads com `tag=X`, só vendas acima de `R$ 10k`).
- Evento que não passa filter: não gera delivery.

## Edge Cases

- **Endpoint offline temporariamente**: retry cobre.
- **Endpoint lento (timeout 10s)**: considerado falha; retry.
- **Endpoint responde 200 mas internamente falha**: responsabilidade do cliente.
- **Payload muito grande** (> 1MB): truncado ou dividido (política).
- **Mudança de secret** sem atualizar endpoint: assinaturas falham; cliente recebe e valida.
- **Endpoint deletado**: deliveries pending canceladas.

## Admin View

- Lista de endpoints com health, deliveries/dia, taxa de sucesso.
- Click em endpoint → list de deliveries.
- Click em delivery → detalhe (payload, response, logs).
- Botão "Replay" em deliveries failed.
- Botão "Test" em endpoint (dispara evento de teste).

## Evolução

- Websockets / SSE para delivery em tempo real (sem retry infra, só "fire and drop" para cliente ouvindo).
- Push notifications para endpoints mobile.
- Multi-destino ordenado (se falhar ao endpoint A, tentar B fallback).
