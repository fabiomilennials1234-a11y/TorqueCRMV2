---
tipo: dominio
entidade: Conversa + Mensagem
---

# Conversa e Mensagem

Histórico persistente de comunicação entre o sistema (membros + agente IA) e um lead, em um canal específico.

## Conversa

### Atributos
- `id`: UUID.
- `organization_id`.
- `lead_id`: lead associado.
- `channel`: tipo do canal (`whatsapp_primary`, `whatsapp_alt`, `messenger`, `webchat`, etc.).
- `channel_instance_id`: instância concreta (ex.: número WhatsApp específico da org que mantém essa conversa).
- `status`: `active` | `archived` | `muted`.
- `agent_id`: agente IA associado (se conversa é atendida por copilot).
- `human_takeover_until`: timestamp até o qual o copilot está pausado (set quando humano envia mensagem).
- `last_message_at`: timestamp da última mensagem.
- `last_inbound_at`: timestamp da última mensagem do lead.
- `last_outbound_at`: timestamp da última mensagem enviada.
- `unread_count`: mensagens do lead não lidas pelo time.
- `assigned_to`: membro que "pegou" a conversa (opcional).
- `summary`: resumo gerado automaticamente (IA; atualizado periodicamente).
- `metadata`: JSON (instance info, labels externos, etc.).

### Invariantes
1. `(lead_id, channel, channel_instance_id)` único.
2. `agent_id` se presente deve pertencer à mesma org.
3. `human_takeover_until` no futuro → agente IA não envia nada.

### Estados
```
active ─┬──► archived (decisão do user ou após X dias sem mensagem)
        ├──► muted (sem notificação mas ativa)
active ─┴──► active (pode voltar de archived/muted)
```

## Mensagem

### Atributos
- `id`: UUID.
- `conversation_id`.
- `organization_id`.
- `direction`: `inbound` (do lead) | `outbound` (do time/agente).
- `sender_type`: `lead` | `member` | `agent` | `system`.
- `sender_id`: id do sender (lead, member, agent, null se system).
- `content_type`: `text` | `audio` | `image` | `document` | `video` | `sticker` | `location` | `contact`.
- `content`: texto (ou legenda).
- `media_url`: URL do blob em storage de objetos (para audio/image/document/video).
- `media_metadata`: JSON (mime, size, duration, transcript, etc.).
- `status`: `pending` | `sent` | `delivered` | `read` | `failed` | `received`.
- `timestamp`: quando ocorreu (importante em inbound, pode vir do provider).
- `external_id`: id da mensagem no provedor de canal (para dedupe).
- `reply_to_message_id`: mensagem respondida (se for reply).
- `metadata`: JSON extra (quote, reactions, forwarded, etc.).
- `error`: detalhe de falha, se `status=failed`.

### Invariantes
1. `external_id` único dentro do canal (dedupe de webhook).
2. Append-only. Nunca atualizar conteúdo — atualizações apenas em `status`.
3. `media_url` aponta para storage com TTL de acesso assinado.

## Fluxo de Entrada (Inbound)

```
Provedor de canal POST /webhooks/canal-X
  → handler autentica, valida payload
    → identifica conversa (ou cria) via (lead_phone, channel_instance)
      → se lead não existe, cria lead com fonte "inbound"
        → dedupe por external_id
          → cria mensagem
            → atualiza conversation.last_message_at, last_inbound_at, unread_count
              → publica realtime
                → aciona agente IA (se ativo e não em takeover)
                  → batch window (agrupa com mensagens subsequentes dentro de 8s)
                    → processa + responde
```

## Fluxo de Saída (Outbound)

```
Usuário/agente cria mensagem
  → validação (permissão, lead tem canal, janela de negócio se configurado)
    → persiste mensagem status=pending
      → enfileira para worker de envio
        → worker chama provedor de canal
          → sucesso → status=sent → update com external_id
          → falha transiente → retry com backoff
          → falha permanente → status=failed + evento de notificação
            → atualiza conversation + realtime
              → provedor emite evento delivered/read → update subsequente
```

