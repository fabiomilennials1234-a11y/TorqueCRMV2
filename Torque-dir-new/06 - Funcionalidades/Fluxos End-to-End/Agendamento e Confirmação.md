---
tipo: fluxo
---

# Agendamento e Confirmação

Fluxo desde o aceite do lead em ter reunião até o comparecimento (ou não). Crítico para reduzir no-show.

## Diagrama

```
Lead aceita reunião
       │
       ▼
[Form de Agendamento]
    - data, hora, duração, local/link
       │
       ▼
Pipeline Entry em Confirmação
  Stage: "reuniao_marcada"
       │
       ▼
[Integração Google Calendar: evento + convite email]
       │
       ▼
Lembretes temporais (workflow + regras de pipe):
  ┌── D-5 ──┬── D-3 ──┬── D-1 ──┬── Dia D ──┐
  ▼          ▼         ▼         ▼           ▼
 confirmar_d5 d3       d1     mesmo_dia    reunião
  │           │        │                    │
  ▼ lead     ▼         ▼                    │
 confirma    confirma  confirma?            │
                                            │
                      Reschedule ──────────►│
                                            │
                      Cancela ────────────►cancelado
                                            │
                                            ▼
                          Após horário: closer marca resultado
                          ┌─────────────┬─────────────┐
                          ▼             ▼             ▼
                      compareceu    nao_compareceu  cancelou última
                          │             │             hora
                          ▼             ▼             │
                 Cria entry       Workflow          ▼
                 em Propostas    recuperação     Tag específica
                                  (novo agenda)
```

## Passo a Passo

### 1. Aceite do Lead
Agente/SDR confirma horário na conversa:
- "Tenho disponibilidade terça 14h. Funciona para você?".
- Lead aceita.

### 2. Form de Agendamento
- Closer/SDR abre no Torque → form.
- Campos: data, hora, duração (default 60 min), local (online / presencial).
- Online: cria link Meet automaticamente (se integração Google ativa).
- Presencial: descrição do local.

### 3. Criação da Entry
- Pipeline Entry em Confirmação stage `reuniao_marcada`.
- `meta.meeting_date`.
- `calendar_event_id` (se integrado).

### 4. Integração Google Calendar
- Evento criado no calendário do closer.
- Convite email ao lead (se email preenchido).
- Lembrete no calendário.

### 5. Transições Temporais
Job cron (15 min) escaneia entries em `reuniao_marcada`/`confirmar_*`:

- `meeting_date - now <= 5d` → move para `confirmar_d5`.
  - Workflow envia lembrete: "Nossa reunião está marcada para {{meeting_date}}. Podemos contar com você?"
- `<= 3d` → `confirmar_d3` + lembrete.
- `<= 1d` → `confirmar_d1` + lembrete.
- Dia → `confirmar_mesmo_dia` + lembrete final (ex.: 2h antes).

### 6. Confirmação do Lead
- Lead responde "Confirmado" → flag `confirmed_at_*` preenchida.
- Lead responde pedindo reagendar → reagendamento (novo form; entry volta a `reuniao_marcada`).
- Lead responde cancelando → stage `cancelado`, entry finalizada.
- Lead não responde → stage continua descendo; comparecimento incerto.

### 7. Dia da Reunião
- Notificação ao closer 15 min antes.
- Meeting link disponível.

### 8. Pós-Reunião
- Closer marca resultado:
  - **Compareceu**: stage `compareceu`, regra cria entry em Propostas stage `preparando_proposta`.
  - **Não compareceu**: stage `nao_compareceu`, workflow:
    - Mensagem automática: "Sentimos sua falta, posso reagendar?".
    - Follow-up criado para closer em 2 dias.
    - Se sem resposta em 7d: tag `noshow`, campanha específica.
  - **Cancelou última hora**: stage específica; possivelmente reagendar.

## Atores

| Ator | Papel |
|---|---|
| Lead | Confirma/reagenda/cancela/comparece. |
| Closer | Conduz reunião, marca resultado. |
| SDR | Agendou inicialmente; recebe visibilidade. |
| Agente IA | Pode automatizar lembretes, aceitar reagendamento, pedir confirmação. |
| Sistema | Transições temporais, integração calendário, notificações. |

## Métricas-chave

- **Show rate (taxa de comparecimento)**: `compareceu / (compareceu + nao_compareceu + cancelou_ultima_hora)`.
- **Taxa de confirmação D-1**: `confirmed_at_d1 not null / total`.
- **Reschedule rate**: `reschedule_count > 0 / total`.
- **No-show por origem**: qual canal tem mais no-show → investigar qualidade do lead.
- **Correlação confirmação D-1 × comparecimento**.

## Regras de Negócio Relevantes

- `meeting_date` futuro obrigatório.
- Reunião fora de horário comercial: aceita mas warning.
- Múltiplas reschedules > N: move para `cancelado` automaticamente (configurável).
- Evento Google deletado externamente → webhook cancela entry.

## Eventos Emitidos

- `MeetingScheduled`.
- `MeetingReminderSent(d5/d3/d1/same_day)`.
- `MeetingConfirmed`.
- `MeetingRescheduled`.
- `MeetingCancelled`.
- `MeetingAttendanceMarked(attended=true/false)`.
- `LeadEnteredPipe(propostas)` (se compareceu).

## Pontos Críticos

- **Lembretes não enviados**: quebra o fluxo. Monitorar execução do workflow.
- **Link Meet quebrado**: teste no Playground da feature.
- **Timezone confusão**: lead e closer em zonas diferentes — UI sempre mostra em ambos.
- **Closer esquece de marcar resultado**: alerta automático 24h após meeting_date.

## Otimizações Possíveis

- **Áudio humanizado em D-1**: agente envia áudio "Oi Maria, lembrando da nossa conversa amanhã...".
- **SMS backup**: além de WhatsApp, SMS D-1 (se lead optou).
- **Confirmação 1-tap**: link com botão "Confirmar" que marca no Torque automaticamente.
- **Double opt-in**: lead confirma no D-3 e D-1 para maximizar compromisso.

## Pitfalls

- **Sem lembretes**: show rate baixa (esperado).
- **Lembretes excessivos** (hora em hora): lead se irrita; cancela.
- **Falta de confirmação final em D-1**: no-show inesperado → pode ser mitigado.
- **Closer não vê D-0**: painel dedicado "Minha agenda de hoje" ajuda.

## Integração com Outras Features

- **Pipeline Confirmação**: central.
- **Workflow**: lembretes, recuperação de no-show.
- **Google Calendar**: sync bidirecional.
- **Chat**: mensagens de lembrete e confirmação.
- **Copilot**: pode automatizar tudo, inclusive receber "sim/não" e mover cards.
- **Analytics**: show rate, taxa de confirmação.
