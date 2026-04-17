---
tags: [arquitetura, realtime, websocket, jobs, async]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Realtime e Jobs

Dois problemas diferentes tratados como temas irmaos: **propagar mudancas ao vivo** e **executar trabalho demorado**. Ambos compartilham a invariante de que a UI nunca espera bloqueada.

## Decisao: WebSocket com hub pattern em Go

Stack: `nhooyr.io/websocket` no servidor, cliente nativo `WebSocket` no navegador.

- Sem STOMP. Protocolo JSON simples resolve o problema sem overhead de framing extra.
- Sem Centrifugo. E uma dependencia externa com operacao propria; nao justifica para o volume do Torque. O hub em Go roda no mesmo binario da API, compartilha autenticacao e tenant-scope.
- Conexao unica por tab. Multiplos consumidores subscrevem ao cliente singleton (`src/lib/ws.ts`).

## Autenticacao do socket

- Upgrade em `wss://api.torque.app/ws` usa o mesmo cookie `__torque_session`.
- O Go extrai JWT no upgrade, resolve `organization_id`, registra conexao no hub com esse tenant como chave de fanout.
- Rejeita com `1008 Policy Violation` se cookie ausente/invalido. Cliente trata como 401, dispara fluxo de refresh e reconecta.

## Protocolo server -> client

```json
{
  "type": "lead.updated",
  "tenant_id": "org_...",
  "entity_id": "lead_...",
  "patch": { "stage_id": "stg_closed_won", "score": 87 },
  "version": 42,
  "occurred_at": "2026-04-15T12:34:56Z"
}
```

- `type` e `<entity>.<verb>`: `lead.created`, `lead.updated`, `lead.deleted`, `conversation.message`, `operation.completed`, `permissions.changed`, `quota.changed`.
- `patch` contem apenas o delta. Se o patch for insuficiente para atualizar a UI, o cliente invalida a query (fallback por correcao, nao por padrao).
- `version` e monotonic crescente por `entity_id`, garantido pelo backend. Ver deduplicacao abaixo.

## Cliente WebSocket

```ts
// src/lib/ws.ts (contrato)
export const ws = {
  connect(): void,
  disconnect(): void,
  on<T extends EventType>(type: T, handler: (msg: Event<T>) => void): Unsubscribe,
  status: Signal<'connecting' | 'open' | 'reconnecting' | 'closed'>,
};
```

Roteamento por evento:
1. Recebe mensagem, faz parse, valida shape.
2. Dedup: se `version <= lastVersionFor(entity_id)`, ignora.
3. Mapeia `type` -> lista de query keys afetadas (`lead.updated` -> `['leads', 'all']`, `['leads', entity_id]`, `['pipeline', stage_id]`).
4. Se `patch` e suficiente: `queryClient.setQueryData(key, prev => merge(prev, patch))` (optimistic).
5. Se patch ausente/ambiguo: `queryClient.invalidateQueries(key)`.

## Deduplicacao

Cache em memoria `Map<entity_id, lastVersion>` dentro do modulo WS. Mensagens com `version` menor ou igual ao ultimo processado sao descartadas silenciosamente. Isso resolve:
- Reconexao com replay recente do servidor.
- Mutations otimistas que ja aplicaram o efeito localmente.
- Ordem fora de sequencia em flaky networks.

## Reconexao

- Backoff exponencial com jitter: `min(1000 * 2^attempt + random(0, 500), 30000)` ms.
- Reset de `attempt` para zero apos 5 minutos de conexao estavel.
- Em reconexao bem-sucedida, cliente emite `sync` com `last_seen_version_per_entity`. Servidor replaya eventos perdidos (janela de 10 minutos em buffer em memoria; alem disso, invalidacao total dos caches afetados).

## Indicador de status

Hook `useWSStatus()` exposto no `TopBar`:
- `open`: badge discreto (ponto verde).
- `reconnecting`: spinner + tooltip "Reconectando...".
- `closed`: banner topo "Sem conexao ao vivo. Tentando reconectar." + botao "Reconectar agora".

Ver [[Principios de Identidade Visual]] para o tratamento visual exato.

## Entidades com subscribe ativo

- `leads` (criacao, update, delete, mudanca de stage)
- `pipe_whatsapp`
- `pipe_confirmacao`
- `pipe_propostas`
- `channel_messages` (mensagens novas em conversas abertas)
- `copilot_agents` (status de execucao em tempo real)
- `operations` (resultados de jobs assincronos)
- `permissions` / `quotas` (eventos administrativos)

Entidades fora dessa lista sao puramente request/response. Nao inflar.

## Jobs assincronos

Frontend nao conhece conceito de cron, fila, worker ou job. Para o frontend, a abstracao e **Operation**.

Fluxo canonico:
```
1. POST /campaigns/:id/dispatch
     -> 202 { operation_id: "op_..." }

2. Cliente armazena operation_id, exibe estado "Em andamento..."

3a. Poll: GET /operations/op_... a cada 2s (timeout 5min, backoff em erro)
3b. OU push: se request inicial passou ?push=true, servidor envia WS
    { type: "operation.completed", operation_id, result }

4. Status terminal: completed / failed / cancelled
   -> cliente atualiza UI, limpa polling
```

Endpoint `GET /operations/:id`:
```json
{
  "id": "op_...",
  "type": "campaign.dispatch",
  "status": "running",
  "started_at": "2026-04-15T12:00:00Z",
  "ended_at": null,
  "error": null,
  "progress": { "done": 340, "total": 1200 }
}
```

- `progress` e opcional; so presente se o job reporta.
- `error` e objeto `{ code, message, details }` identico ao formato de erro HTTP.
- Hook `useOperation(id)` encapsula poll + push + cleanup. Componentes consomem um valor reativo.

## Anti-padroes

- Polling de listas a cada N segundos. **Nao.** Use WS + invalidacao.
- Expor `cron_expression` ou `job_id` no frontend. **Nao.** A abstracao e Operation.
- Manter estado de socket em Context/Redux. **Nao.** Singleton em modulo + hooks = o suficiente.
- Fazer logica de negocio no handler de WS. **Nao.** O handler so atualiza cache. Regras vivem no servidor.

## Referencias

- [[Arquitetura do Sistema]]
- [[Contratos e Boundaries]]
- [[Autenticacao e Autorizacao]]
- [[Seguranca Web]]
- [[Glossario e Vocabulario]]
