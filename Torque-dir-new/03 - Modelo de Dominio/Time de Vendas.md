---
tipo: dominio
entidade: Time Member
---

# Time de Vendas (Time Member)

Usuário pertencente a uma organização, com papel (role) e especialização funcional.

## Atributos Conceituais

### Identificação
- `id`: UUID do membro (distinto do `user_id` global).
- `organization_id`: tenant.
- `user_id`: referência ao usuário global (entidade separada em auth store, possivelmente compartilhada entre orgs).
- `name`: nome exibido.
- `email`: email (deve bater com `user_id` em regra normal).
- `avatar_url`: foto/avatar.

### Papel e Especialização
- `role`: `admin` | `membro`. No código, apenas essas duas variações. Master é referência separada, fora da entidade `team_member`.
- `specialty`: `sdr` | `closer` | `prospectador` | `admin` | `outro`. **Não é role de permissão** — é classificação funcional.
- `is_active`: se está operando no momento.

### Permissões
- Derivadas de `role` + `specialty` + overrides de `member_feature_permissions` (tabela ou estrutura equivalente).
- Engine de permissão consulta todas essas fontes.

### Comissão
- `commission_config`: JSON com regra de comissão específica deste membro (pode sobrescrever default da org).
  - Exemplo: `{base_percent: 10, per_product: {"sku-123": 15, "sku-999": 8}}`.

### Metas
- `goal_config`: JSON com metas individuais definidas pelo admin (opcional; default vem da org).

### Metadados
- `hired_at`: data de entrada no time.
- `last_login_at`: último acesso.
- `phone`: telefone (contato interno, não aparece para leads).
- `bio`: descrição curta (opcional, exibida em perfil público do membro se a org ativar).

## Invariantes

1. `(organization_id, user_id)` único.
2. `(organization_id, email)` único.
3. Toda organização precisa ter ao menos **um membro ativo com role=admin**.
4. `role` só pode ser alterado por outro admin (nunca self-downgrade que deixe a org sem admin).
5. Ao desativar o último admin ativo → operação rejeitada.
6. `specialty` é livre para o admin mudar sem afetar permissões.

## Especializações

| Specialty | Responsabilidades típicas | Features principais |
|---|---|---|
| `admin` | Administra config da org, time, workflows, metas. | Tudo. |
| `sdr` | Qualifica leads novos, primeira abordagem, agendamento. | Pipeline WhatsApp, Chat, Templates, Follow-ups. |
| `closer` | Conduz reunião, prepara proposta, fecha venda. | Pipeline Confirmação/Propostas, Produtos, Calendário. |
| `prospectador` | Outbound ativo, campanhas. | Campanhas, Chat, Dashboard Outbound. |
| `outro` | Genérico — atendente, marketing, financeiro. | Configurável pelo admin via permissões. |

A UI pode oferecer **presets de permissão** baseados em specialty (SDR preset tem Chat + Pipeline WhatsApp). Admin pode customizar.

## Ciclo de Vida

```
convidado (invite pending) → ativo → (suspenso) → inativo → removido (soft delete)
```

### Convite
1. Admin preenche email + nome + role + specialty.
2. Sistema cria `team_member` em estado `pending`, associa a `user_id` global se já existe, ou cria usuário global se novo.
3. Envia email com link de aceitação (token curto).
4. Aceitação: set `is_active=true`, confirma email.
5. Se usuário já existia em outra org, aceitação só associa — não reenvia senha.

### Ativação/Desativação
- `is_active=false` impede login (ou login OK mas acesso a esta org bloqueado se user está em outras).
- Reativação é toggle simples.

### Remoção
- Soft-delete por default (registros de histórico preservam membro).
- Hard-delete master apenas.
- Ao remover, leads atribuídos ficam sem responsável (admin é notificado) ou são redistribuídos conforme regra.

## Operações

### InviteMember (admin)
1. Validar permissão.
2. Validar email não é de membro existente na org.
3. Validar quota de usuários do plano.
4. Criar member em `pending` + enviar invite.
5. Emitir `MemberInvited`.

### AcceptInvite
1. Validar token.
2. Se usuário global não existe, criar.
3. Set password (se novo).
4. Ativar member.
5. Emitir `MemberActivated`.

### UpdateMember (admin)
- Admin pode mudar name, role (respeitando regras), specialty, avatar, permissões override.
- Self-update: member pode editar próprio nome, avatar, password, telefone.

### ChangeRole (admin → admin)
- Garantir que não fique sem admin.
- Auditoria reforçada.
- Emitir `MemberRoleChanged`.

### SuspendMember / ReactivateMember (admin)
- Toggle `is_active`.

### RemoveMember (admin)
- Soft-delete.
- Redistribuir leads atribuídos (regra configurável: assigned-to-admin, round-robin para ativos, manter órfãos).
- Emitir `MemberRemoved`.

## Eventos Emitidos

- `MemberInvited`
- `MemberActivated`
- `MemberRoleChanged`
- `MemberSpecialtyChanged`
- `MemberSuspended`
- `MemberReactivated`
- `MemberRemoved`

## Relações

- **N:1** com Organization.
- **1:N** como responsável/SDR/closer em Lead.
- **1:N** com Follow-up atribuído.
- **1:N** com Conversa (quando member toma takeover).
- **1:N** com Commission Entry (histórico de comissões).
- **1:N** com Goal Progress (progresso de metas).

## Visão do Leads → Membro

- Lead tem três referências a membro: `responsible_id`, `sdr_id`, `closer_id`.
- Em operações simples, os três apontam para o mesmo.
- Em operações separadas: SDR qualifica, passa para closer; responsible é supervisor.
- Distribuição automática configura regra por specialty (ex.: `sdr_id` por round-robin entre SDRs ativos).

## Comissão

- Ver [[04 - Funcionalidades/Equipe/Comissões]].
- `commission_config` do membro sobrescreve regra da org.
- Calculada quando lead vai para stage final positivo em Propostas.
- Cada evento de comissão gera `Commission Entry` ligado ao membro.

## Metas

- Ver [[04 - Funcionalidades/Equipe/Metas]].
- Metas podem ser da org (divididas) ou individuais.
- Progresso é calculado em tempo quase real por agregação.

## Permissões Default por Role/Specialty

**Admin** (role=admin):
- Todas as ações possíveis na organização.

**Membro com specialty=sdr** (default):
- Ver leads atribuídos a si + leads sem responsável.
- Criar lead.
- Mover em pipeline WhatsApp.
- Responder chat.
- Criar follow-up.
- Usar templates.
- Não ver: Configurações, Workflow Builder, Master Admin, Analytics gerais (só próprio).

**Membro com specialty=closer** (default):
- Ver leads atribuídos a si.
- Mover em pipeline Confirmação e Propostas.
- Editar proposta (produtos, valor).
- Marcar venda.
- Acessar calendário integrado.

**Membro com specialty=prospectador** (default):
- Acessar e gerenciar campanhas.
- Ver leads de campanhas ativas.
- Enviar mensagens outbound.

**Membro com specialty=outro**:
- Permissões mínimas (ver próprios leads, editar próprio perfil).
- Admin customiza o que mais for necessário.

Todos os defaults são **overrides-capable** pelo admin via `member_feature_permissions`.

## Métricas por membro

- Leads trabalhados no período.
- Reuniões agendadas.
- Reuniões realizadas.
- Vendas fechadas.
- Valor total vendido.
- Taxa de conversão pessoal.
- Tempo médio de primeira resposta.
- Follow-ups concluídos vs atrasados.
