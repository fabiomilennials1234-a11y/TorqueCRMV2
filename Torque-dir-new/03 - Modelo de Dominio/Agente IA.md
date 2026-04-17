---
tipo: dominio
entidade: Agente IA
criticidade: alta
---

# Agente IA (Copilot)

Entidade configurável que representa um **bot conversacional** treinado no contexto da organização. Interage com leads em tempo real via canais de mensagem. Executa ações estruturadas (mover stage, adicionar tag, agendar, criar follow-up) além de responder.

> **Área crítica**: bugs aqui afetam diretamente a reputação da organização com seu lead. Qualquer mudança exige teste completo do fluxo.

## Atributos

### Identificação
- `id`.
- `organization_id`.
- `name`.
- `avatar_url`.
- `description`.

### Template
- `template_type`: `qualificador` | `sdr` | `followup` | `agendador` | `prospectador` | `custom`.
- Template define presets iniciais de personalidade, skills e objetivo.

### Personalidade
- `personality_tone`: `formal` | `casual` | `amigável` | `técnico` | `consultivo`.
- `personality_style`: `direto` | `conversacional` | `detalhado` | `objetivo`.
- `personality_energy`: `calmo` | `entusiasmado` | `neutro`.
- Combinação desses três compõe a voz do agente.

### Objetivo
- `main_objective`: texto curto (ex.: "Qualificar lead e marcar reunião").
- `objective_composite`: JSON com objetivos compostos (ex.: `{primary: "qualify", secondary: ["answer_faq", "schedule_meeting"]}`).

### Habilidades
- `skills`: array de strings (ex.: `["qualify", "schedule", "answer_pricing", "handle_objection"]`).
- `allowed_topics`: array de tópicos que pode abordar.
- `forbidden_topics`: array de tópicos a evitar (ex.: política, concorrente, comparação de preço com outros clientes).

### Contexto de Negócio
- `business_context`: texto livre sobre a empresa, produto, diferenciais, preços, público. Injetado no prompt.
- `faqs`: lista de pares P&R com embeddings (entidade separada — ver FAQ Embedado).

### Configuração Operacional
- `is_active`: ligado/desligado.
- `is_default`: default da org (no máximo um).
- `channels`: array de instâncias de canal onde o agente atende.
- `batch_window_seconds`: janela de agrupamento (default 8s).
- `takeover_pause_minutes`: quanto tempo pausar após humano responder (default 10).
- `max_response_length_chars`: tamanho máximo da resposta antes de forçar split.

### Voz (TTS)
- `tts_config`: JSON opcional com `voice_id`, `style`, `stability`, `similarity_boost`.
- Se presente, agente envia áudios gerados de texto além de texto.

### Wizard
- `wizard_progress`: JSON com estado do wizard de 20+ steps que admin preencheu.
- `system_prompt`: texto gerado **automaticamente** a partir das configurações acima. NUNCA editado diretamente.

### Limites e Quota
- Contabilizado em quota de agentes do plano.
- Inativo ainda conta na contagem (para prevenir "criar todos e ativar um")? Varia por plano.

## Templates

### `qualificador`
- Objetivo: qualificar lead novo (entender necessidade, orçamento, timing, decisor).
- Skills: `qualify`, `answer_faq`, `add_tag`.
- Kanban rules típicas: em stage `novo` do WhatsApp, objetivo é obter respostas para BANT.

### `sdr`
- Objetivo: qualificar + agendar reunião com closer.
- Skills: `qualify`, `schedule`, `answer_faq`.

### `followup`
- Objetivo: reengajar leads inativos, confirmar reunião, lembrar.
- Skills: `reengage`, `confirm_meeting`.

### `agendador`
- Objetivo primário: marcar reunião no calendário.
- Skills: `schedule`, `reschedule`, `cancel`.

### `prospectador`
- Objetivo: conduzir sequência outbound, tirar primeira resposta.
- Skills: `prospect`, `qualify`.

