---
tipo: feature
dominio: equipe
---

# Metas

## Propósito

Definição e acompanhamento de **objetivos quantitativos** do time (organização, grupo, individual). Alimenta dashboards, gamificação e visão gerencial. Metas são métricas + valor-alvo + período.

## Atores e Permissões

- **Admin**: define, edita, vê todas.
- **Membros**: veem próprias + do time conforme config de transparência.

Ações: `goal.view:own`, `goal.view:team`, `goal.edit`, `goal.create`.

## Dados Envolvidos

- `id`, `organization_id`.
- `name`.
- `scope`: `organization` | `team` | `member` | `specialty`.
- `scope_ref`.
- `metric`: enum de métricas suportadas (ver abaixo).
- `target_value`.
- `period_type`: `day` | `week` | `month` | `quarter` | `year` | `custom`.
- `period_start`, `period_end`.
- `unit`: `count` | `currency` | `percent`.
- `is_active`.
- `visibility`: `public` (todos veem) | `admin_only` | `self_only` (membro vê a sua).
- `reward_id` (opcional, liga a premiação).
- `created_at`.

### Progresso (calculado, cache)
- `current_value`.
- `percent_achieved`.
- `projected_value` (extrapolação por ritmo atual).
- `last_updated_at`.

## Métricas Suportadas

- `leads_created`: leads novos no período.
- `leads_qualified`: leads que avançaram a stage X.
- `meetings_scheduled`: reuniões marcadas.
- `meetings_attended`: compareceram.
- `proposals_sent`: propostas enviadas.
- `proposals_won`: fechadas positivas.
- `revenue`: soma de vendas.
- `avg_ticket`: ticket médio.
- `response_time_median`: tempo mediano de primeira resposta.
- `followups_completed_on_time`: follow-ups concluídos no prazo.
- `conversion_rate_{stage_from}_{stage_to}`: conversão entre stages.
- `customer`: `clients_won` (contagem).

## Regras de Negócio

1. Meta com `period_type=month` recomputa progresso diariamente (ou em eventos).
2. Progresso: `current_value = métrica aplicada ao scope no período`.
3. `percent_achieved = current_value / target_value` (limite 100% para exibição; sobrepor mostrado como `>100%`).
4. Projeção: ritmo atual × dias restantes + current. Heurística.
5. Metas expiradas: status `closed`. Não apagadas.
6. Reabertura: admin pode estender `period_end`.
7. Meta vinculada a reward dispara o reward em `percent_achieved >= 100%`.

## Fluxos do Usuário

### Criar Meta
1. `Metas → Nova`.
2. Form: nome, scope, métrica, target, período, visibilidade, reward opcional.
3. Preview de histórico da métrica (para calibrar target).
4. Salvar.

### Listar Metas
1. Menu → `Metas`.
2. Tabs: Minhas, Time (se admin), Org.
3. Cards: nome, progresso %, barra visual, projeção, dias restantes.
4. Cores: verde (on track), amarelo (risco), vermelho (não atinge).

### Ver Detalhe
- Gráfico de progresso diário vs meta linear.
- Breakdown por membro (se scope=team).
- Lista de eventos que contaram (ex.: vendas individuais).

### Editar
- Admin ajusta target ou período.
- Audit log.

### Fechar (manual)
- Admin pode fechar meta antes de período expirar.

## Automações e Eventos

### Emite
- `GoalCreated`, `GoalUpdated`, `GoalAchieved`, `GoalExpired`, `GoalClosed`.

### Reage
- Eventos que afetam métrica: `LeadCreated`, `MeetingScheduled`, `ProposalWon`, etc.
- Recalcula progresso (cache).
- Se cruzou 100%: emite `GoalAchieved`.

### Reward
- `GoalAchieved` + `reward_id` → dispara reward para membros elegíveis.

## Integrações

- **Analytics**: calcula métricas.
- **Premiações**: trigger.
- **Dashboard**: widgets de metas.
- **Notificações**: alertas em marcos (25%, 50%, 75%, 100%).

## Edge Cases

- **Meta sem progresso por falta de dados** (ex.: membro sem leads): mostra 0% sem erro.
- **Target zero**: aceita, sempre 100% atingida imediatamente.
- **Múltiplas metas concorrentes mesma métrica**: ambas contam independente.
- **Membro removido durante meta team**: contribuição passada é mantida; sem mais contribuição futura.
- **Período com feriados/folgas**: cálculo é calendário puro; admin pode ajustar target considerando.

## Validações

- `target_value >= 0`.
- `period_end > period_start`.
- `scope_ref` existe e pertence à org.
- `metric` válida.

## Métricas (meta-metrics)

- % metas atingidas (da org, por período).
- Tempo médio até 50% de progresso.
- Metas mais/menos atingidas (sinal de calibração).

## UX

- Barra de progresso clara com %.
- Projeção vs meta em tooltip.
- Ranking dentro de meta team.
- Notificação push ao atingir marco.
