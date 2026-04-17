---
tags:
  - produto
  - glossario
  - vocabulario
  - linguagem-ubiqua
  - referencia
created: 2026-04-15
last_updated: 2026-04-16
status: vivo
---

# Glossario

Linguagem ubiqua do Torque. Esses sao os **termos oficiais** do produto. Devem aparecer, exatamente como descritos aqui, em:

- Codigo (nomes de entidades, tabelas, campos, eventos, endpoints).
- UI (labels, microcopy, titulos, notificacoes).
- Documentacao interna e externa.
- Conversas com o cliente.

Divergencia cria bug semantico. Palavra errada no codigo vira tela errada, relatorio errado e metrica errada. Trate como invariante.

Produto em [[Visao do Produto]]. Papeis em [[Personas e ICP]].

---

## Termos oficiais com regras de uso

| Termo | Uso | Onde aparece | Traducoes proibidas |
|---|---|---|---|
| **Lead** | A entidade principal do CRM: uma pessoa/empresa em contato comercial com a org. Existe desde a captura ate virar `Vendido` ou `Perdido`. Permanece `Lead` para sempre — nao "vira cliente" no modelo. | Entidade de dominio, tabela `leads`, ficha do Lead, inbox, kanban, analytics, API | `prospect`, `contato`, `cliente`, `oportunidade`, `negocio`, `deal` (deal e conceito do Pipe, nao substitui Lead) |
| **Pipe** / **Funil** | O fluxo comercial completo de um processo (prospeccao, fechamento, pos-venda). Intercambiaveis na UI em portugues (`Funil`) e em conversa. **Em codigo, usar sempre `pipe`** — consistencia de schema e API. | UI: "Funil de Prospeccao". Codigo: `pipe_id`, `pipes`, `PipeService`. | `pipeline` (em codigo), `funnel` (em codigo), `processo`, `fluxo` |
| **Stage** | Coluna do kanban dentro de um `Pipe`. Representa um estado do `Lead` no processo. Configuravel por org. Transicoes disparam automacoes. | UI: titulo da coluna. Codigo: `stage_id`, `stages`, `StageTransitionEvent`. | `coluna`, `etapa` (em codigo), `fase` (em codigo), `step` |
| **Calor** | Temperatura do deal, inteiro de **1 a 5**. Atributo do `Lead` no contexto de um `Pipe`. 1 = frio, 5 = quente. Usado para priorizacao, ordenacao e Dispatch Rules. | UI: badge / slider de calor na ficha e no card do kanban. Codigo: `calor` (campo inteiro). | `temperatura` (em codigo), `score`, `prioridade`, `heat` |
| **Agendado** | `Stage` (ou flag de `Stage`) indicando que reuniao foi marcada com o `Lead`. Terminal de qualificacao do SDR. | Kanban, analytics de funil, dashboard. | `marcado`, `confirmado` |
| **Compareceu** | `Stage` seguinte a `Agendado`, indicando que o `Lead` apareceu na reuniao. **Nunca** "confirmado". Confirmacao e evento; comparecimento e fato. | Kanban, analytics de funil (taxa de no-show), comissao. | `confirmado`, `presente`, `attended` (em UI pt-BR) |
| **Vendido** | `Stage` terminal positivo do `Pipe`. Dispara integracao com TinyERP (NF-e), comissao, metricas de receita. | Kanban (coluna final), analytics, integracao ERP. | `ganho`, `fechado`, `won`, `convertido` |
| **Perdido** | `Stage` terminal negativo do `Pipe`. Exige motivo obrigatorio (enum configuravel por org). | Kanban, analytics de perda por motivo. | `descartado`, `lost`, `recusado` (motivo, nao status) |
| **Copilot** | O agente de IA conversacional que responde pelo SDR no WhatsApp, qualifica, agenda. Tem tom configuravel por org e guardrails. Nome e marca do produto. | UI: painel do Copilot, sugestoes na inbox. Codigo: `copilot`, `CopilotAgent`, `copilot_runs`. | `bot`, `chatbot`, `assistente`, `IA` (generico), `agente` (generico em UI) |
| **Oraculo Comercial** | O coach de IA no dashboard do gestor. Le numeros da operacao e aponta gargalos e recomendacoes. Diferente do [[Copilot]] — nao conversa com lead, conversa com gestor. | UI: card "Oraculo Comercial" no dashboard do Admin. Codigo: `oraculo`, `OraculoInsight`. | `insights IA`, `recomendador`, `analista IA`, `coach` (generico em UI) |
| **Dispatch Rule** | Regra de disparo automatico de sequencia de mensagens, configurada no Workflow Builder. Acionada por evento (novo `Lead`, transicao de `Stage`, tempo em `Stage`, mudanca de `Calor`). | UI: Workflow Builder, aba Dispatch Rules. Codigo: `dispatch_rules`, `DispatchRuleEngine`. | `cadencia`, `automacao de mensagem` (generico), `sequencia` (generico), `fluxo` (generico), `trigger` (em UI pt-BR) |
| **Master Admin** | O papel da Milennials que opera cross-tenant. Role `master` no sistema. Nome oficial, sempre capitalizado. | UI: area "Master Admin". Codigo: role enum `master`, `master_admin_actions`, `master_audit_log`. | `super admin`, `superadmin`, `root`, `god mode`, `dono`, `owner` (da plataforma) |
| **Org** / **Organizacao** | O tenant. A unidade de isolamento logico do SaaS. Cada cliente contrata uma Org. Toda entidade de dominio carrega `org_id`. | UI: "Organizacao", seletor de org. Codigo: `org`, `org_id`, `organizations`. Em API: `org_id` em todo payload autenticado. | `tenant` (em UI pt-BR), `workspace`, `conta`, `empresa` (ambiguo — empresa pode ser Lead B2B) |
| **SDR** | **Conceito de negocio.** Papel comercial de prospeccao dentro da org. **NAO e role de sistema.** No codigo, role continua `membro`; SDR e derivado de escopo/permissao. | UI: rotulo informativo, relatorios, ranking. | *Nunca usar como enum de role. Nunca criar `role: "sdr"`.* |
| **Closer** | **Conceito de negocio.** Papel comercial de fechamento dentro da org. **NAO e role de sistema.** No codigo, role continua `membro`. | UI: rotulo informativo, relatorios, ranking. | *Nunca usar como enum de role. Nunca criar `role: "closer"`.* |
| **Membro** | **Role de sistema.** Usuario padrao de uma org. Acesso escopado aos `Pipe`s e recursos que lhe foram atribuidos. | UI: "Membro". Codigo: role enum `membro`. | `user` (em enum), `colaborador` (em enum), `operator` |
| **Admin** | **Role de sistema.** Gestor da org. Acesso total dentro da propria org — configura `Pipe`s, Dispatch Rules, seats, billing, analytics. Nao tem acesso cross-tenant. | UI: "Admin da Organizacao". Codigo: role enum `admin`. | `gestor` (em enum — e label de UI), `owner` (em enum), `manager` |
| **Master** | **Role de sistema.** O papel do [[Master Admin]] da Milennials. Acesso cross-tenant, tudo auditado. | Codigo: role enum `master`. | `super`, `superadmin`, `root`, `god` |

