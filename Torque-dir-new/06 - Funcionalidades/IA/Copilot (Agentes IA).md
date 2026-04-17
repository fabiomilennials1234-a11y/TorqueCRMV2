---
tipo: feature
dominio: ia
criticidade: alta
---

# Copilot (Agentes IA)

> **Área crítica**: é o fluxo que mais gera bugs e confusão com usuários. Qualquer mudança exige teste completo: criar agente → configurar → ativar → conversar com lead.

## Propósito

Agentes IA conversacionais que **atendem leads em tempo real** via canais de mensagem, qualificam, respondem dúvidas, agendam reuniões, confirmam presença, executam follow-ups. Substituem ou complementam o SDR humano. Configuráveis sem código pela admin da organização.

## Atores e Permissões

- **Admin**: CRUD e configuração completa, wizard, FAQs, regras, playground.
- **Membros**: não gerenciam agente; podem fazer takeover (assumir conversa manualmente).
- **Copilot ele próprio**: atua em conversas quando ativo.

Ações: `copilot.view`, `copilot.create`, `copilot.edit`, `copilot.activate`, `copilot.delete`, `copilot.test_playground`, `copilot.view_metrics`, `copilot.take_over`.

## Modelo Conceitual

Ver [[02 - Modelo de Domínio/Agente IA]] para entidade completa. Componentes principais:

1. **Agente** — identidade, personalidade, objetivo, business_context.
2. **Template** — preset inicial (qualificador, sdr, followup, agendador, prospectador, custom).
3. **Kanban Rules** — comportamento por stage de pipeline.
4. **Follow-up Rules** — automação proativa.
5. **FAQs Embedados** — base de conhecimento com busca vetorial.
6. **System Prompt** — gerado automaticamente, nunca editado manualmente.
7. **AI Actions** — comandos estruturados que o LLM pode emitir (mover stage, tag, etc.).
8. **TTS** — geração de áudio.
9. **Playground** — área de teste.
10. **Métricas** — dashboard de performance do agente.

## Templates de Agente

| Template | Objetivo | Skills Default | Canais Típicos |
|---|---|---|---|
| `qualificador` | Qualificar lead novo, coletar dados (BANT). | `qualify`, `answer_faq`, `add_tag` | WhatsApp inbound |
| `sdr` | Qualificar + agendar reunião com closer. | `qualify`, `schedule`, `answer_faq` | WhatsApp inbound |
| `followup` | Reengajar leads inativos, confirmar reuniões, lembrar. | `reengage`, `confirm_meeting` | WhatsApp outbound |
| `agendador` | Focado em marcar reunião no calendário. | `schedule`, `reschedule`, `cancel` | WhatsApp |
| `prospectador` | Conduzir sequência outbound, tirar primeira resposta. | `prospect`, `qualify` | WhatsApp outbound |
| `custom` | Zero presets. Admin define tudo. | — | — |

## Wizard de Configuração (20+ steps)

O wizard é o **único caminho** para configurar agente. Gera system_prompt automaticamente.

### Steps principais
1. Template (escolhe acima).
2. Nome + avatar + descrição.
3. Personalidade: tom, estilo, energia (sliders ou chips).
4. Objetivo principal (texto curto).
5. Objetivo composto (principal + secundários).
6. Skills (checkboxes; depende do template).
7. Allowed topics (tags).
8. Forbidden topics.
9. Business context (textarea longa).
10. Exemplos de conversa (alguns pares pergunta-resposta ideais).
11. FAQs (lista; cada FAQ gera embedding assíncrono).
12. Kanban Rules: para cada pipe + stage, definir goal/behavior/allowed/forbidden.
13. Follow-up Rules: gatilhos e comportamento.
14. Canais ativos (quais instâncias de canal o agente atende).
15. Batch window (slider: 4s, 8s, 15s, 30s).
16. Takeover pause (5, 10, 15, 30 min).
17. TTS: ativar? voz, estilo, estabilidade.
18. Limites: max_response_length.
19. Default da org? (radio).
20. Revisão final.
21. Ativar (ou deixar como draft).

