---
tags: [operacional, observabilidade, logs, sentry, seguranca]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Observabilidade e Logs

Postura de observabilidade do Torque-v2. Sem vazamento de PII, com correlação ponta a ponta, feedback visível ao usuário quando o sistema degrada. Complementa [[Riscos e Duvidas]] e [[Seguranca Base]].

## Frontend

### Sentry

- DSN entregue via `GET /api/bootstrap` no boot do app, nunca hard-coded no bundle. Isso permite rotacionar DSN sem rebuild.
- `beforeSend` aplica scrubbing das chaves `token`, `password`, `authorization`, `cookie`, `email` em qualquer profundidade do payload. Implementação recursiva sobre `event.extra`, `event.request.headers`, `event.request.data`, `event.breadcrumbs`.
- Sampling: **10% em produção, 100% em desenvolvimento.** `tracesSampleRate: 0.1` em prod, `1.0` em dev.
- **Session Replay apenas em erros.** `replaysOnErrorSampleRate: 1.0`, `replaysSessionSampleRate: 0`. Replay normal é vigilância, não faz parte do contrato com o usuário.
- User context: `{ id, role, is_master }`. **Nunca incluir `email` nem `phone` nem `name`.** Ver lista de itens não logados adiante.
- Release atrelada ao `git sha` do build.

### Logger cliente

Estrutura única `{ level, message, context, timestamp }`. Níveis `debug | info | warn | error`.

- **Dev:** todos os níveis exibidos no console com formatação colorida.
- **Produção:** apenas `warn` e `error` são enviados ao Sentry. `info` e `debug` são silenciados, sem custo.
- `context` aceita qualquer objeto serializável, passa pelo mesmo scrubbing do `beforeSend`.

API proposta:
```ts
logger.info("lead.created", { leadId, pipeId });
logger.warn("ws.reconnect", { attempt, backoffMs });
logger.error("fetch.failed", { url, status, requestId });
```

### Status de WebSocket visível

Badge no `TopBar` com três estados, rotulados por cor e microcopy:

- **Connected** — verde sutil, sem microcopy ou tooltip "Tempo real ativo"
- **Connecting** — âmbar com pulse, tooltip "Reconectando"
- **Disconnected** — vermelho discreto, tooltip "Sem tempo real, dados podem estar desatualizados"

Badge nunca é intrusivo. Mas existe, consistente, e usuário avançado aprende a usar como sinal de saúde.

### Correlation ID

- Header `X-Request-ID` em **todo** fetch originado do frontend.
- Gerado client-side em ULID (`crypto.randomUUID()` se ULID não disponível).
- Servidor Go ecoa o mesmo ID na resposta e propaga em traces OpenTelemetry.
- Logger cliente inclui `requestId` em todo erro de fetch.

### Core Web Vitals (opcional, fase 3+)

Entrar com `web-vitals` enviando LCP, INP, CLS como Sentry custom metrics. Não é prioridade antes de F04 porque volume de tráfego real ainda é baixo. Depois de F09 vira medição rotineira.

## Backend Go (referência, fora do escopo direto do front)

- Structured logging via `zerolog` ou `slog`. Nunca `fmt.Println`.
- OpenTelemetry traces ponta a ponta, `X-Request-ID` como `trace_id` quando possível.
- `tenant_id` sempre em contexto de log, obrigatoriamente. Log sem `tenant_id` é bug.
- Métricas expostas em `/metrics` (Prometheus format) restritas a rede interna.
- Audit log de mutações sensíveis (criação de usuário, mudança de permissão, mudança de plano) persistido em tabela dedicada com `actor_id`, `action`, `target`, `before`, `after`, `request_id`.

## Lista de itens que NUNCA devem ser logados

- **PII de leads:** `phone`, `email`, `document`, `name` completo
- **Conteúdo de mensagens:** corpo de mensagens de WhatsApp/Instagram/Messenger/Webchat
- **Tokens de qualquer natureza:** JWT, refresh token, session, API keys, webhook secrets
- **Valores financeiros exatos:** valor de proposta, preço de plano negociado, comissão
- **Cookies** completos ou parciais
- **Senhas** em qualquer forma

Se esses campos caem em log por acidente, a investigação é de segurança, não de observabilidade. Ver [[Riscos e Duvidas]] R1 e a lição de [[CONCERN-S do Legado]].

## Princípios

- **Observar o sistema, não o usuário.** Log serve para o engenheiro entender o que quebrou, não para reconstituir comportamento individual.
- **Todo erro tem `requestId`.** Sem correlação, o erro é ruído.
- **Silêncio em prod é deliberado.** `info` e `debug` ficam fora por custo e por ruído, não por esquecimento.
- **Scrubbing é camada de defesa, não contrato.** Nunca confie no scrubbing para deixar passar: não logue PII de origem.
