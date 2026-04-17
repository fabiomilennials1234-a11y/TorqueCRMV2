---
tipo: feature
dominio: automacao
---

# Regras de Pipe (Dispatch Rules)

## Propósito

Regras **associadas a stages de pipeline** que executam automaticamente quando um lead entra ou sai da stage. São **mais simples e focadas** que workflows — menos poder, mas mais diretas. Útil para automações específicas de funil (ex.: "ao entrar em `agendado`, criar entry em Confirmação").

## Atores e Permissões

- **Admin**: CRUD.
- **Sistema**: executa.

Ações: `pipeline.edit_dispatch_rules`.

## Diferença entre Regra de Pipe vs Workflow

| Aspecto | Regra de Pipe | Workflow |
|---|---|---|
| Complexidade | Simples, linear. | Grafo complexo. |
| Escopo | Stage específico. | Qualquer trigger. |
| Estado | Stateless (action imediata). | Stateful (delay, wait). |
| Uso | Automação específica de funil. | Automações gerais. |
| Configuração | Form simples. | Editor visual DAG. |

Recomendação: começar com Regra de Pipe para coisas simples; migrar a Workflow se precisa de timing ou lógica condicional sofisticada.

## Dados Envolvidos

- `id`, `organization_id`, `pipeline_id`, `stage_id`.
- `trigger`: `on_enter` | `on_leave` | `on_sla_expired`.
- `conditions`: expressão booleana opcional (ex.: "tag has Ouro").
- `actions`: array de actions a executar em ordem.
- `order`: quando múltiplas regras match, ordem de execução.
- `is_active`.
- `created_at`, `updated_at`.

## Triggers Disponíveis

- `on_enter`: lead entrou nesta stage.
- `on_leave`: lead saiu desta stage (antes de entrar na próxima).
- `on_sla_expired`: lead está nesta stage há mais que `sla_hours` (config da stage).

## Actions Disponíveis (subset das workflow actions)

- `move_stage` (mesmo pipe ou outro).
- `add_tag`, `remove_tag`.
- `assign_responsible`.
- `create_followup`.
- `create_pipeline_entry` (cria entry em outro pipe — ex.: Propostas).
- `send_message` (template).
- `update_lead_field`.
- `call_webhook` (simples).
- `dispatch_workflow` (dispara workflow específico — ponte para Workflow Builder).

## Exemplos

### Regra 1: "Ao lead entrar em `agendado` no Pipe WhatsApp, criar entry em Confirmação"
- pipeline: WhatsApp; stage: agendado.
- trigger: on_enter.
- conditions: nenhuma.
- actions:
  1. `create_pipeline_entry(pipeline=Confirmacao, stage=reuniao_marcada, meta={meeting_date: lead.custom_fields.meeting_date})`.

### Regra 2: "Ao lead ficar esfriado, disparar workflow de retenção"
- pipeline: WhatsApp; stage: esfriou.
- trigger: on_enter.
- actions:
  1. `add_tag(Esfriado)`.
  2. `dispatch_workflow(workflow_id=RetencaoSlow)`.

### Regra 3: "Se lead em `abordado` há mais de 48h sem resposta, mover para esfriou"
- pipeline: WhatsApp; stage: abordado.
- trigger: on_sla_expired (sla_hours=48 na config da stage).
- actions:
  1. `move_stage(stage=esfriou)`.

### Regra 4: "Ao lead Propostas stage `vendido`, calcular comissão + criar entry em Onboarding Cliente"
- pipeline: Propostas; stage: vendido.
- trigger: on_enter.
- actions:
  1. `trigger_commission_calculation`.
  2. `create_pipeline_entry(pipeline=OnboardingCliente, stage=contrato_assinado)`.
  3. `send_message(template=agradecimento_venda)`.

## Fluxos do Usuário

### Criar Regra
1. Admin abre config do pipeline → stage específica → "Regras".
2. Botão "Nova regra".
3. Form: trigger, condições (opcional), actions (adiciona múltiplas).
4. Validação (referências existentes).
5. Salvar → ativa.

### Listar
- Em cada stage, tab "Regras" lista regras com status (ativa/inativa), último disparo, sucesso/falha.

### Editar / Desativar / Deletar
- Standard CRUD.

### Ver Execuções
- Log de execução de regras (similar a workflow executions).
- Útil para debug ("por que lead X foi movido automaticamente?").

## Ordem de Execução

Quando múltiplas regras matcham:
1. Ordenadas por `order` (ASC).
2. Executadas sequencialmente.
3. Action em uma pode gerar evento que dispara outra → sim, cascata é possível.
4. Proteção contra loop infinito (contador, max 10 hops por evento inicial).

## Automações e Eventos

### Emite
- `DispatchRuleTriggered(rule_id, execution_result)`.
- `DispatchRuleFailed(rule_id, error)`.

### Reage
- `LeadStageChanged`, `LeadEnteredPipe`, `LeadLeftPipe` (dispara on_enter/on_leave).
- Cron de SLA: avalia sla_expired.

## Integrações

- **Pipeline**: dono natural.
- **Workflow**: pode despachar workflow (ponte).
- **Tags, Leads, Follow-ups, Mensagens**: actions comuns.
- **Audit**: cada disparo registrado.

## Edge Cases

- **Regra com action que falha**: log erro, próximas actions da mesma regra podem continuar (config) ou parar.
- **Regras conflitantes** (uma move para stage A, outra para B): ordem define resultado; admin vê warning.
- **Loop**: A dispara B que dispara A → contador bloqueia; alerta.
- **Regra desabilitada durante execução em andamento**: execução em curso completa; novas não disparam.
- **Action referencia entidade deletada** (tag, workflow, stage): skip + log erro.

## Validações

- Trigger válido.
- Conditions sintaxe OK.
- Actions com parâmetros válidos.
- Referências (tags, workflows, stages) existentes.
- Sem ciclo direto óbvio (regra A → move para stage B → regra B move para A).

## Métricas

- Regras ativas por pipeline.
- Disparos por regra (volume).
- Taxa de sucesso.
- Regras mais executadas.
- Regras sem disparo (candidate a remoção).
