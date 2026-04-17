---
tags:
  - features
  - F01
  - funis
  - pipe-whatsapp
  - feature-spec
type: feature-spec
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# F01 — Funis Hub e Pipe WhatsApp

## Visão

F01 é o **vertical slice que prova o pipeline técnico inteiro**: contratos gerados, HTTP autenticado, realtime via WS, DnD acessível, permissões enforçadas, estados de UI completos. Entrega um hub de funis (`/pipes`) e o primeiro funil completo (`/pipes/whatsapp`) com kanban funcional. É a feature em que toda decisão de arquitetura do Sistema Base precisa se provar — se algo estiver errado na base, essa feature expõe.

## Jobs-to-be-done

- Quando recebo um lead novo, quero vê-lo cair no estágio `novo` automaticamente para começar a trabalhar.
- Quando abordo um lead, quero mover o card para `abordado` em um gesto, sem formulário.
- Quando o lead responde, quero ver o card mudar de estágio em tempo real, mesmo sem estar com a aba focada.
- Quando o lead esfria, quero registrar isso para não voltar a ele sem razão.
- Quando o lead topa uma reunião, quero marcar `agendado` e disparar a confirmação.
- Quando um colega de time move um card, quero ver a mudança refletir na minha tela sem F5.

## Telas e rotas

- `/pipes` — **Hub de funis.** Grid de `PipeCard` com 4 pipes estruturais (WhatsApp, Confirmação, Propostas, Pós-venda) + customs da organização.
- `/pipes/whatsapp` — **Kanban do Pipe WhatsApp** com 5 colunas (stages canônicos).

## Stages canônicos do Pipe WhatsApp

```
novo → abordado → respondeu → esfriou → agendado
```

**Não confundir** com os stages genéricos de mockup antigo. Esses 5 são os corretos e estão documentados em `src/lib/vocabulary.ts` e no `manual.ts`.

## Componentes novos

- `PipeCard` — card no hub, exibe nome, ícone, total de leads, delta 7d
- `DraggableKanbanBoard` — orquestrador de DnD com `@dnd-kit/core`
- `KanbanColumn` — coluna com header, contagem, drop target, empty interno
- `LeadCard` — card draggable com nome, canal, calor, última atividade, próxima ação
- `LeadDetailDrawer` — drawer lateral (lazy-loaded) com detalhes, histórico, ações

## Contratos de API

- `GET /leads?cursor=&filter[stage]=`
	- Response: `{ data: Lead[], next_cursor: string | null }`
- `POST /leads`
	- Request: `{ name, channel, source, ... }`
	- Response: `Lead`
- `PATCH /leads/:id`
	- Request: `Partial<Lead>`
	- Response: `Lead`
- `GET /pipes/whatsapp`
	- Response: `{ stages: Stage[], counts: Record<StageId, number> }`
- `PATCH /pipes/whatsapp/:id/stage`
	- Request: `{ stage: StageId, position: number }`
	- Response: `PipeRecord`

Conversão snake↔camel via `contracts/transformers.ts` (definido no Sistema Base).

## Eventos de realtime consumidos

- `lead.created` — inserir card no estágio correspondente (`queryClient.setQueryData` com merge)
- `lead.updated` — patch no card sem refetch
- `pipe_record.moved` — reordenar / mudar coluna com animação sutil

**Patches aplicados direto** no novo modelo. Debounce de 2s do modelo antigo não é mais necessário — o backend já agrega eventos antes de publicar.

## Permissões relevantes

- `pipeline.view` — acessar `/pipes` e `/pipes/*`
- `create_lead` — botão "Novo lead" e ação de criar
- `move_pipe_record` — habilitar DnD (sem essa, cards são read-only visualmente)
- `view_lead` — abrir `LeadDetailDrawer`

Todas checadas via `<PermissionGate>` + `useCanPerformAction`.

## Estados de UI

- **Loading:** skeleton de **5 colunas × 4 cards ghost** no kanban. Hub com 4 `PipeCard` skeleton.
- **Empty global:** `EmptyState` "Você ainda não tem leads" + CTA "Importar leads".
- **Empty com filtro ativo:** `EmptyState` "Nenhum lead bate com esse filtro" + "Limpar filtro".
- **Error:** boundary com botão "Tentar novamente" + link para status.
- **Realtime updating:** pulse sutil (animação de 400ms) no card que acabou de receber patch.
- **Saving / moving:** card fica com opacity 0.8 enquanto a mutation está em voo; rollback com shake se erro.

## Edge cases

- **Lead em múltiplos pipes simultâneos:** mesmo lead pode estar em WhatsApp e Confirmação. `pipe_record` é a entidade de posição; `lead` é única. UI mostra badge indicando presença em outros funis.
- **Lead shadow (`is_shadow: true`):** não aparece no kanban até ser "promovido". Filtro default esconde shadows.
- **Drop em coluna sem permissão:** o DnD até permite o gesto, mas a mutation retorna 403. Toast claro + rollback para posição original.
- **Perda de conexão WS durante drag:** o gesto local conclui, mutation sobe normalmente; reconexão reconcilia.
- **Reordenação intra-coluna:** posição é `integer` com espaçamento — usa estratégia de rebalanceamento no backend quando gaps esgotam.

## Métricas de sucesso

- **Tempo médio de move percebido < 300ms** (optimistic update + animação).
- **Zero perda de state em reconexão WS** (teste: matar conexão, mover card em outra tab, reconectar, ver estado correto).
- **100% das rotas com `<PermissionGate>`** (lint rule assegura).
- **DnD navegável por teclado** (auditoria com axe + teste manual com screen reader).

## Dependências

- **Sistema Base 100%** — [[Checklist Sistema Base]]
- **Contratos:** `Lead`, `Pipe`, `PipeRecord`, `Stage`
- **Componentes UI:** `Skeleton`, `EmptyState`, `Drawer`, `Button`, `Badge`, `Card`, `ChannelBadge`
- **Hooks:** `useSession`, `useCanPerformAction`, `useWSStatus`

## Decisões tomadas

- **DnD:** `@dnd-kit/core` (acessível por teclado, compatível com mobile, sem legado de react-dnd). Sensors: pointer + keyboard. Collision: `closestCenter`.
- **Optimistic update:** atualização local imediata + rollback on error. Sem "loading entre move e confirmação" — o usuário não deve esperar o backend.
- **Paginação:** cursor-based desde o início. Nunca offset.
- **Realtime:** patches aplicados direto; sem debounce no client.
- **Drawer de detalhe:** `lazy-loaded` via `React.lazy` — não pesa bundle de `/pipes/whatsapp`.

## Fora de escopo

- Bulk actions (selecionar múltiplos leads + mover)
- Filtros avançados com combinadores booleanos
- Exportação CSV/XLSX
- Timeline de atividades detalhada (vai para F04/F05)
- Envio de mensagem direto do card (vai para F04 Inbox)

## Checklist de conclusão

Ver [[F01 - Funis Hub e Pipe WhatsApp/Checklist de Conclusao]].

## Referências

- [[00 - Mapa de Features]]
- [[Template - Feature Spec]]
- [[F01 - Funis Hub e Pipe WhatsApp/Design]]
- [[F01 - Funis Hub e Pipe WhatsApp/Tasks]]
- [[F01 - Funis Hub e Pipe WhatsApp/Checklist de Conclusao]]