Wizard pode ser pausado e retomado (progresso persiste).

## Kanban Rules (por stage)

Para cada combinação `(pipe, stage)`:
- `goal`: objetivo do agente nesta stage.
- `behavior`: como agir (tom, passos).
- `allowed_actions`: lista de actions permitidas (catálogo abaixo).
- `forbidden_actions`: lista de actions proibidas.
- `exit_conditions`: quando passar para humano (ex.: "lead pediu supervisor", "lead perguntou por CEO").

Exemplo:

| Pipe | Stage | Goal | Allowed | Forbidden |
|---|---|---|---|---|
| whatsapp | novo | Cumprimentar, entender interesse | `answer`, `add_tag:{topic}` | `move_stage`, `schedule` |
| whatsapp | abordado | Qualificar (BANT) | `answer`, `add_tag`, `move_stage:respondeu` | `schedule` |
| whatsapp | respondeu | Agendar reunião | `answer`, `schedule`, `move_stage:agendado` | — |
| confirmacao | confirmar_d1 | Confirmar comparecimento | `answer`, `reschedule`, `add_tag:confirmado` | `move_stage:direto_compareceu` |

## Follow-up Rules

Disparos proativos sem mensagem do lead:

- `no_response_after_hours`: se lead não respondeu em N horas.
- `no_response_after_days`: similar.
- `reminder_for_meeting`: lembrete D-5/D-3/D-1.
- Filtros: aplicar só em stage X, com tag Y.
- Behavior: qual mensagem, se áudio ou texto, tom.
- Priority: quando múltiplas rules matcham, executa em ordem.

## FAQs Embedados (RAG)

### Estrutura
- `question` + `answer` + `category` opcional.
- Ao salvar, job assíncrono calcula embedding do `question` via serviço de embeddings.
- Vetor armazenado em banco vetorial, indexado por `agent_id` + `organization_id`.

### Uso
1. Lead faz pergunta.
2. Sistema gera embedding da pergunta.
3. Busca kNN no banco vetorial filtrado por `agent_id` + `organization_id`.
4. Top-K FAQs mais similares (geralmente K=5, similaridade mínima 0.7).
5. Injetadas no contexto do LLM para resposta fundamentada.

### Gerenciamento
- CRUD manual no wizard.
- Geração automática: agente pode sugerir FAQs baseando-se em conversas passadas.
- Reindexação: ao mudar modelo de embedding, reprocessa.

## Business Context

Texto longo (até 10.000 chars) descrevendo:
- O que a empresa faz.
- Diferenciais.
- Público-alvo.
- Produtos principais (resumo + preços-chave).
- Linguagem preferida (formal/informal).
- Proibições (não dar desconto, não prometer prazo, etc.).

Injetado no system_prompt. É a **fonte de verdade factual** do agente.

## System Prompt (geração automática)

Template conceitual:

```
Você é {nome}, assistente {template_type} da {org.name}.

PERSONALIDADE:
Tom: {tom}. Estilo: {estilo}. Energia: {energia}.

OBJETIVO PRIMÁRIO:
{main_objective}

OBJETIVOS SECUNDÁRIOS:
{objective_composite.secondary}

CONTEXTO DO NEGÓCIO:
{business_context}

TÓPICOS PERMITIDOS: {allowed_topics}
TÓPICOS PROIBIDOS: {forbidden_topics}

CONTEXTO DA CONVERSA ATUAL:
Lead: {lead.name}, empresa {lead.company}, telefone {lead.phone}
Stage atual: {current_stage} em {current_pipe}
Objetivo neste stage: {kanban_rule.goal}
Como agir: {kanban_rule.behavior}
Ações permitidas: {kanban_rule.allowed_actions}
Ações proibidas: {kanban_rule.forbidden_actions}

FAQs RELEVANTES (via busca vetorial):
{top_5_faqs}

HISTÓRICO DA CONVERSA (últimas N mensagens):
{last_n_messages}

INSTRUÇÕES DE FORMATO:
- Responda em JSON: { "message": "...", "actions": [...] }
- Actions conforme catálogo.
- NUNCA invente dados. Se não sabe, diga.
- Respeite idioma do lead.
- Evite respostas longas; chunkar em Smart Split se necessário.
```

