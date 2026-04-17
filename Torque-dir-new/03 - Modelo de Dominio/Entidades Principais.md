---
tipo: dominio
---

# Entidades Principais — Resumo

Ficha rápida de cada entidade. Detalhe completo nos arquivos individuais.

## Organização

- Representa um tenant.
- Atributos: `id`, `name`, `slug`, `plan_id`, `is_active`, `created_at`, `timezone`, `custom_settings`, `logo_url`, `brand_color`.
- Invariante: slug único globalmente. Um admin inicial sempre vinculado ao criar.

## Time Member

- Usuário da organização com papel e especialização.
- Atributos: `id`, `organization_id`, `user_id`, `name`, `email`, `avatar_url`, `role` (`admin`|`membro`), `specialty` (`sdr`|`closer`|`prospectador`|`admin`|`outro`), `is_active`, `commission_config` (JSON).
- Invariante: `(organization_id, user_id)` único. Um user pode pertencer a múltiplas orgs mas uma vez por org.

## Lead

- Pessoa/empresa prospect.
- Atributos: `id`, `organization_id`, `name`, `company`, `phone`, `email`, `origin` (source livre ou enum), `rating` (1-5, manual), `qualification_score` (0-100, automático), `responsible_id`, `sdr_id`, `closer_id`, `utm_source`, `utm_medium`, `utm_campaign`, `utm_term`, `utm_content`, `custom_fields` (JSON), `created_at`, `updated_at`, `external_id` (para dedupe).
- Invariantes:
  - Nome obrigatório (mínimo 2 chars).
  - Telefone ou email obrigatório (ao menos um).
  - Telefone normalizado (E.164).
  - `qualification_score` sempre em [0,100].
  - `rating` sempre em [1,5].

## Pipeline

- Funil configurável.
- Atributos: `id`, `organization_id`, `name`, `type` (`structural:whatsapp`|`structural:confirmacao`|`structural:propostas`|`custom`), `is_active`, `order`, `description`.
- Invariante: pipelines estruturais existem por padrão e não podem ser deletados; podem ser desabilitados.

## Stage

- Etapa dentro de um pipeline.
- Atributos: `id`, `pipeline_id`, `name`, `order`, `color`, `is_final` (positivo|negativo|null), `auto_rules` (JSON), `sla_hours` (timeout para alertas).
- Invariante: ordem única dentro do pipeline. Nome único dentro do pipeline.

## Pipeline Entry

- Presença de um lead em um pipeline em uma stage.
- Atributos: `id`, `organization_id`, `pipeline_id`, `lead_id`, `current_stage_id`, `entered_at`, `moved_to_current_at`, `meta` (JSON opcional com dados específicos do pipe — ex.: valor da proposta).
- Invariantes:
  - `(pipeline_id, lead_id)` único (um lead aparece 1x por pipe por vez).
  - `current_stage_id` deve pertencer a `pipeline_id`.

## Tag

- Rótulo de segmentação.
- Atributos: `id`, `organization_id`, `name`, `color`, `description`, `category` (opcional).
- Invariante: nome único (case-insensitive) dentro da org.

## Lead-Tag

- Associação N:N.
- Atributos: `lead_id`, `tag_id`, `added_by`, `added_at`.
- Invariante: par único.

## Produto

- Item do catálogo.
- Atributos: `id`, `organization_id`, `external_id` (do ERP), `name`, `sku`, `price`, `currency`, `description`, `is_active`, `metadata` (JSON).
- Invariante: SKU único dentro da org.

## Conversa

- Thread de mensagens com um lead em um canal.
- Atributos: `id`, `organization_id`, `lead_id`, `channel`, `channel_instance_id`, `status` (`active`|`archived`), `human_takeover_until` (timestamp), `agent_id` (nullable), `last_message_at`.
- Invariante: `(lead_id, channel, channel_instance_id)` único (uma conversa por canal-instância por lead).

## Mensagem

- Unidade de comunicação.
- Atributos: `id`, `conversation_id`, `organization_id`, `direction` (`inbound`|`outbound`), `sender_type` (`lead`|`member`|`agent`|`system`), `sender_id`, `content_type` (`text`|`audio`|`image`|`document`|`video`), `content`, `media_url`, `status` (`pending`|`sent`|`delivered`|`read`|`failed`), `timestamp`, `external_id` (id no provedor para dedupe), `metadata` (JSON).
- Invariante: append-only. `external_id` único para dedupe.

## Workflow

- Automação em DAG.
- Atributos: `id`, `organization_id`, `name`, `description`, `is_active`, `trigger_type`, `trigger_config` (JSON), `nodes` (JSON ou tabela separada), `edges` (JSON ou tabela separada), `version`, `created_by`.
- Invariante: grafo acíclico. Exatamente um node trigger.

