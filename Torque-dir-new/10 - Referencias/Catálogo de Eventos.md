---
tipo: referencia
---

# Catálogo de Eventos de Domínio

Lista completa de eventos emitidos pelo sistema. Todo evento segue formato padrão; consumers (workflow engine, webhooks externos, analytics, notificações, audit) se subscrevem a eventos relevantes.

## Formato Padrão

```json
{
  "event_name": "LeadCreated",
  "event_id": "uuid",
  "event_version": 1,
  "organization_id": "uuid",
  "actor_type": "user|agent|system|workflow|webhook",
  "actor_id": "uuid|null",
  "entity_type": "lead|organization|...",
  "entity_id": "uuid",
  "timestamp": "2026-04-15T12:34:56Z",
  "correlation_id": "uuid",
  "payload": { /* específico do evento */ }
}
```

## Eventos por Entidade

### Lead
- `LeadCreated` — novo lead criado.
  - payload: snapshot básico do lead.
- `LeadUpdated` — campos alterados.
  - payload: `{before, after}` diff por campo.
- `LeadAssigned` — responsible/sdr/closer mudou.
  - payload: `{role: "responsible|sdr|closer", from_member_id, to_member_id}`.
- `LeadReassigned` — reatribuição manual.
- `LeadTagAdded` — tag adicionada.
  - payload: `{tag_id, origin: "manual|automation|workflow|agent"}`.
- `LeadTagRemoved`.
- `LeadStageChanged` — moveu de stage em algum pipe.
  - payload: `{pipeline_id, from_stage_id, to_stage_id, triggered_by}`.
- `LeadEnteredPipe` — entrou em um pipe.
  - payload: `{pipeline_id, stage_id}`.
- `LeadLeftPipe` — saiu de um pipe (finalizou entry).
  - payload: `{pipeline_id, reason, final_stage_id}`.
- `LeadScoreRecalculated`.
  - payload: `{before, after, breakdown}`.
- `LeadDeleted` — soft-delete.
- `LeadRestored` — undelete.
- `LeadMerged` — dois leads consolidados (raro).

### Organização
- `OrganizationProvisioned`.
- `OrganizationUpdated`.
- `OrganizationPlanChanged`.
- `OrganizationSuspended` / `OrganizationReactivated`.
- `OrganizationDeactivated` (soft).
- `OrganizationHardDeleted`.
- `OrganizationOnboardingCompleted`.

### Membro / Time
- `MemberInvited`.
- `MemberActivated`.
- `MemberRoleChanged`.
- `MemberSpecialtyChanged`.
- `MemberSuspended` / `MemberReactivated`.
- `MemberRemoved`.
- `PermissionGranted(member_id, action)`.
- `PermissionRevoked`.
- `PermissionsPresetApplied`.

### Pipeline
- `PipelineCreated` / `PipelineUpdated` / `PipelineDeleted`.
- `StageCreated` / `StageUpdated` / `StageDeleted` / `StageReordered`.

### Conversa e Mensagem
- `ConversationStarted`.
- `ConversationArchived` / `ConversationAssigned`.
- `MessageReceived` — inbound.
- `MessageSent` — outbound confirmado.
- `MessageFailed`.
- `MessageRead` — lead leu nossa mensagem.
- `HumanTakeoverStarted`.
- `HumanTakeoverExpired`.

### Follow-up
- `FollowupCreated` / `FollowupCompleted` / `FollowupMissed` / `FollowupCancelled` / `FollowupReassigned`.

### Template
- `TemplateCreated` / `TemplateUpdated` / `TemplateDeleted`.
- `TemplateUsed`.

### Mensagem Agendada
- `ScheduledMessageCreated` / `ScheduledMessageSent` / `ScheduledMessageCancelled` / `ScheduledMessageFailed`.

### Workflow
- `WorkflowCreated` / `WorkflowUpdated` / `WorkflowDeleted`.
- `WorkflowActivated` / `WorkflowDeactivated`.
- `WorkflowExecutionStarted` / `WorkflowExecutionCompleted` / `WorkflowExecutionFailed` / `WorkflowExecutionCancelled`.
- `WorkflowStepFailed`.

