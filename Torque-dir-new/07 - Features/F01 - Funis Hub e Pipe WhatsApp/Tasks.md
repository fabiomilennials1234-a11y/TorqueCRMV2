---
tags:
  - features
  - F01
  - tasks
  - execucao
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# F01 — Tasks

Lista numerada de execução. Cada item é rastreável e marcável.

1. **Gerar contratos** das entidades envolvidas (`Lead`, `Pipe`, `PipeRecord`, `Stage`) em `contracts/manual.ts` até OpenAPI cobrir; rodar `generate:types`.
2. **Criar hook `useLeads()`** com React Query cursor-based em `features/pipes/hooks/useLeads.ts`.
3. **Criar hook `usePipeWhatsapp()`** com realtime subscription (`lead.created`, `lead.updated`, `pipe_record.moved`) + invalidation inteligente.
4. **Criar componente `PipeCard`** em `features/pipes/components/PipeCard.tsx` (nome, ícone, total, delta 7d).
5. **Criar página `/pipes`** com grid responsivo de `PipeCard`, consumindo `useListPipes()`.
6. **Criar página `/pipes/whatsapp`** com 5 colunas canônicas e header sticky.
7. **Criar `DraggableKanbanBoard`** com `@dnd-kit/core` (sensors pointer + keyboard, `closestCenter`).
8. **Criar `LeadCard` + `KanbanColumn`** em `features/pipes/components/`.
9. **Criar `LeadDetailDrawer`** lazy-loaded via `React.lazy` + `Suspense`.
10. **Fiar realtime patches** com invalidation inteligente (setQueryData em vez de invalidateQueries quando possível).
11. **Testes de DnD** — acessibilidade com teclado, leitor de tela, axe-core.
12. **QA end-to-end** — rede lenta (throttle 3G), reconexão WS (matar/reabrir), permission denied mid-drag (simular 403).

## Referências

- [[F01 - Funis Hub e Pipe WhatsApp/Spec]]
- [[F01 - Funis Hub e Pipe WhatsApp/Design]]
- [[F01 - Funis Hub e Pipe WhatsApp/Checklist de Conclusao]]
