---
tipo: feature
dominio: vendas
pipeline: structural:confirmacao
---

# Pipeline Confirmação

## Propósito

Segundo pipeline estrutural. Criado quando lead é agendado (tipicamente vindo de Pipeline WhatsApp stage `agendado`). Objetivo: **confirmar presença** do lead na reunião marcada via lembretes temporais, e marcar o resultado (compareceu / não compareceu).

## Stages Default

| Ordem | Nome | Descrição |
|---|---|---|
| 1 | `reuniao_marcada` | Acabou de entrar. Reunião futura. |
| 2 | `confirmar_d5` | 5 dias antes da reunião. |
| 3 | `confirmar_d3` | 3 dias antes. |
| 4 | `confirmar_d1` | 1 dia antes (véspera). |
| 5 | `confirmar_mesmo_dia` | No dia da reunião. |
| 6 | `compareceu` | Lead compareceu (positivo final). |
| 7 | `nao_compareceu` | No-show (negativo final). |
| 8 | `reagendado` | Lead pediu nova data → volta para `reuniao_marcada`. |
| 9 | `cancelado` | Lead cancelou; não vai remarcar (final negativo). |

Admin pode customizar. As stages `confirmar_*` são **temporais**: lead desce conforme data se aproxima.

## Atores e Permissões

- **Closer**: primário. Conduz a reunião; marca resultado.
- **SDR**: pode acompanhar/fazer lembretes se configurado.
- **Admin**: tudo.
- **Copilot**: agente "followup" ou "agendador" pode operar; envia lembretes, aceita reagendamento.
- **Workflow**: automações temporais principais.

Ações: `pipeline.view:confirmacao`, `pipeline.move_entry:confirmacao`.

## Dados Envolvidos

Pipeline Entry de Confirmação carrega `meta`:

- `meeting_date` (timestamp absoluto, obrigatório).
- `meeting_duration_minutes` (opcional).
- `meeting_location` (string livre: "online — Meet", "presencial — sala 3", etc.).
- `meeting_url` (para videochamadas).
- `calendar_event_id` (se integrado com calendário externo).
- `confirmed_at_d5`, `confirmed_at_d3`, `confirmed_at_d1` (timestamps quando lead confirmou em cada tentativa).
- `reschedule_count`.
- `attendance_marked_at`.
- `attendance_marked_by`.

## Regras de Negócio

1. Entry **só existe** com `meeting_date` válido no futuro.
2. Lead **automaticamente** desce pelas stages `confirmar_*` conforme `meeting_date - now`:
   - `meeting_date - now <= 5d` → move para `confirmar_d5`.
   - `<= 3d` → `confirmar_d3`.
   - `<= 1d` → `confirmar_d1`.
   - dia da reunião (antes do horário) → `confirmar_mesmo_dia`.
3. Lembrete automático em cada transição (ver Automações).
4. Lead responde confirmando → flag `confirmed_at_*` preenchida; stage NÃO muda imediatamente (continua descendo).
5. Lead responde querendo reagendar → stage `reagendado`. Closer agenda nova data → entry atualiza `meeting_date`, volta para `reuniao_marcada` (ou stage equivalente pela proximidade).
6. Lead responde cancelando → stage `cancelado`. Entry finalizada com `cancelled`.
7. Após horário da reunião:
   - Closer marca `compareceu` → cria entry em Pipeline Propostas.
   - Closer marca `nao_compareceu` → pode disparar workflow de recuperação.
   - Se ninguém marca em 24h → sistema dispara lembrete ao closer.
8. Data futura excessiva (> 6 meses) requer confirmação extra (raro).
9. Horário dentro de janela de negócio da org (se configurada para exigir).

## Fluxos do Usuário

### Visualização
1. Kanban com colunas por stage.
2. Card mostra: nome, empresa, data da reunião (destacada), dias restantes, closer, flags de confirmação.
3. Filtro por período (esta semana, próximos 7 dias, atrasados).

