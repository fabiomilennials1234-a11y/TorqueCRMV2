---
tipo: async
---

# Processamento de Mensagens Outbound

Fila e worker responsáveis por **enviar mensagens** via canais externos. Abstrai provedor, aplica rate limit, gerencia retry, respeita janela de negócio.

## Arquitetura

```
Origem:
  - Usuário clicou Enviar no chat.
  - Workflow action send_message.
  - Campanha dispatcher.
  - Agente IA resposta.
  - Mensagem agendada vencendo.
  - Follow-up automático.

  Todas produzem:
  Message record (status=pending) + enfileira job "send_message"

        │
        ▼
Worker (job process-outbound, contínuo ou batch cada 1-5 min):
  - Pega job
  - Valida:
    - Canal conectado?
    - Rate limit OK?
    - Janela de negócio (se respect_business_hours)?
    - Lead não opt-out?
  - Se não valida: reenfileira ou falha
  - Se OK:
    - Chama adaptador de canal → POST ao provedor
    - Timeout 15s
        │
        ▼
Resultado:
  - Sucesso: message.status=sent, armazena external_id
  - Falha transiente: retry com backoff (até 5x)
  - Falha permanente: status=failed + error
        │
        ▼
Webhook posterior do provedor (delivered/read) atualiza status.
```

## Componentes

- **Adaptador de Canal**: tradução canal-específica (Evolution, SZ.Chat, Messenger).
- **Fila**: armazena jobs com prioridade, delay.
- **Worker**: consome e processa.
- **Rate Limiter**: controla envios por canal/tempo.
- **Business Hours Check**: pula envio fora de janela (ou re-agenda).
- **Opt-out Check**: não envia para tag `no-contact`.

## Rate Limiting

- Por instância de canal: max N mensagens/hora (ex.: 100/h default).
- Evita banimento por WhatsApp.
- Configurável por org conforme plano.
- Limite do provedor também respeitado.
- Excedido → job fica em fila, processa quando janela libera.

## Janela de Negócio

- Configurada por org (`business_hours`).
- Worker pula envios fora de janela se flag:
  - `respect_business_hours=true` (default para campanhas).
  - Re-schedula para próximo horário válido.
  - Mensagens urgentes (ex.: resposta direta do agente a lead inbound) podem ignorar.

## Retry

- Falha transiente (5xx, timeout): retry com backoff (30s, 2min, 10min, 1h).
- Falha permanente (4xx — número inválido, formato ruim): status=failed sem retry.
- Após 5 tentativas: dead letter, admin pode reprocessar manualmente.

## Smart Split (Outbound Chunks)

- Mensagem longa (> max_length por canal): split em chunks.
- Worker envia chunks sequenciais com delay entre (2-4s).
- Cada chunk é mensagem distinta no provider.
- UI mostra como mensagens consecutivas no histórico.

## TTS (Áudio)

- Mensagem com `content_type=audio` e `text_to_speak`:
  - Worker chama TTS provider.
  - Recebe áudio.
  - Upload storage.
  - Envia como mídia via canal.

## Opt-out

- Tag `opt-out` ou `no-contact`: mensagem rejeitada (status=failed com reason=opt_out).
- Check antes de chamar provedor.

## Idempotência

- Mensagem tem `id` único no Torque.
- Envio ao provider usa `Idempotency-Key = message.id` (se provedor suporta).
- Duplicata do Torque → provider dedupe.

## Priorização

- Fila tem prioridade:
  - `urgent`: resposta de agente a lead que está ativo (responde imediato).
  - `high`: mensagens de workflow crítico, proposta.
  - `normal`: campanhas, follow-ups.
  - `low`: warmup, broadcasts.
- Worker processa prioridade mais alta primeiro.

## Status Tracking

- `pending`: enfileirada.
- `sending`: worker está chamando provider.
- `sent`: provider aceitou, message_id externo registrado.
- `delivered`: webhook do provider confirma entrega ao device.
- `read`: check azul do WhatsApp.
- `failed`: erro permanente.

## Webhook de Status

Provider envia atualização:
- `delivered`: Torque atualiza message.status.
- `read`: idem.
- `failed` (não entregou): status=failed.

## Edge Cases

- **Canal desconectado ao pegar job**: rejeita, reenfileira com delay, admin alerta.
- **Rate limit do provider retorna 429**: backoff maior, re-agenda.
- **Número inválido / bloqueado / inexistente**: status=failed, tag `invalid_number` ou `blocked`.
- **Mensagem muito antiga em fila** (> 24h): descartada, audit log (lead já pode ter respondido outro canal).
- **Volume extremo** (10k mensagens enfileiradas): respeita rate limit, processa ao longo do tempo; UI mostra ETA.

## Observabilidade

- Log por envio: message_id, channel, duration, result, attempts.
- Métricas:
  - Mensagens enviadas/hora por canal.
  - Taxa de falha.
  - Latência p50/p95.
- Alertas:
  - Canal com taxa de falha > X%.
  - Fila crescendo (backlog > threshold).

## Multi-canal

- Se lead tem múltiplos canais ativos, regra define qual usar:
  - Canal original da conversa.
  - Preferência do lead (campo custom).
  - Default da org.

## Segurança

- Content não logado em plaintext em info.
- Tokens de provider em vault.
- URLs de mídia assinadas (storage).

## Anti-spam

- Respeita opt-out sempre.
- Detecta padrões de bloqueio → pausa automática.
- Mensagem inicial identifica empresa (não spam).

## Performance

- Worker escala horizontal.
- Partição por canal/org.
- Rate limit local ao worker (não chamar N workers para mesmo canal simultâneo — sincronizado).
