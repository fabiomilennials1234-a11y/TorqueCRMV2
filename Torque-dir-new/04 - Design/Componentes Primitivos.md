---
tags: [design, componentes, ui, primitivos, sistema-base]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Componentes Primitivos

Inventário canônico dos primitivos de UI do Torque. Moram em `torque-web/src/ui/` e são **genuinamente custom** — não wrappers finos de shadcn. Radix é usado apenas como camada de acessibilidade/comportamento (focus trap, keyboard nav, ARIA), o visual é 100% Torque.

Categorias neste documento:

1. **17 primitivos existentes na base.**
2. **3 primitivos a promover** para a base (genéricos, reusáveis em futuras features).
3. **12 componentes de domínio** que ficam fora da base, junto das features.

Ver [[Design System Base]], [[Tipografia]], [[Motion e Animacao]].

---

## 1. Primitivos existentes (17)

### Button

Propósito: ação primária, secundária, destructive, ghost, link.
Variantes: `primary` (accent ouro, glow em hover), `secondary` (surface + hairline), `ghost` (transparente, hover revela surface), `destructive` (danger token), `link` (inline, sublinhado em hover).
Tamanhos: `sm` (h-8), `md` (h-9, padrão), `lg` (h-10).
Onde usar: qualquer ação disparada pelo usuário. `primary` máximo 1 por tela/bloco.

### Badge

Propósito: rótulo estático pequeno indicando status, categoria, contagem.
Variantes: `default`, `accent`, `success`, `warning`, `danger`, `info`, `outline`.
Onde usar: chips em cards, estado de registro em listagem, contadores inline.

### Card

Propósito: contêiner base de conteúdo agrupado.
Variantes: `default` (surface + hairline), `elevated` (elevated + elev-2), `interactive` (hover revela hairline accent).
Subcomponentes: `CardHeader`, `CardTitle` (font-display), `CardDescription`, `CardContent`, `CardFooter`.
Onde usar: tudo que é bloco de conteúdo — KPI, item de lista rica, preview.

### Avatar

Propósito: imagem circular de usuário/entidade, com fallback para inicial.
Variantes: `xs` (h-5), `sm` (h-7), `md` (h-9), `lg` (h-12). Fallback com background em `accent-soft` e inicial em `font-display`.
Onde usar: mensagens, headers de conversa, member lists.

### Input

Propósito: campo de texto single-line.
Variantes: `default`, `with-icon`, `password` (toggle de visibilidade).
Estados: focus ring accent, error (hairline danger + mensagem), disabled (ink-dim).
Onde usar: forms, filtros, search.

### Pill

Propósito: chip interativo, mais denso que badge, usado em filtros e seletores inline.
Variantes: `default`, `active` (accent-soft bg + accent ink), `removable` (com X).
Onde usar: tags em lead, filtros ativos, categorias selecionadas.

### Kbd

Propósito: representação visual de tecla de atalho (ex: `⌘K`, `Esc`).
Variantes: única, com tamanho controlado pelo contexto.
Onde usar: tooltips de ações com shortcut, command palette, hints em footer de modal.

### PageHeader

Propósito: header padrão de tela com título `font-display`, breadcrumb opcional, ações à direita.
Variantes: `default`, `compact` (sem description), `hero` (display-xl + description).
Onde usar: topo de toda página de módulo. Obrigatório em rotas principais.

### ScoreMeter

Propósito: gauge radial mostrando um score 0–100 (ou % de meta atingida).
Variantes: `sm` (48px), `md` (64px), `lg` (96px). Ring animado com `stroke-dasharray`, cor em `--accent` ou gradiente stages.
Onde usar: perfil de lead, card de agente, dashboard de performance individual.

### Sparkline

Propósito: micro-gráfico inline mostrando tendência temporal em 1 linha.
Variantes: `default` (accent), `success`, `danger`. Com ou sem ponto final destacado.
Onde usar: cards de KPI ao lado do número, linhas de tabela de performance.

### Dropdown (Radix)

Propósito: menu de ações contextual.
Variantes: `align: start | end`, `side: top | bottom`. Suporta grupos, separadores, items com ícone e Kbd.
Onde usar: menu de ações em row de tabela, botão de perfil, seletores compactos.

### Sheet (Radix)

Propósito: drawer deslizante para edição lateral sem sair da tela base.
Variantes: `right` (padrão, edit forms), `bottom` (mobile, ações), `left` (nav overlay em mobile).
Onde usar: edit de lead, detalhes de conversa expandida, filtros avançados.

### Tabs (Radix)

Propósito: navegação entre seções paralelas de conteúdo.
Variantes: `underline` (padrão, editorial), `pills` (accent-soft no active).
Onde usar: dentro de tela única com múltiplas visões — perfil de lead com tabs Info/Histórico/Arquivos.

### Tooltip (Radix)

