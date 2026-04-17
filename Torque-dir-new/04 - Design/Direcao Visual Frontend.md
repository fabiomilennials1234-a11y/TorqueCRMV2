---
tags: [design, frontend, direcao-visual, sistema-base, shell, login]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
depends_on: [Principios de Identidade Visual, Design System Base, Tipografia, Motion e Animacao, Componentes Primitivos, Criterios de Reprovacao, Vocabulario de UI]
---

# Direcao Visual Frontend -- Sistema Base

Documento de direcao pratica para implementacao visual do Sistema Base. Nao repete definicoes do vault -- aplica-as em decisoes concretas por zona, componente e estado. Este e o input para a spec de redesign.

Referencia cruzada obrigatoria: [[Principios de Identidade Visual]], [[Design System Base]], [[Tipografia]], [[Motion e Animacao]], [[Componentes Primitivos]], [[Criterios de Reprovacao]], [[Vocabulario de UI]].

---

## 1. Linguagem visual aplicada

### AppShell -- a sensacao

O shell e uma maquina de precisao: densa, silenciosa, com brilho controlado apenas onde importa. O usuario abre o Torque e sente um instrumento -- nao um painel administrativo.

**Sidebar** ocupa `surface` (6% luminosidade). Hairline `inset -1px 0 0 0` na borda direita -- nunca `border-r`. O fundo e solido, sem blur, sem transparencia. E a area mais estavel da tela: ancora visual. Largura fixa 232px expandida; 56px colapsada (apenas icones + tooltips).

**TopBar** ocupa `bg/80` com `backdrop-blur-xl`. Hairline `inset 0 -1px 0 0` na borda inferior. O blur cria separacao atmosferica entre chrome e conteudo sem adicionar peso visual. Altura fixa 56px (h-14). E a unica zona com transparencia no shell -- o conteudo abaixo e visivel por detras, criando profundidade.

**Main content** ocupa `bg` puro (4% luminosidade). E o canvas mais escuro da tela. Cards e blocos dentro dele sobem para `surface`; dropdowns e modais sobem para `elevated`. A hierarquia nunca pula nivel: bg -> surface -> elevated, nessa ordem.

**Grain** aparece no `body` inteiro como pseudo-elemento `::before` no root. Opacity 0.035, mix-blend overlay. E permanente e global -- nao e ativado por componente. O grain nao aparece duplicado em elementos internos; o do body atravessa tudo.

**Vignette** NAO aparece no shell diario. E reservada exclusivamente para telas signature: `/login`, TV Dashboard, celebracoes, checkout final. Usar vignette em tela de operacao diaria e reprovado -- polui a sensacao de ferramenta.

**Accent ouro** no shell aparece em: barra lateral de 2px no item ativo da sidebar, icone do item ativo, badge de notificacao (dot 6px), CTA primaria na topbar ("Novo lead"), glow do foco em inputs. NAO aparece em: fundos de secao, labels de grupo, separadores, icones inativos, nem no logo (o logo usa gradient accent -> warm, nao ouro solido).

### Hierarquia de luminosidade -- mapa pratico

```
4%  --bg         Canvas raiz. Body, main area, fundo de modais overlay.
6%  --surface    Cards, sidebar, paineis, blocos de conteudo.
9%  --elevated   Dropdowns, popovers, modais, sheets, toast.
16% --hairline   Bordas via shadow inset. Nunca como fundo.
```

Regra operacional: olhe para o fundo imediatamente atras do componente. Se e `bg`, o componente usa `surface`. Se e `surface`, o componente usa `elevated`. Se e `elevated`, o componente continua em `elevated` (nao sobe mais). Usar `surface` sobre `surface` e invisivel -- reprovado.

### Onde o grain NAO aparece

O grain e global via `body::before`. Componentes que possuem seu proprio `overflow: hidden` com fundo opaco (modais, sheets, dropdowns) naturalmente cortam o grain. Isso e correto -- esses elementos ja tem elevacao propria e nao precisam de textura extra. NAO adicionar `.grain` em componentes internos.

### Onde o accent ouro e PROIBIDO no sistema base

- Fundo de sidebar inteira ou qualquer area > 200px de largura
- Background de cards de KPI (mesmo parcial)
- Icones de navegacao inativos
- Labels de grupo na sidebar
- Separadores
- Background de badges padrao (exceto `badge tone="accent"`)
- Gradientes decorativos em headers
- Fundo de topbar
- Qualquer area que concorra visualmente com o CTA primario da tela

---

## 2. Hierarquia tipografica por zona

### Sidebar

| Elemento | Familia | Tamanho | Peso | Cor | Extra |
|---|---|---|---|---|---|
| Logo "Torque" | `font-display` | 1.05rem | regular | `ink` | `tracking-tightest` |
| Versao "v1.0" | `font-metric` | `text-2xs` | regular | `ink-dim` | `tabular-nums` |
| Label de grupo ("Vendas", "Automacao") | `font-sans` | `text-2xs` | medium (500) | `ink-dim` | uppercase, `tracking-[0.14em]` |
| Item de nav inativo | `font-sans` | 0.8125rem (13px) | regular | `ink-muted` | -- |
| Item de nav ativo | `font-sans` | 0.8125rem (13px) | regular | `ink` | bg `elevated`, left bar accent |
| Badge numerico | `font-metric` | `text-2xs` | regular | `ink-dim` | `tabular-nums` |
| OrgSwitcher nome | `font-sans` | 0.8125rem | medium | `ink` | truncate |
| OrgSwitcher plano | `font-sans` | `text-2xs` | regular | `ink-dim` | uppercase, `tracking-[0.1em]` |
| OrgSwitcher inicial | `font-metric` | 0.7rem | medium | `accent` | em caixa `accent/15` |
| Quota footer label | `font-sans` | `text-2xs` | regular | `ink-dim` | uppercase, `tracking-[0.12em]` |
| Quota footer valor | `font-metric` | text-xs | regular | `ink-muted` | `tabular-nums` |
| Quota footer descricao | `font-sans` | `text-2xs` | regular | `ink-dim` | -- |

