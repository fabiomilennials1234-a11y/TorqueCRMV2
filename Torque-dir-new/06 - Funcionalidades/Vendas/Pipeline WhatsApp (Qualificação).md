---
tipo: feature
dominio: vendas
pipeline: structural:whatsapp
---

# Pipeline WhatsApp (Qualificação)

## Propósito

Primeiro pipeline estrutural do Torque. Entry point de praticamente todo lead no sistema. Objetivo: **qualificar** o lead — entender necessidade, orçamento, timing, decisor — e avançar para reunião agendada.

## Stages Default

| Ordem | Nome | Descrição | Next típico |
|---|---|---|---|
| 1 | `novo` | Lead acabou de entrar, ainda não foi abordado. | abordado |
| 2 | `abordado` | Primeira mensagem enviada ao lead. | respondeu, esfriou |
| 3 | `respondeu` | Lead respondeu ao menos uma vez. | agendado, esfriou |
| 4 | `esfriou` | Lead parou de responder após N horas. | final negativo (via regra) |
| 5 | `agendado` | Reunião marcada. Dispara criação de entry em Confirmação. | final positivo deste pipe |

Admin pode editar nomes, adicionar stages intermediárias (ex.: "aquecendo", "pré-qualificado"), remover stages não-essenciais. Stages `novo` e `agendado` são críticas para automações; remover exige migração explícita.

## Atores e Permissões

- **SDR**: primário. Vê, move, envia mensagens, atribui a si.
- **Admin**: tudo.
- **Agente IA (copilot)**: pode mover stages conforme Kanban Rules (geralmente `novo → abordado` ao iniciar conversa, `abordado → respondeu` ao receber resposta, `respondeu → agendado` ao confirmar reunião).
- **Workflow**: pode mover conforme regras configuradas.

Ações: `pipeline.view:whatsapp`, `pipeline.move_entry:whatsapp`.

## Dados Envolvidos

Pipeline Entry de WhatsApp carrega `meta` com:

- `first_approached_at`: quando enviou primeira msg.
- `first_response_at`: quando lead respondeu.
- `approach_count`: quantas tentativas de abordagem.
- `next_followup_at`: próximo contato agendado.
- `cooling_reason`: motivo de esfriamento (opcional — timeout, lead pediu pausa, etc.).

Ver [[02 - Modelo de Domínio/Pipeline]] para atributos de Stage e Entry em geral.

## Regras de Negócio

1. Lead novo (sem entries) recebe entry em `novo` por default se origem configurada assim.
2. Transição `novo → abordado` pode ser automática ao enviar a primeira mensagem.
3. Transição `abordado → respondeu` é automática ao receber primeira mensagem inbound do lead após abordagem.
4. Transição `respondeu → esfriou` é automática após SLA (ex.: 48h sem resposta, configurável).
5. Transição para `agendado` **sempre** dispara criação de entry em Pipeline Confirmação (via regra de pipe ou workflow built-in).
6. Lead em `agendado` sai do kanban WhatsApp por default (filtrado), para não poluir visão do SDR.
7. Reabertura: se lead em `agendado` cancela reunião (sai de Confirmação para stage negativa), pode voltar a entry em WhatsApp `respondeu` conforme workflow.
8. Respeita janela de negócio nas automações de primeiro contato (ex.: lead entra 23h → abordagem marcada para 9h do dia seguinte).

## Fluxos do Usuário

### Visualização Kanban
1. Abre `Pipeline WhatsApp` no menu.
2. Vê colunas por stage, cards por entry.
3. Card mostra: foto (se aplicável), nome, empresa, tags top, tempo na stage, responsável, badges (workflow count, mensagens não-lidas).
4. Filtros no topo: responsável (meu / todos se admin), tag, origem, período.
5. Drag-drop entre colunas com animação.
6. Clique em card → drawer do lead.

### Abordar Lead
1. SDR vê lead em `novo`.
2. Clica "Iniciar conversa" (abre chat do lead).
3. Envia mensagem (template ou livre).
4. Sistema move entry para `abordado` (auto ou manual).
5. `first_approached_at` registrado.

### Responder
1. Lead responde em WhatsApp.
2. Inbound chega, entry move para `respondeu`.
3. `first_response_at` registrado.
4. Notificação push/in-UI para responsável.

