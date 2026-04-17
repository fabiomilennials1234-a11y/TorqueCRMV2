---
tipo: identidade
---

# Permissões por Feature

Catálogo granular de ações que a engine de permissão conhece. Cada ação tem:

- Nome canônico: `<feature>.<verbo>[:<escopo>]`.
- Descrição.
- Papéis/specialties default autorizadas.
- Observações.

## Convenções

- Escopo `:own` significa "apenas sobre entidades atribuídas ao usuário".
- Escopo `:all` significa "sobre qualquer entidade da organização".
- Ações sem sufixo são irrestritas dentro da org (mais adequado para admin).
- `view` implica leitura; verbos dedicados (`view_sensitive`) para casos especiais.

## Lead

| Ação | Descrição | Default |
|---|---|---|
| `lead.view:own` | Ver leads com user como responsible/sdr/closer. | SDR, Closer, Prospectador |
| `lead.view:all` | Ver todos os leads da org. | Admin |
| `lead.view:unassigned` | Ver leads sem responsável. | SDR |
| `lead.create` | Criar lead manualmente. | SDR, Closer, Prospectador, Admin |
| `lead.update:own` | Editar lead atribuído. | SDR, Closer |
| `lead.update:all` | Editar qualquer lead. | Admin |
| `lead.assign` | Atribuir/reatribuir responsáveis. | Admin |
| `lead.assign:self` | Assumir lead sem responsável. | SDR |
| `lead.delete` | Deletar lead (soft). | Admin |
| `lead.hard_delete` | Remoção permanente. | Admin (com confirmação) |
| `lead.export` | Exportar CSV/JSON. | Admin |
| `lead.tag` | Adicionar/remover tag. | SDR, Closer, Admin |
| `lead.view_score` | Ver qualification_score. | Todos com `lead.view` |
| `lead.edit_score_weights` | Editar pesos do algoritmo. | Admin |
| `lead.view_custom_fields` | Ver campos custom. | Default herda de `lead.view` |
| `lead.edit_custom_fields_schema` | Configurar schema. | Admin |

## Pipeline

| Ação | Default |
|---|---|
| `pipeline.view:all` | Admin |
| `pipeline.view:whatsapp` | SDR, Admin |
| `pipeline.view:confirmacao` | Closer, Admin |
| `pipeline.view:propostas` | Closer, Admin |
| `pipeline.view:{custom_id}` | Configurado por admin |
| `pipeline.move_entry:whatsapp` | SDR, Admin |
| `pipeline.move_entry:confirmacao` | Closer, Admin |
| `pipeline.move_entry:propostas` | Closer, Admin |
| `pipeline.create_custom` | Admin |
| `pipeline.edit_config` | Admin |
| `pipeline.delete_custom` | Admin (confirmação) |
| `pipeline.reorder_stages` | Admin |
| `pipeline.edit_dispatch_rules` | Admin |

## Conversa / Chat

| Ação | Default |
|---|---|
| `conversation.view:own` | Todos membros ativos |
| `conversation.view:all` | Admin |
| `conversation.send_message` | SDR, Closer, Prospectador, Admin |
| `conversation.send_media` | SDR, Closer, Prospectador, Admin |
| `conversation.take_over` | Qualquer com `send_message` |
| `conversation.assign_to_self` | SDR, Closer, Prospectador |
| `conversation.assign_other` | Admin |
| `conversation.archive` | Admin + membros com escopo |
| `conversation.add_note_internal` | Default todos com `view` |
| `conversation.view_summary` | Default todos com `view` |

## Follow-up

| Ação | Default |
|---|---|
| `followup.view:own` | Todos ativos |
| `followup.view:team` | Admin |
| `followup.create:own` | SDR, Closer, Admin |
| `followup.create:for_other` | Admin |
| `followup.complete` | Atribuído ou admin |
| `followup.delete` | Admin |

## Template de Mensagem

| Ação | Default |
|---|---|
| `template.view` | Todos com acesso a chat |
| `template.use` | Todos com acesso a chat |
| `template.create` | Admin (SDR pode criar privados) |
| `template.edit` | Admin (proprietário do privado) |
| `template.delete` | Admin |

## Mensagem Agendada

| Ação | Default |
|---|---|
| `scheduled_message.view:own` | SDR, Closer, Admin |
| `scheduled_message.create` | SDR, Closer, Admin |
| `scheduled_message.cancel:own` | Criador |
| `scheduled_message.cancel:any` | Admin |

## Equipe

| Ação | Default |
|---|---|
| `team.view` | Admin |
| `team.invite` | Admin |
| `team.update:other` | Admin |
| `team.remove` | Admin |
| `team.change_role` | Admin |
| `team.change_specialty` | Admin |
| `team.set_permissions` | Admin |
| `team.view_own_permissions` | Todos |
| `profile.edit:self` | Todos |

## Workflow

| Ação | Default |
|---|---|
| `workflow.view` | Admin |
| `workflow.create` | Admin |
| `workflow.edit` | Admin |
| `workflow.delete` | Admin |
| `workflow.activate` | Admin |
| `workflow.execute_manual` | Admin |
| `workflow.view_executions` | Admin |

## Copilot (Agente IA)

| Ação | Default |
|---|---|
| `copilot.view` | Admin |
| `copilot.create` | Admin |
| `copilot.edit` | Admin |
| `copilot.delete` | Admin |
| `copilot.activate` | Admin |
| `copilot.test_playground` | Admin |
| `copilot.view_metrics` | Admin |
| `copilot.take_over` (conversa) | SDR, Closer, Admin (membros com chat) |