---

## Regras de uso

1. **No codigo, os tres roles sao `admin`, `membro`, `master`.** Fim. Qualquer outra "role" e modelagem errada.
2. **SDR e Closer sao derivados**, nunca enum. Se a UI precisa diferenciar, e via atributo/escopo, nao via role.
3. **`Pipe` em codigo, `Funil` em UI.** Nunca `pipeline` no codigo nosso (`pipeline` ja e termo sobrecarregado em engenharia de dados e CI/CD).
4. **`Lead` e eterno.** Nao existe "virou cliente" no schema. Um `Lead` em `Stage` `Vendido` continua sendo `Lead`.
5. **`Compareceu` nunca e "confirmado".** Confirmacao e outro evento (mensagem enviada na vespera). Comparecimento e o fato no dia.
6. **`Copilot` e nome proprio.** Nao e "bot", nao e "chatbot", nao e "assistente generico". Capitalizado.
7. **`Master Admin` nunca e "super admin".** E nome oficial do papel interno da Milennials.
8. **`Org` em codigo, `Organizacao` em UI.** Nunca "tenant" na UI voltada ao cliente final.

Violacao dessas regras e bug. Corrige antes de shippar.

---

## Referencia expandida por dominio

### Entidades Centrais

- **Lead**: Pessoa ou empresa identificada como oportunidade comercial. Entidade central do sistema.
- **Time Member**: Usuario pertencente ao time comercial de uma organizacao.
- **Produto**: Item do catalogo vendido pela organizacao.
- **Tag**: Rotulo de segmentacao livre aplicavel a lead ou conversa. Relacao N:N com leads.
- **Conversa**: Sequencia persistente de mensagens entre o sistema/time e um lead em um canal.
- **Mensagem**: Unidade atomica de comunicacao dentro de uma conversa.

### Pipelines Estruturais

- **Pipe WhatsApp (Qualificacao)**: Primeiro pipe padrao. Stages tipicos: novo, abordado, respondeu, esfriou, agendado.
- **Pipe Confirmacao**: Segundo pipe padrao. Stages: reuniao marcada, D-5, D-3, D-1, compareceu, nao compareceu.
- **Pipe Propostas**: Terceiro pipe padrao. Stages: proposta enviada, negociando, vendido, perdido.
- **Pipeline Customizado**: Pipeline criado pela organizacao com stages proprias.
- **Distribuicao**: Processo de atribuir leads novos automaticamente entre SDRs/closers segundo regra.