### TopBar

| Elemento | Familia | Tamanho | Peso | Cor | Extra |
|---|---|---|---|---|---|
| Search placeholder | `font-sans` | text-sm | regular | `ink-dim` | -- |
| Kbd shortcuts | `font-mono` | derivado do Kbd | regular | `ink-dim` | -- |
| Botao "Novo lead" | `font-sans` | text-sm | medium | `ink` sobre accent | -- |
| User menu nome | `font-sans` | text-sm | medium | `ink` | -- |
| User menu email | `font-sans` | text-xs | regular | `ink-dim` | -- |
| Menu label de grupo | `font-sans` | `text-2xs` | medium | `ink-dim` | uppercase |
| Menu item | `font-sans` | text-sm | regular | `ink-muted` -> `ink` em hover | -- |
| Menu shortcut | `font-mono` via Kbd | derivado | regular | `ink-dim` | -- |

### PageHeader

| Elemento | Familia | Tamanho | Peso | Cor | Extra |
|---|---|---|---|---|---|
| Eyebrow ("Visao geral . Hoje") | `font-sans` | text-sm | regular | `ink-muted` | -- |
| Titulo principal | `font-display` | text-display-xl (2.75rem) | regular | `ink` | `tracking-tightest`, line-height 1.05, `opsz 144 SOFT 30` |
| Parte dim do titulo | `font-display` | herda | herda | `ink-dim` | usado para segunda linha contextual |
| Descricao | `font-sans` | `text-base` (15px) | regular | `ink-muted` | line-height 1.5 |
| Acoes (botoes) | `font-sans` | text-sm | medium | depende da variante | -- |

### Cards de dashboard (quando existirem)

| Elemento | Familia | Tamanho | Peso | Cor | Extra |
|---|---|---|---|---|---|
| Card title | `font-sans` | text-sm | medium | `ink` | via `CardTitle` quando nao-editorial |
| Card title editorial | `font-display` | text-lg | regular | `ink` | quando o card e destaque |
| KPI valor grande | `font-display` | ~1.9rem | regular | `ink` | `tabular-nums`, `tracking-tightest` |
| KPI unidade/sufixo | `font-metric` | text-sm | regular | `ink-dim` | -- |
| KPI label | `font-sans` | `text-2xs` | medium | `ink-dim` | uppercase, `tracking-[0.14em]` |
| KPI delta | `font-metric` | text-xs | regular | `success` ou `danger` | `tabular-nums` |
| Dados em linha/tabela | `font-metric` | text-sm | regular | `ink` | `tabular-nums slashed-zero` |
| Timestamps | `font-metric` | `text-2xs` | regular | `ink-dim` | `tabular-nums` |

### Pagina /login -- statement tipografico

| Elemento | Familia | Tamanho | Peso | Cor | Extra |
|---|---|---|---|---|---|
| Logo "Torque" | `font-display` | text-lg | regular | `ink` | `tracking-tightest` |
| Eyebrow "Area restrita" | `font-sans` | `text-2xs` | medium | `accent` | uppercase, `tracking-[0.14em]`, com dot animado |
| Heading principal | `font-display` | 2.4rem | regular | `ink` | `tracking-tightest`, line-height 1.02 |
| Subheading | `font-sans` | 0.9375rem | regular | `ink-muted` | line-height relaxed |
| Labels de campo | `font-sans` | `text-2xs` | medium | `ink-dim` | uppercase, `tracking-[0.14em]` |
| Link "Esqueci" | `font-sans` | `text-2xs` | regular | `accent` | normal-case, hover underline |
| CTA "Entrar na operacao" | `font-sans` | text-sm | medium | sobre accent solido | -- |
| Painel direito blockquote | `font-display` | 2rem | regular | `ink` | `tracking-tightest`, line-height 1.15 |
| Painel direito accent text | `font-display` | herda | herda | `accent-gradient` | via `.text-accent-gradient` |
| Painel direito metricas | `font-metric` | 1.5rem | regular | `ink` | `tabular-nums` |
| Painel direito label metrica | `font-sans` | `text-2xs` | regular | `ink-dim` | uppercase, `tracking-[0.12em]` |
| Footer copyright | `font-sans` | `text-2xs` | regular | `ink-dim` | -- |
| Footer hint de comando | `font-sans` + `Kbd` | `text-2xs` | regular | `ink-dim` | -- |

---

## 3. Cores por contexto

### Quando usar cada nivel de canvas