## Human Takeover

- Quando membro envia mensagem numa conversa atendida por agente IA:
  - `human_takeover_until = now + 10min` (configurável).
  - Agente pausa envios durante essa janela.
  - Nova mensagem do lead durante takeover não aciona agente; fica para o humano.
- Takeover expira automaticamente; agente retoma.
- Admin pode forçar "retomar agente" antes do fim da janela.

## Batch Window (Agentes IA)

- Objetivo: não responder a cada mensagem do lead se ele está digitando sequência (natural em WhatsApp).
- Mecanismo:
  - Primeira mensagem do lead: timer inicia (ex.: 8s).
  - Mensagens subsequentes dentro da janela acumulam.
  - Ao expirar a janela, agente processa TODAS juntas.
- Implementação tipicamente por debounce em fila.

## Smart Split (Agentes IA)

- Resposta longa do LLM é chunkada em mensagens consecutivas humanizadas.
- Heurística: 1-3 parágrafos por mensagem, quebras em limites naturais, pequeno delay entre mensagens (tipo "digitando..." simulado).
- Nunca parte no meio de palavra ou link.

## Status de Entrega

- `pending`: enfileirada.
- `sent`: aceita pelo provedor.
- `delivered`: entregue ao device (duplo check no WhatsApp).
- `read`: lida (check azul).
- `failed`: erro permanente.
- `received`: inbound recebida (uso interno).

Provedores podem não reportar todos. Fallback: após X minutos em `sent` sem update, assume `delivered`.

## Notas Internas (dentro da conversa)

Veja [[04 - Funcionalidades/Comunicação/Notas Internas]]. Entidade separada ou tipo especial de mensagem (`sender_type=system, content_type=note, not sent externally`).

## Resumo Automático

- Job periódico ou disparado após N mensagens gera resumo da conversa via LLM.
- Armazenado em `conversation.summary`.
- Útil para passar conversa entre membros sem ler tudo.

## Busca

- Por lead.
- Por texto em mensagens (full-text).
- Por canal.
- Por status.
- Por período.

## Eventos

- `ConversationStarted`.
- `MessageReceived` (inbound).
- `MessageSent` (outbound confirmado).
- `MessageFailed`.
- `MessageRead` (lead leu nossa mensagem).
- `HumanTakeoverStarted`.
- `HumanTakeoverExpired`.
- `ConversationArchived`.
- `ConversationAssigned`.

## Relações

- **N:1** com Lead.
- **1:N** com Mensagem.
- **1:N** com Nota Interna.
- **N:1** com Instância de Canal.
- **N:1** com Agente IA (opcional).
- **N:1** com Time Member (assigned_to, opcional).

## Segurança e Privacidade

- Conteúdo de mensagem é dado pessoal sensível.
- Nunca logar em plaintext em níveis acima de `debug` em produção.
- Retenção: mensagens retidas pelo prazo legal (LGPD). Deleção em cascata com hard-delete de lead.
- Criptografia em repouso para colunas sensíveis quando viável.
- Mídia em storage com acesso via URL assinada de curto tempo.

## Limites

- Tamanho máximo de texto por mensagem: ~4000 chars (canais têm limites próprios; WhatsApp ~65k, mas split recomendado antes de 1000).
- Mídia: limite do canal (WhatsApp aceita até ~16MB vídeo, ~100MB documento).
- Mensagens por conversa: sem limite hard; performance via paginação na UI.

## Performance

- Lista de conversas: índice por `(organization_id, last_message_at DESC)` + filtro status.
- Mensagens de uma conversa: índice por `(conversation_id, timestamp DESC)`.
- Full-text search via índice dedicado ou integração com serviço de busca.
- Carregamento progressivo (infinite scroll) para conversas longas.
