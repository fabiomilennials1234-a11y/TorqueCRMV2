---
tipo: dominio
entidade: Organização
---

# Organização (Tenant)

Unidade de isolamento do sistema. Cada cliente do Torque possui uma organização, e **nenhum dado cruza fronteiras entre organizações** (exceto via Master Admin, com auditoria).

## Atributos Conceituais

### Identificação
- `id`: UUID.
- `slug`: identificador amigável único globalmente (ex.: `fabrica-xyz`).
- `name`: razão social ou nome fantasia.
- `display_name`: nome exibido na UI.

### Dados
- `cnpj`: CNPJ validado (opcional em plano grátis).
- `phone`: telefone de contato.
- `email`: email de contato principal.
- `logo_url`: logo da organização.
- `brand_color`: cor de destaque (HSL ou hex).
- `timezone`: fuso horário (default `America/Sao_Paulo`).
- `locale`: idioma (default `pt-BR`).
- `address`: endereço (JSON com campos estruturados).

### Plano e Estado
- `plan_id`: referência ao plano de assinatura.
- `plan_activated_at`.
- `plan_expires_at`.
- `is_active`: se pode operar (false bloqueia acesso).
- `suspension_reason`: motivo da suspensão (inadimplência, decisão master, pedido do cliente).
- `trial_until`: data de fim de trial (opcional).

### Configurações
- `business_hours`: janela de negócio (ex.: segunda-sexta 09:00-18:00, com exceções).
- `custom_settings`: JSON com configurações diversas (cor do brand no copilot, mensagem de boas-vindas default, fuso horário, etc.).
- `feature_flags`: overrides explícitos (raramente usados, geralmente features vêm do plano).
- `quota_overrides`: raros overrides de quota negociados com cliente específico.

### Metadados
- `created_at`.
- `updated_at`.
- `onboarded_at`: quando completou onboarding inicial.
- `last_activity_at`: última ação administrativa relevante.

## Invariantes

1. `slug` único globalmente.
2. `name` e `display_name` não-vazios.
3. Ao menos um admin vinculado em todo momento (não pode ficar sem admin).
4. `is_active = false` bloqueia todas as operações de cliente, mas mantém dados acessíveis ao master.
5. Organização com `plan_expires_at` no passado entra em estado de **expirada** — acesso read-only até renovação.

## Ciclo de Vida

```
provisionamento → onboarding → ativa → (suspensa | expirada)* → desativada → arquivada (soft delete)
                                                                           → hard-delete (master + janela de carência)
```

### Provisionamento
1. Master admin ou checkout automatizado cria organização.
2. Gera `id`, `slug`, registra `plan_id`.
3. Cria admin inicial (convite por email ou provisionado).
4. Cria entidades default: pipelines estruturais, tags básicas, templates iniciais, plano de custom fields vazio.
5. Inicializa contadores de quota para o período vigente.
6. Emite evento `OrganizationProvisioned`.

### Onboarding
- Wizard guia o admin através de: dados da empresa, conexão de canal, criação de primeiro workflow/agente, convite do time.
- Estado de progresso persiste — admin pode sair e voltar.
- Marca `onboarded_at` ao concluir.

### Suspensão
- Automática: inadimplência após N tentativas (webhook do provedor de pagamento → estado `suspensa`).
- Manual: master admin com justificativa.
- Efeito: login possível mas todas as ações de escrita bloqueadas; UI mostra banner claro.

### Expiração
- Quando `plan_expires_at` é excedido sem renovação.
- Read-only automático. Dados preservados. Comunicação externa suspensa (envio de mensagens bloqueado, webhooks de saída pausados).

### Reativação
- Pagamento confirmado → evento `OrganizationReactivated` → `is_active = true`, `plan_expires_at` atualizado.

### Desativação / Arquivamento
- Cliente pede cancelamento → `is_active = false`, dados preservados por período legal (default 6 meses).
- Admin é notificado por email em cada etapa.

### Hard-delete
- Apenas master admin.
- Janela de carência obrigatória (≥30 dias após desativação).
- Deleção cascata: membros, leads, conversas, mensagens, workflows, campanhas, agentes, FAQs, objetos em storage, vetores.
- Log de auditoria retido fora do tenant (não é deletado).

## Relações

- **1:N** com Time Member.
- **1:N** com Lead.
- **1:N** com Pipeline.
- **1:N** com Workflow.
- **1:N** com Campanha.
- **1:N** com Agente IA.
- **1:N** com Instância de Canal.
- **1:N** com Webhook Endpoint.
- **1:N** com Integração (Google Calendar conectado, TinyERP conectado, etc.).
- **N:1** com Plano.

## Operações

### CreateOrganization (master ou checkout)
Ver "Provisionamento" acima.

### UpdateOrganizationSettings
- Admin da org edita dados.
- Campos protegidos (plan_id, is_active) só via caminho dedicado.

### UpgradePlan / DowngradePlan
- Validar quotas atuais contra novos limites.
- Downgrade que violaria (ex.: mais agentes que o novo plano permite) exige que admin primeiro reduza.
- Efeito imediato em quotas; cobrança pro-rata a critério de negócio.

### SuspendOrganization (master)
- Marcar `is_active = false`, registrar motivo.
- Emitir `OrganizationSuspended`.

### ReactivateOrganization
- Toggle de volta + auditoria.

### DeleteOrganization (soft)
- Pedido do cliente via UI (com dupla confirmação) ou master.
- Emite `OrganizationDeactivated`.

### HardDeleteOrganization (master)
- Janela de carência obrigatória.
- Dispara job de remoção em cascata.

## Isolamento

- Ver [[01 - Arquitetura Conceitual/Multi-tenancy]].
- Toda entidade filha carrega `organization_id`.
- RLS/policies no armazenamento garantem que consulta cross-org devolve zero linhas.

## Eventos Emitidos

- `OrganizationProvisioned`
- `OrganizationUpdated`
- `OrganizationPlanChanged`
- `OrganizationSuspended`
- `OrganizationReactivated`
- `OrganizationDeactivated`
- `OrganizationOnboardingCompleted`
- `OrganizationDeleted` (hard)

## Configurações detalhadas

### Janela de Negócio
- Estrutura: array de regras `{day_of_week, start_time, end_time}` + exceções `{date, closed | custom_hours}`.
- Usada por: mensagens agendadas com "enviar só em horário comercial", nodes `wait_business_window` em workflow, follow-ups automáticos de agente.

### Custom Settings
- `welcome_message`: mensagem padrão de boas-vindas de qualquer novo lead.
- `default_agent_id`: agente IA default.
- `lead_score_weights`: pesos para engine de score (quando org customiza).
- `notification_preferences`: quais eventos geram notificação push, email, ou nada.
- `commission_default_rule`: regra de comissão default.
- `working_days`: dias úteis (para SLA).

### Feature Flags e Overrides
- Em regra, features vêm do plano. Flags são **overrides pontuais** concedidos pelo master.
- Exemplos: dar acesso a feature em beta a uma org específica, subir quota temporariamente.
- Cada override tem `granted_by` e `reason`.

## Dados derivados visíveis ao admin

- Quota consumida vs limite no período.
- Número de membros ativos.
- Data de expiração do plano.
- Histórico de pagamentos.
- Lista de integrações conectadas e status.
- Saúde operacional (mensagens enviadas/falhas, webhooks falhos, etc.).

## Segurança

- Organização não tem "senha" — acesso é sempre via usuário autenticado.
- API keys são por organização, não por usuário.
- Rotação de API key é operação visível no audit log.
- Senha de admin primeiro é gerada aleatória e enviada por email no provisionamento, forçando troca no primeiro login.