| Token | Quando usar | Quando NAO usar |
|---|---|---|
| `bg` (4%) | Body background, main content area, overlay de modal (com alpha), fundo de paginas fullscreen (login, 404) | Nunca como fundo de card ou componente flutuante |
| `surface` (6%) | Cards, sidebar, paineis laterais, blocos de conteudo sobre bg, linhas de tabela, areas agrupadas | Nunca sobre outro surface (invisivel), nunca como fundo de dropdown |
| `elevated` (9%) | Dropdowns, popovers, modais, sheets, toast, itens de nav hover, OrgSwitcher fundo, estado hover de card | Nunca como fundo de pagina, nunca como bg do body |

### Quando usar cada nivel de tinta

| Token | Quando usar |
|---|---|
| `ink` (91%) | Texto principal: titulos, nomes de lead, valores de KPI, texto de botao, item de nav ativo, conteudo primario que o usuario le primeiro |
| `ink-muted` (64%) | Texto secundario: descricoes, subtitulos, labels de secao, item de nav inativo, texto de suporte que o usuario le segundo |
| `ink-dim` (40%) | Texto terciario: timestamps, metadata, hints, labels de campo uppercase, icones inativos, versao, copyright. Texto que o usuario consulta quando precisa, mas nao le proativamente |

### Accent e accent-soft

| Token | Onde aparece no sistema base |
|---|---|
| `accent` (44 93% 54%) | CTA primario (fundo), barra lateral de item ativo (2px), icone de item ativo na sidebar, dot de notificacao, glow de focus ring, dot animado no login, badge de contagem em item de nav (apenas quando critico), progress bar de quota, link "Esqueci", accent gradient no painel de login |
| `accent-soft` (44 50% 18%) | Fundo de avatar fallback no OrgSwitcher, fundo de Pill ativa, fundo de tab ativa (variante pills), fundo discreto de badges accent. NUNCA como fundo de area grande |
| `accent/10` ou `accent/15` | Micro-areas: caixa de inicial no OrgSwitcher, avatar mini na command palette, hover sutil em elementos accent |

### Shadow glow-accent

Aparece EXCLUSIVAMENTE em:
- Hover do Button `primary` (CTA ouro)
- Focus state do Button `primary`
- PIX QR ativo (futuro, fora da base)
- Podio 1o lugar (futuro, fora da base)
- Torque-tick em animacao signature

NAO aparece em: cards, sidebar, topbar, inputs, badges, nem em qualquer estado padrao de componente.

### Semanticos no sistema base

| Token | Onde aparece |
|---|---|
| `success` | Badge de WS status "conectado", delta positivo de KPI, dot de atividade tipo "won", bulk completion feedback |
| `warning` | Card de shortcut com urgencia (ex: "3 conversas aguardando > 10min"), countdown badge futuro, quota acima de 70% |
| `danger` | Item "Sair" no menu de usuario, error state de input (hairline danger + mensagem), error boundary retry, delta negativo de KPI, toast de erro |
| `info` | Dot de atividade tipo "new lead", badge de status neutro em andamento, tooltip informativo |

---

## 4. Espacamento e densidade

### Regras canonicas

**Cards:**
- Padding interno: `p-5` (20px) para cards de KPI e conteudo rico. `p-4` (16px) para cards compactos em listas.
- Nunca `p-6` ou maior em cards de operacao. `p-6` so em heroes editoriais.

**Botoes:**
- `sm`: `h-8`, `px-3`, `py-1.5`, `text-sm`
- `md` (padrao): `h-9`, `px-4`, `py-2`, `text-sm`
- `lg`: `h-10`, `px-5`, `py-2.5`, `text-sm`
- Nunca `h-12` ou `py-3` em botao padrao. `lg` e o teto.
- `icon`: `h-9 w-9` padrao, `h-8 w-8` small.

**Inputs:**
- Altura: `h-9` (36px). Nunca `h-10` ou `h-12`.
- Padding horizontal: `px-3`. Com icone: `pl-9 pr-3`.
- Gap entre label e input: `mb-1.5` (6px).
- Gap entre campos em form: `space-y-3` (12px).

**Listas e stacks:**
- Gap entre items de lista: `space-y-0.5` (2px) para nav sidebar, `space-y-1` (4px) para listas densas, `space-y-4` (16px) para timeline de atividade.
- Gap entre cards em grid: `gap-6` (24px) para grid principal, `gap-px` (1px) com bg-hairline para strip de KPI.

**Secoes:**
- Gap entre secoes no dashboard: `mt-8` (32px) entre blocos maiores, `mt-6` (24px) entre blocos secundarios.
- Padding lateral do main content: `px-8` (32px). Responsivo para mobile.

**Container:**
- `max-w-[1400px]` e o teto canonico. Centrado com `mx-auto`.
- TV Dashboard e OperationsCenter sao excecoes: usam `w-full`.

**Sidebar:**
- Largura expandida: `w-[232px]`.
- Padding do branding: `px-4 pt-4 pb-3`.
- Padding do OrgSwitcher: `px-3 pb-3`.
- Padding da nav: `px-2`.
- Altura de item de nav: `h-8`.
- Gap entre icone e label no item: `gap-2.5`.

**TopBar:**
- Altura: `h-14` (56px).
- Padding lateral: `px-6`.
- Gap entre elementos: `gap-3` para blocos, `gap-1.5` para grupo de icones.
- Search bar: `h-9`, `w-[340px]`.

**CommandPalette:**
- Largura: `max-w-[640px]`.
- Posicao: `top-[22%]`, centrada horizontalmente.
- Padding do input: `px-4 pt-4 pb-3`.
- Padding da lista: `p-2`.
- Altura max da lista: `max-h-[420px]`.
- Item: `px-2 py-2`, `gap-3`.