### Criar Entry (manual)
1. Em drawer do lead: botão "Agendar reunião".
2. Form: data, horário, duração, local/link, observações.
3. Validação: data futura, dentro de horário comercial (warning se não).
4. Opcional: criar evento em calendário integrado (Google Calendar).
5. Entry criado em `reuniao_marcada`; lembrete automático se configurado.

### Enviar Lembrete
1. Workflow automático dispara em cada transição temporal.
2. Template: "Olá {{lead.name}}, confirmando nossa reunião em {{days}} dias. Podemos contar com você?".
3. Lead responde → confirmação registrada.

### Confirmar/Rejeitar/Reagendar
1. Copilot ou SDR/Closer processa resposta.
2. Confirma → flag atualizada.
3. Reagenda → novo form de data, entry atualizada.
4. Cancela → move para `cancelado`, registra motivo opcional.

### Marcar Compareceu / Não Compareceu
1. Após `meeting_date`, card aparece com botão "Marcar resultado".
2. Closer escolhe: Compareceu / Não compareceu / Cancelou de última hora.
3. Compareceu → cria entry em Propostas, move entry para `compareceu` final.
4. Não compareceu → move para `nao_compareceu`, dispara workflow opcional de reagendamento.

## Automações e Eventos

### Workflows temporais built-in
- Job de cron a cada 15 min (ou similar) escaneia entries ativos.
- Para cada entry em `reuniao_marcada`/`confirmar_*`:
  - Se cruzou threshold de D-5/D-3/D-1: mover stage + enviar lembrete (respeitando janela de negócio).
  - Se `meeting_date` passou e ainda não marcou presença: alerta ao closer.

### Eventos
- `MeetingScheduled`, `MeetingRescheduled`, `MeetingCancelled`, `MeetingConfirmed`, `MeetingReminderSent`.
- `LeadStageChanged(confirmacao, ...)`.
- `MeetingAttendanceMarked(attended)`.

### Reagem
- `MeetingAttendanceMarked(attended=true)` → cria entry em Propostas.
- `MeetingAttendanceMarked(attended=false)` → pode disparar campanha de recuperação.

## Integrações

- **Google Calendar** (ou calendário equivalente): criação/atualização de evento em calendário do closer.
- **Copilot**: agente "agendador" ou "followup" opera.
- **Pipeline WhatsApp**: origem da entrada.
- **Pipeline Propostas**: destino em `compareceu`.
- **Workflow**: triggers temporais e de stage.
- **Chat**: lembretes são mensagens enviadas.

## Edge Cases

- **Reunião fora do horário comercial**: aceita mas dispara warning.
- **Reschedule para data já passada**: rejeitado.
- **Lead cancela repetidamente**: após N reschedules (configurável), move para `cancelado` automaticamente e desativa lembretes.
- **Evento no Google Calendar deletado externamente**: webhook sincroniza e marca entry como `cancelado`.
- **Múltiplas reuniões para mesmo lead**: entries separadas no pipe (permitido).
- **Lead não confirmou nem respondeu lembrete D-1**: closer é notificado (risco de no-show).
- **Fuso horário**: armazena em UTC; exibe no fuso da organização.

## Validações

- `meeting_date` futura.
- `meeting_date` dentro de período razoável (ex.: até 1 ano à frente).
- Se `meeting_url`: URL válida.
- Reschedule: nova data ≠ antiga + >= 15 min à frente.

## Métricas

- Taxa de comparecimento: `compareceu / (compareceu + nao_compareceu + cancelado)`.
- Taxa de confirmação em D-1: `confirmed_at_d1 is not null / total`.
- Reschedule rate: `reschedule_count > 0 / total`.
- Tempo entre `reuniao_marcada` e `compareceu`: lead time do agendamento.
- Por closer: comparecimento, cancelamentos, reschedules.
- Correlação entre confirmação D-1 e comparecimento efetivo.
