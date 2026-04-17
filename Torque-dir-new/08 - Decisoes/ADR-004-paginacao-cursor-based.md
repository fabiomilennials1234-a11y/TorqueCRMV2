---
tags: [adr, architecture]
created: 2026-04-15
last_updated: 2026-04-15
status: accepted
id: ADR-004
---

# ADR-004 — Paginação cursor-based em todas as listas

## Contexto

As listas principais do produto — leads, conversas, mensagens, atividades — podem crescer para milhões de registros por tenant em alguns dos perfis de cliente-alvo. Offset/limit é o padrão mental mais comum, mas tem dois problemas estruturais que não se resolvem com ajuste: o custo no banco cresce com `OFFSET N` porque o motor ainda precisa varrer e descartar os N registros anteriores, e o cursor lógico (número da página) é invalidado a cada insert ou delete que acontece durante a sessão do usuário.

O segundo problema é particularmente ruim em um produto com realtime. Enquanto o usuário olha a página 3, uma inserção na tabela faz o item que estava no topo da página 4 aparecer duplicado na próxima navegação, ou um delete faz um item da página 2 desaparecer silenciosamente da lista renderizada. Paginação numérica força uma escolha entre experiência consistente e dados frescos, e a resposta correta em um produto moderno é não ter que escolher.

Cursor-based paginação resolve ambos: o ponteiro é semântico (uma posição no sort, não um número), estável a inserts e deletes fora da janela, e permite índices adequados para custo constante por página independente da profundidade.

A decisão também precisa definir o shape do cursor (opaco vs. estruturado), o contrato de resposta, e como filtros e sort compõem na query string.

## Decisão

Todas as listas paginadas usam cursor opaco, com contrato uniforme de resposta e filtros declarativos na query string.

- Contrato de resposta: `{ items: T[], next_cursor: string | null, total?: number }`. `next_cursor` é `null` quando não há mais páginas. `total` é opcional e só é computado quando o cliente passa `include=total`.
- Cursor é opaco: base64 de `{id, sort_value, sort_direction}` assinado ou checado no servidor; o cliente trata como string impenetrável. Expor chaves de ordenação diretamente vazaria detalhes de schema e permitiria manipulação.
- Filtros: `filter[campo]=valor` para igualdade, `filter[campo][op]=valor` para operadores (`gte`, `lte`, `in`, `like`).
- Sort: `sort=-created_at,name` — prefixo `-` para descendente, vírgula separa tie-breakers; o `id` é sempre implicitamente o último tie-breaker para garantir estabilidade.
- Cliente: `useInfiniteQuery` do React Query alimenta todas as listas; helper `useInfiniteList(key, queryFn)` padroniza assinatura.
- Backend: todas as colunas usadas em sort têm índice composto com `id` para suportar keyset pagination eficiente.

## Alternativas consideradas

- **Offset/limit** — descartado porque não escala em profundidade (custo linear com offset) e quebra a cada mutação concorrente na tabela.
- **Page numbers com total count** — descartado porque força `COUNT(*)` caro a cada request, UX de paginação numérica fica ruim com realtime invalidando a contagem, e não resolve o problema de inserts deslocarem páginas.
- **Keyset sem cursor opaco** — cliente manda o último `id` e `created_at` explicitamente — descartado porque expõe detalhes de schema, permite manipulação (cliente pode pedir qualquer ponto arbitrário), e impede mudanças futuras na chave de ordenação sem breaking change.
- **Timestamp-based sem id tie-breaker** — descartado porque colisões de timestamp em alta concorrência causam itens duplicados ou ausentes na borda.

## Consequências

**Positivas**

- Custo por página constante em relação à profundidade, desde que o sort tenha índice.
- Estável sob inserts e deletes fora da janela atual do usuário.
- Back/forward nativos via histórico de cursors no cliente (cada página guarda o cursor que a trouxe).
- Contrato uniforme simplifica o front: um único hook, um único shape, zero casos especiais.

**Negativas**

- Jumping to arbitrary page (ir para página 37) não existe como primitiva; aceitar isso como decisão de produto, não como limitação.
- Exige disciplina de índices no banco; sort em coluna sem índice degrada rápido e silenciosamente.
- `total` opcional é um leve ajuste de expectativa para UIs que queiram mostrar "X resultados"; quando pedido, vem com custo.
- Cursor opaco dificulta debugging manual (precisa decodificar para entender); aceitar em troca do encapsulamento.

## Impacto

- `src/api/*` — todos os clients de recurso implementam o contrato uniforme.
- Hooks de lista — `useLeads`, `useConversations`, etc. passam por `useInfiniteList`.
- Endpoints Go — handler helper padronizado para serializar cursor, ler filtros e sort.
- SQL — índices compostos em colunas de sort mais usadas (leads por `created_at,id`, conversations por `updated_at,id`).
- OpenAPI — convenção documentada e aplicada em todos os endpoints de lista.

## Próximos passos

1. Documentar formato interno do cursor (campos, encoding, assinatura se houver).
2. Implementar helper `useInfiniteList(key, queryFn)` com integração a `useInfiniteQuery`.
3. Codificar convenção em OpenAPI (parâmetros padrão `cursor`, `filter[*]`, `sort`, `include`).
4. Criar checklist de índices para o time de banco antes de cada novo endpoint de lista.

## Links

- [[Arquitetura do Front]]
- [[ADR-001-contratos-openapi-snake-camel]]
- [[ADR-002-realtime-websocket-go-hub]]
- [[Padrões de API]]
