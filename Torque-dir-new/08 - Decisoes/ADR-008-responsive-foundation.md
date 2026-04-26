---
tags:
  - adr
  - frontend
  - design-system
  - responsivo
status: aceito
date: 2026-04-26
sprint: S56
---

# ADR-008 — Responsive Foundation (mobile-first adaptive shell)

## Status

Aceito. Sprint S56 (Onda 1) — `2026-04-26`.

## Contexto

Frontend Torque foi construido como aplicacao desktop-only ate S54. Auditoria pre-S56 revelou apenas 22 arquivos em `src/` com utilities `sm:|md:|lg:|xl:|2xl:` num codebase de >300 arquivos React/TSX. Sintomas concretos:

- AppShell `flex h-screen w-screen overflow-hidden` + Sidebar `w-[232px]` fixa em **todas** as viewports → mobile recebe ~40% da largura util consumida pela sidebar fixa.
- TopBar com search bar `w-[340px]` literal + 5 botoes ghost lado a lado → quebra em viewports `<sm`.
- Modais via `<SheetContent side="right" max-w-[520px]">` → forma errada para mobile (slide lateral em tela 360px e estranho; padrao mobile-native e bottom sheet ou fullscreen).
- Touch targets: `Button size="sm"` = `h-8` (32px). Apple HIG e Material 3 exigem **44x44px minimo**.
- Tipografia: classes Tailwind fixas (`text-sm`, `text-base`, `text-lg`, `text-2xl`...) sem escala fluida. Display type estoura em mobile, fica pequeno em telas grandes.
- Sem safe-area iOS: app instalado como PWA em iPhone com notch sobrepoe TopBar.

A construcao para escala de decadas (CLAUDE.md global) **exige** que o produto responda a qualquer viewport com a mesma postura editorial dark-first cinematografica. Mobile-only ou desktop-only sao amputacoes.

## Decisao

**Adotar arquitetura adaptiva mobile-first com 5 breakpoints Tailwind canonicos, 3 primitivos UI mobile-aware, escala tipografica fluida e safe-area nativa em utilities.**

### 1. Breakpoints (mantidos, default Tailwind)

| Token | Min-width | Persona |
|-------|-----------|---------|
| (none) | 0px | mobile (iPhone SE 360px → iPhone Pro Max 430px) |
| `sm` | 640px | small tablet portrait |
| `md` | 768px | iPad portrait, viewport "comfort" |
| `lg` | 1024px | iPad landscape, MacBook 13" |
| `xl` | 1280px | desktop standard |
| `2xl` | 1536px | desktop large / external monitor |

**Regra de uso:** mobile-first. Estilos default = mobile. Prefixos `sm:`/`md:`/`lg:`/`xl:` sobrescrevem para cima. Nunca `<sm:` (Tailwind 4 nao suporta).

**Threshold critico — Sidebar fixa:** `lg` (≥1024px). Abaixo, virar drawer.

### 2. Shell adaptativo

**AppShell** — grid de 1 coluna em mobile. A `Sidebar` fixa so existe em `>=lg` (`hidden lg:flex`). Em `<lg`, `AppShell` controla um state `sidebarOpen` que abre `Sheet` Radix com `<SidebarContent />`. Um `useEffect` em `location.pathname` fecha o drawer ao navegar (UX mobile esperada).

**TopBar** — responsabilidade tripla:

1. Hamburger trigger (`<lg` only) abre o drawer.
2. Search button colapsa: `flex-1` em `<sm`, `w-[260px] sm:` em `>=sm`, `w-[340px] md:` em `>=md`.
3. Acoes secundarias (UiModeToggle, theme toggle desktop) escondem em `<md` e migram pro `UserMenu` dropdown (theme toggle exposto como item).

**Sidebar** — exporta dois componentes:
- `Sidebar` — `<aside hidden lg:flex>` (renderizado pelo AppShell desktop).
- `SidebarContent` — body sem wrapper `<aside>`, aceita `onNavigate?` para o drawer fechar ao clicar.

Links: `h-11` (44px) por default, `lg:h-9` (36px desktop). Touch comfort em mobile, density em desktop sem regredir.

### 3. Primitivos UI mobile-aware

**Button** — adicionado `size: 'touch' | 'icon-touch'` (44x44px) preservando os 6 sizes existentes. Pattern: usar `touch` em fluxos exclusivos mobile ou `lg:size-md` para tamanho adaptativo. Nao redimensionar `sm`/`md`/`lg` legados (rompe densidade desktop).

**Sheet (Radix Dialog)** — extendido com 2 variants:
- `side="bottom"` — bottom sheet mobile (padrao iOS/Android). Pinned bottom, max 90dvh, rounded top, `pb-safe` para notch inferior.
- `side="fullscreen"` — substituto mobile do Dialog. 100dvh edge-to-edge, `pt-safe pb-safe`. Para wizards e modais densos onde modal centrado nao serve.

`SheetContent` close button ganha `min-h-[44px] min-w-[44px]` em todas variants — touch target inviolavel.