Regenerado sempre que config relevante muda.

## AI Actions (catálogo)

| Action | Parâmetros | Descrição |
|---|---|---|
| `answer` | — | Apenas responder (nenhuma ação lateral). |
| `move_stage` | `pipe`, `stage` | Move entry do lead. |
| `add_tag` | `tag_name` | Adiciona tag. |
| `remove_tag` | `tag_name` | Remove. |
| `assign_responsible` | `member_id` ou `strategy` | Atribui. |
| `create_followup` | `title`, `due_in_hours`, `assigned_to` | Cria follow-up. |
| `schedule_message` | `content`, `scheduled_for` | Agenda. |
| `schedule_meeting` | `date`, `duration`, `location` | Integra calendário. |
| `update_lead_field` | `field`, `value` | Atualiza custom field. |
| `pause_conversation` | `reason` | Pede humano. |
| `end_conversation` | `reason` | Agente encerra. |
| `add_note_internal` | `content` | Nota interna. |
| `play_audio` | `audio_id` ou `text` (TTS) | Envia áudio. |

Executor:
1. Valida action contra `allowed_actions` da Kanban Rule corrente.
2. Valida plano.
3. Executa em ordem; cada action isolada.
4. Falha de uma não bloqueia as outras.
5. Audit log.

## Batch Window (Agrupar Mensagens)

- Lead pode mandar "Oi", "tudo bem?", "queria saber do produto" em 3 mensagens em 5s.
- Agente espera `batch_window_seconds` (default 8s) antes de responder.
- Todas as mensagens desse burst são processadas em uma ida ao LLM.
- Humaniza: agente não responde a cada keystroke.

## Human Takeover

- Quando membro envia mensagem na conversa → `human_takeover_until = now + takeover_pause_minutes`.
- Durante janela: agente não envia nada.
- Lead escreve: mensagem fica para humano, não aciona agente.
- Ao expirar: agente retoma automaticamente.
- UI sinaliza takeover ativo.

## Smart Split

- Se resposta do LLM > `max_response_length_chars`: split em chunks consecutivos.
- Heurística: 1-3 parágrafos por chunk, quebra em pontos naturais (final de frase, parágrafo).
- Delay 2-4s entre chunks para humanizar.
- Nunca parte em palavra ou URL.
- Máximo N chunks por resposta (evita resposta interminável).

## TTS (Áudio)

- Se `tts_config` presente, agente pode enviar áudios.
- LLM retorna action `play_audio` com texto ou escolhe áudio pré-gravado.
- Texto passa por TTS (serviço externo, ex.: ElevenLabs).
- Áudio gerado em storage, enviado como mensagem de áudio.
- Preview no playground.

## Playground

- Área onde admin **conversa com o agente** sem afetar produção.
- Lead simulado com dados fictícios configuráveis.
- Exibe:
  - System prompt completo (expandível).
  - Mensagens recentes como contexto.
  - FAQs recuperadas (top-K com score).
  - Resposta do LLM (texto + actions).
  - Custo estimado da chamada.
- Não cria conversa real, não executa actions.
- Útil para validar antes de ativar.

## Ativação / Default

- Agente tem `is_active=true/false`.
- Org pode ter 0 ou 1 agente `is_default=true`.
- Default opera em qualquer conversa não atribuída a agente específico.
- Conversa pode ser explicitamente vinculada a agente específico (via Campanha ou regra).

## Métricas (Dashboard do Agente)

- Mensagens enviadas/recebidas.
- Taxa de resposta do lead.
- Taxa de qualificação (% movidos a stage objetivo).
- Taxa de takeover (% precisaram humano).
- Tempo médio de resposta (primeiro → última).
- Custo de LLM por conversa.
- Ações executadas por categoria.
- Avaliação de qualidade (LLM avaliador) — NPS-like 1-5.
- FAQ hits (quais FAQs mais usadas).

## Avaliação Automática de Qualidade

