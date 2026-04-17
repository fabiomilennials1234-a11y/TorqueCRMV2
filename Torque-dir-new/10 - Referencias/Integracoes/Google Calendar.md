---
tipo: integracao
direcao: bidirectional
criticidade: media
---

# Google Calendar

Integração com Google Calendar para sincronização de reuniões agendadas pelos closers com leads. Bidirecional: Torque cria eventos no calendário do closer; mudanças no Google refletem no Torque.

## Propósito

- Closer marca reunião no Torque → evento aparece no Calendar dele.
- Reagendamento/cancelamento no Calendar reflete no Torque.
- Lembretes automáticos via Calendar (complementares aos do Torque).
- Integração com Google Meet (link de vídeo auto-gerado).

## Contrato

- OAuth com Google.
- Uso de Calendar API v3.
- Webhook (push notifications) para receber updates.

## Autenticação

- OAuth 2.0.
- Scope: `https://www.googleapis.com/auth/calendar.events`.
- Cada closer conecta individualmente (vinculação a membro, não a org global).
- Token refresh automático.

## Conexão por Membro

1. Closer em `Meu Perfil → Integrações → Conectar Google Calendar`.
2. OAuth fluxo.
3. Callback retorna code.
4. Torque armazena tokens cifrados por membro.

## Endpoints Consumidos

### Calendar API
- `POST /calendar/v3/calendars/primary/events` — criar evento.
- `GET /calendar/v3/calendars/primary/events/:id` — detalhe.
- `PATCH /calendar/v3/calendars/primary/events/:id` — editar.
- `DELETE /calendar/v3/calendars/primary/events/:id` — cancelar.
- `POST /calendar/v3/calendars/primary/events/watch` — subscrição de updates (webhook).

### Payload criação (exemplo)
```json
{
  "summary": "Reunião comercial — Maria Silva (Indústria XYZ)",
  "description": "Lead de Meta Ads. Interessada em produto X.\nLink do CRM: https://torque.com.br/leads/{lead_id}",
  "start": {"dateTime": "2026-04-20T14:00:00-03:00"},
  "end": {"dateTime": "2026-04-20T15:00:00-03:00"},
  "attendees": [
    {"email": "maria@empresa.com.br"}
  ],
  "conferenceData": {
    "createRequest": {
      "requestId": "uuid",
      "conferenceSolutionKey": {"type": "hangoutsMeet"}
    }
  },
  "reminders": {
    "useDefault": false,
    "overrides": [{"method": "popup", "minutes": 15}]
  }
}
```

## Webhooks Recebidos (push notifications)

- Google envia push quando evento muda.
- Torque recebe e chama `GET` para obter estado atualizado.
- Dedupe por `etag`.

## Fluxos

### Criar Reunião no Torque
1. Closer marca reunião no Pipeline Confirmação (ver [[04 - Funcionalidades/Vendas/Pipeline Confirmação]]).
2. Form: data, horário, local (online/presencial), duração, criar link de Meet?.
3. Sistema cria Pipeline Entry.
4. Se integração conectada: chama Calendar API para criar evento.
5. Armazena `calendar_event_id`.
6. Evento aparece no Google Calendar do closer + convite ao lead (se email preenchido).

### Reagendamento
1. Closer reagenda no Torque (ou direto no Google).
2. Torque atualiza Pipeline Entry + chama Calendar `PATCH` (ou recebe webhook).
3. Novo convite enviado ao lead.

### Cancelamento
1. Similar — sync bidirecional.

### Google → Torque
1. Closer muda evento no Google (move horário, cancela).
2. Push notification.
3. Torque consulta evento, compara, atualiza Entry.
4. Se cancelado: move entry para stage `cancelado`.

## Regras de Negócio

1. Integração por membro (não por org) — cada closer conecta seu.
2. Evento no Google tem link para o lead no Torque.
3. Lead recebe convite email (se email preenchido) com convite do Google.
4. Sincronização bidirecional.
5. Se closer desconecta: eventos existentes permanecem no Google, mas não sincronizam mais.
6. Evento sem calendar_event_id: fallback para lembrete interno.

## Refresh de Token

- Cron diário.
- Se refresh falha: membro é notificado para reconectar.

## Edge Cases

- **Sem email do lead**: evento criado sem convidado; apenas lembrete do closer.
- **Conflito de horário**: Google avisa closer; Torque não bloqueia por default.
- **Evento deletado no Google**: webhook sincroniza; entry marcado `cancelado`.
- **Timezone diferente entre closer e lead**: mostra em ambos; evento armazena em UTC.
- **Meet indisponível** (Workspace admin bloqueou): fallback para evento sem Meet; admin escolhe link alternativo.
- **Múltiplos calendários**: por default usa `primary`; futuro permite escolher.

## Rate Limits

- Calendar API: 1M requests/dia/projeto; Torque respeita.
- Push: limite de canais ativos.

## Fallback

- Sem integração: admin marca reunião manualmente; lembretes internos do Torque suficientes.

## Segurança

- Tokens por membro em vault cifrado.
- Scope mínimo (apenas `calendar.events`).
- Nunca acessa calendário de outros membros.
- Audit de acesso.

## Observabilidade

- Log por chamada.
- Métricas: eventos criados/dia, sync failures.

## LGPD

- Dados do lead (nome, email) compartilhados com Google via evento → consentimento implícito (lead agendou reunião).
- Opção de reunião "sem convidado" para lead que não quer ser visível em calendários externos.
