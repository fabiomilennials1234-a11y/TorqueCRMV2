---
tags:
  - features
  - F01
  - design
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# F01 — Design

Stub. Preencher durante execução.

## Wireframes

_A ser desenhado._

- `/pipes` — hub: grid responsivo de `PipeCard` (cols: 1 / 2 / 3 / 4 por breakpoint)
- `/pipes/whatsapp` — kanban com header sticky, 5 colunas scrolláveis horizontalmente em viewports menores
- `LeadDetailDrawer` — largura 480px em desktop, fullscreen em mobile

## Tokens específicos usados

- **Stages (Pipe WhatsApp):** `--stage-whatsapp-novo`, `--stage-whatsapp-abordado`, `--stage-whatsapp-respondeu`, `--stage-whatsapp-esfriou`, `--stage-whatsapp-agendado`
- **Canal:** `--channel-whatsapp` (usado em `ChannelBadge`)
- **Calor:** `--heat-frio`, `--heat-morno`, `--heat-quente`, `--heat-muito-quente`
- **Surface elevada** nos cards do kanban
- **Hairline shadows** em vez de borders

## Motion patterns

- **Drag start:** `scale(1.02)` + sombra elevada em 120ms ease-out
- **Drag over coluna:** background da coluna aquece sutilmente
- **Drop:** `spring` com pequeno bounce (stiffness 300, damping 22)
- **Realtime patch:** pulse de 400ms (background accent → surface)
- **Rollback:** shake horizontal curto (3 oscilações, 200ms total)

## Skeleton variants

- **Kanban loading:** 5 colunas × 4 cards ghost (heights variados para não parecer repetitivo)
- **Hub loading:** 4 `PipeCard` skeleton
- **Drawer loading:** header + 3 blocos de conteúdo

## Empty state variants

- **Zero leads globalmente:** ilustração minimal + "Você ainda não tem leads" + CTA primária "Importar leads" + secundária "Criar lead manualmente"
- **Zero no filtro atual:** sem ilustração (mais leve) + "Nenhum lead bate com esse filtro" + CTA "Limpar filtro"
- **Coluna vazia:** texto curto e cinza no centro da coluna ("Nada aqui ainda.")

## Decisões visuais pendentes

- [ ] Ícone de cada estágio (explorar minimalismo cinematográfico — não usar emojis)
- [ ] Microcopy exato dos empty states (casar tom com referências)
- [ ] Animação do `PipeCard` no hub ao hover (elevação vs tilt)
- [ ] Tratamento visual quando lead está em múltiplos funis (badge discreto vs marcador)

## Referências

- [[F01 - Funis Hub e Pipe WhatsApp/Spec]]
- [[F01 - Funis Hub e Pipe WhatsApp/Tasks]]
- [[Escopo do Sistema Base]]
