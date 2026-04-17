---
tags: [design, exploracao, alternativa, light-dark, glassmorphism]
created: 2026-04-15
last_updated: 2026-04-15
status: exploracao
type: proposal
---

# Exploracao Visual Alternativa

**Status:** EXPLORACAO — nao revoga principios documentados. Implementada no trunk para avaliacao pelo fundador. Pode ser revertida integralmente.

## Contexto

O fundador solicitou explorar uma direcao visual alternativa testando: light/dark mode funcional, tipografia mais empresarial, glassmorphism/neumorphism controlados, reducao de poluicao visual e uso de assets oficiais da marca.

Os principios documentados em [[Principios de Identidade Visual]] e [[Criterios de Reprovacao]] permanecem intactos como referencia. Esta exploracao testa se uma direcao mais corporativa/sobria agrada antes de revogar qualquer decisao.

## O que mudou

### 1. Light/Dark mode funcional
- `ThemeProvider` em `src/providers/ThemeProvider.tsx`
- Toggle Sun/Moon no TopBar
- Persistencia em localStorage (`torque-theme`)
- Deteccao de preferencia do sistema como fallback
- Script anti-FOUC no `index.html`
- Tokens `:root` (light) e `.dark` (dark) em `globals.css`

**Dark ajustado (cinza, nao preto puro):**
- bg: `220 15% 8%` (era `222 14% 4%`)
- surface: `220 14% 11%` (era `222 13% 6%`)
- elevated: `220 13% 15%` (era `222 13% 9%`)
- hairline: `220 10% 22%` (era `222 10% 16%`)

**Light (novo):**
- bg: `220 15% 97%` (off-white azulado)
- surface: `0 0% 100%` (branco puro para cards)
- elevated: `220 15% 97%`
- hairline: `220 10% 88%`
- ink: `220 20% 12%`

### 2. Tipografia
- **Display:** DM Serif Display (serif classico empresarial, substituiu Fraunces)
- **Body:** Inter (sans geométrica, substituiu Instrument Sans)
- **Metric:** JetBrains Mono (mantido)

Justificativa da exploracao: Fraunces e editorial-artsy, DM Serif Display e classico-profissional. Inter e o padrao do mercado mas transmite seriedade corporativa. A exploracao testa se o fundador prefere diferenciacao (Fraunces) ou sobriedade (DM Serif).

### 3. Glassmorphism controlado
Aplicado apenas em:
- **Sidebar:** `bg-surface/80 backdrop-blur-[8px]`
- **TopBar:** ja tinha blur, mantido
- **CommandPalette:** `bg-elevated/90 backdrop-blur-xl` no modal e overlay
- **Dropdown:** `bg-elevated/90 backdrop-blur-xl`
- **Sheet overlay:** `backdrop-blur-[16px]`

NAO aplicado em: badges, pills, tooltips, separadores, skeletons.

### 4. Neumorphism controlado
- **Cards hover:** `shadow-neu-hover` (shadow externo suave + inset hairline)
- **Input focus:** `shadow-neu-focus` (glow accent sutil)
- **Button primary:** `shadow-btn-primary` (flutuante suave em vez de forte)

### 5. Reducao de poluicao visual
- Grain overlay removido (`.grain::before` eliminado do globals.css)
- `ruler-top` decorativo removido do kanban
- Empty states simplificados (removido bg-dot-grid)
- PageHeader eyebrow: de accent para ink-dim (menos gritante)
- Nav items: h-8 para h-9, space-y-0.5 para space-y-1
- NavGroups: mb-4 para mb-6
- Cards: padding aumentado para mais respiro
- Body line-height: 1.5 para 1.6

### 6. Logo oficial do Torque
- `torque-logo.png` (texto branco) para dark mode
- `torque-logo-dark.png` (texto preto) para light mode
- `torque-icon.png` como marca central no login
- `torque-hexagons.png` como pattern sutil no painel editorial do login
- TorqueMark.tsx SVG mantido como componente secundario (breadcrumbs, error pages)

