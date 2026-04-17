# STATE.md — Torque CRM v2

> Last updated: 2026-04-17

## Current Phase
Sistema Base Frontend — Final Etapa 6 (paginas-casca), before Etapa 7 (validation).

## Key Decisions

| ID | Date | Decision | Rationale |
|----|------|----------|-----------|
| D001 | 2026-04-15 | Vault Obsidian criado como fonte de verdade | Substituir vault legado, comecar limpo |
| D002 | 2026-04-15 | 6 ADRs formalizados (OpenAPI, WS, Auth, Cursor, Fonts, Jobs) | Decisoes arquiteturais travadas antes de implementar |
| D003 | 2026-04-15 | 9 agent profiles documentados no vault | Conductor, Architect, Backend, Frontend, DBA, QA, Infra, Automation, AI |
| D004 | 2026-04-15 | Etapa 0 blockers (B1-B6) identificados e 5/6 resolvidos | B6 (ESLint) continua sem regras reais |
| D005 | 2026-04-16 | Plano Mestre de Finalizacao criado (30 sprints) | Roadmap operacional completo ate producao |
| D006 | 2026-04-16 | Vault reorganizado (12 secoes, zero dualidades) | Duas taxonomias paralelas mergeadas em uma |
| D007 | 2026-04-17 | CLAUDE.md + .specs/ + agent skills criados | Protocolos de agentes e skills extraidos do legado e adaptados para Go |

## Current Blockers
- B6: ESLint com zero regras (pendente Sprint S00)
- Git: Projeto NAO e repositorio Git (pre-requisito urgente)

## Lessons Learned
- Documentacao do vault pode ficar significativamente desatualizada vs codigo
- Sempre verificar estado real do codigo antes de confiar em status documentados
- Duas taxonomias paralelas no vault causam confusao — cada numero deve ter UMA pasta

## Preferences
- CTO padrao: world-class, dark-first, Go backend, seguranca desde commit 1
- Vault e fonte de verdade para decisoes e specs
- Agentes usados proativamente, com briefings densos e autossuficientes
- Uma feature por vez (regra cardinal)