---

## 5. Navegacao e interacao no shell

### Sidebar

**4 grupos de navegacao:**

1. **Vendas** -- Visao geral, Funis, Conversas
2. **Automacao** -- Fluxos, Campanhas
3. **Inteligencia** -- Agentes IA, Analytics
4. **Equipe** -- (placeholder vazio no sistema base, preenchido em features futuras)

Mais footer: Configuracoes, Ajuda.

NOTA: o codigo atual usa "Operacao" como grupo unico. A direcao e migrar para 4 grupos. No sistema base, os grupos Automacao, Inteligencia e Equipe podem existir com items placeholder ou simplesmente com o label de grupo visivel e nenhum item (sinalizando que sera preenchido).

**Icones:** Lucide, `strokeWidth={1.75}`, `h-4 w-4`. Inativos em `ink-dim`, hover em `ink-muted`, ativos em `accent`.

**Estados do item de nav:**

| Estado | Fundo | Texto | Icone | Extra |
|---|---|---|---|---|
| Inativo | transparente | `ink-muted` | `ink-dim` | -- |
| Hover | `elevated/50` | `ink` | `ink-muted` | transition-colors 150ms ease-out-soft |
| Ativo | `elevated` | `ink` | `accent` | `shadow-hairline`, barra esquerda 2px `bg-accent` arredondada, h-4 |
| Focus-visible | -- | -- | -- | ring accent 2px com offset 2px bg |

**Barra accent no ativo:** posicionada `absolute left-0 top-1/2 -translate-y-1/2`, `w-[2px] h-4 rounded-full bg-accent`. E o unico ponto de accent na sidebar alem do icone.

**Colapsado vs expandido:**
- Expandido (232px): logo + nome "Torque" + versao + OrgSwitcher + nav com labels + footer + quota.
- Colapsado (56px): apenas TorqueMark centralizado + icones centralizados + tooltips ao hover (via Tooltip side="right") + avatar do user no footer.
- Transicao entre estados: 280ms `ease-in-out-precise`, width animado. Labels fazem fade-out em 150ms antes do width encolher; fade-in em 150ms apos o width expandir.

**Master area (footer da sidebar):**
Bloco de quota do plano. Mostra barra de progresso com fill accent, label de plano (uppercase, `text-2xs`), valor percentual em `font-metric`, descricao em `text-2xs ink-dim`. O bloco inteiro vive em `bg-elevated/50` com `shadow-hairline` e `rounded-md`.

### TopBar

**Search trigger:** botao que mimetiza um input. `h-9`, `w-[340px]`, fundo `surface`, `shadow-hairline`. Contem icone Search (`ink-dim`), placeholder ("Buscar leads, conversas, acoes..."), e dois Kbd (`Cmd` + `K`). Hover muda hairline para `ink-dim`. Nao e um input real -- clique abre CommandPalette.

**Botao "Novo lead":** `Button variant="secondary" size="sm"` com icone Plus. Posicionado a direita com `ml-auto`. O texto e "Novo lead" (L minusculo conforme vocabulario).

**Notificacoes:** `Button variant="ghost" size="icon"` com icone Bell. Badge dot de 6px (`h-1.5 w-1.5 rounded-full bg-accent ring-2 ring-bg`) posicionado `absolute top-1.5 right-1.5`. O dot so aparece quando ha notificacoes nao lidas. Sem numero -- apenas presenca.

**User menu:** Avatar `md` como trigger. Dropdown com: nome + email, separator, grupo "Conta" (Perfil, Preferencias, Dispositivos), separator, grupo "Organizacao" (Time e permissoes, Integracoes, Faturamento), separator, "Sair" em `text-danger`.

**Badge de WS status:** (a implementar) Pequeno dot antes do search ou no canto esquerdo da topbar. Verde `success` quando conectado, amarelo `warning` quando reconectando, vermelho `danger` quando desconectado. Tooltip explica o estado. Nunca ocupa espaco significativo.

### CommandPalette

**Estrutura de grupos:**

1. **Navegacao** -- todas as rotas principais (Visao geral, Funis, Conversas, Fluxos, Campanhas, Agentes IA, Analytics, Configuracoes). Cada item com icone + label + seta ArrowRight que aparece apenas no selecionado.
2. **Acoes rapidas** -- Novo lead (N), Novo contato, Adicionar tag. Com Kbd de shortcut quando aplicavel.
3. **Leads recentes** -- Ultimos 4-6 leads acessados, com mini-avatar (caixa 20px com iniciais em `font-metric text-accent` sobre `accent/10`).

**Input da paleta:** `font-sans` (nao display -- corrigir implementacao atual que usa `font-display` no input). Placeholder: "Busque ou digite um comando...". Dot animado accent a esquerda como indicador de ativo.

**Navegacao por teclado:** Setas cima/baixo, Enter para selecionar, Esc para fechar. Footer mostra hints de teclado em Kbd + texto `text-2xs ink-dim`. `loop` ativo no cmdk para ciclar pelo fim da lista.

**Visual:** Fundo `elevated`, `shadow-elev-3` + `shadow-hairline`, `rounded-lg`. Overlay do fundo: `bg-bg/70 backdrop-blur-sm`. Entrada com `animate-scale-in` (180ms ease-out-soft). Item selecionado: `bg-surface text-ink`, icone muda para `text-accent`.

### Transicoes entre rotas

