---
tags:
  - features
  - F01
  - checklist
  - conclusao
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# F01 — Checklist de Conclusão

**Só avança para F02 quando 100% marcado.**

- [ ] Contratos das 3 entidades (`Lead`, `Pipe`, `PipeRecord`) finalizados e gerados via `generate:types`
- [ ] Hook `useLeads` com paginação cursor testado (unit + integration)
- [ ] Hook `usePipeWhatsapp` com subscribe + invalidation inteligente
- [ ] `/pipes` renderiza 4 cards estruturais + customs da organização
- [ ] `/pipes/whatsapp` com 5 stages corretos (`novo / abordado / respondeu / esfriou / agendado`)
- [ ] DnD funcional com teclado (setas + espaço, conforme `@dnd-kit` a11y)
- [ ] DnD com reordenação intra-coluna e movimento cross-coluna
- [ ] Optimistic update com rollback em erro server
- [ ] Permission gate em drop (403 de permissão vira toast + rollback)
- [ ] `LeadDetailDrawer` lazy-loaded (verificado no bundle analyzer)
- [ ] `Skeleton` em loading, `EmptyState` em vazio, error boundary em erro
- [ ] Realtime: abrir 2 tabs, mover em uma, ver refletir na outra em < 1s
- [ ] Reconexão WS: matar conexão, voltar, estado consistente com servidor
- [ ] Testes e2e mínimos em Playwright: criar lead, mover card, abrir drawer
- [ ] Documentado em [[05 - Features/F01 - Funis Hub e Pipe WhatsApp/Spec|F01 Spec]] completo
- [ ] Code review independente aprovado
- [ ] Zero uso de `border` do Tailwind (só `shadow-hairline-*`)
- [ ] Zero violação dos [[Criterios de Reprovacao]]

---

## Critério de saída

**100% marcado. Sem exceção.**

Ao bater 100%, abrir o próximo item em [[00 - Mapa de Features]] (F02 — Pipe Confirmação).

## Referências

- [[F01 - Funis Hub e Pipe WhatsApp/Spec]]
- [[F01 - Funis Hub e Pipe WhatsApp/Design]]
- [[F01 - Funis Hub e Pipe WhatsApp/Tasks]]
- [[00 - Mapa de Features]]