## Assets encontrados e catalogados

| Asset | Origem | Uso |
|---|---|---|
| torque-logo.png | v8milennialsb2bv2/src/assets/ | Sidebar dark mode |
| torque-logo-dark.png | v8milennialsb2bv2/src/assets/ | Sidebar light mode |
| torque-icon.png | v8milennialsb2bv2/src/assets/ | Login brand mark |
| torque-hexagons.png | v8milennialsb2bv2/src/assets/ | Login background pattern |
| logo-dark.png (MB2B) | v8milennialsb2bv2/src/assets/ | NAO USADO — marca antiga Milennials |
| logo-light.png (MB2B) | v8milennialsb2bv2/src/assets/ | NAO USADO — marca antiga Milennials |

## Arquivos criados (1)
- `src/providers/ThemeProvider.tsx`

## Arquivos modificados (19)
- `index.html` — removido class="dark" hardcoded, adicionado script anti-FOUC
- `src/main.tsx` — adicionado ThemeProvider
- `src/styles/globals.css` — tokens dual-theme, nova tipografia, removido grain
- `tailwind.config.ts` — fontes, shadows glass/neomorph
- `src/shell/Sidebar.tsx` — logo oficial, glassmorphism, respiro
- `src/shell/TopBar.tsx` — toggle de tema
- `src/shell/CommandPalette.tsx` — glassmorphism
- `src/shell/AppShell.tsx` — limpeza
- `src/ui/button.tsx` — shadow-btn-primary
- `src/ui/card.tsx` — hover neumorfico, padding
- `src/ui/input.tsx` — focus neumorfico
- `src/ui/empty-state.tsx` — simplificado
- `src/ui/page-header.tsx` — eyebrow simplificado
- `src/ui/dropdown.tsx` — glassmorphism
- `src/ui/sheet.tsx` — backdrop-blur overlay
- `src/features/auth/LoginPage.tsx` — logo oficial + hexagons pattern
- `src/features/dashboard/DashboardPage.tsx` — respiro KPIs
- `src/features/pipeline/KanbanPage.tsx` — removido ruler, respiro
- `src/features/errors/*.tsx` — removido grain

## Conflitos com principios documentados

| Principio vigente | O que esta exploracao faz | Revogacao necessaria |
|---|---|---|
| Dark-first sem toggle | Light/dark funcional | Sim — se aprovado |
| Fraunces + Instrument Sans | DM Serif Display + Inter | Sim — ADR-005 precisa ser revogada |
| Grain permanente | Grain removido | Sim — principio "sensibilidade cinematografica" |
| Density-first (px-2.5, py-1.5) | Mais respiro (padding aumentado) | Parcial — principio de compacidade |
| Hairline-only visual | Glassmorphism/neumorphism adicionado | Parcial — novo vocabulario visual |

## Decisao pendente do fundador

Se aprovado: atualizar [[Principios de Identidade Visual]], [[Design System Base]], [[Tipografia]], [[Motion e Animacao]], [[Criterios de Reprovacao]], revogar ADR-005, criar ADR-007 com nova direcao.

Se rejeitado: reverter os 19 arquivos modificados + 1 criado. Principios intactos.

Se parcial: cherry-pick itens aprovados (ex: logo oficial sim, Inter nao).

## Validacao

- Build: sucesso em 5.26s
- Zero erros introduzidos (pre-existentes em tipos Lucide permanecem)
- Light mode: funcional, testavel
- Dark mode: cinzas ajustados, menos preto puro
- Toggle: funcional com persistencia

## Referencias

- [[Principios de Identidade Visual]] (nao revogado)
- [[Criterios de Reprovacao]] (nao revogado)
- [[ADR-005-tipografia-self-hosted]] (nao revogado, mas conflita)
