---
tags: [design, identidade, principios, dark-first]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Principios de Identidade Visual

A identidade do Torque CRM não é uma camada estética aplicada sobre o produto. É uma restrição de design que atravessa todas as telas, todos os componentes, todas as decisões de espaçamento e tipografia. Este documento fixa os princípios inegociáveis.

## Dark-first é o design, não uma opção

O Torque é pensado, construído e enviado em dark. Não existe toggle de tema. Não existe light mode sendo desenvolvido em paralelo. Escolhemos dark porque:

- A operação comercial do cliente acontece em ambientes de baixa luz (escritório, TV em sala fechada, notebook à noite).
- Dark permite hierarquia por luminosidade (elevated surface, hairline sutil) sem ruído cromático.
- Permite o uso cinematográfico do accent ouro como ponto de fuga — impossível em light sem parecer amarelo genérico.

Light mode é falha de design. Se aparecer proposta de toggle, reprovou. Ver [[Criterios de Reprovacao]].

## Referências

Toda decisão visual é comparada contra cinco referências. Nenhuma é copiada. Todas são entendidas.

- **Apple** — hierarquia por luminosidade e peso, tipografia óptica, restraint obsessivo.
- **Airbnb** — warmth editorial, fotografia como vocabulário, tipografia Fraunces como statement.
- **Linear** — densidade informacional sem ruído, keyboard-first, feedback instantâneo.
- **Stripe** — clareza tabular de dados, tipografia monoespaçada para métricas, motion discreto.
- **Vercel** — minimalismo técnico, grid backgrounds, preto como canvas, accent como convite.

Se uma tela poderia pertencer ao dashboard padrão de qualquer SaaS B2B, reprovou.

## Tipografia editorial + sensibilidade cinematográfica

Não usamos uma única família geométrica para tudo. A hierarquia é obrigatória:

- **Fraunces** (serif editorial com optical sizing) para headings, números KPI grandes, elementos de statement.
- **Instrument Sans** (humanista) para body, labels, navegação.
- **JetBrains Mono** (`tabular-nums slashed-zero`) para qualquer número que precise alinhar ou ser lido como dado.

Detalhes em [[Tipografia]].

A sensibilidade cinematográfica aparece em três decisões: vignette radial em telas signature (login, TV dashboard, celebrations), grain sutil permanente no canvas, e motion com eases soft/precise que lembram transições de câmera, não salto de UI.

## Sofisticação, diferenciação, encantamento

Estes três vetores são o teste final de qualquer tela antes de shippar:

- **Sofisticação** — hierarquia clara, sem ruído, sem excesso decorativo. Parece um produto caro.
- **Diferenciação** — reconhecível como Torque em um screenshot sem logo. Não genérico.
- **Encantamento** — tem um detalhe que faz o usuário sorrir. ScoreMeter animado, torque-tick no login, vignette no dashboard TV, celebration effect no fechamento.

Se uma tela passa no primeiro e falha nos outros dois, é template. Reprovou.

## Hairline-first

Não usamos `border` do Tailwind padrão. Bordas são ruído cromático em dark. Usamos `shadow-hairline` — um inset shadow de 1px com `hsl(var(--hairline))` em alpha baixo. O resultado é uma linha que existe mas não grita. Ver tokens em [[Design System Base]].

```css
.shadow-hairline {
  box-shadow: inset 0 0 0 1px hsl(var(--hairline) / 0.6);
}
```

Regra: se você escreveu `border border-white/10`, substituiu pelo caminho errado. Reprovou.

## Compacto, nunca espaçoso demais

A densidade informacional é parte da identidade. O Torque é uma ferramenta de operação, não um site institucional. Padrões:

- Padding horizontal de card: `px-2.5` a `px-4`. Nunca `px-8`.
- Padding vertical de botão: `py-1.5` a `py-2`. Nunca `py-3` em CTA padrão.
- Gap entre elementos em stack: `gap-1.5` a `gap-3`. Nunca `gap-6` em listas.
- Altura de input: `h-9`. Nunca `h-12`.

Espaço existe para respirar onde importa (headers editoriais, empty states, celebrações), não em todo lugar.

## Accent restraint

O ouro `hsl(44 93% 54%)` é a cor mais cara do sistema. Usar errado é queimá-lo.

Usar em: CTAs primárias, active states de navegação, highlight de KPI crítico, progresso em gauge, selos de conquista, pix QR border.

Nunca usar em: fundos grandes, banners, gradientes decorativos, ícones genéricos, estados hover não-críticos.

Se o ouro está em dois blocos grandes da mesma tela, reprovou um deles.

## Grain e vignette

O canvas do Torque tem um grain sutil permanente (`opacity: 0.035`, `mix-blend-mode: overlay`). Ele existe para remover o aspecto plástico do flat dark e adicionar textura cinematográfica. Não é decorativo — é pele.

Vignette radial é aplicada apenas em telas signature: login, TV dashboard, checkout final, celebrações. É um foco dramático, não um filtro.

## Motion restraint

Animação serve legibilidade, não decoração. Ver [[Motion e Animacao]].

- Eases custom (`--ease-out-soft`, `--ease-in-out-precise`). Nunca `ease` linear do CSS padrão.
- Duração: 150ms para micro-interação, 280ms sweet spot, 300ms teto. Nunca acima de 350ms.
- Motion (framer-motion) só quando Tailwind transition não alcança — stagger, layout, exit animations.

## Grid backgrounds

Áreas de canvas e editor (WorkflowCanvas, OperationsCenter) usam `bg-grid` ou `bg-dot-grid` como fundo. Não é ornamento — é sistema de referência espacial para o usuário ancorar componentes arrastados. O grid tem a mesma cor do hairline, alpha baixíssimo.

## Resumo operacional

Antes de shippar qualquer tela, rode a checklist:

1. É dark? Único modo?
2. Tipografia tem display + body + metric onde há números?
3. Hairline via shadow, nunca border?
4. Accent ouro restrito a CTAs/highlights?
5. Espaçamento compacto (px-2.5, py-1.5, gap-1.5 como base)?
6. Motion com ease custom, sub-300ms?
7. Se tirar o logo, ainda parece Torque?

Falhou em qualquer item, volta. Ver [[Criterios de Reprovacao]] e [[Vocabulario de UI]].
