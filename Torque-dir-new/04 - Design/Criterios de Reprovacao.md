---
tags: [design, qa, reprovacao, red-flags, review]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Critérios de Reprovação

Dez red flags que reprovam uma tela automaticamente. Não é sugestão, não é preferência, não é "a gente pode discutir". São as linhas que definem o padrão Torque. Se qualquer item abaixo é verdadeiro, a tela volta.

Este documento é a última checagem antes de shippar. Ver [[Principios de Identidade Visual]], [[Design System Base]], [[Tipografia]], [[Motion e Animacao]], [[Componentes Primitivos]], [[Vocabulario de UI]].

---

## 1. Parece template SaaS genérico

**Reprovado.**

Cards brancos arredondados com ícone colorido no canto, header de nav com logo à esquerda e avatar à direita, botão roxo com gradiente. Se a tela poderia estar num concorrente, num boilerplate de Vercel, num template do Tailwind UI sem modificação significativa — é template.

**Certo:** hierarquia por luminosidade (surface/elevated), tipografia editorial com display+body+metric, hairline via shadow, accent restrito. Identidade reconhecível em screenshot sem logo. Ver [[Principios de Identidade Visual]].

---

## 2. Poderia pertencer a qualquer produto

**Reprovado.**

Critério de especificidade. Se você abre a tela, tira os textos literais (nomes de entidades, labels) e troca por lorem ipsum, e ela ainda funcionaria como dashboard de qualquer CRM/ERP/project-manager — falta identidade.

**Certo:** a tela tem elementos que são Torque: vocabulário ("Funis", "Conversas", "Oráculo Comercial"), microcopy editorial, uso de Fraunces em KPIs, torque-tick em momentos-chave, grain sutil no canvas. Ver [[Vocabulario de UI]].

---

## 3. Escolheu opção visual segura em vez da certa

**Reprovado.**

Escolher Inter porque "todo mundo usa". Escolher border sólido padrão porque "é o default". Escolher azul porque "não ofende ninguém". Escolher spinner porque "é mais rápido que fazer skeleton". Essas escolhas individualmente parecem inofensivas, acumuladas matam a identidade.

**Certo:** toda decisão visual tem justificativa. Instrument Sans porque diferencia. Hairline via shadow porque não polui. Accent ouro porque é carro-chefe cromático. Skeleton porque mantém forma enquanto carrega. Defaults são escolhas — e defaults do mercado não são os defaults do Torque.

---

## 4. Light mode ou toggle de tema

**Reprovado.**

Dark-first não é "começamos em dark e depois fazemos light". É "dark é o design". Implementar light adiciona dívida técnica (duplicar toda a paleta), quebra a sensibilidade cinematográfica (vignette, grain, hairline), e sinaliza que a equipe não acredita na decisão.

**Certo:** zero CSS de light mode no bundle. Zero `ThemeProvider` com opção de switch. `<html class="dark">` hard-coded. Ver [[Principios de Identidade Visual]].

**Exceção documentada:** páginas de impressão (PDF gerado, invoice imprimível) podem usar estilo invertido específico, claramente marcado, não acessível via toggle.

---

## 5. Rainbow para stages ou states

**Reprovado.**

Pegar as 7 cores semânticas (danger, warning, accent, success, info...) e atribuir uma para cada estágio de funil é amadorismo cromático. Parece Trello. Perde toda comunicação semântica (o que o usuário entende quando vê vermelho? erro? stage 5?).

**Certo:** paleta `--stage-1` a `--stage-7` é curada — progressão de frios para quentes, acentuando no ouro no stage 6 (fechamento). `--heat-1` a `--heat-5` mesma lógica. Cores semânticas (danger, success) ficam para semântica real. Ver [[Design System Base]].

---

## 6. Tipografia sem hierarquia display/body/metric

**Reprovado.**

Tela com tudo em Inter (ou tudo em Instrument Sans). KPI grande sem Fraunces. Números em coluna sem JetBrains Mono `tabular-nums`. Nenhum label em uppercase tracking-wider. Isso é tipografia de dashboard genérico.

**Certo:** toda tela relevante tem as três vozes. Título em `font-display` (Fraunces). Body e labels em `font-sans` (Instrument Sans). Qualquer número que forme coluna ou que seja KPI em `font-metric` (JetBrains Mono tabular). Ver [[Tipografia]].

---

## 7. Animação com ease linear ou >350ms

**Reprovado.**

`transition: all 0.4s ease` é o padrão do mercado. Linear é duro, 400ms é lento, `all` é impreciso. Três erros em uma linha.

**Certo:** ease custom (`--ease-out-soft` ou `--ease-in-out-precise`), duração entre 150ms e 300ms (sweet spot 280ms), propriedade específica (`transition-[transform,opacity]`, não `all`). Ver [[Motion e Animacao]].

---

## 8. Número sem font-metric tabular-nums

**Reprovado.**

Tabela com coluna de valores monetários onde `R$ 1.234,56` e `R$ 987,00` têm larguras diferentes entre linhas. Impossível escanear, impossível comparar. Amador.

**Certo:** toda coluna numérica, todo KPI, todo counter, todo timestamp usa `font-metric tabular-nums slashed-zero`. Dígitos alinham perfeitamente. O olho do usuário faz scan vertical sem tropeçar. Ver [[Tipografia]].

---

## 9. Loading com spinner "Carregando…" em vez de skeleton

**Reprovado.**

Spinner no centro de um container vazio força o usuário a esperar às cegas. Não sabe o que vai aparecer, não sabe quando, não tem contexto. Parece um produto dos anos 2010.

**Certo:** skeleton com a forma exata do conteúdo que vai carregar. Card com 3 linhas? Skeleton com 3 linhas. Tabela com 10 rows? Skeleton com 10 rows. Animação `shimmer` em loop. O usuário já vê a estrutura, só espera o preenchimento. Ver [[Componentes Primitivos]] primitivo `Skeleton` e [[Motion e Animacao]].

**Exceção:** ações discretas de <800ms (salvar botão, confirmar modal) podem usar spinner pequeno inline dentro do botão — mas não tela cheia.

---

## 10. `border` Tailwind padrão em vez de shadow-hairline

**Reprovado.**

`border border-white/10` cria uma linha dura de 1px com contraste alto que grita em dark. Polui a tela, compete com conteúdo, destrói a sensibilidade cinematográfica.

**Certo:** `shadow-[inset_0_0_0_1px_hsl(var(--hairline)/0.6)]` ou a classe utilitária `shadow-hairline`. Linha existe, é legível, mas não briga com o conteúdo. Hairline-first é princípio fundamental do visual Torque. Ver [[Principios de Identidade Visual]] e [[Design System Base]].

---

## Processo de review

Antes de abrir PR de qualquer tela nova ou ajuste visual:

1. Rodar mentalmente os 10 itens acima.
2. Se qualquer um falha, ajustar antes de pedir review.
3. Se há dúvida em algum item, ler a nota linkada e decidir.
4. Em review, reviewer escreve o número do item que falhou (ex: "Falha em 7 — ease linear na transição do tab"). Autor corrige e re-submete.

## Referência cruzada

- [[Principios de Identidade Visual]] — fundação conceitual.
- [[Design System Base]] — tokens exatos.
- [[Tipografia]] — hierarquia obrigatória.
- [[Motion e Animacao]] — eases e durações.
- [[Componentes Primitivos]] — catálogo de uso correto.
- [[Vocabulario de UI]] — microcopy e tom.