Sem page transition complexa. O conteudo do `<Outlet>` simplesmente monta e desmonta. Skeleton aparece imediatamente se a rota carrega dados. A unica transicao e o `fade-in` de 280ms no conteudo carregado apos o skeleton resolver (via Suspense ou loading state do React Query).

Justificativa: transicoes de pagina em app de operacao sao friccionantes. O usuario navega dezenas de vezes por hora -- 300ms de animacao a cada troca e 15+ segundos perdidos por sessao. O feedback de navegacao e dado pela mudanca de active state na sidebar, que e instantanea.

---

## 6. Estados universais

### Loading

**Regra:** skeleton sempre, spinner nunca (exceto inline em botao durante acao < 800ms).

**Skeleton:**
- Fundo base: `surface`.
- Shimmer: gradient horizontal de `surface` -> `elevated` -> `surface`, animado em loop 2.4s linear.
- Forma: replica a estrutura do conteudo que vai carregar. Card de KPI = retangulo com 3 linhas de texto skeleton. Lista = N linhas com avatar skeleton + texto.
- Reduced motion: shimmer desativado, skeleton estatico em `elevated` solido.

**Botao com loading inline:**
- Conteudo do botao esconde via `opacity-0`.
- Spinner SVG de 16px aparece centralizado, animado em rotate 1s linear.
- Botao fica `pointer-events-none`, opacidade do fundo reduz para 80%.
- Duracao maxima: se >3s, algo errou -- exibir toast de erro.

### Empty

**EmptyState:**
- Centralizado no container, `py-32` para espaco vertical respirado.
- Icone (Lucide, 40px, `ink-dim`, `strokeWidth={1.25}`) opcional, nunca ilustracao colorida.
- Titulo: `font-display text-xl tracking-tightest text-ink`. Uma linha. Editorial, direto.
- Descricao: `font-sans text-sm text-ink-muted`. Uma a duas frases. Sem exclamacao, sem emoji.
- CTA: um ou dois botoes. Primario accent + secundario outline. Nunca mais de dois.
- Tom: conforme [[Vocabulario de UI]] -- "Nenhum lead por aqui. Importe uma planilha ou crie manualmente."

### Error

**Error Boundary:**
- Fundo: `bg` puro, centrado como EmptyState.
- Icone: AlertTriangle ou similar, 40px, `text-danger`, `strokeWidth={1.25}`.
- Titulo: `font-display text-xl text-ink`. Ex: "Algo quebrou."
- Descricao: `font-sans text-sm text-ink-muted`. Ex: "Nao foi possivel carregar esta secao. Tente novamente ou volte ao inicio."
- Acoes: botao "Tentar novamente" (primary) + "Voltar ao inicio" (ghost).
- Sentry e notificado automaticamente em background.

**Error de campo (input):**
- Hairline muda de `--hairline` para `--danger`.
- Mensagem de erro: `font-sans text-2xs text-danger`, imediatamente abaixo do input, `mt-1`.
- Icone de alerta opcional dentro do input (direita, 14px, `text-danger`).

### Focus

- Ring: `0 0 0 2px hsl(var(--bg)), 0 0 0 3px hsl(var(--accent))`. O gap de 2px em `bg` entre o elemento e o ring cria separacao limpa.
- Aparece com transition: `box-shadow 100ms ease-out-soft`.
- Nunca muda cor de fundo do elemento. O ring e a unica indicacao.
- Tab navigation funciona em toda a interface. `:focus-visible` apenas (nao `:focus`).

### Disabled

- Texto: `ink-dim` (40%).
- Fundo: inalterado ou com opacity 50%.
- `pointer-events: none`.
- `cursor: not-allowed` no wrapper se necessario para feedback visual antes do click.
- Sem hairline extra, sem icone de cadeado. O dim e suficiente.
- `aria-disabled="true"`.

### Hover

| Componente | Comportamento de hover |
|---|---|
| Item de nav sidebar | `bg-elevated/50`, texto `ink-muted -> ink`, icone `ink-dim -> ink-muted`, 150ms |
| Card interativo | `shadow-elev-1 -> shadow-elev-2`, `-translate-y-px`, 200ms ease-out-soft |
| Linha de lista | `bg-elevated/50`, 150ms |
| Botao primary | `shadow-glow` (glow accent), 150ms |
| Botao secondary | hairline brighten (ink-dim -> ink-muted na borda), 150ms |
| Botao ghost | `bg-elevated/50`, 150ms |
| Link inline | underline aparece, 100ms |
| Search trigger topbar | hairline muda para `ink-dim`, 150ms |

---

## 7. Pagina /login -- design signature

### Layout

Split horizontal: form a esquerda (flex-1), painel editorial a direita (flex-1, hidden em mobile < lg).

**Painel esquerdo (form):**
- Fundo: `bg` puro.
- Estrutura vertical: logo no topo, form centrado verticalmente (`justify-between`), footer no rodape.
- Form: `max-w-[380px]`, centrado com `mx-auto`.

**Painel direito (editorial):**
- Fundo: `surface` com camadas atmosfericas empilhadas:
  1. `bg-grid` com `background-size: 56px` e `opacity: 0.4` -- referencia tecnica, nao decorativa.
  2. Gradient de fade do `bg` subindo de baixo (`h-2/3`).
  3. `.vignette` -- radial gradient criando foco central.
  4. `.grain` -- textura cinematografica.