## Workflow Execution

- Instância executando.
- Atributos: `id`, `organization_id`, `workflow_id`, `lead_id`, `status` (`pending`|`running`|`waiting`|`completed`|`failed`|`cancelled`), `started_at`, `completed_at`, `context` (JSON — variáveis acumuladas), `current_node_id`, `error` (opcional).

## Execution Step

- Passo granular dentro de uma execução.
- Atributos: `id`, `execution_id`, `node_id`, `status`, `started_at`, `completed_at`, `input` (JSON), `output` (JSON), `error`.
- Invariante: append-only.

## Campanha

- Processo outbound.
- Atributos: `id`, `organization_id`, `name`, `objective`, `start_date`, `end_date`, `agent_id` (opcional), `distribution_rule`, `message_sequence` (JSON ou tabela), `is_active`, `team_goal` (JSON), `owner_id`.

## Campanha Stage

- Etapa lógica da campanha (similar a stage de pipe mas no escopo campanha).
- Atributos: `id`, `campaign_id`, `name`, `order`, `wait_hours`, `message_template_id`.

## Agente IA

- Bot conversacional.
- Atributos: `id`, `organization_id`, `name`, `template_type`, `personality_tone`, `personality_style`, `personality_energy`, `skills` (array), `allowed_topics` (array), `forbidden_topics` (array), `main_objective`, `objective_composite` (JSON), `business_context` (text), `system_prompt` (text gerado), `is_active`, `is_default`, `tts_config` (JSON opcional).
- Invariante: no máximo um `is_default = true` por org.

## Agente FAQ

- Par pergunta-resposta do agente.
- Atributos: `id`, `agent_id`, `organization_id`, `question`, `answer`, `position`, `embedding` (vetor), `category`.

## Kanban Rule (por stage do agente)

- Atributos: `id`, `agent_id`, `pipe_type`, `stage_name`, `goal`, `behavior`, `allowed_actions` (array), `forbidden_actions` (array).

## Follow-up Rule (automação do agente)

- Atributos: `id`, `agent_id`, `name`, `trigger_type`, `priority`, `filters` (JSON), `behavior` (JSON), `is_active`.

## Follow-up (tarefa)

- Atributos: `id`, `organization_id`, `lead_id`, `assigned_to`, `title`, `description`, `due_at`, `status` (`pending`|`done`|`missed`), `created_by`, `completed_at`, `origin` (`manual`|`auto`|`workflow`|`agent`).

## Lead History

- Audit de um lead.
- Atributos: `id`, `organization_id`, `lead_id`, `actor_type` (`user`|`agent`|`system`|`workflow`), `actor_id`, `action` (enum), `before` (JSON), `after` (JSON), `timestamp`, `context` (JSON).
- Invariante: append-only.

## Webhook Endpoint

- Destino externo configurado pela org.
- Atributos: `id`, `organization_id`, `name`, `url`, `secret`, `events` (array subscrito), `is_active`, `headers_extra` (JSON).

## Webhook Delivery

- Tentativa de entrega.
- Atributos: `id`, `endpoint_id`, `event_id`, `status`, `attempts`, `last_attempt_at`, `response_status`, `response_body_snippet`, `next_retry_at`, `payload` (JSON).

## Plano de Assinatura

- Definição de plano.
- Atributos: `id`, `name`, `features` (array de flags), `limits` (JSON — leads/mês, mensagens/mês, agentes, FAQs, usuários), `price_monthly`, `price_yearly`.

## Quota Counter

- Contador por org por recurso por período.
- Atributos: `organization_id`, `resource`, `period_start`, `period_end`, `value`, `limit`.

## Nota Interna

- Atributos: `id`, `organization_id`, `conversation_id`, `author_id`, `content`, `created_at`.

## Mensagem Agendada

- Atributos: `id`, `organization_id`, `lead_id`, `channel`, `content`, `scheduled_for`, `status` (`scheduled`|`sent`|`cancelled`|`failed`), `created_by`.

## Template de Mensagem

- Atributos: `id`, `organization_id`, `name`, `category`, `content`, `placeholders` (array), `is_active`.

## Instância de Canal

- Atributos: `id`, `organization_id`, `channel_type`, `provider`, `name`, `external_id` (ex.: número WhatsApp), `status` (`connected`|`disconnected`|`error`), `credentials_ref`, `last_sync_at`.

## Observações gerais

- Todos os atributos listados são **conceituais**. Implementação escolhe tipos concretos (varchar vs text, timestamp com ou sem tz, etc.).
- Nomes em inglês são convenção do domínio; na UI são em português-BR.
- Tipos complexos marcados como `(JSON)` podem ser document, JSONB, subtabela, conforme store.
