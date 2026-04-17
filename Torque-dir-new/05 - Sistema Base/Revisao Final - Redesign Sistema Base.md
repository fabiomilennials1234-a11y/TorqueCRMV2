---
tags: [sistema-base, revisao, quality-gate, bloqueadores]
created: 2026-04-15
last_updated: 2026-04-15
status: resolved
type: review-report
---

# Revisao Final — Redesign Sistema Base

Auditoria independente cruzando 4 superficies: documentacao do Obsidian, Analise Pratica do codebase, Direcao Visual Frontend e Escopo do Sistema Base.

**Veredicto: REPROVADO COM RESSALVAS CRITICAS**

A proposta visual esta aprovada. O codebase tem debitos nao detectados pela analise pratica inicial que bloqueiam a execucao.

---

## Bloqueadores criticos — TODOS RESOLVIDOS (2026-04-15)

Etapa 0 executada com sucesso. Build limpo apos todos os fixes.



### B1 — `* { @apply border-hairline }` global em globals.css

`globals.css` linha 49 aplica `border-hairline` em absolutamente todos os elementos do DOM. Todo `<div>`, `<span>`, `<p>`, `<svg>` recebe o inset shadow. Causa acumulacao de shadows e conflitos visuais. Nenhum doc de design valida uso global.

**Fix:** Remover a regra global. Aplicar shadow-hairline individualmente nos componentes que precisam.

### B2 — `torque-tick` implementado como rotacao, docs definem como escala

`Motion e Animacao.md` define: "escala 1.0 -> 1.04 -> 1.0, 280ms, ease-in-out-precise, fade do glow accent."
`tailwind.config.ts` implementa: rotacao 0deg -> 180deg -> 0deg, 1.8s, ease diferente.

Sao dois erros: natureza (rotacao vs escala) e duracao (1.8s vs 280ms). A animacao signature do produto esta errada.

**Fix:** Reescrever keyframe torque-tick como scale(1) -> scale(1.04) -> scale(1), 280ms, ease-in-out-precise.

### B3 — shimmer e caret-blink com duracoes erradas

Docs: shimmer 1.2s, caret-blink 1.06s.
Implementado: shimmer 2.4s (2x mais lento), caret-blink 1.25s.

Shimmer a 2.4s parece loading pesado, contradiz a sensacao de ferramenta rapida.

**Fix:** Corrigir duracoes no tailwind.config.ts.

### B4 — `border border-hairline` disseminado em 14+ arquivos (nao 2)

A Analise Pratica identificou 2 locais. Auditoria encontrou 20+ ocorrencias em:
- Primitivos: `button.tsx` (variante outline), `tabs.tsx`, `sheet.tsx`
- Shell: `CommandPalette.tsx`, `Sidebar.tsx`
- Pages: `KanbanPage.tsx`, `InboxPage.tsx`, `AgentsPage.tsx`, `DashboardPage.tsx`, `CampaignsPage.tsx`, `LoginPage.tsx`, `KanbanColumn.tsx`

[[Criterios de Reprovacao]] item 10 e explicito: qualquer `border` Tailwind reprova.

**Fix:** Substituir todos os `border border-hairline` por `shadow-hairline` equivalente. ~20 edits em ~14 arquivos.

### B5 — Google Fonts CDN ativo (bloqueador CSP)

`index.html` tem 3 tags de Google Fonts. @fontsource nao instalado. Bloqueia CSP `font-src 'self'`.

**Fix:** Instalar @fontsource, importar no entry point, remover `<link>` tags.

### B6 — ESLint sem regras reais (bloqueador de seguranca)

`.eslintrc.cjs` contem apenas parser e env, sem nenhuma regra. Sem `no-restricted-imports` nem ban de `dangerouslySetInnerHTML`.

**Fix:** Configurar regras reais.

---

## Desvios entre proposta visual e identidade

### D6 — Stage tokens divergem entre CSS e docs

`Design System Base.md`: accent no stage-6 (fechamento), progressao fria->quente.
`globals.css`: accent no stage-5 (proposta), sequence completamente diferente.
Direcao Visual nao resolve qual e correto.

**Decisao necessaria:** o CSS atual ou os docs sao a fonte de verdade? Precisa ser fixado antes de F01.

### D7 — dropdown.tsx inverte hierarquia de luminosidade

Fundo do dropdown e `elevated` (9%). Estado `data-[highlighted]` usa `bg-surface` (6%) — mais escuro que o fundo. Inverte a hierarquia.

**Fix:** Trocar para `bg-elevated/80` ou adicionar opacidade sobre elevated.

---

## Vazamentos de escopo encontrados

### V1 — QuotaGauge: contradicao interna nos docs

Escopo lista `QuotaGauge` como primitivo da base. Escopo tambem lista "Quotas visuais" no que NAO entra.

