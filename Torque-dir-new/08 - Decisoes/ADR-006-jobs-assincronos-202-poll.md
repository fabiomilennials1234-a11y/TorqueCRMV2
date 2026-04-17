---
tags: [adr, architecture]
created: 2026-04-15
last_updated: 2026-04-15
status: accepted
id: ADR-006
---

# ADR-006 — Jobs assíncronos via 202 Accepted e poll

## Contexto

Várias operações do produto têm duração inerente não-trivial: importação de leads a partir de CSV ou planilha externa, provisioning de uma nova organização com seus seeds e configurações iniciais, cálculo de embeddings para o Copilot sobre histórico do tenant, recomputação de métricas agregadas. Segurar a requisição HTTP aberta por minutos não é opção — timeout do load balancer, experiência do usuário, custo de conexões ociosas no backend — e rodar síncrono em background sem feedback ao cliente é pior ainda, porque some a visibilidade de progresso e de erro.

O legado tratava isso de forma implícita, misturando conceitos de "cron job" e "operação longa" no mesmo modelo mental. Isso vazou para o front: havia UIs que mostravam "próximo job às X:XX", conceitos de schedule na interface, e assunções de que certas coisas aconteciam em lotes. Para o domínio do front, cron é detalhe de infra — o que importa é se a operação começou, em que ponto está, e se terminou (com sucesso ou erro).

A decisão precisa combinar duas coisas: padrão HTTP claro para operações longas, e uma estratégia de notificação que seja resiliente a desconexão do cliente (trocar de rede no meio, fechar e reabrir a aba) sem perder o resultado.

## Decisão

Padrão `202 Accepted` com poll, otimizado por push via WebSocket quando disponível.

- Operações longas respondem `202 Accepted` com body `{ operation_id, status: "pending", push: true }`.
- Cliente imediatamente faz `GET /operations/:id` e começa a poll a cada 2 segundos, com timeout máximo de 5 minutos.
- Quando `push: true` está presente e a conexão WS está ativa, o cliente escuta o evento `operation.completed` (ou `operation.failed`, `operation.progress`) e encerra o poll assim que recebe. Isso torna o caminho feliz quase instantâneo sem depender só de poll.
- Se a WS cair durante o processamento, o poll continua como fallback — o cliente nunca perde o resultado porque `GET /operations/:id` é a fonte de verdade.
- Shape de `Operation`: `{ id, type, status, progress: 0..1 | null, started_at, ended_at, error: { code, message } | null, result: object | null }`.
- Status: `pending`, `running`, `completed`, `failed`, `cancelled`.
- Cron jobs do legado deixam de existir como conceito no front; qualquer UI que precise refletir algo periódico consome o resultado via operações ou recursos normais.

## Alternativas consideradas

- **Push-only via WebSocket** — servidor dispara evento ao terminar, sem endpoint de status — descartado porque se a conexão WS cair durante o processamento, o cliente perde o evento e não tem como recuperar o estado sem sinal explícito.
- **Long-polling HTTP** — request segura aberta até a operação terminar — descartado por gastar conexão ociosa, esbarrar em timeouts de proxy/LB, e não escalar bem em número de operações simultâneas.
- **Callbacks registrados** — cliente passa URL para receber webhook ao terminar — descartado porque é excessivo para um front SPA (precisaria de endpoint público, autenticação de callback, relay); apropriado para integrações server-to-server, não para UI.
- **Polling puro sem otimização por push** — descartado porque piora latência percebida e gasta requests desnecessários no caminho feliz; o push via WS que já existe (ADR-002) é essencialmente gratuito.
- **Cron-as-domain** — manter o conceito de cron no front, como no legado — descartado porque vaza detalhe de infra e complica o modelo mental sem benefício ao usuário.

## Consequências

**Positivas**

- Resiliente a reconexão: o poll é a garantia, o push é a otimização.
- Padrão uniforme para todas as operações longas — uma UI (`OperationStatus`), um hook (`useOperation`), um shape.
- Cancelamento fica possível via `DELETE /operations/:id` com o mesmo modelo.
- O conceito de "cron" some do domínio do front, simplificando o modelo mental e os componentes.

**Negativas**

- Operações que terminam em menos de 2 segundos ainda pagam um poll tick antes de serem vistas como concluídas (mitigado pelo push via WS no caminho feliz).
- Servidor precisa manter um store de operações (Postgres ou Redis) com TTL; novo recurso para operar e observar.
- Front precisa lidar com o caso de operação desaparecida (TTL expirado) com UX clara, não com erro genérico.
- Progresso (`progress: 0..1`) é opcional e fica a cargo do backend emitir com granularidade útil — há risco de operações que só reportam 0 e 1.

## Impacto

- Endpoint `GET /operations/:id` e `DELETE /operations/:id` no Go, mais store de operações.
- Eventos `operation.progress`, `operation.completed`, `operation.failed` no protocolo WS (ADR-002).
- Componente `OperationStatus` no front — visual consistente de pending/running/progress/completed/failed.
- Hook `useOperation(id)` combinando poll + listener WS, com API `{ operation, isLoading, error, cancel }`.
- Handlers do Go: todos os handlers de operações longas retornam `202` com `operation_id` em vez de processar síncrono.

## Próximos passos

1. Definir shape exato de `Operation` em OpenAPI e espelhar no front.
2. Implementar store de operações no backend (Postgres com TTL ou Redis com expiração).
3. Implementar hook `useOperation(id)` com poll + bridge WS + cancel.
4. Criar componente `OperationStatus` cobrindo todos os estados com estética cinematográfica (não spinner genérico).
5. Migrar primeiro caso concreto (importação de leads) para validar o padrão end-to-end.

## Links

- [[Realtime e Jobs]]
- [[ADR-002-realtime-websocket-go-hub]]
- [[ADR-001-contratos-openapi-snake-camel]]
- [[Arquitetura do Front]]
