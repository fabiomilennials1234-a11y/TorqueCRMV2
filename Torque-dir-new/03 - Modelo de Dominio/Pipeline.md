---
tipo: dominio
entidade: Pipeline
---

# Pipeline

Funil configurável representando um processo comercial. Leads avançam por stages (etapas) em direção a um desfecho positivo ou negativo.

## Componentes

Pipeline é composto de três entidades relacionadas:

1. **Pipeline** (definição do funil).
2. **Stage** (etapas do funil).
3. **Pipeline Entry** (presença de um lead específico em um pipeline específico).

## Pipeline

### Atributos
- `id`: UUID.
- `organization_id`.
- `name`: nome exibido.
- `description`: propósito.
- `type`: classificação — `structural:whatsapp`, `structural:confirmacao`, `structural:propostas`, ou `custom`.
- `is_active`: se está em uso.
- `order`: ordem de exibição na navegação.
- `color`: cor visual.
- `icon`: ícone.
- `default_entry_stage_id`: stage inicial para leads novos.
- `settings`: JSON com configs (sla_default, distribution_rule, etc.).

### Tipos

- **Estruturais** (3 pipelines padrão do Torque):
  - WhatsApp (qualificação)
  - Confirmação (de reunião)
  - Propostas (negociação/fechamento)
- **Customizado**: criado pela organização, totalmente configurável.

### Invariantes
- Tipos estruturais existem por default em toda organização.
- Estrutural não pode ser deletado; pode ser desativado.
- Custom pode ser criado, editado e deletado (com confirmação).

## Stage

### Atributos
- `id`: UUID.
- `pipeline_id`.
- `organization_id`.
- `name`: nome (ex.: "novo", "abordado").
- `label`: nome exibido (pode diferir do name técnico).
- `order`: posição no pipe.
- `color`: cor da coluna no kanban.
- `is_final`: `positive` | `negative` | `null`.
- `sla_hours`: tempo máximo antes de alerta (opcional).
- `auto_move_to_stage_id`: após `sla_hours`, mover automaticamente para essa stage (opcional).
- `description`: para ajuda inline.
- `goal`: objetivo do lead nesta stage (texto; usado pelo copilot).

### Invariantes
- `order` único dentro do pipeline.
- `name` único (case-insensitive) dentro do pipeline.
- Exatamente um stage com `is_final=positive` e um com `is_final=negative` tipicamente (custom pode ter zero ou vários).
- `auto_move_to_stage_id` deve apontar para stage do mesmo pipeline.

### Operações em Stage

- Criar, editar, reordenar (drag), deletar (confirmação — move entries para stage default).
- Ao deletar stage com entries: UI força escolha de stage destino.

## Pipeline Entry

### Atributos
- `id`.
- `organization_id`.
- `pipeline_id`.
- `lead_id`.
- `current_stage_id`.
- `entered_at`: quando entrou no pipeline.
- `moved_to_current_at`: quando chegou na stage atual.
- `meta`: JSON com dados específicos do pipe (ex.: no Propostas, valor da proposta; no Confirmação, data da reunião).
- `last_action_at`.
- `finished_at` (opcional, se chegou a stage final).
- `finished_reason` (opcional: `won`, `lost`, `abandoned`, etc.).

### Invariantes
- `(pipeline_id, lead_id)` único por entrada **ativa**. Histórico pode ter múltiplas entradas finalizadas.
- `current_stage_id` pertence a `pipeline_id`.
- Uma vez em stage final, entry é considerado fechado. Pode ser reaberto (nova entry ou reset de `finished_at`).

## Transições de Stage

### Movimentação manual
1. Usuário arrasta card no kanban (ou botão "avançar").
2. UI envia `MoveStage(entry_id, new_stage_id)`.
3. Backend valida: permissão, stage pertence ao pipe, entry existe, transição permitida pelas regras.
4. Atualiza `current_stage_id`, `moved_to_current_at`.
5. Registra `Lead History`.
6. Emite evento `LeadStageChanged(pipeline_id, from_stage, to_stage)`.
7. Consumers: regras de pipe (dispatch rules), workflows subscritos a `stage_changed`, distribuição (se a transição desencadeia reatribuição), analytics.

### Movimentação automática
- Via workflow (action `move_stage`).
- Via SLA expirado (`auto_move_to_stage_id`).
- Via regra de pipe (dispatch rule).
- Via ação do agente IA (com permissão, Kanban Rules).