### Automacao

- **Workflow**: Automacao modelada como grafo aciclico dirigido (DAG) de nodes conectados por edges.
- **Node**: Unidade do grafo de workflow. Tipos: trigger, action, condition, delay, wait_response, split_ab, copilot, webhook_call, wait_business_window.
- **Edge**: Conexao entre nodes que define fluxo de execucao. Pode ter condicao.
- **Trigger**: Node inicial. Tipos: lead_created, stage_changed, tag_added, tag_removed, cron, manual, webhook_received, message_received.
- **Action**: Node que executa um efeito (enviar mensagem, mover stage, adicionar tag, atribuir responsavel, etc.).
- **Execution**: Instancia de execucao de um workflow para um lead ou entidade especifica.
- **Campanha**: Processo paralelo aos pipes. Tem objetivo, deadline, agente IA, sequencia de mensagens, metas de time e distribuicao de leads.

### Inteligencia Artificial

- **Agente IA**: Entidade configurada com personalidade, objetivo, skills, contexto de negocio, regras por stage.
- **Template de Agente**: Preset inicial do agente — qualificador, SDR, followup, agendador, prospectador, custom.
- **Personalidade**: Trio tom + estilo + energia que modela voz do agente.
- **Business Context**: Bloco de texto sobre o negocio da organizacao injetado no prompt.
- **FAQ Embedado**: Par pergunta-resposta convertido em vetor para busca por similaridade (RAG).
- **Kanban Rule**: Configuracao por stage do pipe: objetivo do agente nesse stage, acoes permitidas/proibidas.
- **Batch Window**: Janela curta (ex.: 8s) em que mensagens consecutivas do lead sao agrupadas antes do agente responder.
- **Human Takeover**: Momento em que um humano responde na conversa, causando pausa automatica do agente (ex.: 10 min).
- **Smart Split**: Fragmentacao inteligente de resposta longa em multiplas mensagens consecutivas humanizadas.
- **Acao IA (AI Action)**: Instrucao estruturada gerada pelo agente que o executor aplica no dominio.
- **Lead Score**: Pontuacao 0-100 automatica baseada em sinais do lead e interacao.

### Comunicacao

- **Canal**: Meio de comunicacao com lead. Ex.: WhatsApp, Messenger, Instagram.
- **Instancia**: Conexao concreta de um canal pertencente a uma organizacao.
- **Template de Mensagem**: Texto parametrizado reutilizavel, com placeholders dinamicos.
- **Mensagem Agendada**: Mensagem cujo envio foi programado para momento futuro.
- **Nota Interna**: Comentario visivel apenas para o time dentro de uma conversa.

### Integracoes Externas

- **Orquestrador Externo**: Ferramenta como n8n para fluxos de ingestao.
- **ERP Externo**: Sistema de gestao do cliente (ex.: TinyERP) para sincronizar produtos e pedidos.
- **Provedor de Pagamento**: Servico externo que processa cobranca recorrente (ex.: Asaas).
- **Calendario Externo**: Servico externo (ex.: Google Calendar) para sincronizar agendamentos.
- **Modelo LLM**: Servico externo de linguagem usado pelos agentes IA.
- **TTS (Text-to-Speech)**: Servico de geracao de audio a partir de texto.

### Processos e Infraestrutura

- **Job Recorrente**: Processo executado em intervalo fixo por agendador.
- **Worker**: Processo em background que consome de fila ou agendamento.
- **Fila**: Estrutura de mensagens a processar, com retry e dead letter.
- **Dead Letter**: Fila de mensagens que falharam todos os retries.
- **Webhook Delivery**: Tentativa de entregar um evento de saida a um endpoint externo.
- **Idempotency Key**: Chave que garante que uma mesma operacao repetida nao gera efeitos duplicados.
- **Audit Log**: Registro imutavel append-only de acoes relevantes.
- **Janela de Negocio**: Intervalo de horas uteis configurado pela organizacao.

### Metricas e Analytics

- **CPL**: Custo Por Lead. Investimento em aquisicao / numero de leads.
- **CAC**: Custo de Aquisicao de Cliente. Investimento / clientes novos.
- **ROAS**: Return On Ad Spend. Receita por anuncio / custo do anuncio.
- **UTM**: Parametros de URL para atribuicao de origem (source, medium, campaign, term, content).
- **Conversao por Stage**: % de leads que passam de uma stage para a proxima.
- **Ticket Medio**: Valor medio de venda.
- **Pipeline Value**: Soma do valor potencial dos leads em stages nao-finais.
- **MRR**: Monthly Recurring Revenue — receita recorrente mensal do Torque.

### Siglas

CRM, SaaS, B2B, SDR, Closer, ICP, RAG, LLM, DAG, RBAC, LTV, TTS, SDD.
