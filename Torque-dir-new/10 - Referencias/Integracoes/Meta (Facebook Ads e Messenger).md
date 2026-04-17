---
tipo: integracao
direcao: bidirectional
criticidade: alta
---

# Meta (Facebook Ads, Instagram, Messenger)

Integração com plataforma Meta Business: captura de leads de anúncios (Facebook/Instagram Lead Ads) e atendimento via Messenger.

## Propósito

- **Ingestão de leads** de campanhas Lead Ads do Facebook/Instagram: quando usuário preenche formulário em anúncio, Meta envia lead para o Torque.
- **Atendimento Messenger**: conversas no Messenger integradas ao chat unificado.
- Opcional futuro: consumo de métricas de campanhas para analytics de ROAS.

## Contrato

- OAuth com Meta Business.
- Subscrição de eventos via Graph API webhooks.
- Chamadas à Graph API.

## Autenticação

- Fluxo OAuth 2.0 com Meta.
- Token de longa duração (~60 dias), refreshable.
- Refresh automático antes do vencimento.
- Rotação documentada.

## OAuth Setup (por organização)

1. Admin em `Configurações → Integrações → Meta Business → Conectar`.
2. Redireciona para Meta OAuth (lista de páginas/ad accounts que admin autoriza).
3. Callback retorna code.
4. Torque troca code por access token.
5. Armazena token cifrado.
6. Subscreve webhooks para páginas/ad accounts.

## Endpoints Consumidos

### Graph API
- `GET /{page-id}/leadgen_forms` — listar forms.
- `GET /{form-id}/leads` — puxar leads (fallback).
- `POST /{page-id}/messages` — enviar mensagem no Messenger.
- `GET /{ad-account-id}/insights` — métricas (futuro).

## Webhooks Recebidos

Eventos configurados:
- `leadgen`: novo lead preenchido em form de anúncio.
- `messages`: nova mensagem no Messenger.
- `messaging_postbacks`: botão clicado.
- `page_feed` (opcional): posts/comentários.

### Payload leadgen
```json
{
  "object": "page",
  "entry": [{
    "id": "page_id",
    "changes": [{
      "field": "leadgen",
      "value": {
        "leadgen_id": "lead_id_no_meta",
        "ad_id": "...",
        "form_id": "...",
        "adgroup_id": "...",
        "page_id": "...",
        "created_time": 1712345678
      }
    }]
  }]
}
```

Torque:
1. Recebe webhook (valida via signature).
2. Chama Graph API para puxar detalhes do lead (`GET /{leadgen_id}`).
3. Normaliza campos para formato interno.
4. Chama caso de uso `CreateLead` com origin=`meta_ads`, utm preenchidos.

### Payload messages
Mensagem de lead via Messenger:
```json
{
  "object": "page",
  "entry": [{
    "messaging": [{
      "sender": {"id": "psid_do_usuario"},
      "recipient": {"id": "page_id"},
      "timestamp": 1712345678,
      "message": {
        "mid": "msg_id",
        "text": "Olá"
      }
    }]
  }]
}
```

## Autenticação de Webhook

- Meta envia `X-Hub-Signature-256` com HMAC.
- Torque valida antes de processar.

## Fluxos

### Lead de Meta Ads → Torque
1. Usuário preenche form em anúncio Facebook/Instagram.
2. Meta envia webhook `leadgen`.
3. Torque puxa detalhes via Graph API.
4. Normaliza (nome, email, phone, custom fields do form).
5. Cria lead com UTMs preenchidos (campaign=`{campaign_name}`).
6. Lead aparece no pipe configurado.

### Messenger
1. Lead manda mensagem → webhook `messages`.
2. Torque identifica conversa (por psid + page_id).
3. Persiste msg.
4. UI atualiza + agente pode atender.

### Enviar via Messenger
1. User envia mensagem pelo chat do Torque.
2. Torque chama Graph API `POST /{page-id}/messages`.
3. Respeita janela de 24h do Messenger (após essa janela, só mensagens `tagged` permitidas).

## Configuração

- Admin conecta via OAuth.
- Autoriza páginas + ad accounts.
- Escolhe quais forms rastrear.
- Define mapping de campos do form para campos de lead (ex.: campo "orçamento" → custom_field `budget`).

## Rate Limits

- Graph API: limites por aplicação + por usuário.
- Messenger: 24h window para envio não-pago.
- Torque respeita e retry com backoff.

## Refresh de Token

- Cron diário verifica tokens próximos de expirar (< 7 dias).
- Refresha automaticamente.
- Se falha: admin é notificado para reconectar.

## Edge Cases

- **Token expirado**: recupera via refresh; se falha, admin reconecta manualmente.
- **Meta rate limit**: backoff.
- **Form descontinuado**: leads desse form param de vir; admin é notificado.
- **Mudança de schema do form**: admin ajusta mapping.
- **Messenger 24h window**: após janela, envio falha com erro específico; usar tagged messages apenas se configurado.
- **Webhook signature inválida**: ignora payload, alerta.

## Fallback

- Se Meta down: leads não chegam. Reenvio manual via admin.
- Alternativa: captura via formulário no site → webhook direto.

## Segurança

- HMAC validation.
- Token cifrado em vault.
- Nunca expor token ao cliente.
- Page tokens têm escopo mínimo necessário.

## Observabilidade

- Log por webhook recebido.
- Log por chamada Graph API.
- Métricas: leads/dia de Meta por org.
- Dashboard de saúde.

## LGPD / Privacidade

- Lead entra no Torque com consentimento dado no form de Meta (declarado no próprio form).
- Torque preserva opt-out.
- Conforme LGPD, titular pode pedir exclusão.

## Evolução

- Adicionar ingestão direta de métricas de campanhas para ROAS automático.
- Suporte a WhatsApp Business Platform (via Meta Cloud API) como canal principal.
- Automatic A/B test analysis via campaign insights.
