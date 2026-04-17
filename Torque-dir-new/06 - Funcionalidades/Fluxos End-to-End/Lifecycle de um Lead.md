---
tipo: fluxo
---

# Lifecycle de um Lead (End-to-End)

Jornada completa de um lead desde entrada até desfecho. Atravessa múltiplas features — este doc é a "pauta" que orienta como elas se combinam.

## Visão Macro

```
  ENTRADA           QUALIFICAÇÃO         CONFIRMAÇÃO          NEGOCIAÇÃO         DESFECHO
     │                    │                    │                     │                  │
     ▼                    ▼                    ▼                     ▼                  ▼
  Fonte           Pipe WhatsApp        Pipe Confirmação       Pipe Propostas      Vendido /
  externa         novo→abordado→        reuniao_marcada        proposta_enviada    Perdido /
  (n8n,           respondeu→            →D-5→D-3→D-1→          →negociando→         Descartado
  form, ads,      agendado              compareceu              vendido/perdido
  manual)           │                    │                     │
     │              ▼                    ▼                     ▼
     │         Agente IA             Lembretes                Comissão
     │         conduz                automáticos              calculada
     │         conversa              (workflow)
     ▼
  Lead criado
  + Pipeline Entry default
  + Evento LeadCreated
```

## Etapa 1: Entrada

### Fontes
- Webhook de ingestão (n8n / orquestrador).
- Formulário web público.
- Meta Ads Lead Ads (via webhook Meta).
- Mensagem inbound de número desconhecido (cria lead + conversa).
- Cadastro manual (admin/SDR no UI).
- API direta (integrador que usa API key).
- Upload CSV (ad-hoc).

### Ações do sistema
1. Valida input.
2. Dedupe (external_id, phone, email).
3. Normaliza phone e email.
4. Cria `Lead` entity.
5. Aplica tags iniciais.
6. Cria `Pipeline Entry` default em Pipe WhatsApp stage `novo`.
7. Emite `LeadCreated`.
8. Event consumers:
   - Distribuição atribui `sdr_id` (se configurada).
   - Calculadora de score computa `qualification_score` inicial.
   - Workflows subscritos a `lead_created` iniciam execução.
   - Notificações ao responsável.
   - Webhook de saída ao endpoint da org configurado.

### Decisão de distribuição
- Se `assigned_user_id` veio no payload: atribui direto.
- Se não e há regra de distribuição: aplica.
- Se ninguém online e regra permite: lead fica `unassigned`.

## Etapa 2: Qualificação

### Abordagem
- SDR (ou agente IA) vê lead em `novo`.
- Envia mensagem (template ou livre).
- Transição `novo → abordado` (auto ou manual).

### Batch + Agente
- Se agente IA ativo: lead responde → batch 8s → agente responde.
- Agente segue Kanban Rule da stage (objetivo: qualificar).
- Pode: adicionar tags, pedir dados, agendar reunião.

### Escalamento
- Humano assume (takeover) a qualquer momento — agente pausa.
- Agente retoma após 10 min sem nova mensagem humana.

### Esfriamento
- Lead para de responder > 48h → SLA move para `esfriou`.
- Workflow de retenção pode disparar (mensagem automática, áudio, tentativa de reativação).

### Agendamento
- SDR/agente confirma data de reunião.
- Transição `respondeu → agendado`.
- Regra de pipe cria entry em Pipeline Confirmação.
- Lead é removido do board ativo WhatsApp (filter padrão).

## Etapa 3: Confirmação

### Entrada
- Entry criado em Confirmação stage `reuniao_marcada`.
- `meeting_date` armazenada.
- Se integrado: evento no Google Calendar do closer + convite ao lead.

### Descida temporal
- `meeting_date - 5d` → move para `confirmar_d5` + envia lembrete.
- Similar D-3, D-1, dia.

### Interação do lead
- Confirma → flag preenchida, segue descendo.
- Reagenda → nova data, entry atualizada, volta a `reuniao_marcada`.
- Cancela → move para `cancelado`, entry finalizada.

### Comparecimento
- Dia da reunião passa.
- Closer marca resultado:
  - `compareceu` → cria entry em Propostas.
  - `nao_compareceu` → workflow de recuperação.
  - `cancelado_última_hora` → tag específica.

## Etapa 4: Negociação (Propostas)

### Entrada
- Entry em Pipeline Propostas stage `preparando_proposta`.

### Montagem
- Closer adiciona items (produtos + qty + preço + desconto).
- Sistema calcula subtotal, total.
- Define validade, condições de pagamento.

### Envio
- Closer envia ao lead (mensagem + link/PDF).
- Transição `proposta_enviada`.

### Negociação
- Lead responde → ajustes.
- Transições entre `negociando` e `proposta_enviada` conforme versões.

### Fechamento
- `vendido`:
  - `meta.total`, items obrigatórios.
  - Emite `LeadSold`.
  - Cálculo de comissão (SDR split + Closer split).
  - Workflow de pós-venda: tag cliente, notificação, entry em Onboarding Cliente, criação de pedido no ERP.
- `perdido`:
  - `lost_reason` obrigatório.
  - Workflow de reciclagem (follow-up em 6 meses, campanha de reativação).

## Etapa 5: Pós-venda (opcional)

- Entry em pipe "Onboarding Cliente" (custom).
- Upsell: ao cruzar condições, oportunidade gerada (ver [[04 - Funcionalidades/Vendas/Upsell]]).
- Renovação: antes de plano expirar (se assinatura), sequência de renovação.

## Estado Paralelo

Lead pode estar:
- Em múltiplos pipes customizados simultaneamente.
- Em campanha (outbound) ativa enquanto também em pipe.
- Com múltiplas conversas abertas (canais diferentes).

## Métricas do Lifecycle

- **Lead time total**: criação → vendido.
- **Tempo por etapa**: entrada → qualif → reunião → proposta → fechamento.
- **Conversão por etapa**: taxa de passagem.
- **Taxa de fechamento global**: vendidos / leads criados.
- **Origem → conversão**: qual canal qualifica mais.

## Eventos Emitidos ao longo

Lista não-exaustiva:

`LeadCreated` → `LeadAssigned` → `LeadTagAdded` → `LeadScoreRecalculated` → `ConversationStarted` → `MessageReceived` → `MessageSent` → `LeadStageChanged(whatsapp, novo→abordado)` → ... → `MeetingScheduled` → `MeetingConfirmed` → `MeetingAttendanceMarked(attended=true)` → `ProposalCreated` → `ProposalSent` → `ProposalWon` / `LeadSold` → `CommissionCalculated` → [opcional `UpsellOpportunityCreated` meses depois].

## Edge Cases do Lifecycle

- **Lead que pula etapas**: acontece (SDR senior fecha direto sem reunião). Sistema permite mas tracking de etapas pode ser incompleto — analytics mostra.
- **Lead que volta à qualificação após perder**: cria novo entry, mantém histórico.
- **Lead com múltiplas conversas** (WhatsApp + Messenger): tratado como um só lead, conversas separadas.
- **Lead migrado de sistema anterior**: import via CSV + backdated `created_at`.
- **Merge de leads** (admin descobre que são a mesma pessoa): feature rara, exige confirmação + audit reforçado.

## Instrumentação

- Cada transição crítica emite evento.
- Audit log preserva sequência.
- Dashboard mostra funil visual.
- Oráculo Comercial responde perguntas sobre lifecycle ("quanto tempo demora em média?").