- Job diário examina conversas finalizadas do último dia.
- LLM avaliador separado analisa:
  - Agente seguiu persona?
  - Respondeu objetivamente?
  - Usou FAQs relevantes?
  - Executou ações corretas?
  - Lead terminou satisfeito (inferido)?
- Score 1-5 + justificativa.
- Flagadas como "ruim" vão para revisão humana.
- Agregação → admin ajusta.

## Fluxo Conversacional (End-to-End)

```
1. Lead envia msg em WhatsApp.
2. Webhook provedor → persiste msg inbound → publica realtime.
3. Scheduler detecta: agente ativo + não takeover.
4. Inicia batch window (8s).
5. Lead envia mais msgs dentro da janela → acumula.
6. Janela expira.
7. Sistema monta contexto:
   a. Lead + stage atual + pipe.
   b. Kanban rule ativa.
   c. Mensagens recentes (últimas 10-30).
   d. Embedding da última(s) mensagem(ns) do lead.
   e. Top-K FAQs por similaridade.
   f. System prompt completo.
8. Chama LLM com prompt.
9. LLM retorna JSON estruturado: {message, actions}.
10. Valida resposta:
    a. JSON válido? Senão, fallback (retry ou pause).
    b. Actions em allowed? Senão, filtra e log.
11. Executa AI Actions (em ordem).
12. Envia message:
    a. Se > max_length: Smart Split.
    b. Se TTS ativo e texto apropriado: gera áudio.
    c. Envia via canal → mensagem outbound persistida → realtime.
13. Persist agent_message audit.
14. Monitor: custo, latência, tokens consumidos.
```

## Edge Cases

- **LLM timeout** (> 30s): retry 1x; se falha, pausa conversa + alerta admin.
- **LLM retorna JSON inválido**: parse falha → retry com instruction mais clara; se falha, pausa.
- **Action com parâmetro inválido** (ex.: stage inexistente): action skipped, log erro, envia só mensagem.
- **Conversa tem > 200 mensagens**: contexto trunca para últimas N relevantes + resumo.
- **Lead em stage sem Kanban Rule**: fallback para rule genérica da pipe; warning para admin.
- **Agente sem business_context**: respostas genéricas; warning visível na UI.
- **FAQ sem embedding** (recém-criada): exclui da busca até indexação; indexação assíncrona.
- **Lead muda de idioma**: agente pode perceber (LLM infere) e responder no idioma do lead se allowed.
- **Custo de LLM atingindo quota**: agente pausa automaticamente ao cruzar budget; admin é alertado.
- **Lead diz palavra-chave de STOP** (ex.: "PARAR", "SAIR"): agente faz `pause_conversation` + `add_tag:opt-out`.

## Quotas e Planos

- Número de agentes IA por org: limite do plano.
- Número de FAQs por agente: limite do plano.
- Mensagens processadas por mês: limite global do plano.
- TTS: feature gated (plano com áudio).
- Avaliação automática: feature gated (plano premium).

## Segurança

- Agente não executa action fora de `allowed_actions`.
- Actions críticas (ex.: `update_lead_field` com campo sensível): podem exigir aprovação de admin via feature flag.
- Dados do lead não vazam entre orgs — contexto é sempre per-org.
- Contrato com provedor LLM: não usar conteúdo para treinar modelos (quando plano comercial permite).
- Conteúdo de conversa não é logado em nível `info` em produção.

## Observabilidade

- Log de cada interação com LLM: tokens, latência, resultado.
- Log de cada action executada.
- Alerta se taxa de takeover sobe acima de X% (possível problema com agente).
- Alerta se latência média cresce (possível problema com provider LLM).

## Configuração Recomendada (Best Practices)

- Business context rico é o maior fator de qualidade.
- FAQs cobrem as top 20 perguntas reais (não teóricas).
- Kanban rules específicas por stage > regra genérica.
- Teste exaustivo no playground antes de ativar em produção.
- Monitorar primeira semana — taxa de takeover e qualidade.
- Iterar: revisar conversas ruins, atualizar FAQs e rules.