- Conteudo: meter decorativo no topo (relogio/gauge), blockquote editorial no centro com accent-gradient, 3 metricas no rodape separadas por `border-t border-hairline`.

### Uso de Fraunces como statement

O heading "Bem-vindo de volta." e Fraunces em 2.4rem com `tracking-tightest` e `line-height: 1.02`. E a primeira coisa que o usuario le. O tamanho e deliberadamente grande para um form de login -- e statement, nao utilidade.

O blockquote no painel direito tambem e Fraunces 2rem. A combinacao de dois blocos Fraunces (form esquerdo + editorial direito) cria unidade sem repeticao. O form e pessoal ("Bem-vindo de volta"), o editorial e institucional ("Se parece template, reprovou").

### Vignette + grain

Vignette e grain aparecem APENAS no painel direito. O painel esquerdo (form) e limpo, sem textura -- contraste intencional entre "area de acao" (limpa) e "area de identidade" (atmosferica).

### TorqueMark

SVG de 24px no canto superior esquerdo do form, junto ao nome "Torque" em `font-display`. O mark e geometrico (gear-inspired, nao literal), com stroke em `ink-dim` e fill gradient accent. Tamanho contido -- nao e hero, e assinatura.

### Form

- Eyebrow "Area restrita" em `text-2xs uppercase tracking-wide accent` com dot animado (pulse).
- Dois campos: email com icone Mail, senha com icone Lock. Labels uppercase `text-2xs ink-dim`.
- Link "Esqueci" em `accent`, alinhado a direita do label de senha.
- CTA: `Button primary lg w-full` com texto "Entrar na operacao" e seta ArrowRight. `justify-between` para separar texto da seta.
- Separador "ou" com linhas hairline.
- Botao SSO: `Button outline lg w-full` "Entrar com SSO do time".
- Nota de seguranca: `text-xs ink-dim text-center` "Protegido por 2FA . SSO . Trilha de auditoria completa."

### Estados do form

- **Loading:** botao CTA entra em estado loading (texto some, spinner inline).
- **Erro de credencial:** toast no canto superior direito "Credenciais invalidas" em tom danger. Campos nao mudam de estado (nao e erro de validacao, e erro de autenticacao).
- **Erro de campo:** hairline muda para danger, mensagem abaixo do campo.
- **Sucesso:** `torque-tick` animation no TorqueMark (scale 1.0 -> 1.04 -> 1.0 em 280ms com glow accent fade), seguido de redirect para `/`.

### Diferenciacao editorial

O que faz esta pagina NAO parecer um template NextAuth:
1. Split com painel editorial -- nao e form centrado sozinho.
2. Blockquote com Fraunces -- nao e feature list ou ilustracao stock.
3. Metricas reais no painel -- nao e depoimento generico.
4. Grid background + vignette + grain -- camadas atmosfericas proprias.
5. Eyebrow "Area restrita" com dot accent -- tom de operacao, nao de boas-vindas.
6. Meter decorativo no canto -- linguagem visual de instrumento de precisao.
7. Tom direto sem exclamacao -- "Bem-vindo de volta." com ponto, nao "Bem-vindo de volta!".
8. Footer com hint de `Cmd+K` -- sinaliza keyboard-first desde o primeiro contato.

---

## 8. Lacunas de identidade encontradas

### 8.1 Empty state: ilustracao ou texto+icone?

Os docs definem EmptyState com "ilustracao + titulo + descricao + CTA" mas nao ha catalogo de ilustracoes, nem estilo definido.

**Opcoes:**

A. **Apenas icone + texto (recomendado).** Icone Lucide 40px `ink-dim`, titulo `font-display`, descricao `font-sans`. Sem ilustracao. Consistente com a identidade minimalista. Nao requer assets extras. E o caminho Linear/Vercel.

B. **Ilustracoes abstratas SVG monocromaticas.** Linhas finas em `ink-dim`, accent como highlight pontual. Estilo tecnico/esquematico (como blueprint). Requer criacao de set de ilustracoes custom.

C. **Icone animado.** O icone faz uma microanimacao (fade-in + leve rotate) ao montar. Adiciona encantamento sem requer asset extra.

**Recomendacao:** A para o sistema base, evoluir para C conforme features surgem. B so com designer dedicado.

### 8.2 CommandPalette: visual especifico ou generico cmdk?

A implementacao atual ja tem visual customizado (elevated, hairline, accent em selecionado). Porem:

**Lacunas:**
- O input usa `font-display tracking-tight` -- nao e consistente com inputs normais que usam `font-sans`. Recomendacao: mudar para `font-sans`.
- Nao ha estado de loading (quando busca e assincrona).
- Nao ha feedback de "acao executada" (ex: "Novo lead" deveria ter confirmacao visual).
- "Alternar tema" esta listado nas acoes mas toggle de tema e reprovado.

**Opcoes:**

A. **Manter visual atual, corrigir font do input e remover "Alternar tema".** Minimo viavel.

B. **Adicionar skeleton de busca assincrona + confirmacao de acao com torque-tick inline.** Preparado para futuro.

**Recomendacao:** A para sistema base, B ao conectar com backend real.

### 8.3 Notification badge: estilo

Definido como dot 6px na topbar. Mas:

**Lacunas:**
- Quando ha contagem, mostra numero ou apenas dot?
- Dropdown de notificacoes: qual e o layout?
- Badge de contagem na sidebar (ex: "9" em Conversas): `font-metric text-2xs ink-dim` ou badge accent?

