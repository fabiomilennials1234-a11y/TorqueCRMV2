---
tipo: feature
dominio: automacao
criticidade: alta
---

# Workflow Builder

## Propósito

Editor visual que permite ao admin construir automações complexas em forma de **grafo acíclico dirigido (DAG)** — sem código. Cada node representa uma unidade (trigger, ação, decisão, espera) e edges representam fluxo. Ao ser disparado, o motor de execução percorre o grafo.

## Atores e Permissões

- **Admin**: CRUD e gestão.
- **Membros**: sem acesso (permissão override possível).
- **Master**: acesso via impersonação.

Ações: `workflow.view`, `workflow.create`, `workflow.edit`, `workflow.activate`, `workflow.delete`, `workflow.execute_manual`, `workflow.view_executions`.

## Modelo Conceitual

Ver [[02 - Modelo de Domínio/Workflow]] para entidade. Resumo:

- **Workflow** (definição): DAG de nodes + edges.
- **Node**: unidade (trigger, action, condition, delay, wait_response, split_ab, copilot, webhook_call, wait_business_window, loop, parallel, join, end).
- **Edge**: conexão direcional; pode ter porta (true/false para condition, named branches para split).
- **Execution**: instância rodando para um lead.
- **Execution Step**: audit granular.

## Catálogo Completo de Nodes

### Trigger (obrigatório, exatamente 1)

| Tipo | Config |
|---|---|
| `lead_created` | filtros: origin, tags, pipeline_default. |
| `lead_entered_pipe` | `pipeline_id`, `stage_id` opcional. |
| `lead_left_pipe` | `pipeline_id`, `reason`. |
| `stage_changed` | `pipeline_id`, `from_stage_id`, `to_stage_id`. |
| `tag_added` | `tag_id`. |
| `tag_removed` | `tag_id`. |
| `message_received` | `channel`. |
| `message_not_responded` | `hours`. |
| `cron` | `cron_expression`, `timezone`. |
| `manual` | — (admin clica para disparar). |
| `webhook_received` | token da org. |
| `form_submitted` | `form_id`. |
| `meeting_scheduled` | — |
| `meeting_canceled` | — |
| `payment_received` | — |
| `payment_failed` | — |
| `followup_overdue` | `hours_overdue`. |

### Actions

#### Mensagem
- `send_message` — `channel`, `template_id` ou `content`, mídia opcional, `respect_business_hours`.
- `send_audio` — `audio_id` (pré-gravado) ou `text_to_tts`.
- `schedule_message` — `content`, `scheduled_for`.

#### Pipeline
- `move_stage` — `pipeline_id`, `stage_id`.
- `create_pipeline_entry` — criar entry em pipe específico.
- `close_pipeline_entry` — finalizar com status.

#### Lead
- `add_tag` — `tag_id`.
- `remove_tag` — `tag_id`.
- `update_lead_field` — `field`, `value` (pode referenciar variável).
- `assign_responsible` — `strategy`, `member_id` opcional.
- `assign_sdr`, `assign_closer`.
- `calculate_score` — força recálculo.

#### Follow-up
- `create_followup` — título, prazo, atribuído.
- `complete_followup` — marca concluído.

#### Agente IA
- `start_copilot` — `agent_id`.
- `end_copilot`.
- `pause_copilot_temporary` — `minutes`.

#### Integrações Externas
- `call_webhook` — URL, método, headers, body template.
- `create_calendar_event` — integra calendário.
- `sync_to_erp` — envia para ERP.

#### Notificação
- `notify_member` — push/email a membro.
- `notify_lead` — canal + conteúdo.

### Condition
- `condition` — expressão booleana, duas portas (true/false).

### Timing
- `delay` — `delay_type` (`relative` em horas/minutos, `absolute` timestamp, `until_business_hour`).
- `wait_response` — pausa até lead enviar mensagem; timeout configurável.
- `wait_business_window` — pausa até próxima janela de negócio.

### Controle de Fluxo
- `split_ab` — branches probabilísticos com pesos.
- `loop` — repete um trecho N vezes ou até condição.
- `parallel` — bifurca em múltiplas branches concorrentes.
- `join` — reúne branches; modo `all` (espera todas) ou `any` (primeira).
- `end` — encerra execução (com reason opcional).

## Editor Visual

### Interface
- Canvas infinito com zoom e pan.
- Paleta de nodes à esquerda (arrastar para canvas).
- Inspector à direita (configura node selecionado).
- Mini-mapa no canto.
- Botões: Play (teste), Save, Activate, Version History.

### Interação
- Drag node da paleta ao canvas.
- Click em output port e arraste para input de outro node → cria edge.
- Click em edge para deletar.
- Copy/paste nodes (Ctrl+C/V).
- Undo/redo.

### Validação Visual
- Node não conectado: warning amarelo.
- Trigger ausente: error vermelho — não pode ativar.
- Grafo com ciclo: error.
- Config obrigatória faltando: warning no node.

### Versionamento
- Cada Save incrementa versão.
- Painel de histórico mostra últimas N versões com diff.
- Revert para versão anterior.

## Variáveis e Templating

