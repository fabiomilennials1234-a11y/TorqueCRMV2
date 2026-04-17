---
tipo: feature
dominio: admin
---

# Webhooks (de Saída)

## Propósito

Permite à organização **registrar endpoints externos** que receberão eventos do Torque em tempo real. Útil para integrar o Torque com ERPs, BI, automações n8n/Zapier custom, notificação em Slack/Teams, etc.

## Atores e Permissões

- **Admin**: CRUD.
- **Membros**: sem acesso.

Ações: `webhook.view`, `webhook.create`, `webhook.edit`, `webhook.delete`, `webhook.view_deliveries`, `webhook.replay_delivery`.

## Dados Envolvidos

### Webhook Endpoint
- `id`, `organization_id`.
- `name`: nome legível.
- `url`: endpoint HTTP(S).
- `secret`: shared secret para HMAC (gerado, rotacionável).
- `events`: array de event names subscritos (catálogo em [[09 - Referências/Catálogo de Eventos]]).
- `filters`: JSON opcional (ex.: `{tag_id: "xxx", pipeline_id: "yyy"}`).
- `headers_extra`: JSON de headers adicionais (ex.: Authorization custom).
- `is_active`.
- `created_at`, `updated_at`.
- `created_by`.

### Webhook Delivery
- `id`, `endpoint_id`, `event_id`.
- `status`: `pending` | `success` | `failed` | `retrying`.
- `attempts`: N.
- `last_attempt_at`, `next_retry_at`.
- `response_status` (HTTP).
- `response_body_snippet` (primeiros 500 chars).
- `duration_ms`.
- `payload` (JSON completo enviado).
- `error_message`.

## Fluxo de Entrega

### Evento
1. Sistema emite evento de domínio.
2. Fan-out: cada webhook endpoint subscrito cria Webhook Delivery em `pending`.
3. Worker pega pending, prepara payload, chama URL.

### Chamada
```
POST <url>
Content-Type: application/json
X-Torque-Signature: hmac-sha256 do payload com secret
X-Torque-Event: <event_name>
X-Torque-Delivery-Id: <delivery_id>
X-Torque-Timestamp: <iso8601>
[headers_extra]

{
  "event_name": "LeadCreated",
  "event_id": "...",
  "organization_id": "...",
  "timestamp": "...",
  "data": { ...payload específico do evento... }
}
```

### Resposta
- 2xx: success. `status=success`, `duration_ms`.
- 4xx: failed (não retry — configuração incorreta do endpoint).
- 5xx: retrying (retry com backoff).
- Timeout (10s): retry.

### Retry
- Backoff exponencial: 30s, 2min, 10min, 1h, 6h.
- Máximo 5 tentativas.
- Após esgotar: `status=failed`, vai para dead letter.

### Dead Letter
- Admin vê em UI.
- Pode inspecionar payload, response, error.
- Pode reprocessar manualmente ("Replay").

## Regras de Negócio

1. Secret exibido **uma vez** na criação; depois fica masked.
2. Admin pode rotacionar secret (novo gerado; subscriber precisa atualizar).
3. URL deve ser HTTPS em produção (HTTP rejeitado ou warning).
4. Eventos subscritos: apenas os da org (não pode subscrever cross-org).
5. Filters: reduzem fan-out (evento que não passa filter não gera delivery).
6. Rate limit por endpoint: max N deliveries/segundo para não derrubar o destino.
7. SSRF: URLs não podem apontar para ranges privados (localhost, 10.x, 192.168.x) em produção.

## Fluxos do Usuário

### Criar Webhook
1. `Configurações → Webhooks → Novo`.
2. Form: nome, URL, eventos (multi-select), filters opcionais, headers extra.
3. Sistema gera secret → exibe uma vez.
4. Admin clica "Testar" → envia evento `WebhookTestEvent` → vê resposta do endpoint (status, body).
5. Ativa.

### Editar
- Tudo editável exceto secret (rotação separada).

### Ver Deliveries
- Tab "Entregas" por endpoint.
- Lista paginada: timestamp, evento, status, duração, código HTTP.
- Click em delivery → detalhe: payload, response, headers.

### Reprocessar Delivery
- Dead-letter com click "Replay".
- Cria nova delivery.

### Desativar / Deletar
- Soft-delete. Deliveries futuras não enviadas. Histórico preservado.

## Automações

### Emite
- `WebhookDeliverySucceeded`, `WebhookDeliveryFailed`, `WebhookEndpointHealthDegraded`.

### Reage
- Consome todos eventos de domínio; fan-out.

## Integrações

- **Todos eventos** do Torque.
- **Audit**.
- **Alertas**: se % de falha de um endpoint > X, alerta ao admin.

## Edge Cases

- **Endpoint 100% offline**: após 5 falhas consecutivas, endpoint marcado `health_degraded`; admin notificado.
- **Endpoint devolvendo 200 mas não processando**: responsabilidade do receptor; Torque considera success se código OK.
- **Payload muito grande** (> 1MB): truncado ou dividido (política).
- **Eventos em burst** (100 leads criados em 1s): rate limit enfileira; delivery pode atrasar segundos.
- **Mudança de secret sem atualizar cliente**: assinaturas vão falhar; cliente recebe e valida; admin repete deploy.
- **URL quebrada por typo**: testar criação evita; se passa, deliveries falham.

## Validações

- URL válida, HTTPS.
- Eventos existem no catálogo.
- Filters sintaxe correta.
- Headers não conflitam com headers padrão do Torque (X-Torque-*).
- SSRF check.

## Métricas

- Webhooks ativos.
- Deliveries por dia, por endpoint.
- Taxa de sucesso por endpoint.
- Duração média.
- Dead-letter size.

## Segurança

- HMAC obrigatório — integrador verifica que payload veio do Torque.
- Secret rotacionável.
- Rate limit.
- SSRF block.
- Payload não inclui secrets do Torque (só dados de domínio).

## UX

- Editor de filters com preview.
- Test delivery antes de ativar.
- Logs legíveis.
- Deep-link do evento de domínio para deliveries relacionadas.

## Contrato Documentado

- Admin pode ler [[API Docs]] com specs de cada payload por evento.
- Playground opcional para testar localmente.