## Campanha

| Ação | Default |
|---|---|
| `campaign.view` | Prospectador, Admin |
| `campaign.create` | Prospectador, Admin |
| `campaign.edit:own` | Prospectador |
| `campaign.edit:all` | Admin |
| `campaign.activate` | Prospectador, Admin |
| `campaign.pause` | Prospectador, Admin |
| `campaign.delete` | Admin |

## Produto

| Ação | Default |
|---|---|
| `product.view` | Closer, Admin |
| `product.create` | Admin |
| `product.edit` | Admin |
| `product.delete` | Admin |
| `product.add_to_proposal` | Closer |
| `product.import_from_erp` | Admin |

## Proposta (dentro de Pipeline Propostas)

| Ação | Default |
|---|---|
| `proposal.view` | Closer, Admin |
| `proposal.edit_items` | Closer, Admin |
| `proposal.apply_discount:up_to_10%` | Closer |
| `proposal.apply_discount:above_10%` | Admin |
| `proposal.send_to_lead` | Closer |
| `proposal.mark_won` | Closer |
| `proposal.mark_lost` | Closer |

## Comissão

| Ação | Default |
|---|---|
| `commission.view:own` | Todos com venda |
| `commission.view:all` | Admin |
| `commission.edit_rules` | Admin |
| `commission.approve` | Admin |
| `commission.export` | Admin |

## Meta

| Ação | Default |
|---|---|
| `goal.view:own` | Todos |
| `goal.view:team` | Admin |
| `goal.edit` | Admin |
| `goal.create` | Admin |

## Premiação

| Ação | Default |
|---|---|
| `reward.view` | Todos |
| `reward.edit_rules` | Admin |

## Analytics

| Ação | Default |
|---|---|
| `analytics.view:own` | Todos |
| `analytics.view:team` | Admin |
| `analytics.view:company` | Admin |
| `analytics.export` | Admin |
| `analytics.view:financial` | Admin |
| `analytics.view:outbound` | Prospectador, Admin |

## TV Dashboard

| Ação | Default |
|---|---|
| `tv_dashboard.view` | Admin (conta de TV dedicada) |
| `tv_dashboard.configure` | Admin |

## Configurações da Organização

| Ação | Default |
|---|---|
| `organization.view` | Admin |
| `organization.edit` | Admin |
| `organization.upload_logo` | Admin |
| `organization.edit_business_hours` | Admin |
| `plan.view` | Admin |
| `plan.change` | Admin |
| `billing.view` | Admin |
| `billing.update_payment_method` | Admin |

## Integrações

| Ação | Default |
|---|---|
| `integration.view` | Admin |
| `integration.connect:{provider}` | Admin |
| `integration.disconnect` | Admin |
| `integration.refresh_token` | Admin |
| `integration.view_logs` | Admin |

## Webhook

| Ação | Default |
|---|---|
| `webhook.view` | Admin |
| `webhook.create` | Admin |
| `webhook.edit` | Admin |
| `webhook.delete` | Admin |
| `webhook.view_deliveries` | Admin |
| `webhook.replay_delivery` | Admin |

## API Key

| Ação | Default |
|---|---|
| `apikey.view` | Admin |
| `apikey.create` | Admin |
| `apikey.revoke` | Admin |

## Master (somente para master)

| Ação | Default |
|---|---|
| `master.list_organizations` | Master |
| `master.provision_organization` | Master |
| `master.suspend_organization` | Master |
| `master.reactivate_organization` | Master |
| `master.hard_delete_organization` | Master |
| `master.impersonate_admin` | Master |
| `master.view_cross_org_metrics` | Master |
| `master.rotate_global_secret` | Master |
| `master.manage_plans` | Master |
| `master.view_audit_cross_org` | Master |

## Feature Gating

Cada ação está ligada a uma feature. Se a feature não está habilitada no plano da organização, toda ação associada é automaticamente **negada** com motivo `feature_not_available`.

| Feature | Ações associadas (sample) |
|---|---|
| `pipeline_structural` | pipe WhatsApp/Confirmação/Propostas |
| `pipeline_custom` | pipeline.create_custom |
| `workflow_builder` | workflow.* |
| `copilot` | copilot.* |
| `campaign` | campaign.* |
| `analytics_advanced` | analytics.view:company, analytics.export |
| `api_public` | apikey.*, webhook.* |
| `google_calendar` | integration.connect:google_calendar |
| `erp_sync` | integration.connect:tinyerp |
| `asaas_billing` | integration.connect:asaas |
| `tts_audio` | copilot.tts_enable |
| `tv_dashboard` | tv_dashboard.* |

## Overrides (exemplos)

**Exemplo 1**: SDR promovido para "SDR sênior" ganha ver todos os leads.
- Override: `{member_id: x, action: "lead.view:all", allow: true}`.

**Exemplo 2**: Closer júnior não pode aprovar desconto acima de 15%.
- Override: `{member_id: y, action: "proposal.apply_discount:above_10%", allow: false, constraints: {max_discount_percent: 10}}`.

**Exemplo 3**: Atendente (specialty=outro) pode acessar chat.
- Overrides: `{action: "conversation.view:all", allow: true}`, `{action: "conversation.send_message", allow: true}`.

## Teste de Permissão

Suite automatizada deve validar, para cada ação:

1. Default para cada combinação (role, specialty).
2. Efeito de cada override (permit, deny).
3. Feature gating.
4. Quota enforcement.
5. Escopo multi-tenant (usuário org A não acessa org B).
