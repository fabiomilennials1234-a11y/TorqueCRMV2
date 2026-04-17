---
tipo: feature
dominio: comunicacao
---

# Mensagens Agendadas

## Propósito

Permite ao membro **agendar envio de mensagem** para um lead em um momento futuro específico, sem depender de workflow complexo. Útil para: lembrete pontual, mensagem em horário do fuso do lead, envio em janela de negócio.

## Atores e Permissões

- **Membros com permissão de chat**: criam, veem, cancelam suas próprias.
- **Admin**: vê e cancela todas.

Ações: `scheduled_message.create`, `scheduled_message.view:own`, `scheduled_message.cancel:own`, `scheduled_message.cancel:any` (admin).

## Dados Envolvidos

- `id`, `organization_id`, `lead_id`, `conversation_id` (opcional; resolvido no envio).
- `channel`, `channel_instance_id` (preferido; auto se um só).
- `content_type` (text/audio/image/etc.).
- `content`, `media_url`.
- `template_id` (opcional; se veio de template, snapshot de placeholders para resolver no envio).
- `scheduled_for` (timestamp).
- `status`: `scheduled` | `sent` | `cancelled` | `failed`.
- `created_by`.
- `sent_at`, `failure_reason`.
- `respect_business_hours`: bool.
- `sent_message_id` (linka à mensagem real depois de enviar).

## Regras de Negócio

1. `scheduled_for` deve ser futuro no momento da criação.
2. Limite de mensagens agendadas por org/período (plano).
3. Se `respect_business_hours=true` e `scheduled_for` fora de janela: agenda para próxima janela válida.
4. Canal/instância ativo no momento do envio; se inativo, marca `failed` e alerta criador.
5. Lead arquivado/deletado antes do envio: cancelada automaticamente.
6. Cancelamento permitido até segundos antes (window mínima de ~30s para execução do scheduler).

## Fluxos do Usuário

### Criar
1. No chat, alternativo ao "Enviar agora" → botão "Agendar".
2. Modal: conteúdo (text/template/mídia), data/hora, fuso (default org).
3. Opção: "Respeitar janela de negócio".
4. Validação → save.
5. Mensagem aparece em lista "Agendadas" no drawer do lead e na página dedicada.

### Listar
1. Menu → `Mensagens Agendadas` (ou tab no chat).
2. Filtros: minhas / todas (admin) / próximas / canceladas / falhadas / período.
3. Lista com: lead, canal, prévia, horário, status.

### Cancelar
1. Click em agendada → botão "Cancelar".
2. Confirmação.
3. Status = `cancelled`.

### Editar
- Cancelar + recriar (mais simples que editar estado).

### Envio Automático
- Job de cron (ex.: a cada minuto) pesca mensagens com `scheduled_for <= now` e `status=scheduled`.
- Para cada, tenta enviar via canal:
  - Sucesso → status=sent, linka à mensagem criada.
  - Falha → retry N vezes; após esgotar, status=failed.
- Respeita janela de negócio ao pescar (se flag).

## Automações e Eventos

### Emite
- `ScheduledMessageCreated`, `ScheduledMessageSent`, `ScheduledMessageCancelled`, `ScheduledMessageFailed`.

### Reage
- Job de dispatcher. Ver [[07 - Processos Assíncronos/Processamento de Mensagens Outbound]].

## Integrações

- **Chat**: envio final como mensagem normal.
- **Templates**: uso opcional.
- **Janela de negócio**: ajuste automático.
- **Notificações**: criador notificado em envio/falha se configurado.

## Edge Cases

- **Horário fora de DST (daylight saving)**: armazena em UTC; exibe no fuso org.
- **Agendada para segundos no futuro**: aceito, mas scheduler pode pegar só no próximo tick (atraso de até ~1 min).
- **Conflito com outra agendada**: sem conflito — várias podem existir para mesmo lead.
- **Lead entrou em opt-out (tag `no-contact`)**: agendada é cancelada automaticamente ao detectar.
- **Template com placeholder que mudou** entre agendamento e envio: resolve no momento do envio com valor atual (não snapshot).
- **Scheduling em massa** (workflow cria 1000 agendadas): respeita rate limit de canal; pode demorar a esvaziar.

## Validações

- `scheduled_for`: futuro, ≥ 60s do presente.
- `content` ou `media_url` preenchido.
- `content` ≤ limite de canal.
- `lead_id` e canal válidos para a org.

## Métricas

- Total agendadas/enviadas/canceladas/falhadas por período.
- Lead time médio (agendamento → envio).
- Taxa de cancelamento.
- Taxa de falha por canal.
- Uso por membro/agente.

## UX

- Ícone de relógio na prévia indicando mensagem agendada.
- Countdown ou timestamp absoluto na lista.
- Notificação opcional X minutos antes do envio (para criador revisar).
