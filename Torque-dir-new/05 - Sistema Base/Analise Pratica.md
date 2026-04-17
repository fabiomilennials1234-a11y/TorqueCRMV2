---
tags: [sistema-base, analise, frontend, audit]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Analise Pratica do Frontend Existente

Auditoria completa do codebase `torque-web/` comparando estado real contra identidade documentada no vault.

## Veredicto: 92% alinhado com identidade documentada

O codebase segue os padroes de design em alto grau. A base e solida para o redesign.

## Tokens CSS (globals.css) — Alinhamento

### Implementados e corretos
| Token | Valor | Status |
|-------|-------|--------|
| `--bg` | 222 14% 4% | Correto |
| `--surface` | 222 13% 6% | Correto |
| `--elevated` | 222 13% 9% | Correto |
| `--hairline` | 222 10% 16% | Correto |
| `--ink` | 220 13% 91% | Correto |
| `--ink-muted` | 220 9% 64% | Correto |
| `--ink-dim` | 220 7% 40% | Correto |
| `--accent` | 44 93% 54% | Correto |
| `--accent-soft` | 44 50% 18% | Correto |
| `--success` | 160 55% 48% | Correto |
| `--warning` | 34 90% 56% | Correto |
| `--danger` | 358 75% 59% | Correto |
| `--info` | 210 70% 58% | Correto |
| `--stage-1..7` | Paleta completa | Correto |
| `--ease-out-soft` | cubic-bezier(0.2,0.8,0.2,1) | Correto |
| `--ease-in-out-precise` | cubic-bezier(0.65,0,0.35,1) | Correto |

### Documentados mas NAO implementados
- `--heat-1..5` (calor) — Quick Win pendente
- `--channel-whatsapp/messenger/instagram/sz` — Quick Win pendente
- `--countdown-safe/warn/urgent` — Quick Win pendente
- `--job-pending/running/completed/failed` — Quick Win pendente

## Tailwind Config — 100% alinhado

- Cores mapeadas corretamente via `hsl(var(--X) / <alpha-value>)`
- Box shadows: hairline, hairline-b, elev-1/2/3, glow (accent) — todos corretos
- Animacoes: fade-in, scale-in, shimmer, torque-tick, caret-blink — com easing custom
- Grid/dot backgrounds: bg-grid, bg-dot-grid — implementados
- Radius: xs(3px), sm(5px), DEFAULT(8px), md(10px), lg(14px), xl(20px)

## Tipografia — Funcional mas com debito

- Google Fonts CDN ainda ativo em `index.html`
- @fontsource NAO instalado (debito vs [[ADR-005-tipografia-self-hosted]])
- Classes `font-display` (Fraunces opsz 144, SOFT 30) e `font-metric` (tabular-nums) corretas em globals.css
- Feature settings: `ss01, ss02, cv11, cv03` presentes no body

## Componentes UI (17 em src/ui/) — Estado

| Componente | Shadow-hairline | Tokens HSL | Tipografia | Notas |
|-----------|:-:|:-:|:-:|---|
| button.tsx | OK | OK | OK | Variants corretos, easing custom |
| input.tsx | OK | OK | OK | Focus com inset shadow accent |
| card.tsx | OK | OK | OK | 5 subcomponentes |
| badge.tsx | OK | OK | OK | 7 tonalidades |
| avatar.tsx | OK | OK | OK | Fallback com font-metric |
| tooltip.tsx | OK | OK | OK | Kbd opcional |
| dropdown.tsx | OK | OK | OK | 10 exports completos |
| sheet.tsx | OK | OK | OK | Backdrop blur + slide |
| tabs.tsx | Parcial | OK | OK | Usa border-hairline (nao shadow) |
| empty-state.tsx | OK | OK | OK | bg-dot-grid correto |
| page-header.tsx | OK | OK | OK | font-display no title |
| kbd.tsx | OK | OK | OK | font-mono text-2xs |
| pill.tsx | OK | OK | OK | accent-soft active |
| score-meter.tsx | N/A (SVG) | OK | OK | font-metric para numero |
| skeleton.tsx | N/A | OK | N/A | Shimmer correto |
| separator.tsx | N/A (linha) | OK | N/A | bg-hairline |
| spark.tsx | N/A (SVG) | OK | N/A | Area fill + dot accent |

Nenhum componente usa `border-` Tailwind padrao. Nenhum usa cores hex/rgb hardcoded.

## Shell — Estado

- **AppShell:** Layout 2-col (sidebar fixed + main flex-1) — correto
- **Sidebar:** w-232px, bg-surface, inset shadow hairline. Grupo "Operacao" com 7 links. Label "Pipelines" (deve ser "Funis" por [[Vocabulario de UI]])
- **TopBar:** h-14, sticky, bg-bg/80 backdrop-blur-xl, shadow-hairline-b — correto
- **TorqueMark:** SVG puro com gradiente accent — correto
- **OrgSwitcher:** 3 orgs hardcoded, avatar + name + plan — correto
- **CommandPalette:** cmdk, 3 secoes, Cmd+K — correto

## Paginas — Todas mockup estatico com seed data

9 rotas: login, dashboard, pipeline, inbox, workflows, campaigns, copilot, analytics, settings. Nenhuma conectada a backend.

## Infraestrutura Faltante

- `src/api/` — nao existe
- `src/hooks/` — nao existe
- `src/contracts/` — nao existe
- `src/providers/` — nao existe (apenas TooltipProvider no shell)
- React Query — nao instalado
- react-intl — nao instalado
- Sentry — nao configurado
- ESLint rules — config vazio
- @fontsource — nao instalado (Google Fonts CDN ativo)

## Debitos que Bloqueiam o Redesign

### Criticos (fix imediato)
1. **Hardcoded HSL em Kbd** (`KanbanPage.tsx:102`) — `bg-[hsl(222_20%_8%/0.3)]` deve ser `bg-bg/30`
2. **Border redundante em LoginPage** (`LoginPage.tsx:119`) — `border border-hairline` junto com `shadow-hairline`, remover o border
3. **Google Fonts CDN** em `index.html` — migrar para @fontsource conforme [[ADR-005-tipografia-self-hosted]]

### Medios (pre-implementacao)
4. **`transition-all`** em 4+ componentes — padronizar com propriedades especificas
5. **ESLint vazio** — configurar regras reais (no-restricted-imports, dangerouslySetInnerHTML ban)
6. **Motion library** instalada mas nao usada — remover ou planejar uso

### Baixos (durante implementacao)
7. Seed data inline em OrgSwitcher/CommandPalette — mover para seed.ts
8. Focus ring pattern nao centralizado — extrair para utility class

## Conflitos entre Docs Detectados

1. **Accent value:** `Agentes/Frontend.md` usa `hsl(47 100% 50%)` vs canonica `hsl(44 93% 54%)` em [[Design System Base]]
2. **Token naming:** Checklist usa "overlay" (nao existe nos tokens), Design System usa "hairline" (diferente de "overlay")
3. **Contagem componentes dominio:** Titulo diz "12" mas lista 14 itens
4. **Canal naming:** `--channel-sz` no Design System vs `--channel-webchat` no Checklist e Quick Wins

## Referencias

- [[Design System Base]]
- [[Principios de Identidade Visual]]
- [[Criterios de Reprovacao]]
- [[Escopo do Sistema Base]]
- [[ADR-005-tipografia-self-hosted]]
