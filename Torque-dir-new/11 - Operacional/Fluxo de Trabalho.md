---
tags: [operacional, fluxo, processo, governanca]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Fluxo de Trabalho

Oito passos obrigatórios para toda etapa de construção. Não negociável. Pular passos compromete a confiança na base e reintroduz débito técnico do tipo que [[Riscos e Duvidas]] documenta explicitamente. Complementa [[Uso de Agentes]] e [[00 - Mapa de Features]].

## 1. Entender contexto

Antes de propor qualquer coisa, ler o que já existe.

- [ ] Ler notas relevantes do vault `Torque-dir-new` ligadas à feature em jogo
- [ ] Conferir o [[00 - Mapa de Features]] para localização da feature no roadmap
- [ ] Conferir [[Escopo do Sistema Base]] se for trabalho de base
- [ ] Consultar legado via [[Top 15 Docs Prioritarios]] quando houver feature correspondente
- [ ] Invocar agente `feature-dev:code-explorer` se o espaço a varrer for extenso ou disperso

## 2. Documentar contexto

O contexto vira nota antes de virar código.

- [ ] Criar nota dedicada ou atualizar nota existente com resumo do entendimento
- [ ] Formato: `o que existe + o que falta + o que é requisito + premissas explícitas`
- [ ] Linkar wikilinks para todas as referências consultadas
- [ ] Marcar frontmatter `status: draft`

## 3. Propor estrutura

Plano antes de execução.

- [ ] Escrever plano em markdown com decomposição de tarefas
- [ ] Incluir arquivos afetados, dependências, critérios de aceite por tarefa
- [ ] Invocar agente `feature-dev:code-architect` quando a decisão tem impacto estrutural
- [ ] Consolidar divergências de opinião antes de seguir; nunca partir com ambiguidade não resolvida

## 4. Validar escopo

O plano volta contra os guardrails do projeto.

- [ ] Revisar contra [[Escopo do Sistema Base]] se é trabalho de base
- [ ] Revisar contra [[00 - Mapa de Features]] para confirmar que não está misturando features
- [ ] Se o plano abraça dois escopos distintos, quebrar em dois planos antes de executar
- [ ] Confirmar que a feature anterior está 100% fechada (regra cardinal)

## 5. Executar

Implementação disciplinada.

- [ ] Uma feature por vez, nunca duas em paralelo pela mesma pessoa ou pelo mesmo agente
- [ ] Tasks numeradas, ordem respeitada salvo independência comprovada
- [ ] Agentes em paralelo somente onde tarefas são independentes de fato
- [ ] Commits atômicos com mensagens descritivas em PT-BR
- [ ] Nunca pular testes; ver [[Estratégia de Testes]]

## 6. Revisar

Revisão independente, não auto-aprovação.

- [ ] Invocar agente `feature-dev:code-reviewer` separado da execução
- [ ] Briefing do reviewer: arquivos alterados, critérios de aceite, pontos de atenção conhecidos
- [ ] Resposta a todos os apontamentos antes do merge
- [ ] Zero auto-aprovação: quem executou não aprova

## 7. Documentar resultado

Sem rastro, sem progresso.

- [ ] Registrar decisões tomadas em ADR quando estruturais
- [ ] Atualizar [[00 - Mapa de Features]] marcando progresso
- [ ] Atualizar checklist da feature
- [ ] Log de contexto da sessão com `o que + por quê + impacto + próximos passos`
- [ ] Mudar frontmatter `status` para `done` onde aplicável

## 8. Só então avançar

- [ ] Checklist da feature 100% marcado
- [ ] Nenhum item aberto em [[Riscos e Duvidas]] bloqueando
- [ ] Próxima feature só começa após todos os itens acima confirmados

## Regras gerais

- **Nunca trabalhe ad hoc.** Todo trabalho passa pelos oito passos, inclusive quick wins.
- **Nunca misture base com feature.** Trabalho de Sistema Base vive em ciclo próprio, com seu próprio checklist. Mistura obriga retrabalho quando a feature mudar e o pedaço de base tiver que voltar para o isolamento.
- **Sempre deixe rastro documental.** Decisão não documentada é decisão perdida. Vault é memória externa, não arquivo.
- **Sempre use o formato `o que + por quê + impacto + próximos passos`.** Vale para ADRs, logs, descrição de PR, commits relevantes. Quem ler em três meses precisa entender sem precisar perguntar.