**Opcoes:**

A. **Dot-only no sino + contagem numerica na sidebar em `font-metric text-2xs ink-dim`.** O dot sinaliza presenca, o numero da sidebar sinaliza volume. Nao ha badge accent numerico -- evita poluicao.

B. **Dot no sino + badge accent com numero no sino quando > 9.** O threshold evita badge por qualquer notificacao, so por acumulacao.

C. **Dot no sino + dropdown de notificacoes com lista.** Requer design do dropdown que esta fora do sistema base.

**Recomendacao:** A para o sistema base. O dropdown de notificacoes e feature futura.

### 8.4 User menu: avatar + nome ou apenas avatar?

A implementacao atual mostra apenas Avatar como trigger.

**Opcoes:**

A. **Apenas avatar (atual).** Compacto, funcional. O nome aparece dentro do dropdown ao abrir. Padrao Linear/Stripe.

B. **Avatar + primeiro nome.** Ocupa mais espaco, mas da contexto imediato. Padrao Airbnb.

C. **Avatar + role badge.** Avatar com borda accent para master, outline para admin, nenhuma para membro. Comunica hierarquia visualmente.

**Recomendacao:** A (apenas avatar) na topbar. O nome aparece no dropdown. No sidebar colapsado, o avatar migra para o footer da sidebar. C e interessante mas complexifica a base sem necessidade.

### 8.5 Error pages (/404, /401-403): editorial ou funcional?

A implementacao atual do 404 e funcional minimalista (eyebrow metric, titulo display, descricao com hint de Cmd+K).

**Opcoes:**

A. **Manter funcional minimalista (atual).** Consistente com o tom direto do produto. "Nao existe aqui." e Torque puro.

B. **Editorial com painel atmosferico.** Similar ao split de login, com grain + vignette + blockquote na metade da tela. Mais impressionante, mas potencialmente exagerado para uma pagina de erro.

C. **Funcional com micro-detalhe signature.** Manter layout centrado, mas adicionar: TorqueMark animado (tick lento), grid background com opacity baixa, eyebrow com numeracao de erro em `font-metric`.

**Recomendacao:** C. O 404 e uma oportunidade de encantamento discreto sem overhead de layout complexo. Exemplo:
- Eyebrow: `404 . NOT IN SCOPE` em `font-metric text-2xs ink-dim tracking-[0.2em]`.
- TorqueMark 48px centralizado com animacao tick lenta (uma vez, nao loop).
- Titulo: "Nao existe aqui." em `font-display text-4xl`.
- Descricao com hint de `Cmd+K`.
- Grid background `opacity-20` atras, confinado ao bloco central.

Para `/401-403`:
- Eyebrow: `401 . ACESSO NEGADO` ou `403 . SEM PERMISSAO`.
- Titulo: "Acesso restrito." em `font-display`.
- Descricao: "Esta area exige permissao que voce nao tem. Fale com o admin da sua organizacao."
- Acoes: "Voltar" (ghost) + "Sair" (danger outline).

### 8.6 Sidebar colapsada: definicao incompleta

Os docs mencionam sidebar colapsada mas nao definem:
- Trigger de colapso (botao? keyboard? auto em breakpoint?)
- Onde fica o OrgSwitcher colapsado
- Como o quota block se comporta

**Opcoes:**

A. **Trigger manual via botao na sidebar (hamburger/chevron) + auto em breakpoints < 1024px.**

B. **Apenas auto em breakpoints, sem toggle manual.**

C. **Toggle manual + persistent preference em localStorage.**

**Recomendacao:** C. O usuario de operacao que usa tela de 1920px pode preferir maximizar area de conteudo. Persistir a preferencia respeita a intencao. O trigger e um ChevronLeft 14px no rodape da sidebar (expandida) ou ChevronRight (colapsada). OrgSwitcher colapsado mostra apenas a caixa de iniciais 28px. Quota block desaparece no colapsado (visivel apenas na topbar como tooltip ou no user menu).

### 8.7 Tokens ausentes no globals.css

Comparando o `globals.css` atual com a especificacao em [[Design System Base]]:

**Ausentes no CSS (presentes nos docs):**
- `--heat-1` a `--heat-5` (calor)
- `--channel-whatsapp`, `--channel-messenger`, `--channel-instagram`, `--channel-sz`
- `--countdown-safe`, `--countdown-warn`, `--countdown-urgent`
- `--job-pending`, `--job-running`, `--job-completed`, `--job-failed`
- `--radius-sm`, `--radius`, `--radius-lg`, `--radius-xl`, `--radius-full`
- `--shadow-elev-1`, `--shadow-elev-2`, `--shadow-elev-3` como custom properties (existem em tailwind config mas nao como CSS vars)
- `--shadow-glow-accent` como custom property

**Divergencias:**
- Stages no CSS: 7 tokens com valores diferentes dos docs. CSS tem `--stage-1: 220 9% 64%` (reusa ink-muted), docs tem `--stage-1: 220 30% 55%`. Os docs sao a fonte de verdade mais recente -- reconciliar.
- `torque-tick` no tailwind config faz `rotate 0 -> 180 -> 0` mas os docs definem como `scale 1.0 -> 1.04 -> 1.0 com glow fade`. Reconciliar com os docs.
- Shimmer no tailwind e 2.4s, docs dizem 1.2s. Reconciliar.
- `caret-blink` no tailwind e 1.25s, docs dizem 1.06s. Reconciliar.

