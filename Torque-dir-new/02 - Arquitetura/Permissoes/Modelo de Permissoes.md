---
tipo: identidade
---

# Modelo de Permissões

Engine central que decide se `usuário U`, com papel `R`, na organização `O`, pode executar `ação A` sobre `entidade E`.

## Arquitetura em 3 Camadas

```
          ┌────────────────────────────┐
     1.   │     MASTER ADMIN           │  Bypass total com auditoria. Raramente invocado.
          │ (empresa-dona do Torque)   │
          └──────────┬─────────────────┘
                     │
          ┌──────────▼─────────────────┐
     2.   │   ORGANIZATION ADMIN       │  Dentro da própria org.
          │ (admin dentro da org)      │
          └──────────┬─────────────────┘
                     │
          ┌──────────▼─────────────────┐
     3.   │  FEATURE PERMISSIONS       │  Features disponíveis pelo plano
          │   +  ROLE MATRIX           │  + papel do membro + overrides.
          │   + MEMBER OVERRIDES       │
          └────────────────────────────┘
```

## Componentes

### Papel (role)
- `master`: super-admin da empresa-dona. Existe fora do modelo de membro da org (vínculo separado).
- `admin`: admin de uma organização específica.
- `membro`: usuário padrão da organização.

### Especialização (specialty)
- `sdr`, `closer`, `prospectador`, `admin`, `outro`.
- **Não** determina permissão per se. Apenas define **presets** sugeridos que admin pode aplicar.

### Feature
- Funcionalidade do sistema (ex.: `workflow_builder`, `copilot`, `analytics_avançado`, `api_pública`, `campanhas`).
- Feature **habilitada pelo plano** → disponível para a organização.
- Feature **desabilitada** → toda a organização não vê/não usa.

### Ação (action)
- Verbo granular sobre entidade (ex.: `lead.create`, `lead.update`, `lead.assign`, `pipeline.move_entry`, `workflow.edit`, `team.invite`, `integration.connect`).
- Catálogo completo em [[Permissões por Feature]].

### Member Override
- Tabela/estrutura onde admin concede ou nega ação específica a um membro.
- Override vence regra padrão do papel.

## Fluxo de Decisão

```
pode(usuario, acao, entidade) =
  1. usuario é master? → SIM (registra audit).
  2. entidade pertence à organização do usuario? Não → NÃO.
  3. feature da ação está habilitada no plano da org? Não → NÃO (motivo: feature_not_available).
  4. quota da feature atingida e ação requer quota? → NÃO (motivo: quota_exceeded).
  5. existe override explícito (member_id, action) que NEGA? → NÃO.
  6. existe override explícito (member_id, action) que PERMITE? → SIM.
  7. role=admin e ação não está em "apenas_master"? → SIM.
  8. role=membro e ação está em "permitido_para_membro" OU preset da specialty do membro permite? → SIM.
  9. default → NÃO.
```

## Catálogo de Ações (alto nível)

Ver [[Permissões por Feature]] para catálogo completo. Resumo:

### Lead
- `lead.view`, `lead.view_all`, `lead.view_own`, `lead.create`, `lead.update`, `lead.delete`, `lead.assign`, `lead.tag`, `lead.export`.

### Pipeline
- `pipeline.view`, `pipeline.move_entry`, `pipeline.edit` (config de pipe), `pipeline.create_custom`, `pipeline.delete_custom`.

### Time
- `team.view`, `team.invite`, `team.update`, `team.remove`, `team.change_role`.

### Workflow
- `workflow.view`, `workflow.create`, `workflow.edit`, `workflow.delete`, `workflow.activate`, `workflow.execute_manual`.

### Copilot
- `copilot.view`, `copilot.create`, `copilot.edit`, `copilot.delete`, `copilot.activate`, `copilot.test` (playground), `copilot.take_over` (pausar agente e assumir conversa).

### Campanha
- `campaign.view`, `campaign.create`, `campaign.edit`, `campaign.delete`, `campaign.activate`.

### Integrações
- `integration.view`, `integration.connect`, `integration.disconnect`, `integration.configure`.

### Analytics
- `analytics.view_own`, `analytics.view_team`, `analytics.view_company`, `analytics.export`.

### Configurações
- `organization.view`, `organization.edit`, `plan.view`, `plan.change` (admin), `billing.view`.

### Comissão
- `commission.view_own`, `commission.view_all`, `commission.edit_rules`, `commission.approve`.

### Webhook
- `webhook.view`, `webhook.create`, `webhook.edit`, `webhook.delete`, `webhook.view_deliveries`.

### API Key
- `apikey.create`, `apikey.revoke`.