Nodes usam `{{ var }}`:
- `{{ lead.name }}`, `{{ lead.phone }}`, `{{ lead.custom_fields.key }}`
- `{{ now }}`, `{{ today }}`
- `{{ trigger.message.content }}` (no contexto de `message_received`)
- `{{ context.<key> }}` — variáveis acumuladas (set por action `set_variable`)
- `{{ member.name }}`

Engine expande antes de executar action.

## Condition — DSL

Expressão booleana:
```
(lead.custom_fields.budget > 10000 AND lead.has_tag("B2B"))
OR
(lead.origin == "referral")
```

Operadores listados em [[02 - Modelo de Domínio/Workflow]]#condition--operadores.

## Execução (Motor)

### Disparo
1. Evento ocorre (ex.: `lead_created`).
2. Engine consulta workflows ativos com trigger matching (inclui filters).
3. Para cada match, cria `Execution` com `context` inicial (lead, trigger data).
4. Worker consome.

### Processamento Node a Node
1. Engine pega `current_node`.
2. Executa handler do tipo:
   - `trigger`: no-op, captura data.
   - `action.*`: chama caso de uso domínio → aplica efeito.
   - `condition`: avalia expressão → escolhe edge (true ou false).
   - `delay`: marca `waiting`, agenda resume.
   - `wait_response`: marca `waiting`, registra listener em conversa.
   - `split_ab`: sorteia edge com pesos.
   - `parallel`: cria sub-executions para cada branch.
   - `join`: aguarda condição (all/any).
   - `end`: marca completed.
3. Cria `Execution Step` com input/output.
4. Segue edge ao próximo node.

### Estados
```
pending → running ↔ waiting → completed
                 ↓
              failed
                 ↓
             cancelled
```

### Retry
- Actions com efeito externo (`send_message`, `call_webhook`, `create_calendar_event`): retry 5x com backoff.
- Após esgotar: step=failed.
- Workflow config: `on_failure` edge opcional ou `fail_execution`.

### Timeout
- Execução total timeout configurável (default 7 dias).
- Cronjob garbage-collector fecha estagnadas.

### Concorrência
- Mesmo lead pode ter N executions ativas.
- Flag `mutex_per_lead` → apenas 1 ativa por lead; nova substitui (cancela anterior) ou é rejeitada (configurável).

## Ativação

- Workflow em `draft` não dispara.
- Admin clica "Ativar":
  - Validação: grafo válido, trigger OK, todos os nodes configurados.
  - Se sim: `is_active=true`.
  - Alerta: "Este workflow disparará para todos os futuros [trigger]. Continuar?"
- Execuções em andamento sobrevivem a ativação/desativação.

## Observabilidade

### Lista de Executions
- Filtro por workflow, status, período, lead.
- Cada execution mostra: início, fim, status, duração, nodes percorridos.

### Detalhe da Execution
- Grafo renderizado com highlight dos nodes executados (cor: verde=sucesso, vermelho=falha, azul=em-progresso, cinza=não-executado).
- Click em step → detalhe: input, output, duração, erro.

### Métricas por Workflow
- Total disparado.
- Taxa de sucesso.
- Duração média.
- Node gargalo (onde mais falha/demora).
- Taxa de cada branch (em split_ab).

## Edge Cases

- **Lead deletado durante execução**: steps subsequentes falham com erro explicativo; execution vai para failed ou cancelada.
- **Canal desconectado durante `send_message`**: retry; se persiste, dead letter + alerta.
- **Condition que referencia campo inexistente**: avalia como null → expressão é false (ou erro, configurável).
- **Loop infinito acidental**: detecção por contador `max_iterations` (default 100).
- **Webhook externo timeout**: retry; se > 5 falhas, step=failed.
- **Execução em pausa há > 30 dias**: garbage-collected.
- **Workflow editado enquanto executions em curso**: executions carregam `workflow_version`; edits não retroagem.
- **Trigger `cron` em org com timezone diferente**: expression respeita timezone da org.

## Templates de Workflow (biblioteca)

Org não precisa começar do zero. Templates incluídos:

- "Lembrete automático de reunião (D-1)".
- "Follow-up de proposta não respondida em 48h".
- "Sequência de boas-vindas para novo lead".
- "Alertar SDR se lead qualificado sem abordagem em 1h".
- "Recuperação de lead esfriado".
- "Nutrição educacional (drip campaign)".
- "Encaminhar lead enterprise para sênior automaticamente".

Admin importa, customiza, ativa.

## Validações

- Grafo: exatamente 1 trigger, sem ciclos.
- Nodes: config obrigatória preenchida.
- Edges: conectados a portas válidas.
- Referências: templates, tags, stages, members existentes.
- Permissão: admin.

## Limites e Performance

- Nodes por workflow: limite alto (ex.: 200); UI avisa em 50+.
- Workflows ativos por org: limite do plano.
- Execuções concorrentes por org: limite (previne abuso).
- Fila de execução: prioridade por idade.
- Worker escala horizontalmente.

## Segurança

- `call_webhook` só para URLs allowlisted (protege contra SSRF).
- `update_lead_field` respeita permissões de campo.
- Secrets (para headers de webhook) em vault, não visíveis no editor.

## Métricas

- Workflows ativos.
- Execuções por dia.
- Taxa de sucesso global.
- Custo estimado por workflow (em $ se reliable).
- Workflows mais ativados.
- Workflows "inertes" (ativos mas nunca disparam) — sinal de config ruim.