**Dialog (Radix)** — nao criamos primitivo separado. Reuso de `Sheet` com `side="fullscreen"` em mobile + `side="right"` em desktop e suficiente. Princípio: **um primitivo, multiplas faces**, em vez de 3 componentes paralelos com mesmo papel logico.

### 4. Tipografia fluida

`globals.css` adiciona 4 utilities em `@layer utilities`:

| Classe | clamp() | Substitui |
|--------|---------|-----------|
| `.text-fluid-display` | `clamp(1.75rem, 4.5vw + 0.5rem, 3rem)` | `text-3xl` / `text-4xl` em headers de hero |
| `.text-fluid-h1` | `clamp(1.5rem, 3.5vw + 0.4rem, 2.25rem)` | `text-2xl` em PageHeader |
| `.text-fluid-h2` | `clamp(1.25rem, 2.2vw + 0.3rem, 1.75rem)` | `text-xl` em sections |
| `.text-fluid-h3` | `clamp(1.0625rem, 1.4vw + 0.25rem, 1.375rem)` | `text-lg` em card headings |

Letter-spacing e line-height alinhados ao Spec Sistema Base (display = -0.02em, line-height ≤1.1). Body type permanece em `text-sm`/`text-base`/`text-lg` legacy (pequena variacao de eixo nao justifica clamp e a escala fluida em copy quebra ritmo vertical).

**Aplicacao em S57+:** substituir headers de pages e secoes em `features/*` por `.text-fluid-h*`. Nao tocar copy, badges, micro-text.

### 5. Safe-area iOS

`globals.css` adiciona 6 utilities:

```css
.pt-safe { padding-top: env(safe-area-inset-top); }
.pb-safe { padding-bottom: env(safe-area-inset-bottom); }
.pl-safe / .pr-safe / .px-safe / .py-safe
.min-h-screen-safe { min-height: 100dvh; }
```

`100dvh` (dynamic viewport height) substitui `100vh` em containers full-height — `100vh` em iOS Safari inclui a barra de URL escondida e quebra layout quando ela aparece.

`index.html` ja tinha `viewport-fit=cover` (validado, nao alterado).

### 6. Out of scope (Onda 1)

S56 entrega o **chassi**. Componentes feature-level (Pipes Kanban, Inbox, Tabelas, Charts, Forms, Wizard Copilot) nao foram tocados — entregaveis das ondas 2-5:

- **S57** — Pipes Kanban scroll-snap horizontal mobile, Inbox 3-col → 1-col stack, LeadDetail mobile.
- **S58** — Tabelas → Cards mobile (`md:hidden` table + `hidden md:block` cards), Dashboard, Charts ResponsiveContainer / Visx ParentSize.
- **S59** — Forms 1-col mobile, Wizard Copilot step-por-tela mobile, Settings sections.
- **S60** — Playwright mobile viewports, axe a11y, touch target audit final, hover→focus migration.

## Consequencias

### Positivas

- Shell + primitivos prontos para qualquer viewport sem refactor por feature. Onda 1 destrava trabalho paralelo nas ondas seguintes.
- Touch targets compliant com Apple HIG / Material 3 — pre-empts findings de auditoria de a11y mobile.
- Safe-area utilities resolvem PWA iOS de uma vez. Nao mais workaround por componente.
- Tipografia fluida em utilities (`.text-fluid-h*`) e opt-in — nao quebra nenhuma copy existente.
- `Sheet` extendido (em vez de criar `Dialog` novo) mantem inventario de primitivos enxuto.

### Negativas

- 1 snapshot test (`sheet.test.tsx`) reescrito (close button ganhou min-h/min-w). Esperado, atualizado em commit S56 QA.
- Fluxo mobile real ainda requer S57-S60. Onda 1 e foundation, **nao** declaracao de "responsivo entregue". Plano Mestre §8 explicita as 4 ondas remanescentes para evitar declaracao prematura.
- `100dvh` nao tem polyfill — Chrome ≥108, Safari ≥16. Cobertura ~96% globalmente. Aceitavel: Torque e B2B com publico recente.

### Reversibilidade

- Toda mudanca e additive nas utilities (`.text-fluid-*`, `.pt-safe`, `Button size="touch"`, `Sheet side="bottom"|"fullscreen"`). Remover Onda 1 nao quebra nenhum codigo legacy.
- Sidebar/TopBar/AppShell tem mudancas estruturais minimas; reverter a behavior desktop-only e um revert de 4 arquivos.

## Referencias

- Spec Sistema Base — `Torque-dir-new/05 - Sistema Base/Spec - Redesign Sistema Base.md`
- ADR-005 (tipografia self-hosted) — fontes ja vem do `@fontsource`, nao precisa rede.
- Apple Human Interface Guidelines — Touch Targets (44pt).
- Material Design 3 — Touch target (48dp; Apple 44pt e o piso seguro multi-plataforma).
- Tailwind CSS v4 docs — Responsive design, breakpoints.
- CSS `env(safe-area-inset-*)` — CanIUse 96.4%.
- Plano Mestre `§8 Sprint S56` — bloco com out-of-scope explicito ondas 2-5.
- STATE.md `D076` — registro de decisao.
