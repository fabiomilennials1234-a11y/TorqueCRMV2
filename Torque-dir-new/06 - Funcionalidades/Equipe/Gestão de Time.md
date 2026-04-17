---
tipo: feature
dominio: equipe
---

# Gestão de Time

## Propósito

CRUD e administração dos membros da organização: convite, ativação, edição de dados, alteração de papel e especialização, remoção. Base para distribuição de leads, atribuição de follow-ups, comissão, metas.

## Atores e Permissões

- **Admin**: CRUD completo.
- **Membros**: editam próprio perfil.
- **Master**: via impersonação.

Ações: `team.view`, `team.invite`, `team.update:other`, `team.remove`, `team.change_role`, `team.change_specialty`, `team.set_permissions`, `profile.edit:self`.

## Dados Envolvidos

Ver [[02 - Modelo de Domínio/Time de Vendas]] para entidade completa. Resumo:

- Identificação: nome, email, avatar, phone.
- Vínculo: organization_id, user_id.
- Papel: role (admin / membro), specialty (sdr / closer / prospectador / admin / outro).
- Estado: is_active.
- Configurações: commission_config, goal_config, working_hours (opcional override).
- Permissões: member_feature_permissions (overrides).
- Metadata: hired_at, last_login_at, bio.

## Fluxos do Usuário

### Listar Time
1. Admin acessa `Equipe`.
2. Tabela: avatar, nome, email, role, specialty, ativo/inativo, último login, leads ativos (em entries de pipe), vendas do mês, performance rank.
3. Filtros: specialty, role, ativo/inativo, busca.
4. Ordenação por qualquer coluna.

### Convidar Membro
1. Admin clica "Convidar".
2. Form: email, nome (opcional — pode ser preenchido no aceite), role, specialty, permissões (presets ou customizar).
3. Validações: email não-cadastrado na org, quota de usuários do plano OK.
4. Envia email com link de aceite (token curto, 7 dias).
5. Member criado em status `pending`.

### Aceitar Convite (novo usuário)
1. Usuário clica link no email.
2. Página de cadastro: define senha, confirma nome.
3. Aceita ToS.
4. `is_active=true`, membro ativo.
5. Recebe onboarding em-app.

### Aceitar Convite (usuário existente em outra org)
1. Clica link.
2. Login (se não logado).
3. Confirma aceite na nova org.
4. Membro ativo em ambas (user pode trocar contexto).

### Editar Dados
- Admin: pode editar qualquer campo exceto senha.
- Self: pode editar nome, avatar, phone, bio, password.

### Alterar Role
- Admin promove/rebaixa.
- Bloqueado se seria último admin ativo rebaixado.
- Audit crítico.
- Email de notificação.

### Alterar Specialty
- Admin escolhe nova specialty.
- Perguntar se aplicar preset de permissão da nova specialty (sobrescreve custom).

### Set Permissões Custom
- Para membro, admin pode ajustar ações granulares (grants/denies).
- Interface lista permissões agrupadas por feature.
- Checkbox 3-estados: herda default / concede / nega.

### Suspender / Reativar
- Toggle is_active.
- Suspenso: não pode logar. Leads atribuídos ficam congelados (admin redistribui).
- Reativar: volta ao estado anterior.

### Remover
- Admin clica "Remover".
- Confirmação com impacto listado (X leads atribuídos, Y follow-ups pendentes, Z conversas atribuídas).
- Escolha de redistribuição:
  - Manter órfão (admin redistribui manualmente depois).
  - Atribuir ao admin.
  - Round-robin entre ativos.
- Soft-delete; hard-delete apenas por master.

## Regras de Negócio

1. Email único dentro da org.
2. Admin ativo mínimo: 1 em todo momento.
3. Self-rebaixamento: proibido se único admin.
4. Specialty não afeta role; só afeta presets/UI.
5. Removido: leads redistribuídos conforme política.
6. Suspenso: leads não redistribuídos (aguarda decisão).
7. Quota de usuários por plano enforced.
8. Member pode pertencer a múltiplas orgs (mesmo user_id global).
9. Profile edits por self respeitam mínimos de integridade (não pode remover nome todo).

## Automações e Eventos

### Emite
- `MemberInvited`, `MemberActivated`, `MemberRoleChanged`, `MemberSpecialtyChanged`, `MemberSuspended`, `MemberReactivated`, `MemberRemoved`.
- `PermissionGranted`, `PermissionRevoked`.

### Reage
- `MemberActivated` → onboarding em-app (welcome tour).
- `MemberRemoved` → redistribui leads, cancela follow-ups, arquiva conversas atribuídas.

## Integrações

- **Autenticação**: login via email+senha.
- **Permissões**: role+specialty+overrides.
- **Lead**: responsible/sdr/closer referências.
- **Follow-up**: assigned_to.
- **Conversa**: assigned_to.
- **Comissão**: calculado por member.
- **Metas**: progresso per-member.
- **Distribuição**: leads novos entram conforme members ativos.
- **Audit**: toda mudança registrada.
- **Email**: invite, mudança de role, reset senha.

## Edge Cases

- **Email típico do domínio da empresa**: sugerir configuração de SSO futuro (não implementado).
- **Usuário deletou conta global**: membros órfãos — admin deve remover.
- **Convite enviado a email já cadastrado em org diferente**: aceite apenas adiciona vínculo; não cria nova conta.
- **Convite expirou (> 7 dias)**: admin clica "Reenviar convite" → novo token.
- **Admin removido enquanto logado**: sessão invalidada; vê tela de "você foi removido desta org".
- **Mudança de email**: exige verificação por código enviado ao novo email.
- **Suspender admin ≠ rebaixar**: admin suspenso ainda conta como admin na contagem. Admin suspenso removido requer que haja outro admin ativo.

## Validações

- Nome: 2-100 chars.
- Email: formato válido, domínio válido.
- Phone: E.164 se preenchido.
- Bio: ≤ 500 chars.
- Avatar: ≤ 2MB, JPG/PNG.
- Role/specialty: enum válido.
- Commission config: schema válido (ver [[Comissões]]).

## Métricas

- Total membros ativos por specialty.
- Taxa de convites aceitos (aceites / enviados).
- Tempo médio até aceite.
- Rotatividade (remoções / tempo).
- Último login distribution (quantos dias desde último acesso — inatividade).

## UX

- Card do membro mostra specialty badge com cor distinta.
- Quick actions: suspender, redistribuir, editar permissões.
- Tooltip com última atividade.
- Filtro rápido "Inativos 30+ dias" para limpeza.
