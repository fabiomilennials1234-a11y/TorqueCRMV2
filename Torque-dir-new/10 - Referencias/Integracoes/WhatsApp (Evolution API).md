---
tipo: integracao
direcao: bidirectional
criticidade: alta
---

# WhatsApp (via Evolution API)

Canal de mensagens WhatsApp suportado via provedor open-source "Evolution API". Multi-device: cada organização pode conectar múltiplos números (instâncias). Canal principal da operação do Torque.

## Propósito

Permitir que organizações enviem e recebam mensagens WhatsApp em escala, integradas ao CRM. Substituir WhatsApp Web + planilha + follow-up manual.

## Contrato (o que Torque precisa do provider)

- Conexão inicial via QR code (emparelhamento).
- Envio de mensagens: texto, imagem, vídeo, áudio, documento, localização, contato.
- Recebimento de mensagens: idem + ack de entrega/leitura.
- Status de conexão (connected/disconnected/error).
- Desconexão remota.
- Listagem de contatos (opcional).
- Grupos: suporte básico (receber em grupos, enviar se necessário).
- Rate limiting documentado pelo provider.

## Autenticação

- Provider instalado tem API key por instância.
- Torque armazena (em vault cifrado) a URL do provider + API key.
- Cada instância tem URL/key próprios.

## Endpoints Consumidos (do Torque → provider)

### Gerenciar Instância
- `POST /instance/create` — cria instância.
- `GET /instance/connect/:name` — retorna QR code para emparelhamento.
- `GET /instance/status/:name` — status (connected/disconnected).
- `DELETE /instance/:name` — remove.

### Enviar Mensagem
- `POST /message/sendText/:instance` — texto.
- `POST /message/sendMedia/:instance` — mídia (image, video, audio, document).
- `POST /message/sendAudio/:instance` — áudio voice-note.
- `POST /message/sendLocation/:instance` — localização.
- `POST /message/sendContact/:instance` — vCard.

Cada endpoint recebe:
- `number`: telefone do destinatário (E.164).
- `options`: delay, presence (typing before), etc.
- Conteúdo específico.

Resposta típica:
- 200 com `message_id` do provider.
- 4xx com erro (número inválido, instância offline).

### Ler Mensagens Históricas (opcional)
- `GET /chat/findMessages/:instance` — paginado.

## Webhooks Recebidos (provider → Torque)

Provider faz POST para URL configurada (com token específico da org).

### Eventos
- `messages.upsert`: nova mensagem (inbound ou outbound echo).
- `messages.update`: status de delivery/read update.
- `connection.update`: status da instância.
- `contacts.upsert`: novo contato.
- `chats.upsert`: nova conversa.

### Payload típico (messages.upsert)
```json
{
  "event": "messages.upsert",
  "instance": "numero-principal",
  "data": {
    "key": {
      "remoteJid": "5511987654321@s.whatsapp.net",
      "fromMe": false,
      "id": "MSG_ID_NO_PROVIDER"
    },
    "pushName": "Maria Silva",
    "message": {
      "conversation": "Olá, tudo bem?",
      // ou imageMessage, videoMessage, audioMessage, etc.
    },
    "messageTimestamp": 1712345678
  }
}
```

Torque:
1. Autentica webhook (token na URL).
2. Identifica organização + instância.
3. Normaliza payload para formato interno.
4. Dedupe por `data.key.id`.
5. Persiste Mensagem.
6. Cria/atualiza Conversa.
7. Se `fromMe=false` e canal ativo com agente → aciona copilot.

## Fluxos de Dados

### Outbound
```
User clicka Enviar → backend valida → persiste msg status=pending → enfileira job
Worker → chama Evolution sendText/sendMedia → sucesso → update status=sent + message_id externo
Provider entrega ao WhatsApp → evento `messages.update` com status delivered/read
Webhook → update status da msg
```

### Inbound
```
Lead escreve → Evolution recebe → webhook para Torque
Torque dedupe, persiste, publica realtime
UI atualiza + agente IA aciona
```

## Configuração por Organização

1. Admin em `Configurações → Canais → Adicionar WhatsApp`.
2. Torque cria instância no provider via API.
3. Mostra QR code.
4. Admin escaneia com WhatsApp do número.
5. Emparelhamento → instância `connected`.
6. Admin nomeia ("Número da Vendas"), salva.
7. Pode adicionar múltiplas instâncias (plano permitir).

## Tratamento de Erros

### Desconexão
- Webhook `connection.update` com `connected=false`.
- UI alerta admin.
- Mensagens outbound novas falham com erro explicativo.
- Retry manual: admin re-escaneia QR.

### Rate Limit
- Provider rate-limita.
- Torque enfileira; worker respeita delay configurado.

### Mensagem rejeitada (número inválido)
- Update status=failed + error.

### Bloqueio pelo lead
- Sem evento explícito; detectado por mensagens sucessivas em failed.
- Após N falhas, marca contato como "possivelmente bloqueado".

## Rate Limits

- Provider / WhatsApp têm limites não-oficiais (banimento por spam).
- Convenção: ≤ 100 mensagens/hora por número recém-conectado; aumenta gradualmente.
- Rate limit por campanha mais restritivo.

## Fallback

- Canal alternativo: SZ.Chat (provider alternativo).
- Admin pode conectar ambos; sistema roteia conforme instância ativa.
- Se primário falha, admin migra para alternativo temporariamente.

## Segurança

- Webhook URL tem token único por org (rotacionável).
- Credenciais (API key do provider) em vault.
- Nunca expor URL interna do provider ao cliente.
- Conteúdo de mensagem sensível — não logado em plaintext em info.

## Observabilidade

- Log por chamada: instance, endpoint, latência, status.
- Métricas: mensagens enviadas/dia por org, taxa de falha, latência p95.
- Dashboard de saúde do canal.

## Limitações

- WhatsApp (como plataforma) pode banir números por comportamento agressivo.
- Provider open-source: sem SLA oficial.
- Mudanças no WhatsApp quebram o provider ocasionalmente.
- Sem API oficial do WhatsApp Business (fora do escopo atual).

## Evolução

- Planejamento para usar API oficial WhatsApp Business (Cloud API) como alternativa — com suas próprias restrições (templates pré-aprovados, 24h window).