### Agendar
1. SDR ou copilot confirma reunião.
2. Preenche data/hora (integra com calendário).
3. Move card para `agendado` (manual ou action).
4. Sistema cria entry em Confirmação com `meeting_date`.
5. Entry em WhatsApp é finalizada (status positivo local).

### Esfriar
1. Lead não responde em 48h.
2. Job de SLA detecta, move para `esfriou`.
3. Workflow de retenção pode disparar (mensagem automática, áudio, etc.).

## Distribuição

- Leads novos são distribuídos entre SDRs ativos via regra configurada no pipe:
  - `round_robin` entre `is_active=true` com `specialty=sdr`.
  - `load_based`: quem tem menos leads em `novo` + `abordado`.
  - `tag_based`: tag "Ouro" → sênior; tag "Latão" → júnior.
- Se nenhum SDR online, leads acumulam em `unassigned`.
- Admin pode reatribuir em bulk.

## Automações e Eventos

### Emite
- `LeadEnteredPipe(whatsapp, novo)`.
- `LeadStageChanged(whatsapp, from, to, actor)`.
- `LeadLeftPipe(whatsapp, reason)`.

### Consome
- Novo lead criado → cria entry no pipe (se configurado).
- Mensagem outbound enviada → move `novo → abordado` (regra).
- Mensagem inbound recebida → move `abordado → respondeu`.
- Timer de SLA expira → move `abordado/respondeu → esfriou`.
- Cancelamento de reunião → volta lead para `respondeu`.

### Regras de Pipe (dispatch rules típicas)
- `on_enter(agendado)` → `create_entry(pipeline=confirmacao)`.
- `on_enter(esfriou)` → `add_tag(Esfriado)` + `assign_workflow(RetencaoSlow)`.
- `on_sla_expired(abordado, 48h)` → `move_stage(esfriou)`.

## Integrações

- **Pipeline Confirmação**: criado ao agendar.
- **Chat Multi-canal**: origem de conversas.
- **Copilot**: agente pode operar este pipe.
- **Workflow Builder**: triggers `stage_changed`, `lead_entered_pipe`.
- **Analytics**: funnel de qualificação (conversão por stage).
- **Follow-ups**: criados automaticamente por workflow nesta pipe.
- **Campanhas**: lead em `esfriou` pode entrar em campanha de reengajamento.

## Edge Cases

- **Lead sem telefone**: aparece em `novo` mas não pode ser abordado via WhatsApp. UI mostra warning, sugere coletar telefone antes.
- **Múltiplos SDRs mesmo lead**: evitado por `responsible_id` único; se `assigned_to_self` conflita, primeiro ganha, segundo vê toast "já atribuído".
- **Lead cancelou reunião em Confirmação**: volta para `respondeu` ou `agendado` conforme política; workflow decide.
- **Reabrir lead `esfriou`**: enviar mensagem move de volta para `abordado` automaticamente (ou stage específica via workflow).
- **Drag durante realtime update**: otimista — UI aplica, backend confirma. Conflito → UI reverte e mostra erro.
- **Stage deletada com entries ativas**: admin força escolha de stage destino antes.

## Validações

- Stage destino pertence ao pipeline.
- Permissão para mover.
- Transição não-proibida por regra (ex.: de `esfriou` direto para `agendado` sem passar por `respondeu`: talvez OK, mas workflow pode bloquear).

## Métricas

- **Conversão por stage**: `moved_to_next / entered_stage`.
- **Tempo médio por stage**: distribuição e p50/p90.
- **Leads abordados vs novos recebidos**: taxa de atendimento.
- **Taxa de resposta**: `respondeu / abordado`.
- **Taxa de agendamento**: `agendado / respondeu`.
- **Taxa de esfriamento**: `esfriou / abordado`.
- **SDR scorecard**: leads trabalhados, taxa de resposta, taxa de agendamento, tempo médio.
- **Origem → taxa de agendamento**: identifica origens mais qualificadas.

## Performance

- Kanban carrega top-N entries por stage (paginado se > 200).
- Realtime debounce 2s.
- Índices por `(organization_id, pipeline_id, current_stage_id)` e `(organization_id, pipeline_id, moved_to_current_at)`.
