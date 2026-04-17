---
tipo: dominio
entidade: Workflow
---

# Workflow

Automação modelada como DAG (grafo acíclico dirigido) de nodes conectados por edges. Configurada visualmente pelo admin. Disparada por triggers e executada por um motor de workflow.

## Componentes

1. **Workflow** — a **definição** (template).
2. **Node** — unidade do grafo (trigger, action, condition, delay, etc.).
3. **Edge** — conexão direcional entre nodes.
4. **Execution** — uma **instância** executando para um lead/entidade específica.
5. **Execution Step** — registro de um node executado dentro de uma execution.

## Workflow (definição)

### Atributos
- `id`.
- `organization_id`.
- `name`.
- `description`.
- `is_active`: se pode ser disparado.
- `trigger_node_id`: id do node trigger único do grafo.
- `nodes`: JSON ou tabela separada.
- `edges`: JSON ou tabela separada.
- `version`: versão (incrementa a cada save).
- `associated_stage_id`: opcional — stage do kanban onde o workflow aparece como badge.
- `created_by`, `updated_by`, `created_at`, `updated_at`.
- `statistics`: cache de métricas (total executions, success rate, avg duration).

### Invariantes
- Grafo é acíclico.
- Exatamente um trigger node.
- Todo node (exceto trigger) tem ao menos um edge de entrada.
- Edges respeitam compatibilidade entre portas (saída booleana de condition → 2 edges `true`/`false`; split_ab → N edges nomeados).
- Workflow inativo não dispara, mas execuções em andamento continuam até o fim.

## Node

### Atributos comuns
- `id`.
- `workflow_id`.
- `type`: enum (abaixo).
- `position`: coordenadas no editor visual.
- `config`: JSON específico do tipo.
- `display_name`.
- `output_ports`: array de port names (depende do tipo).

### Tipos de Node

| Tipo | Descrição | Config típico |
|---|---|---|
| `trigger` | Entrada do workflow. Escuta evento. | `trigger_type`, `filters` |
| `action.send_message` | Envia mensagem ao lead. | `channel`, `template_id` OU `content`, `media_url` |
| `action.send_audio` | Envia áudio (TTS ou pré-gravado). | `audio_id` OU `text_to_speak` |
| `action.move_stage` | Move lead entre stages. | `pipeline_id`, `stage_id` |
| `action.add_tag` | Adiciona tag. | `tag_id` |
| `action.remove_tag` | Remove tag. | `tag_id` |
| `action.assign_responsible` | Atribui membro. | `strategy` (`fixed`, `round_robin`, `load_based`), `member_id` (opcional) |
| `action.create_followup` | Cria tarefa de follow-up. | `title`, `due_in_hours`, `assigned_to` |
| `action.update_lead_field` | Atualiza campo do lead. | `field`, `value` (pode referenciar variáveis) |
| `action.call_webhook` | Chama endpoint externo. | `url`, `method`, `headers`, `body_template` |
| `action.start_copilot` | Aciona agente IA para tomar a conversa. | `agent_id` |
| `action.end_copilot` | Pausa agente IA. | — |
| `action.schedule_message` | Agenda mensagem para futuro. | `scheduled_for`, content fields |
| `condition` | Avalia expressão; divide fluxo. | `expression` (AND/OR de predicados) |
| `delay` | Pausa por tempo fixo ou até timestamp. | `delay_type` (`relative` / `absolute` / `until_business_hour`), valor |
| `wait_response` | Pausa até lead responder (com timeout). | `timeout_hours` |
| `split_ab` | Divide fluxo probabilisticamente entre branches. | `branches`: `[{name, weight}]` |
| `copilot` | Delega branch ao agente IA com objetivo específico. | `agent_id`, `objective`, `exit_condition` |
| `wait_business_window` | Pausa até próxima janela de negócio. | — |
| `loop` | Volta para um node anterior N vezes ou até condição. | `max_iterations`, `exit_condition` |
| `parallel` | Divide em múltiplas branches executadas concorrentemente. | — |
| `join` | Re-une branches paralelas. | `wait_for`: `all` | `any` |
| `end` | Encerra execução em ponto específico. | opcional: `reason` |

## Edge

- `id`.
- `workflow_id`.
- `from_node_id`.
- `from_port` (para condition e split_ab).
- `to_node_id`.
- `label` (opcional).

## Triggers Disponíveis

| Trigger | Evento | Filtros típicos |
|---|---|---|
| `lead_created` | Novo lead em qualquer origem. | `origin`, `tags`, `source_pipeline` |
| `lead_entered_pipe` | Lead entra em pipe específico. | `pipeline_id`, `stage_id` opcional |
| `lead_left_pipe` | Lead sai de pipe (finalizado). | `pipeline_id`, `reason` |
| `stage_changed` | Lead muda de stage dentro de um pipe. | `pipeline_id`, `from_stage_id`, `to_stage_id` |
| `tag_added` | Tag adicionada a lead. | `tag_id` |
| `tag_removed` | Tag removida. | `tag_id` |
| `message_received` | Mensagem inbound de lead. | `channel` |
| `message_not_responded` | Lead não respondeu em X horas. | `hours`, aplicado a conversa |
| `cron` | Agendado por cron. | `cron_expression` (ex.: "0 9 * * MON-FRI") |
| `manual` | Disparado manualmente. | — |
| `webhook_received` | Webhook externo dispara. | identificação por token |
| `form_submitted` | Form público da org preenchido. | `form_id` |
| `meeting_scheduled` | Reunião marcada no calendário. | — |
| `meeting_canceled` | Reunião cancelada. | — |
| `followup_overdue` | Follow-up atrasado. | `hours_overdue` |
| `payment_received` | Provedor de pagamento confirma. | — |
| `payment_failed` | Provedor reporta falha. | — |