Acao: Bloco 1 do [[Plano de Execucao]] resolve todas essas lacunas.

### 8.8 `border` no codigo atual

O `Sidebar.tsx` usa `border-t border-hairline` no footer e o `LoginPage.tsx` usa `border-t border-hairline` em separadores. Conforme [[Criterios de Reprovacao]] item 10, isso deveria ser `shadow-[inset_0_1px_0_0_hsl(var(--hairline))]` ou equivalente. Corrigir no redesign.

---

## 9. Checklist de aderencia

Antes de implementar qualquer tela do Sistema Base, o implementador verifica cada item abaixo. Falha em qualquer item bloqueia merge.

### Identidade

- [ ] Tela e 100% dark. Zero referencia a light mode, zero `ThemeProvider` com toggle.
- [ ] Se tirar o logo e o nome "Torque", a tela ainda e reconhecivel como Torque (tipografia editorial, density, accent ouro, grain).
- [ ] Nao parece nenhum template SaaS generico (Tailwind UI, shadcn demo, NextAuth default).

### Tipografia

- [ ] Headings/titulos usam `font-display` (Fraunces) com `opsz 144 SOFT 30 tracking-tightest`.
- [ ] Body, labels, nav usam `font-sans` (Instrument Sans).
- [ ] Numeros em coluna, KPIs, timestamps, IDs usam `font-metric` (JetBrains Mono) com `tabular-nums slashed-zero`.
- [ ] Fontes sao self-hosted via `@fontsource`. Zero referencia a Google Fonts CDN.
- [ ] Nenhum uso de `font-bold` redundante em display grande.
- [ ] Nenhuma mistura inline de Fraunces + Instrument Sans no mesmo span.
- [ ] Nenhum numero com `font-sans` em vez de `font-metric`.

### Cores

- [ ] Todas as cores via tokens CSS (`--bg`, `--surface`, `--ink`, etc.). Zero hex hardcoded, zero `text-white` direto.
- [ ] Hierarquia de canvas respeitada: bg -> surface -> elevated, sem pular nivel.
- [ ] Accent ouro restrito a CTAs, active states, highlights pontuais. Sem fundo grande accent.
- [ ] Cores semanticas (success/warning/danger/info) usadas apenas para semantica real, nunca para estetica de stages.
- [ ] Stages usam `--stage-1` a `--stage-7`, nunca rainbow.

### Bordas e sombras

- [ ] Zero uso de `border` do Tailwind para separadores visuais. Tudo via `shadow-hairline` ou equivalente `shadow-[inset_...]`.
- [ ] Elevacao correta: card lista -> elev-1, dropdown -> elev-2, modal -> elev-3.
- [ ] `shadow-glow` apenas em CTA primary hover e estados signature.

### Espacamento

- [ ] Padding de card: `p-4` a `p-5`. Nunca `p-6` em card de operacao.
- [ ] Padding de botao: `py-1.5` a `py-2.5`. Nunca `py-3`.
- [ ] Gap entre items de lista: `space-y-0.5` a `space-y-4` conforme contexto. Nunca `gap-6` em lista densa.
- [ ] Altura de input: `h-9`. Nunca `h-12`.
- [ ] Container: `max-w-[1400px]`.

### Motion

- [ ] Eases customizadas: `--ease-out-soft` ou `--ease-in-out-precise`. Zero `ease` padrao CSS.
- [ ] Duracao: 150ms micro, 280ms sweet spot, 300ms teto. Nunca > 350ms.
- [ ] Propriedade especifica em transitions (`transition-[transform,opacity]`). Nunca `transition-all` com duracao longa.
- [ ] Loading: skeleton com shimmer. Nunca spinner fullscreen.
- [ ] `prefers-reduced-motion: reduce` respeitado.

### Acessibilidade

- [ ] Contraste minimo 4.5:1 para texto normal, 3:1 para texto grande.
- [ ] Focus visible com ring accent em todos os elementos interativos.
- [ ] Keyboard navigation funcional em sidebar, topbar, command palette, menus.
- [ ] `aria-label` em botoes icon-only.
- [ ] `aria-hidden` em icones decorativos.
- [ ] Touch targets >= 44px em contextos moveis.
- [ ] Labels visiveis em todos os campos de formulario.

### Componentes

- [ ] Componentes usam primitivos de `@/ui/`, nao criam variantes ad-hoc.
- [ ] Radix usado apenas para comportamento (focus trap, ARIA, keyboard). Visual e 100% custom.
- [ ] Nenhum componente de dominio importado no shell ou nas paginas-casca.

### Vocabulario

- [ ] "Funis" (nunca "Pipelines" em nav).
- [ ] "Conversas" (nunca "Chat" em nav).
- [ ] "Agentes IA" (nunca "AI Agents", "Bot").
- [ ] "Novo lead" (L minusculo).
- [ ] Tom: direto, sem exclamacao, sem emoji, sem "com sucesso", sem "voce".

---

## Referencias

- [[Principios de Identidade Visual]]
- [[Design System Base]]
- [[Tipografia]]
- [[Motion e Animacao]]
- [[Componentes Primitivos]]
- [[Criterios de Reprovacao]]
- [[Vocabulario de UI]]
- [[Escopo do Sistema Base]]
- [[Checklist Sistema Base]]
- [[Pendencias e Lacunas]]
