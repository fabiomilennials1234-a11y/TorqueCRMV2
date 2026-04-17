---
tags: [design, tipografia, fonts, editorial]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Tipografia

Tipografia no Torque é hierárquica e editorial. Três famílias, três funções, sem mistura. Nunca uma família única geométrica para tudo — isso é o default do mercado e reprova automaticamente. Ver [[Principios de Identidade Visual]].

## As três famílias

### `font-display` — Fraunces

Serif editorial com optical sizing variável. Usada para:

- Headings de tela (`h1`, `h2`, `h3` editoriais).
- Números KPI grandes (cards de métrica, dashboard).
- Elementos de statement: título de login, nome de usuário no perfil, títulos de celebração.

Configuração obrigatória:

```css
.font-display {
  font-family: "Fraunces", ui-serif, Georgia, serif;
  font-optical-sizing: auto;
  font-variation-settings: "opsz" 144, "SOFT" 30;
  letter-spacing: -0.02em;
}
```

O `opsz 144` é o que dá o caráter editorial — contraste maior nos traços, serifas mais presentes. `SOFT 30` arredonda o esqueleto. Sem isso, Fraunces parece um Times qualquer.

### `font-sans` — Instrument Sans

Humanista com personalidade discreta (não é Inter). Usada para:

- Body text (parágrafos, descrições).
- Labels, hints, placeholders.
- Navegação, menus, breadcrumbs.
- Botões, badges, pills.
- UI geral.

```css
.font-sans {
  font-family: "Instrument Sans", ui-sans-serif, system-ui, sans-serif;
  letter-spacing: -0.005em;
}
```

Porque Instrument Sans e não Inter: Inter é o default do mercado (Linear, Vercel, milhares). Instrument Sans tem um ligeiro caráter humanista nas terminações que diferencia o Torque imediatamente sem ser exótico. Decisão de diferenciação.

### `font-mono` / `font-metric` — JetBrains Mono

Monoespaçada técnica. Usada para:

- Números em tabelas (colunas de métrica).
- Timestamps, datas formatadas.
- IDs, códigos, chaves.
- Valores monetários em listagens densas.
- Código e logs.

**Regra crítica:** qualquer número que o usuário vai comparar visualmente entre linhas (valores em coluna, KPIs empilhados, counters) usa `font-metric` com `tabular-nums slashed-zero`:

```css
.font-metric {
  font-family: "JetBrains Mono", ui-monospace, SFMono-Regular, monospace;
  font-variant-numeric: tabular-nums slashed-zero;
  font-feature-settings: "zero" 1, "ss01" 1;
}
```

Sem `tabular-nums`, números em coluna dançam de largura e ilegíveis em scan rápido. Falha em [[Criterios de Reprovacao]].

## Self-host é decisão

**Decisão arquitetural:** removemos Google Fonts CDN. Fontes são self-hosted via `@fontsource` (pacotes npm) ou download manual com subset `latin-ext`.

Razões:

1. **CSP estrita.** Policy não aceita `fonts.googleapis.com` e `fonts.gstatic.com` sem relaxar. Self-host permite `font-src 'self'`.
2. **Performance.** Zero round-trip DNS, fonte servida do mesmo origin com cache imortal.
3. **Soberania.** Não dependemos de CDN externa para um ativo de identidade.
4. **Privacidade.** Google Fonts loga IP de todo request.

Implementação:

```ts
// torque-web/src/app/layout.tsx
import "@fontsource-variable/fraunces/index.css";
import "@fontsource-variable/instrument-sans/index.css";
import "@fontsource-variable/jetbrains-mono/index.css";
```

Ou download manual dos `.woff2` em `torque-web/public/fonts/` quando precisar de subset custom.

## Escala tipográfica

Definida em tokens Tailwind, mapeada para classes utilitárias.

### Display (Fraunces)

- `text-display-2xl` → `clamp(3rem, 6vw, 4.5rem)`, line-height 1.0, letter-spacing -0.03em. Login, TV, celebrações.
- `text-display-xl` → `2.75rem` / 1.05 / -0.025em. Página hero, dashboard title.
- `text-display-lg` → `2rem` / 1.1 / -0.02em. Seções.

### Body (Instrument Sans)

- `text-base` → `0.9375rem` (15px) / 1.5. Corpo padrão.
- `text-sm` → `0.8125rem` (13px) / 1.45. Texto secundário em UI densa.

### Label (Instrument Sans)

- `text-label-2xs` → `0.6875rem` (11px) / 1.2, uppercase, `tracking-wider`. Labels de campo, seção headers em cards.

### Metric (JetBrains Mono)

- `text-metric-xl` → `2rem`, `tabular-nums`. KPIs grandes.
- `text-metric` → `0.9375rem`, `tabular-nums`. Valores em tabela.
- `text-metric-sm` → `0.8125rem`. Badges numéricos.

### Code (JetBrains Mono)

- `text-code` → `0.75rem` (12px). IDs, chaves, logs.

## Hierarquia obrigatória por tela

Toda tela que mostra dados precisa ter três níveis tipográficos visíveis:

1. **Display** para o título da tela ou do bloco principal.
2. **Body/Sans** para descrição, labels, navegação.
3. **Metric/Mono** para os números — sempre que houver números.

Tela que tem só uma família ou só um tamanho cai em [[Criterios de Reprovacao]] item 6.

## Anti-padrões

- `font-bold` em Fraunces display grande é redundante — o peso já vem do tamanho e do opsz. Reprovado.
- Mistura Instrument Sans + Fraunces no mesmo inline (ex: título com pedaço serif e pedaço sans) — reprovado salvo em cases editoriais explícitos.
- Número com `font-sans` em vez de `font-metric` — reprovado item 8.
- Letter-spacing positivo em display — reprovado, editorial é tight.
- `font-family: Inter` em qualquer lugar — reprovado, não é nosso sistema.

## Kerning e features tipográficas

Ligaduras padrão ligadas (`"liga" 1, "calt" 1`). Stylistic sets de Fraunces (`ss01` nas alternativas que suavizam o `a` e `g`) ligados globalmente para manter o caráter editorial consistente.

## Referência cruzada

- [[Design System Base]] para tokens de tinta (`--ink`, `--ink-muted`, `--ink-dim`).
- [[Componentes Primitivos]] para quais primitivos usam qual família.
- [[Criterios de Reprovacao]] para os red flags tipográficos.