### `custom`
- Zero presets; admin define tudo.

## Kanban Rules (comportamento por stage)

Entidade separada `agent_kanban_rule`:

- `agent_id`.
- `pipe_type`: `whatsapp` | `confirmacao` | `propostas` | `custom:<pipe_id>`.
- `stage_name`: nome da stage.
- `goal`: objetivo do agente nesta stage (texto).
- `behavior`: como agir (texto; ex.: "Seja breve, peça apenas o telefone se ainda não tiver").
- `allowed_actions`: array (ex.: `["answer", "move_stage:agendado", "add_tag:qualificado"]`).
- `forbidden_actions`: array (ex.: `["discuss_price", "promise_discount"]`).
- `exit_on`: condição para encerrar conversa IA e passar para humano (ex.: "lead pede por humano", "lead pergunta por CEO").

Ao responder, agente consulta a regra da stage atual do lead e segue.

## Follow-up Rules (automação de follow-up)

Entidade `agent_followup_rule`:

- `agent_id`.
- `name`.
- `trigger_type`: `no_response_after_hours` | `no_response_after_days` | `reminder_for_meeting`.
- `priority`: 1-10 (ordem quando múltiplas rules matcham).
- `filters`: JSON (stage, tags, origin).
- `behavior`: JSON com instrução do follow-up (template, tom, se manda áudio).
- `is_active`.

## FAQ Embedado

Entidade `agent_faq`:

- `id`, `agent_id`, `organization_id`.
- `question`: texto.
- `answer`: texto.
- `category`: opcional.
- `position`: ordem (opcional).
- `embedding`: vetor (dimensão fixa, ex.: 1536).
- `embedding_model`: qual modelo gerou.
- `last_indexed_at`.

Uso: ao gerar resposta, agente embeda pergunta do lead, busca kNN nas FAQs desta organização desta agente, inclui top-K no contexto do LLM.

## Ações IA (AI Actions)

LLM retorna resposta estruturada que pode incluir ações:

```json
{
  "message": "Perfeito! Vou confirmar com nosso consultor.",
  "actions": [
    {"type": "move_stage", "pipeline": "whatsapp", "stage": "agendado"},
    {"type": "add_tag", "tag_name": "Qualificado"},
    {"type": "create_followup", "assigned_to": "sdr_1", "due_in_hours": 24}
  ]
}
```

Executor de AI Actions:
1. Valida action permitida pela Kanban Rule corrente.
2. Valida action permitida pelo plano.
3. Executa em ordem.
4. Se action falha, loga erro mas envia mensagem (não trava a resposta).
5. Audit log registra cada action.

### Catálogo de Actions

- `move_stage`: mover lead.
- `add_tag` / `remove_tag`.
- `assign_responsible`.
- `create_followup`.
- `schedule_message`.
- `schedule_meeting` (integra calendário).
- `update_lead_field`.
- `pause_conversation` (pede humano).
- `end_conversation` (se objetivo cumprido).
- `start_audio_response`.
- `add_note_internal`.

## Batch Window (ver [[Conversa e Mensagem]])

- Default 8s.
- Configurável por agente.
- Motivo: humanização — não responde a cada keystroke.

## Smart Split

- Resposta do LLM excede X chars → split em mensagens consecutivas.
- Delay entre chunks (ex.: 2-4s cada, aumentando com tamanho) simulando digitação.
- Nunca parte em palavra/link.
- Áudio (TTS) é enviado como mensagem única.

## Human Takeover

- Detectado quando mensagem outbound vem de `sender_type=member` na conversa.
- Pausa agente por `takeover_pause_minutes`.
- Conversa mostra badge "humano no controle" até expirar.

## Teste (Playground)

- Interface onde admin conversa com o agente sem afetar produção.
- Exibe prompt enviado, ações sugeridas, resposta.
- Útil para validar configuração antes de ativar.

