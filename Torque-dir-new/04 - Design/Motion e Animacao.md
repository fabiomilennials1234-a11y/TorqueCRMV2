---
tags: [design, motion, animation, eases]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Motion e Animação

Animação no Torque serve legibilidade, não decoração. Cada transição tem um motivo — orientar o olho, comunicar hierarquia temporal, confirmar ação. Frivolidade reprova. Ver [[Principios de Identidade Visual]] e [[Criterios de Reprovacao]].

## Eases canônicas

Duas curvas custom cobrem 95% dos casos. `ease` padrão CSS é proibido.

```css
--ease-out-soft:      cubic-bezier(0.2, 0.8, 0.2, 1);
--ease-in-out-precise: cubic-bezier(0.65, 0, 0.35, 1);
```

### `--ease-out-soft`

Desaceleração forte no final. Usar para:

- **Reveals** — entrada de card na lista, aparecimento de toast, fade-in de skeleton.
- **Expansões** — accordion abrindo, dropdown descendo, sheet deslizando.
- **Entradas em foco** — quando algo chega para ser visto, não para sair.

Sensação: o elemento "pousa". Desacelera como se tivesse inércia.

### `--ease-in-out-precise`

Simétrica, acelera e desacelera de forma controlada. Usar para:

- **Transições de estado** — toggle de tab, swap de conteúdo.
- **Modais** — abrir e fechar (a mesma curva nos dois sentidos).
- **Slides direcionais** — carrossel, step progress avançando.
- **Movimentos onde ambos os endpoints importam**.

Sensação: movimento deliberado, cinematográfico. Lembra um corte de câmera bem editado.

## Duração — regra dos 280ms

Hierarquia canônica:

- **Micro-interação (hover, press, focus)** → 120–180ms. Sweet spot 150ms.
- **Transições de estado (tab, toggle)** → 180–220ms.
- **Reveals (card, dropdown, toast)** → 220–300ms. Sweet spot 280ms.
- **Modais e sheets** → 240–300ms.
- **Page transitions** → 300ms teto.

**Nunca acima de 350ms.** Uma duração de 400ms não é "suave", é lenta. Parece janky no primeiro uso e intolerável na centésima. Reprovado em [[Criterios de Reprovacao]] item 7.

**Nunca abaixo de 100ms** em transição visível — o olho não registra, parece salto. Instantaneidade é para mudanças que o usuário não precisa acompanhar (ex: troca de estado interno que não muda visual).

## Animações custom catalogadas

Definidas em `globals.css` e Tailwind config. Reutilizar, nunca recriar.

### `fade-in`

Reveal padrão. Opacidade 0 → 1 com leve translate-y de 4px. 220ms, `ease-out-soft`. Usada em cards entrando na lista, conteúdo carregado após skeleton.

### `scale-in`

Entrada de elemento com ênfase. Scale 0.96 → 1.0 + opacity 0 → 1. 200ms, `ease-out-soft`. Usada em dropdown, popover, tooltip, toast.

### `shimmer`

Loop infinito para estados de carregamento em skeletons. Gradient horizontal atravessando o elemento, 1.2s de ciclo, linear (exceção justificada — loops precisam ser lineares para não parecer pulsação desigual). Ver `Skeleton` em [[Componentes Primitivos]].

### `torque-tick`

Animação signature do Torque. Um "tick" sutil que marca um momento: login completo, transação confirmada, lead fechado. Escala de 1.0 → 1.04 → 1.0 em 280ms com `ease-in-out-precise`, acompanhado de fade do glow accent. Não é celebração ruidosa — é um aceno.

### `caret-blink`

Piscar do cursor em inputs custom (command palette, wizard de copilot). 1.06s de ciclo, `ease-in-out-precise`. Levemente mais lento que o blink nativo do OS para parecer intencional.

## Library — Motion (framer-motion) com restraint

Motion (ex-framer-motion) é usado **apenas** onde Tailwind transition não resolve:

- **Stagger** — lista de cards aparecendo em cascata.
- **Layout animations** — reordenação de items (drag and drop em funil).
- **Exit animations** — elemento precisa animar ao sair (modal fechando antes de desmontar). `AnimatePresence`.
- **Gestos** — drag, spring no heat slider.
- **Orquestração complexa** — celebration effect com múltiplas camadas sequenciadas.

Para 90% dos casos (hover, focus, toggle, reveal simples), Tailwind `transition-all duration-200 ease-[var(--ease-out-soft)]` é suficiente. Importar Motion sem motivo é bloat.

## Princípios operacionais

### Animação comunica causalidade

Quando algo muda, a animação mostra como — de onde vem, para onde vai. Um card que desaparece sem transição some. Um card que fade-out + scale-down 0.98 em 180ms saiu. O usuário não pensa nisso, mas sente.

### Repouso é tão importante quanto movimento

Tela parada tem que estar viva mas não agitada. Não animar KPIs sem motivo, não fazer loops decorativos em ícones, não ter elementos piscando fora do caret.

### Reduced motion

Respeitar `prefers-reduced-motion: reduce`. Substituir transições por fades curtos (100ms) ou suprimir onde o movimento é ornamental (torque-tick, shimmer). Nunca cortar feedback funcional (expand/collapse).

```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
```

Mas manter os reveals essenciais com 100ms de fade, não zero absoluto.

### Estado de foco é animado

Focus ring (accent ouro em outline 2px) aparece com fade-in de 100ms. Entra e sai suave — não pisca. Acessibilidade não precisa ser visualmente agressiva.

## Anti-padrões

- `transition: all 0.4s ease` — reprovado em três pontos (duração alta, ease linear, `all` sem precisão).
- Bounce em CTA — parece brinquedo. Reprovado salvo em celebration effect explícito.
- Hover scale > 1.03 — exagero, parece inchaço.
- Loop infinito em elemento estático sem motivo de carregamento — reprovado, polui a tela.
- Fade-in de 500ms em dashboard load — o usuário já esperou a rede, não faz ele esperar a animação.

## Referência cruzada

- [[Componentes Primitivos]] — quais primitivos têm animação interna.
- [[Criterios de Reprovacao]] — item 7 sobre ease linear e duração.
- [[Design System Base]] — shadow-glow-accent usado em torque-tick.
