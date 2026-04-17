---
tipo: feature
dominio: vendas
---

# Follow-ups

## Propósito

Tarefa de acompanhamento atribuída a um membro com prazo e descrição. Garante que nenhum lead seja esquecido — lembrete proativo para o time retomar contato. Pode ser criado manualmente, por workflow, por agente IA, ou por regra de pipe.

## Atores e Permissões

- **Membros**: criam, concluem próprios. Veem follow-ups atribuídos.
- **Admin**: vê todos, reatribui, cria para outros.
- **Copilot**: pode criar via AI Action.
- **Workflow**: pode criar via action `create_followup`.

Ações: `followup.view:own`, `followup.view:team`, `followup.create`, `followup.complete`, `followup.delete`.

## Dados Envolvidos

- `id`, `organization_id`, `lead_id`.
- `assigned_to`: membro responsável.
- `title`: resumo (ex.: "Ligar para confirmar orçamento").
- `description`: detalhe opcional.
- `due_at`: timestamp do prazo.
- `priority`: `low` | `normal` | `high` | `urgent`.
- `status`: `pending` | `done` | `missed` | `cancelled`.
- `completed_at`, `completed_by`.
- `missed_reason` (opcional).
- `origin`: `manual` | `workflow` | `agent` | `rule` | `system`.
- `context`: JSON (ex.: workflow_execution_id se veio de workflow).

## Estados e Transições

```
pending ─┬─► done (usuário conclui)
         ├─► missed (prazo passou sem conclusão → job automático)
         └─► cancelled (admin ou workflow cancela)

missed pode voltar a pending (reabrir).
done não volta.
```

## Regras de Negócio

1. Criar follow-up exige lead e assigned_to válidos pertencentes à mesma org.
2. `due_at` futuro (pode ser passado em casos de registro retroativo; warning).
3. Follow-up em status `pending` com `due_at` passado é `overdue` na UI (destacado) — job pode marcar `missed` conforme política.
4. Conclusão exige nota opcional (resultado da ação).
5. Múltiplos follow-ups por lead são permitidos.
6. Follow-up atribuído a membro removido: cai como órfão; admin reatribui.

## Fluxos do Usuário

### Criar Manual
1. Drawer do lead → aba Follow-ups → "+ Novo".
2. Form: title, descrição, data/hora, prioridade, atribuído a (self default, admin pode escolher outro).
3. Salvar → aparece em lista.

### Ver Pendentes (página dedicada)
1. Menu → `Follow-ups`.
2. Filtros: meus / todos (admin) / vencidos / hoje / próximos 7d / urgentes.
3. Lista com lead, título, prazo, prioridade.
4. Clique → vai para lead.

### Concluir
1. Na lista ou no drawer do lead.
2. Botão "Concluir".
3. Dialog opcional: "Qual foi o resultado?" (texto livre, para audit).
4. Status = `done`, `completed_at` registrado.

### Marcar Perdido / Reabrir
- Admin pode marcar `missed` explicitamente (com reason) ou sistema faz automaticamente.
- Reabrir: edit com novo `due_at`, volta a `pending`.

### Automação (criação)
- Workflow cria via action `create_followup(title, due_in_hours, assigned_to)`.
- Agente IA cria via AI Action.
- Regra de pipe pode criar ao entrar em stage específica (ex.: "Ao entrar em `proposta_enviada`, criar FUp para daqui a 48h").

## Automações e Eventos

### Emite
- `FollowupCreated`, `FollowupCompleted`, `FollowupMissed`, `FollowupCancelled`, `FollowupReassigned`.

### Reage
- `LeadStageChanged`: workflow pode criar follow-up.
- `MessageReceived` inbound: se follow-up do lead está pending, considerar concluído (opcional).
- Cron de SLA: marca missed em follow-ups vencidos.

## Integrações

- **Leads**: todo follow-up pertence a um lead.
- **Workflow**: action `create_followup`.
- **Copilot**: action via agente.
- **Regras de Pipe**: criação automática em transições.
- **Notificações**: push/email ao atribuído quando criado; lembrete próximo do prazo.
- **Analytics**: métricas de produtividade (follow-ups no prazo).

## Edge Cases

- **Lead deletado** (soft): follow-ups pendentes ficam visíveis mas com lead marcado como deletado. Ao concluir, pede confirmação.
- **Membro removido com follow-ups**: admin é alertado, redistribuição manual ou regra automática.
- **Follow-up no passado criado retroativamente**: aceita, warning, registra.
- **Concluir após `missed`**: volta status para `done` + nota de "recuperado".
- **Criação em massa** via workflow: respeitar rate limit.

## Validações

- `title`: 3-200 chars.
- `description`: até 2000 chars.
- `due_at`: aceita qualquer, mas UI alerta se muito no passado/futuro.
- `priority`: enum válido.
- `assigned_to`: membro ativo da mesma org.

## Métricas

- **Taxa de conclusão no prazo**: `done com completed_at <= due_at / total`.
- **Taxa de atraso**: `done com completed_at > due_at / done`.
- **Taxa de perda**: `missed / total`.
- **Backlog**: pendentes com `due_at` passado.
- **Por membro**: produtividade (criados, concluídos, no prazo).
- **Por origem**: quais canais criam mais follow-ups (pipe rules, workflows, manual).
- **Impacto no funnel**: leads com follow-up concluído em dia têm maior conversão?

## Notificações

- Atribuição → push/email ao membro.
- X horas antes do prazo (configurável) → lembrete.
- Ao vencer → alerta ao membro + opcional admin.
- Diário: membro recebe resumo "Hoje você tem N follow-ups".

## UX

- Badge no menu com contagem de follow-ups pendentes/vencidos.
- Widget no dashboard: próximos follow-ups.
- Notificação sonora opcional.