## Condition — Operadores

Condition avalia expressão booleana sobre **variáveis do contexto** (lead, mensagem, campos custom, dados acumulados).

Operadores:
- Comparação: `equals`, `not_equals`, `greater_than`, `less_than`, `greater_or_equal`, `less_or_equal`.
- String: `contains`, `starts_with`, `ends_with`, `matches_regex`, `is_empty`, `is_not_empty`.
- Lista: `in`, `not_in`, `any_of`, `all_of`.
- Tag/relação: `has_tag`, `not_has_tag`, `has_responsible`, `is_in_pipe`, `in_stage`.
- Tempo: `is_business_hour`, `is_weekend`, `hour_between`.
- Custom: `custom_field_equals`, `custom_field_contains`.

Combinadores: `AND`, `OR`, `NOT`, grupos aninhados.

## Execution

### Atributos
- `id`.
- `organization_id`.
- `workflow_id`.
- `workflow_version`: versão do workflow no momento do dispatch (snapshot).
- `entity_type`: geralmente `lead`.
- `entity_id`.
- `status`: `pending` | `running` | `waiting` | `completed` | `failed` | `cancelled`.
- `started_at`, `completed_at`.
- `current_node_id`: onde está (quando running/waiting).
- `context`: JSON de variáveis acumuladas.
- `triggered_by`: evento que disparou.
- `error`: detalhe de falha se `failed`.

### Estados
```
pending → running ─┬─► waiting (delay ou wait_response)
                   │    └─ retorna a running quando timer expira
                   ├─► completed
                   ├─► failed
                   └─► cancelled (admin ou evento externo)
```

## Execution Step

### Atributos
- `id`.
- `execution_id`.
- `node_id`.
- `started_at`, `completed_at`.
- `status`: `running` | `succeeded` | `failed` | `skipped`.
- `input`: snapshot dos valores usados.
- `output`: saída (para condition: qual branch; para action: resultado).
- `error`: detalhe se falhar.
- `attempts`: tentativas (retries).

### Invariante
- Append-only. Mesmo em retry, novo step é criado (ou campo `attempts` incrementado mantendo histórico de tentativas).

## Motor de Execução

### Fluxo
1. Evento disparador ocorre.
2. Engine consulta workflows ativos cujos triggers combinam (evento + filtros).
3. Para cada workflow match, cria `Execution` com status=pending.
4. Worker consome execuções pendentes:
   - Set status=running.
   - Executa trigger node (geralmente no-op; extrai contexto inicial).
   - Avança para próximo node via edges.
   - Para cada node, executa handler do tipo, cria step.
   - Delays e wait_response mudam status=waiting, setam `resume_at` ou listener.
   - Condition dá `true` ou `false` → segue edge correspondente.
   - Parallel cria branches que podem rodar concorrentes.
5. Quando chega a `end` ou último node, status=completed.
6. Falha em node crítico → retry com backoff; após N → failed.

### Retry
- Nodes com efeito externo (`send_message`, `call_webhook`) tentam até 5x com backoff.
- Nodes internos (`move_stage`) geralmente 1-2x.
- Após esgotar: step=failed, execution=failed (ou continua se edge `on_failure` definido).

### Timeout
- Execução total tem timeout (default 7 dias). Excedeu → cancelled.
- Configurável por workflow.

### Concorrência
- Mesmo lead pode ter múltiplas executions ativas.
- Workflow pode ter flag `mutex_per_lead` → só 1 execution por lead (nova dispara substitui ou aborta).

## Variáveis e Templating

Nodes podem referenciar variáveis via sintaxe `{{ var }}`:

- `{{ lead.name }}`
- `{{ lead.phone }}`
- `{{ lead.custom_fields.budget }}`
- `{{ now }}`
- `{{ trigger.message.content }}`
- `{{ context.some_value }}`
- `{{ member.name }}` (quando aplicável)

Engine expande antes de executar cada action.

## Versionamento

- Edit de workflow cria nova versão.
- Executions em curso carregam versão snapshot — não afetadas por edits.
- Admin pode reverter para versão anterior.
- Limite de versões retidas (ex.: 20 últimas).

## Permissões

- `workflow.view`: ver lista e detalhe.
- `workflow.create`: criar.
- `workflow.edit`: editar.
- `workflow.activate`: ativar/desativar.
- `workflow.delete`: deletar.
- `workflow.execute_manually`: disparar manualmente.

Por default: admin tem tudo; membro não tem acesso.

## Eventos

- `WorkflowCreated`, `WorkflowUpdated`, `WorkflowActivated`, `WorkflowDeactivated`, `WorkflowDeleted`.
- `WorkflowExecutionStarted`.
- `WorkflowExecutionCompleted`.
- `WorkflowExecutionFailed`.
- `WorkflowExecutionCancelled`.
- `WorkflowStepFailed` (granular).

## Observabilidade

- Lista de executions do workflow com filtros.
- Visualização do grafo com highlights em tempo real (onde execução está).
- Detalhe de cada step (input, output, duração, erro).
- Métricas por workflow: taxa de sucesso, duração média, gargalos.