Propósito: hint textual curto ao hover/focus.
Variantes: tamanho único, `side` configurável. Aparece com 200ms de delay, desaparece em 100ms.
Onde usar: ícones-only, abreviações, botões com label implícito.

### Separator

Propósito: divisor horizontal ou vertical em hairline.
Variantes: `horizontal` (padrão), `vertical`. Usa `shadow-hairline` lógica.
Onde usar: entre seções em dropdown, entre grupos em sidebar, separação de conteúdo em card.

### Skeleton

Propósito: placeholder animado durante loading.
Variantes: `text` (linha com altura de texto), `card`, `avatar`, `custom` (com className).
Animação: `shimmer` em loop. Cor base `surface`, shimmer em `elevated`.
Onde usar: substituindo spinner em **qualquer** estado de loading de conteúdo. Ver [[Criterios de Reprovacao]] item 9.

### EmptyState

Propósito: tela/bloco quando não há dados para mostrar.
Variantes: `default` (ilustração + título display + descrição + CTA), `compact` (inline em card).
Tom: editorial, direto, sem exclamações. Ver [[Vocabulario de UI]].
Onde usar: listagem vazia, filtro sem resultados, primeiro acesso de módulo.

---

## 2. Primitivos a promover para a base (3)

Componentes nascidos em features específicas que se mostraram genéricos o suficiente para serem promovidos a primitivos. Devem migrar para `torque-web/src/ui/`.

### StepProgress

Propósito: stepper horizontal editorial, mostra etapas de um wizard com estado (completa, atual, futura).
Variantes: `numbered` (com número no círculo), `iconed` (ícone custom por step), `minimal` (apenas barra com marcadores).
Nó atual usa accent; completos usam `success`; futuros usam `ink-dim`. Linha de conexão é hairline.
Onde usar: CopilotWizard, OnboardingWizard, CheckoutWizard, qualquer fluxo multi-step futuro.

### QuotaGauge

Propósito: visualização genérica de `current / effective_limit` com flag `can_add` controlando estado visual e estado de CTA.
Variantes: `bar` (horizontal), `ring` (compacto, reusa ScoreMeter visual), `inline` (texto `12/20` com bullet de cor).
Transições de cor: verde até 70%, warning 70–90%, danger acima. `can_add=false` ativa estado travado.
Onde usar: limite de leads, limite de funis, limite de membros, qualquer feature com plano/quota.

### ChannelBadge

Propósito: badge padronizado de canal de origem (WhatsApp, Messenger, Instagram, SZ.Chat), com ícone e cor canônica.
Variantes: `icon-only` (círculo com ícone), `labeled` (ícone + nome), `dot` (apenas pontinho indicador).
Cor vem de `--channel-*` — ver [[Design System Base]].
Onde usar: lista de conversas, header de chat, filtros, funil com origem visível.

---

## 3. Componentes de domínio (12, fora da base)

Estes são específicos de feature e moram em `torque-web/src/modules/<feature>/`. Não são primitivos e não devem ser importados fora do escopo.

- **CountdownBadge** — badge com tempo regressivo e mudança de cor via `--countdown-*`. Usado em SLA de conversa e prazo de proposta.
- **HeatSlider** — slider 1–5 de calor de lead usando `--heat-*`. Pipe Propostas.
- **HeatIndicator** — read-only do heat atual, para exibição em listagens.
- **FunnelChart** — visualização de funil por estágio com conversão entre etapas. Dashboard.
- **CompetitionPodium** — pódio 1º/2º/3º em gamificação comercial. Premia performance.
- **AchievementBadge** — selo de conquista desbloqueada pelo agente. Gamificação.
- **CelebrationEffect** — overlay full-screen com confete + torque-tick em fechamento de venda.
- **TVLayout** — layout fullscreen para dashboard de TV, vignette radial, tipografia display-2xl.
- **WorkflowCanvas** — canvas de automação com bg-dot-grid, drag-and-drop de nodes. Automations.
- **CopilotWizard** — wizard específico da configuração do Copilot (agente IA).
- **OnboardingWizard** — wizard de primeiro acesso do master.
- **CheckoutWizard** — fluxo de assinatura/upgrade com PixQRCode integrado.
- **PixQRCode** — componente especializado de QR de pagamento com animação de expiração.
- **OperationsCenter** — painel master com jobs, logs, fila. Canvas com bg-grid.

---

## Regra de ouro

Se um componente é usado por **2+ features não relacionadas**, é candidato a primitivo da base. Se é específico de uma feature, mora lá. Promoção é explícita e sempre passa por refatoração das variants para remover acoplamento.

## Referência cruzada

- [[Design System Base]] — tokens que os primitivos consomem.
- [[Tipografia]] — famílias usadas em cada primitivo.
- [[Motion e Animacao]] — animações internas (fade-in, scale-in, shimmer).
- [[Vocabulario de UI]] — textos dos EmptyStates e estados.
- [[Criterios de Reprovacao]] — red flags de uso errado.