### Transições bloqueadas
- Se stage atual exige "requisito" (ex.: proposta precisa ter valor > 0 pra ir a "enviada"), bloqueia e informa motivo.
- Regras customizáveis por stage em `auto_rules`.

## Pipelines Estruturais — Especificação

### Pipeline WhatsApp (Qualificação)
- Stages default: `novo`, `abordado`, `respondeu`, `esfriou`, `agendado`.
- `agendado` tem regra de "ao entrar, criar entry em Confirmação".
- Ver [[04 - Funcionalidades/Vendas/Pipeline WhatsApp (Qualificação)]].

### Pipeline Confirmação
- Stages default: `reuniao_marcada`, `confirmar_d5`, `confirmar_d3`, `confirmar_d1`, `compareceu`, `nao_compareceu`.
- Movimentação D-5/D-3/D-1 é temporal: lead desce automaticamente conforme `meeting_date` se aproxima.
- `compareceu` cria entry em Propostas.
- Ver [[04 - Funcionalidades/Vendas/Pipeline Confirmação]].

### Pipeline Propostas
- Stages default: `proposta_enviada`, `negociando`, `vendido`, `perdido`.
- `meta.value` obrigatório para chegar a `vendido`.
- `vendido` dispara cálculo de comissão e evento `LeadSold`.
- Ver [[04 - Funcionalidades/Vendas/Pipeline Propostas]].

## Pipelines Customizados

- Organização cria novo pipe com próprio nome, próprias stages.
- Útil para processos complementares (pós-venda, recuperação de inadimplente, onboarding de cliente).
- Lead pode estar em múltiplos customs simultaneamente.
- Sem regras especiais built-in; apenas o que org configurar via workflows/dispatch rules.

## Relações

- **1:N** entre Pipeline e Stage.
- **1:N** entre Pipeline e Pipeline Entry.
- **1:1** entre Pipeline Entry e Lead (+Pipeline) — no presente.
- **N:1** entre Pipeline Entry e Stage.
- **N:N** entre Stage e Workflow (workflows podem estar associadas a stages — mostra badge de contagem no kanban).

## Eventos Emitidos

- `PipelineCreated`
- `PipelineUpdated`
- `PipelineDeleted`
- `StageCreated`
- `StageUpdated`
- `StageDeleted`
- `StageReordered`
- `LeadEnteredPipe(pipeline_id, stage_id)`
- `LeadStageChanged(pipeline_id, from_stage_id, to_stage_id, triggered_by)`
- `LeadFinishedInPipe(pipeline_id, stage_id, reason)`

## Distribuição

- Regra de distribuição configurada no pipe ou por stage específica.
- Tipos:
  - `round_robin` (entre membros com specialty X).
  - `load_based` (para o membro com menos leads ativos no momento).
  - `custom_rule` (regra configurável — ex.: leads tag=Ouro → apenas para seniors).
  - `none` (sem distribuição automática).
- Executada na entrada do lead no pipe (ou na stage específica).
- Ver [[07 - Processos Assíncronos/Distribuição de Leads]].

## SLA e Alertas

- Stage pode ter `sla_hours`: se o lead fica mais que isso, alerta.
- Alerta: notificação ao responsável + opcionalmente mover stage automaticamente.
- Dashboard de admin destaca entradas vencidas.

## Visualização

- **Kanban**: colunas verticais por stage, cards por entry.
- **Lista**: tabela com filtro por stage.
- **Funil**: gráfico de conversão stage-a-stage.
- **Busca**: por nome do lead, telefone, tags.

## Permissões

- Ver pipe: membro com permissão `pipeline.{pipe_id}.view` ou preset (SDR vê WhatsApp; Closer vê Confirmação e Propostas).
- Mover entry: permissão `pipeline.{pipe_id}.move`.
- Editar configuração do pipe (stages, regras): admin apenas.
- Criar pipe customizado: admin apenas.

## Limites

- Número de pipelines customizados por org limitado por plano.
- Stages por pipeline: limite alto (ex.: 30), mas UI avisa que kanban fica pesado com > 10.

## Métricas

- **Conversão por stage**: `entries_que_saíram_para_próxima / entries_que_entraram`.
- **Tempo médio por stage**: `média(moved_to_current_at - entered_at)` por stage.
- **Throughput**: entries fechadas por período.
- **Win rate**: `entries_positivas / (positivas + negativas)`.
- **Velocity**: tempo médio do `entered_at` inicial ao `finished_at` final.