## Métricas por Agente

- Mensagens enviadas.
- Taxa de resposta do lead.
- Taxa de qualificação (% que avançou para stage objetivo).
- Taxa de takeover (% que precisou humano).
- Tempo médio de resposta.
- Custo de LLM por conversa.
- Satisfação (NPS de lead ou feedback do time).
- Ações executadas por categoria.

## Avaliação Automática de Qualidade

- Job periódico examina conversas finalizadas.
- LLM avaliador (modelo separado) pontua qualidade da interação (1-5) e razão.
- Flags conversas ruins para revisão humana.
- Agrega em dashboard para admin ajustar agente.

## Geração do System Prompt

Algoritmo conceitual:

```
prompt = f"""
Você é {agent.name}, um assistente {template_type} da {organization.name}.

PERSONALIDADE:
Tom: {personality_tone}
Estilo: {personality_style}
Energia: {personality_energy}

OBJETIVO PRINCIPAL:
{main_objective}

CONTEXTO DO NEGÓCIO:
{business_context}

TÓPICOS PERMITIDOS: {allowed_topics}
TÓPICOS PROIBIDOS: {forbidden_topics}

CONTEXTO ATUAL:
Lead: {lead.name}, empresa {lead.company}
Stage atual: {current_stage.name} no pipe {current_pipe.name}
Objetivo nesta stage: {kanban_rule.goal}
Comportamento esperado: {kanban_rule.behavior}
Ações permitidas: {kanban_rule.allowed_actions}
Ações proibidas: {kanban_rule.forbidden_actions}

FAQS RELEVANTES (recuperadas via busca vetorial):
{top_k_faqs}

HISTÓRICO RECENTE DA CONVERSA:
{last_n_messages}

INSTRUÇÕES DE FORMATO:
- Responda em JSON estruturado com campos "message" e "actions".
- Se precisa de humano, inclua action "pause_conversation" com razão.
- Nunca invente dados. Se não sabe, diga.
- Respeite idioma do lead (PT-BR default).
"""
```

Regenerado automaticamente sempre que config relevante muda.

## Eventos

- `AgentCreated`, `AgentUpdated`, `AgentActivated`, `AgentDeactivated`, `AgentDeleted`.
- `AgentConversationStarted`, `AgentMessageSent`, `AgentActionExecuted`.
- `AgentHandoffToHuman(reason)`.
- `AgentEvaluationCompleted(conversation_id, score)`.

## Relações

- **N:1** Organization.
- **1:N** FAQ.
- **1:N** Kanban Rule.
- **1:N** Follow-up Rule.
- **1:N** Audio (TTS pré-gerado, opcional).
- **1:N** Conversa (quando agente está ativo nela).

## Quotas e Planos

- Número de agentes por org: limite do plano.
- Número de FAQs por agente: limite do plano.
- Mensagens processadas/mês: limite do plano (ultrapassar bloqueia envio novo).

## Edge Cases

- **Lead em stage sem kanban rule configurada** → usa defaults genéricos; alerta admin.
- **Lead sem telefone** → não inicia conversa, fica em estado `no_channel`.
- **Agente sem business_context** → respostas genéricas; admin é avisado ao ativar.
- **LLM timeout** → não envia resposta, agenda retry em 30s, se falha de novo pausa conversa + notifica admin.
- **LLM retorna action inválida** (stage inexistente) → registra erro, envia só a mensagem sem ação, audit flagged.
- **FAQ sem embedding** (recém-criada) → job de embedding pendente; RAG funciona com FAQs indexadas.

## Segurança

- Agente nunca executa action fora de `allowed_actions`.
- Admin deve aprovar agentes que podem propor preços/descontos (opt-in).
- Agente não acessa dados de outra organização (enforçado por contexto).
- Conteúdo de conversa não é usado para treinar modelos externos (verificar contrato do provedor LLM).