### Agente IA
- `AgentCreated` / `AgentUpdated` / `AgentActivated` / `AgentDeactivated` / `AgentDeleted`.
- `AgentConversationStarted`.
- `AgentMessageSent`.
- `AgentActionExecuted(action_type)`.
- `AgentHandoffToHuman(reason)`.
- `AgentEvaluationCompleted(score)`.

### Campanha
- `CampaignCreated` / `CampaignActivated` / `CampaignPaused` / `CampaignCompleted` / `CampaignArchived`.
- `LeadEnrolledInCampaign`.
- `LeadCampaignStageChanged`.
- `LeadExitedCampaign(reason)`.

### Produto
- `ProductCreated` / `ProductUpdated` / `ProductDeactivated` / `ProductDeleted`.
- `ProductSyncStarted` / `ProductSyncCompleted` / `ProductSyncFailed`.

### Proposta (pipe Propostas)
- `ProposalCreated`.
- `ProposalUpdated` (edição de items).
- `ProposalSent`.
- `ProposalWon` (=> dispara `LeadSold`).
- `ProposalLost` (=> dispara `LeadLost`).
- `LeadSold(amount, items, closer, sdr)`.
- `LeadLost(reason)`.
- `CommissionCalculated(member, value, rule)`.
- `CommissionApproved` / `CommissionPaid` / `CommissionCancelled` / `CommissionReversed`.

### Confirmação / Reunião
- `MeetingScheduled`.
- `MeetingReminderSent(d5|d3|d1|same_day)`.
- `MeetingConfirmed`.
- `MeetingRescheduled`.
- `MeetingCancelled`.
- `MeetingAttendanceMarked(attended)`.

### Meta
- `GoalCreated` / `GoalUpdated` / `GoalClosed`.
- `GoalAchieved` (quando cruza 100%).
- `GoalExpired`.

### Premiação
- `RewardGranted(member, reward)`.

### Tag
- `TagCreated` / `TagUpdated` / `TagDeleted`.

### Integração
- `IntegrationConnected` / `IntegrationDisconnected`.
- `IntegrationTokenRefreshed` / `IntegrationTokenRefreshFailed`.
- `IntegrationCallSucceeded` / `IntegrationCallFailed`.

### Webhook Endpoint / Delivery
- `WebhookEndpointCreated` / `WebhookEndpointUpdated` / `WebhookEndpointDeleted`.
- `WebhookDeliverySucceeded` / `WebhookDeliveryFailed`.
- `WebhookEndpointHealthDegraded`.

### API Key
- `ApiKeyCreated` / `ApiKeyRevoked`.

### Upsell
- `UpsellOpportunityCreated` / `UpsellWon` / `UpsellRejected` / `UpsellPostponed`.

### Master
- `MasterAction(actor, org, action, reason)`.
- `ImpersonationStarted` / `ImpersonationEnded`.

### Pagamento / Billing
- `SubscriptionCreated` / `SubscriptionActivated` / `SubscriptionCancelled`.
- `InvoicePaid` / `InvoiceFailed`.
- `PlanUpgraded` / `PlanDowngraded`.

### Sistema
- `SystemAlert(severity, detail)`.
- `JobExecuted(job_name, result)`.

## Consumers dos Eventos

| Consumer | Eventos típicos |
|---|---|
| Workflow engine | Todos (decide quais workflows disparar). |
| Webhook outbound | Todos (org pode subscrever). |
| Analytics | Lead*, Proposal*, Meeting*, Campaign*, Commission*. |
| Audit log | Todos. |
| Notifications | Assignments, mentions, milestones. |
| Realtime UI | Entity mutations (Lead*, Stage*, Message*). |
| TV Dashboard | ProposalWon, GoalAchieved, RewardGranted. |

## Versionamento

- Evento tem `event_version`. Default 1.
- Mudança breaking no payload: incrementa versão.
- Consumers lidam com múltiplas versões em transição.

## Garantias

- **At-least-once**: evento pode ser entregue > 1 vez.
- Consumers idempotentes.
- Ordem preservada por entidade quando crítica (ex.: stage changes).

## Schema Registry

- Idealmente: schema (JSON Schema ou similar) por evento em repo central.
- Versionamento controlado.
- Clientes validam contra schema.