**Resolucao:** QuotaGauge entra como gauge generico de progresso (current/limit), SEM dados de plano hardcoded. O componente e generico; o uso em quotas de plano e feature.

### V2 — StepProgress: unico uso documentado e em wizards (que estao fora do escopo)

**Resolucao:** StepProgress e primitivo generico reutilizavel. Implementacao na base e valida como investimento antecipado — sera usado em F06 (Copilot), F13 (Onboarding), F14 (Checkout). Nao e prematura; e preparacao fundamentada.

### V3 — Sidebar ja inclui items de produto

Escopo diz "grupos vazios, sem modulos de produto". Direcao Visual popula com items reais (Dashboard, Funis, Conversas, etc.).

**Resolucao:** items de navegacao representam a arquitetura de informacao (IA) do produto, nao features implementadas. E aceito que a sidebar mostre o mapa completo com items desabilitados/placeholder apontando para empty states. A IA e parte do shell, nao feature.

---

## Debitos subestimados na Analise Pratica

| Debito | Analise dizia | Realidade | Impacto |
|--------|--------------|-----------|---------|
| `border` Tailwind | 2 locais | 20+ em 14+ arquivos | Esforco de fix 7x maior |
| `* { border-hairline }` global | Nao detectado | Afeta render tree inteiro | Fix de maior impacto |
| button.tsx outline | "Nenhum primitivo usa border" | Usa `border border-hairline` | Contamina todas instancias de Button outline |
| torque-tick | Mencionado como divergencia | Natureza completamente errada (rotacao vs escala) | Animacao signature quebrada |
| shimmer duracao | Nao mencionado | 2x mais lento que docs (2.4s vs 1.2s) | Loading parece pesado |
| ThemeContext vs Criterios | Nao mencionado | Tensao entre "preparado para multi-tema" e "zero toggle" | Ambiguidade para implementador |

---

## Seguranca

| Item | Status |
|------|--------|
| Google Fonts CDN removido | FAIL — 3 tags ativas |
| @fontsource instalado | FAIL — nao instalado |
| CSP font-src 'self' viavel | FAIL — depende de remocao CDN |
| Headers de seguranca | FAIL — nenhum configurado |
| ESLint com regras de seguranca | FAIL — config vazio |
| VITE_* com secrets | PASS — nenhum encontrado |
| dangerouslySetInnerHTML | PASS — nenhum uso encontrado (mas sem lint guard) |

---

## Recomendacoes ordenadas por prioridade

### Pre-requisito (antes de qualquer redesign)
1. **R1** — Remover `* { @apply border-hairline }` do globals.css
2. **R2** — Corrigir torque-tick: scale 1.0->1.04->1.0, 280ms, ease-in-out-precise
3. **R3** — Corrigir shimmer para 1.2s e caret-blink para 1.06s
4. **R4** — Instalar @fontsource, remover Google Fonts CDN
5. **R5** — Auditar e corrigir TODOS os `border border-hairline` (~20 edits, 14 arquivos)

### Paralelo com redesign
6. **R6** — Corrigir dropdown.tsx: highlighted = bg-elevated/80
7. **R7** — Documentar que ThemeContext dark-only e permitido; toggle nunca
8. **R8** — Fixar stage tokens: CSS atual ou docs? Decisao antes de F01
9. **R9** — ESLint com regras reais (no-restricted-imports, dangerouslySetInnerHTML ban)
10. **R10** — Clarificar QuotaGauge: gauge generico, nao visual de plano

---

## Ordem de execucao atualizada

O [[Plano de Execucao Granular]] precisa de uma **Etapa 0 — Fix de bloqueadores** antes de tudo:

```
Etapa 0: Fix bloqueadores (B1-B6)      [ANTES de tudo]
  0.1 Remover * { border-hairline } global
  0.2 Corrigir torque-tick, shimmer, caret-blink
  0.3 Substituir TODOS border border-hairline por shadow-hairline (~14 arquivos)
  0.4 Instalar @fontsource + remover Google Fonts CDN
  0.5 ESLint com regras minimas
  0.6 Corrigir dropdown.tsx highlighted state

Etapa 1: Preparacao     [PARALELO com Etapa 2, apos Etapa 0]
Etapa 2: Fundacao visual [PARALELO com Etapa 1, apos Etapa 0]
...resto conforme plano original
```

---

## Referencias

- [[Spec - Redesign Sistema Base]]
- [[Analise Pratica]]
- [[Direcao Visual Frontend]]
- [[Principios de Identidade Visual]]
- [[Criterios de Reprovacao]]
- [[Motion e Animacao]]
- [[Design System Base]]
- [[Escopo do Sistema Base]]
- [[Plano de Execucao Granular]]
