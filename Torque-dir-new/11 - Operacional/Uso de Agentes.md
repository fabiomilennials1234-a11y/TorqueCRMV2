---
tags: [operacional, agentes, paralelismo, processo]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Uso de Agentes

Agentes são força de trabalho. Usados certo, aceleram e aumentam qualidade. Usados errado, consomem contexto sem retorno. Este documento define quando invocar cada um e como briefar. Complementa [[Fluxo de Trabalho]].

## Catálogo de agentes disponíveis

### `feature-dev:code-explorer`
**Para quê.** Varrer código e documentação dispersa. Extrair padrões recorrentes. Mapear features existentes e suas dependências. Primeira linha de investigação quando o território é desconhecido.
**Quando invocar.** Antes de planejar feature nova que toca área do código pouco explorada. Quando a pergunta é "o que existe aqui e como está organizado".
**Quando NÃO invocar.** Para leitura de poucos arquivos específicos já conhecidos (use `Read` direto).

### `feature-dev:code-architect`
**Para quê.** Planos de implementação detalhados. Decisões arquiteturais com alternativas ponderadas. Sequenciamento de tarefas com dependências explicitadas.
**Quando invocar.** Passo 3 do [[Fluxo de Trabalho]] quando a feature tem impacto estrutural. Sempre que houver mais de um caminho viável e a escolha não for óbvia.
**Quando NÃO invocar.** Para decisões já tomadas ou escopo trivial.

### `feature-dev:code-reviewer`
**Para quê.** Revisão independente. Validação contra convenções do projeto. Detecção de bugs, vulnerabilidades, más decisões de performance.
**Quando invocar.** Passo 6 do [[Fluxo de Trabalho]] obrigatoriamente. Sempre separado de quem executou.
**Quando NÃO invocar.** Nunca pule. Não há "obviedade suficiente" para dispensar revisão.

### `Explore`
**Para quê.** Busca rápida no codebase. Confirmação de estrutura. Localização de arquivos por padrão de nome ou conteúdo.
**Quando invocar.** Quando `Grep` e `Glob` diretos bastam, mas você quer respostas sintetizadas. Paralelo ao trabalho principal.
**Quando NÃO invocar.** Para investigação que exige síntese profunda (use `code-explorer`).

### `general-purpose`
**Para quê.** Consolidação de achados em documentação estruturada. Tarefas multi-passo abertas que não se encaixam nos agentes especializados. Escrita longa de notas técnicas.
**Quando invocar.** Quando o trabalho é documentar, sintetizar ou executar sequência aberta.
**Quando NÃO invocar.** Para tarefas onde um agente especializado resolve melhor.

## Regras de uso

- **Paralelize sempre que possível.** Múltiplos agentes independentes em uma única mensagem com múltiplos tool calls. Tempo de espera é tempo perdido. Só sequencialize quando existe dependência real de output.
- **Delegue investigação antes de implementar.** Mandar agente de exploração enquanto você pensa no plano é padrão. Não fique lendo o codebase manualmente quando um `code-explorer` faz em paralelo.
- **Revisão obrigatória por agente independente antes de avançar.** Sem exceção. Autoaprovação é anti-padrão.
- **Briefing denso e autossuficiente.** O agente não vê o histórico da conversa. Tudo que ele precisa tem que estar no briefing. Objetivo, contexto, restrições, formato de saída.
- **Sempre consolide o resultado em documentação no vault.** Output de agente vira nota linkada, não fica no histórico de mensagens.

## Padrões de briefing

Briefing bom tem quatro seções:

1. **Objetivo.** Uma frase. "Mapear como o hook `useLeads` é usado em todas as páginas do app."
2. **Contexto.** Links para notas relevantes, paths absolutos dos arquivos principais, o que já foi descartado como hipótese.
3. **Restrições.** Budget de palavras do output. Formato esperado (tabela, bullets, prosa). O que não precisa fazer.
4. **Entregável.** Formato e local. "Responder em markdown, máximo 500 palavras, com uma tabela no final listando arquivo, linha e tipo de uso."

### Exemplo de briefing denso

> **Objetivo.** Extrair todos os padrões de uso de React Query no projeto Torque-v2 para consolidar em [[Padroes React Query]].
>
> **Contexto.** Projeto em `C:/Users/torch/Desktop/milennials/Torque-v2`. Hooks ficam em `src/hooks`. Query client em `src/providers/QueryProvider.tsx`. Já foi descartada hipótese de uso de Zustand em paralelo (ver [[Decisao Estado Global]]).
>
> **Restrições.** Não propor mudanças, apenas documentar o que existe. Output máximo 800 palavras. Formato: seções por tipo de padrão (query key, invalidation, optimistic update, placeholder data).
>
> **Entregável.** Markdown pronto para virar conteúdo da nota [[Padroes React Query]]. Usar wikilinks para qualquer arquivo referenciado.

Briefing ruim é o briefing que obriga o agente a adivinhar. Se ele volta fazendo perguntas, o briefing falhou.
