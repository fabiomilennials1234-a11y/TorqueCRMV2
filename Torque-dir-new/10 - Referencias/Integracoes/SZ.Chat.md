---
tipo: integracao
direcao: bidirectional
criticidade: media
---

# SZ.Chat (Canal WhatsApp Alternativo)

Canal de mensagens WhatsApp alternativo via provedor SZ.Chat (Alamaster). Similar ao Evolution mas de terceiros; usado como redundância ou quando a org prefere esse provider.

## Propósito

- Redundância ao canal primário (Evolution).
- Suporte a orgs que já usam SZ.Chat.
- Multi-instância.

## Contrato

- Conexão via QR code.
- Envio de mensagens (texto, mídia).
- Recebimento + ack de entrega.
- Webhook de eventos.
- Multi-número por conta.

## Autenticação

- Cada instância: token/API key do SZ.Chat.
- Torque armazena em vault por org.

## Endpoints Consumidos

- `POST /send/text` — texto.
- `POST /send/media` — mídia.
- `POST /send/audio` — áudio.
- `GET /instance/status` — status.
- `POST /instance/qrcode` — QR code.

Formatos de payload diferem do Evolution; adaptador interno traduz para formato unificado.

## Webhooks Recebidos

Eventos principais:
- `message.received`: nova mensagem.
- `message.status`: status update.
- `instance.connected/disconnected`.

Payload exemplo:
```json
{
  "event": "message.received",
  "instance_id": "...",
  "message": {
    "id": "...",
    "from": "5511987654321",
    "type": "text",
    "text": "Olá",
    "timestamp": 1712345678
  }
}
```

## Fluxos

Idênticos ao Evolution em nível conceitual (outbound/inbound). Diferença é só no adaptador.

## Rate Limits

Conforme contrato do SZ.Chat (tipicamente similar ao Evolution).

## Fallback

- Evolution é default.
- Admin pode ter ambos; escolher qual usar em cada campanha/conversa.

## Segurança

- Webhook com token.
- Credenciais em vault.
- HTTPS.

## Observabilidade

- Logs separados do Evolution.
- Dashboard de saúde.

## Edge Cases

- **Provider instável**: circuit breaker pausa tentativas.
- **Mudança de contrato** (breaking change): adaptador versionado.
- **Usuário migra de Evolution para SZ**: admin desconecta um, conecta outro; conversas persistem em ambos.

## Evolução

- Consideração futura: usar um de múltiplos providers como "canal primário" via feature flag por org.