### Master (somente para master)
- `master.provision_organization`, `master.impersonate`, `master.hard_delete`, `master.view_cross_org_metrics`.

## Presets por Specialty

### SDR
Permissões default:
- `lead.view_own`, `lead.view_unassigned`, `lead.create`, `lead.update`, `lead.tag`, `lead.assign_to_self`.
- `pipeline.view` (WhatsApp), `pipeline.move_entry` (WhatsApp).
- `conversation.view`, `conversation.send_message`.
- `followup.create`, `followup.complete_own`.
- `template.use`, `template.create_personal`.
- `analytics.view_own`.

Sem:
- Workflow builder, Configurações, Analytics da empresa, Time.

### Closer
Default:
- `lead.view_own`, `lead.update`, `lead.tag`.
- `pipeline.view` (Confirmação, Propostas), `pipeline.move_entry` (Confirmação, Propostas).
- `proposal.edit`, `proposal.send`.
- `product.view`, `product.add_to_proposal`.
- `calendar.view_own`, `calendar.schedule`.
- `commission.view_own`.
- `analytics.view_own`.

### Prospectador
Default:
- `campaign.view`, `campaign.edit` (próprias), `campaign.activate`.
- `lead.view_in_my_campaigns`, `conversation.send_message`.
- `analytics.view_outbound`.

### Admin
Tudo dentro da org (exceto ações `master.*`).

### Membro "outro"
Permissões mínimas:
- `lead.view_own`, profile.edit.
- Admin customiza.

## Member Overrides

Admin pode, por membro, **conceder** ações não-default ou **negar** ações default.

Exemplo: SDR específico pode acessar pipe Propostas → admin concede `pipeline.view:propostas` e `pipeline.move_entry:propostas`.

Exemplo: closer júnior não pode dar desconto acima de 10% → override nega `proposal.discount_above_10`.

UI mostra claramente permissões default + overrides.

## Enforcement em Camadas

### UI
- Esconde/desabilita botões conforme engine.
- Mostra tooltip explicativo ao hover em botão desabilitado.
- Nunca confia — backend revalida.

### Aplicação
- Antes de executar caso de uso, chama engine de permissão.
- Se negado: retorna 403 com código estruturado (`forbidden`, `quota_exceeded`, `feature_not_available`).

### Domínio
- Invariantes assumem autorização já checada. Não duplica.

### Persistência
- RLS garante isolamento de tenant; não implementa permissão fina (isso é na aplicação/domínio).

## Cache de Permissão

- Decisões são baratas (memória), mas engine pode cachear por `(user_id, permissions_version)`.
- Invalidação: `permissions_version` incrementa ao mudar papel/override.
- Cliente recebe em token ou via header e força refresh quando necessário.

## Motor (conceitual)

Interface:

```
can(user_context, action: string, resource?: {type, id, data}) -> Decision

Decision = {
  allowed: bool,
  reason: string | null,  // "role_admin", "member_override_deny", "feature_disabled", "quota_exceeded", etc.
  constraints?: object    // ex.: {max_discount_percent: 10}
}
```

## Auditoria

- Toda mudança de permissão (grant/revoke, role change) entra em audit log.
- Master impersonation é audit crítico.
- Negação de ação não é auditada por default (volume); ações sensíveis negadas sim (ex.: tentativa de acessar org alheia).

## Ações Sensíveis

Algumas ações exigem **reconfirmação** (pedido de senha novamente, code MFA, ou simplesmente confirmação dupla):

- Remover membro.
- Deletar organização.
- Revogar API key em produção.
- Hard-delete de lead.
- Mudar plano com impacto em quota.
- Master impersonation (exige justificativa).

## Delegação

- Admin pode delegar permissões específicas a membros específicos via override.
- Preset "SDR sênior" tipicamente inclui `lead.reassign` (pode mudar responsável de outros), default do plano enterprise.

## Planos e Features

- Plano define **feature flags** disponíveis.
- Engine consulta plano → se feature não disponível, decisão negativa com `feature_not_available`.
- UI mostra upsell quando aplicável ("Essa feature faz parte do plano Pro — clique para fazer upgrade").

## Quotas

- Consumo contado por organização por período.
- Exemplos: leads criados/mês, mensagens enviadas/mês, agentes IA ativos.
- Antes de criar recurso, engine consulta quota.
- Quando esgotada: decisão negativa com `quota_exceeded`.
- Reset no ciclo (mensal tipicamente).

## Interop: UI sabendo da decisão

- API pública `/me/permissions` retorna mapa `{action: bool}` para a organização ativa, considerando todas as camadas.
- UI usa para renderização condicional.
- Revalidado em mudanças de contexto ou `permissions_version`.
