---
tags: [adr, architecture]
created: 2026-04-15
last_updated: 2026-04-15
status: accepted
id: ADR-002
---

# ADR-002 — Realtime via WebSocket com hub em Go

## Contexto

O Supabase Realtime entregava eventos `postgres_changes` com a row completa no payload, o que tornava trivial alimentar caches do front: qualquer alteração em uma tabela chegava com o estado pós-mudança, e bastava invalidar ou substituir a entrada local. A saída do Supabase remove essa primitiva e obriga o novo backend Go a definir um protocolo de realtime próprio, já que não existe equivalente drop-in em Go.

O espaço de soluções é amplo — STOMP, Centrifugo, SSE, WebSocket raw, MQTT — e cada opção tem implicações diferentes sobre complexidade operacional, controle do protocolo, ergonomia no cliente, e capacidade de cobrir cenários bidirecionais (takeover de lead, typing indicator em threads, cursor compartilhado em colaboração futura). A decisão precisa escolher a primitiva que permite o controle mais fino com o menor peso operacional, e definir o shape do protocolo que vai durar anos.

Há uma segunda questão embutida: os eventos não podem mais entregar rows completas por default. Patches são mais baratos no fio, mais simples de autorizar campo-a-campo, e alinham melhor com a estratégia de cache baseada em React Query. O front, portanto, precisa ser reescrito para pensar em patches versionados, não em estado completo sobrescrito.

## Decisão

WebSocket bidirecional simples, com hub pattern em Go usando `nhooyr.io/websocket`, protocolo JSON próprio e broadcast filtrado por tenant no servidor.

- Biblioteca: `nhooyr.io/websocket` pela API context-first idiomática e superfície menor que `gorilla/websocket`.
- Hub: uma goroutine por conexão; registro central por `tenant_id`; broadcast sempre passa por filtro de autorização antes de escrever no socket.
- Protocolo: mensagens JSON com shape `{ type, tenant_id, entity_id, patch, version, occurred_at }`. `type` segue convenção `<recurso>.<ação>` (ex.: `lead.updated`, `conversation.message_appended`). `patch` é um objeto parcial — nunca a entidade inteira. `version` é inteiro monotônico por entidade, usado para dedup e ordenação.
- Autorização no servidor: o filtro por `tenant_id` é aplicado antes do broadcast; o cliente nunca recebe eventos fora do seu tenant, independente do que pede.
- Cliente: `src/lib/ws.ts` encapsula conexão, reconnect exponencial com jitter, dedup por `version`, e faz bridge para invalidação/patching de caches do React Query por recurso.

## Alternativas consideradas

- **Centrifugo** — servidor realtime dedicado com SDKs maduros e canais bem modelados — descartado pela dependência operacional extra (mais um binário, mais um ponto de falha, mais um sistema para observar) sem ganho funcional que justifique em um sistema onde já existe backend Go próprio.
- **STOMP sobre WebSocket** — protocolo estruturado com conceito de frames e subscriptions — descartado por ser superdimensionado para o caso de uso e por não existir biblioteca Go server-side idiomática e bem mantida.
- **Server-Sent Events** — stream HTTP unidirecional, simples de operar, reconexão nativa — descartado porque é unidirecional e bloqueia cenários bidirecionais de produto (takeover de lead com lock otimista, typing em threads, presença).
- **gorilla/websocket** — biblioteca mais antiga e popular — descartado porque `nhooyr.io/websocket` tem API menor, context-first, e melhor ergonomia com goroutines modernas; popularidade não é critério.

## Consequências

**Positivas**

- Protocolo simples, totalmente sob controle; mudanças não dependem de roadmap de terceiros.
- Autorização por tenant acontece no ponto certo (servidor), sem confiar em filtragem do cliente.
- Bidirecional desde o dia 1, sem rework para presença, typing, takeover.
- Payload de patch é ordens de magnitude menor que row completa em entidades grandes.

**Negativas**

- Responsabilidades que antes eram do Supabase (reconnect, backoff, dedup) agora são do cliente e precisam ser implementadas com rigor.
- Eventos entregam patches, não rows completas — a lógica de merge no cache é nova e precisa lidar com ordem, idempotência e versioning.
- Um hub em processo não escala horizontalmente sem broker (Redis pub/sub ou NATS) em múltiplas instâncias; aceitar monolito vertical até sinal concreto de necessidade horizontal.
- Sem biblioteca cliente pronta; `ws.ts` é código de infra que o time precisa manter e testar.

## Impacto

- `src/lib/ws.ts` — conexão, reconnect, dedup, bridge para React Query.
- Hub no servidor Go — provavelmente em `internal/realtime/` com `hub.go`, `client.go`, `protocol.go`.
- Hooks de recursos no front — consumem patches em vez de refetch cego.
- [[Realtime e Jobs]] — documentação do protocolo e do fluxo.

## Próximos passos

1. Esqueleto de `src/lib/ws.ts` com reconnect exponencial + jitter e handler registry por `type`.
2. Implementação do hub em Go com filtro por `tenant_id` e testes de concorrência.
3. Protocolo versionado e documentado em [[Realtime e Jobs]] com exemplos de cada `type`.
4. Estratégia de merge de patches no cache React Query para os 3 recursos iniciais (leads, conversations, messages).

## Links

- [[Realtime e Jobs]]
- [[Arquitetura do Front]]
- [[ADR-001-contratos-openapi-snake-camel]]
- [[ADR-006-jobs-assincronos-202-poll]]
