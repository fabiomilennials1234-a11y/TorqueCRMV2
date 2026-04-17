---
tags: [design, tokens, design-system, css, hsl]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Design System Base

Este é o contrato canônico de tokens do Torque CRM. Tudo vive como CSS custom properties em HSL puro (sem `hsl()` wrapper) para permitir composição com alpha do Tailwind via `hsl(var(--token) / 0.5)`. RGB nunca. Hex nunca nos tokens (só em ativos pontuais).

Arquivo fonte da verdade: `torque-web/src/styles/globals.css`. Qualquer novo token nasce ali antes de ser usado em componente.

Ver também [[Principios de Identidade Visual]] e [[Tipografia]].

## Canvas — luminosidade base

A hierarquia visual em dark é feita por luminosidade, não por cor. Quatro níveis:

```css
--bg:        222 14% 4%;   /* canvas raiz, body background */
--surface:   222 13% 6%;   /* cards, panels, sidebar */
--elevated:  222 13% 9%;   /* modais, popovers, dropdowns */
--hairline:  222 10% 16%;  /* bordas via shadow-hairline */
```

Regra: componente sobre `--bg` vai em `--surface`; componente sobre `--surface` vai em `--elevated`. Nunca pular níveis.

## Type — escala de tinta

Tinta (texto) também é hierarquizada em três pesos:

```css
--ink:       220 13% 91%;  /* texto principal, headings, KPIs */
--ink-muted: 220  9% 64%;  /* texto secundário, labels, descrições */
--ink-dim:   220  7% 40%;  /* texto terciário, metadata, timestamps */
```

Nunca usar `text-white` direto. Sempre via token. Se precisar de um quarto nível, provavelmente a hierarquia está errada.

## Accent — ouro quente

```css
--accent:      44 93% 54%;  /* ouro quente, CTA, highlights, active */
--accent-soft: 44 50% 18%;  /* fundo tonalizado para selos, active bg */
```

Regra de uso em [[Principios de Identidade Visual]] — accent restraint.

## Semânticos

Cores de estado do sistema. Nunca misturar com stages ou canal.

```css
--success: 160 55% 48%;  /* confirmação, concluído, online */
--warning:  34 90% 56%;  /* atenção, pendência, prazo próximo */
--danger:  358 75% 59%;  /* erro, falha, urgência, destructive */
--info:    210 70% 58%;  /* informativo, em andamento neutro */
```

## Stages — sequência curada de funil

Sete tokens para representar sequência de estágios no pipeline. Decisão explícita: nunca rainbow. A paleta progride de tons frios para quentes, passando pelo accent no final, comunicando "avanço".

```css
--stage-1: 220  30% 55%;  /* início, lead novo */
--stage-2: 200  45% 55%;  /* qualificação */
--stage-3: 175  45% 52%;  /* nutrição */
--stage-4: 155  50% 52%;  /* proposta em curso */
--stage-5:  90  40% 55%;  /* negociação */
--stage-6:  44  93% 54%;  /* fechamento (accent) */
--stage-7:  20  70% 55%;  /* pós-venda / ativo */
```

Nunca atribuir cor de stage ad-hoc. Se precisa de um oitavo estágio, a curva é revisada inteira — não se adiciona um verde aleatório.

## Calor — Pipe Propostas

Tokens novos introduzidos para o componente `HeatSlider` (1–5) no Pipe Propostas. Progressão frio → quente que comunica temperatura de lead, não estágio.

```css
--heat-1: 210 60% 52%;  /* frio, reativo */
--heat-2: 170 55% 48%;  /* morno */
--heat-3:  44 93% 54%;  /* quente (accent) */
--heat-4:  34 90% 56%;  /* muito quente */
--heat-5: 358 75% 59%;  /* escaldante, fechamento iminente */
```

## Canal — origem de conversa

Cores institucionais dos canais, com ajuste de saturação para combinar com o canvas dark. Não são as oficiais exatas — são derivadas que convivem com o resto do sistema.

```css
--channel-whatsapp:  142 70% 45%;
--channel-messenger: 221 89% 60%;
--channel-instagram: 329 80% 55%;
--channel-sz:        260 65% 58%;  /* SZ.Chat, canal proprietário */
```

Uso via `ChannelBadge` — ver [[Componentes Primitivos]].

## Countdown — urgência temporal

Usado em `CountdownBadge` para prazos de proposta, SLA de resposta.

```css
--countdown-safe:   160 55% 48%;  /* >24h */
--countdown-warn:    34 90% 56%;  /* <24h */
--countdown-urgent: 358 75% 59%;  /* <1h ou vencido */
```

A transição entre bandas é visual discreta (fade no badge), não brusca.

## Job status — processos assíncronos

Usado em painéis de jobs, imports, integrações.

```css
--job-pending:   44 50% 55%;
--job-running:  210 70% 58%;
--job-completed:160 55% 48%;
--job-failed:   358 75% 59%;
```

## Shadow e elevação

Sombra em dark é delicada. Usamos três níveis de elevação + glow do accent + hairline inset.

```css
--shadow-elev-1: 0 1px 2px 0 hsl(0 0% 0% / 0.4);
--shadow-elev-2: 0 4px 12px -2px hsl(0 0% 0% / 0.5),
                 0 2px 4px -1px hsl(0 0% 0% / 0.3);
--shadow-elev-3: 0 16px 32px -8px hsl(0 0% 0% / 0.6),
                 0 8px 16px -4px hsl(0 0% 0% / 0.4);

--shadow-glow-accent: 0 0 24px 0 hsl(var(--accent) / 0.25),
                      0 0 8px 0 hsl(var(--accent) / 0.35);

--shadow-hairline: inset 0 0 0 1px hsl(var(--hairline) / 0.6);
```

Regra: card de lista → elev-1; dropdown → elev-2; modal → elev-3. Glow só em CTA primária em hover e em estados signature (pix QR ativo, podium top 1).

## Espaçamento

Sistema em múltiplos de 4px, mas a prática mais usada é `0.5` (2px), `1` (4px), `1.5` (6px), `2` (8px), `2.5` (10px), `3` (12px), `4` (16px), `6` (24px), `8` (32px), `12` (48px).

Tudo que for maior que `8` (32px) precisa de justificativa de design.

## Radius

```css
--radius-sm:   4px;   /* pills, badges pequenos */
--radius:      8px;   /* botões, inputs, cards padrão */
--radius-lg:   12px;  /* cards com conteúdo rico */
--radius-xl:   16px;  /* panels, modais */
--radius-full: 9999px;
```

## Container

Largura máxima canônica do layout: **1400px**. Nada acima. TV dashboards e Operations Center usam `100%` intencionalmente — são exceção documentada.

## Regra de HSL puro

Todos os tokens de cor são string `H S% L%` sem wrapper. Isso permite:

```html
<div class="bg-[hsl(var(--accent)/0.1)] text-[hsl(var(--ink))]">
```

Se alguém escrever `--accent: hsl(44 93% 54%)` com wrapper, quebra o alpha composition do Tailwind. Reprovado em code review.

## Mapping para Tailwind

O `tailwind.config.ts` mapeia todos esses tokens para as keys do tema (`colors.accent`, `colors.surface`, etc.) usando a função helper `hsl(var(--token))`. Componentes usam as classes semânticas (`bg-surface`, `text-ink-muted`), nunca as variáveis diretas exceto em estilo arbitrário com alpha.

## Arquivo canônico

```
torque-web/src/styles/globals.css
```

Qualquer divergência entre este documento e o arquivo — o arquivo vence. Este documento é atualizado em seguida.
