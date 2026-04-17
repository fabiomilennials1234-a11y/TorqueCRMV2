---
tipo: dominio
entidade: Campanha
---

# Campanha

Processo paralelo aos pipelines que organiza ação **outbound**: uma iniciativa com objetivo, prazo, público-alvo, sequência de mensagens, distribuição para o time e (opcionalmente) um agente IA.

## Conceitos-chave

- Campanha é **diferente de pipeline**: pipeline representa o processo orgânico de qualificação/fechamento. Campanha representa um **esforço pontual**, com começo e fim.
- Lead pode estar em campanha E em pipeline ao mesmo tempo.
- Vários leads são enrolados em uma campanha — distribuição round-robin ou regra.
- Cada lead-em-campanha percorre **stages de campanha** (não as stages de pipeline).

## Atributos

- `id`.
- `organization_id`.
- `name`.
- `objective`: texto livre descrevendo objetivo (ex.: "Reativar leads perdidos com desconto").
- `status`: `draft` | `active` | `paused` | `completed` | `archived`.
- `start_date`, `end_date` (end_date opcional; se ausente, campanha é contínua).
- `agent_id`: agente IA associado (opcional).
- `target_filter`: filtro de público-alvo (ver "Público-alvo" abaixo).
- `distribution_rule`: regra de distribuição.
- `team_goal`: meta de time (JSON — ex.: `{metric: "leads_closed", target: 20}`).
- `per_member_goal`: meta individual (opcional).
- `owner_id`: membro responsável pela campanha.
- `statistics`: cache de métricas agregadas.
- `created_at`, `updated_at`.

## Campanha Stage (etapa dentro da campanha)

Diferente de stage de pipeline. Representa os passos da sequência outbound.

### Atributos
- `id`.
- `campaign_id`.
- `order`.
- `name` (ex.: "Abordagem inicial", "Follow-up 1", "Última tentativa").
- `wait_hours`: tempo desde entrada na stage anterior (ou desde início) até executar esta.
- `message_template_id` OU `message_content` (um ou outro).
- `alternative_variants`: array para split A/B (opcional).
- `respect_business_hours`: bool.
- `exit_conditions`: quando remover lead da stage/da campanha (ex.: respondeu, tag X adicionada, comprou).

## Campanha Entry (lead na campanha)

- `id`.
- `campaign_id`.
- `lead_id`.
- `assigned_to`: membro que herdou este lead da distribuição.
- `current_stage_id`: campanha stage atual.
- `entered_at`.
- `stage_entered_at`.
- `status`: `active` | `completed` | `exited` (saiu por respondeu/comprou) | `failed`.
- `last_message_at`.
- `exit_reason`.

## Público-alvo (`target_filter`)

Filtros combináveis:
- **Tags**: `has_any_of`, `has_all_of`, `not_has`.
- **Pipeline/stage**: leads em pipeline X stage Y, ou NÃO em pipeline X.
- **Origem**: `origin in [...]`.
- **Custom fields**.
- **Período de criação**: criados entre X e Y.
- **Sem atividade**: última mensagem há mais de N dias.
- **Outros**: score > X, rating >= Y, tag de segmento.

Resultado: conjunto dinâmico ou snapshot estático (admin escolhe).

- **Dinâmico**: leads novos que passam a satisfazer o filtro entram automaticamente.
- **Snapshot**: foto no momento da ativação; novos leads não entram.

## Distribuição

Regras:
- `round_robin` entre membros específicos ou com specialty X.
- `load_based`: membro com menos leads na campanha.
- `weighted`: pesos por membro (ex.: A:60%, B:40%).
- `manual`: admin atribui um a um.
- `none`: sem atribuição (agente IA toma).

Algoritmo round-robin: estado persistente (último atribuído), próximo lead vai para próximo da lista em ordem.

## Sequência de Mensagens

Exemplo de campanha com 5 stages:

```
Stage 1 "Abordagem": dispara imediatamente no entry. Template: "Olá {{lead.name}}, tudo bem?".
Stage 2 "Seguir-se-silêncio": 24h após stage 1 se lead não respondeu. Template: "Só queria confirmar..."
Stage 3 "Oferta": 48h após stage 1. Template: "Temos uma oferta..."
Stage 4 "Pressão amigável": 72h. Template: "Esse é o último..."
Stage 5 "Desistência": 120h. Ação: mover para pipeline X stage Y; adicionar tag "Campanha-X-sem-resposta".
```

Exit conditions em cada stage:
- Lead respondeu → sai da campanha, vai para pipeline WhatsApp novamente (ou pipe configurado).
- Lead aceitou reunião → sai da campanha, vai para Confirmação.
- Lead pediu para parar (STOP) → exit + tag `opt-out`.
- Lead comprou → exit + tag `convertido-via-campanha-X`.

## Agentes IA em Campanhas

- Campanha pode delegar conversa ao agente IA.
- Quando lead responde à primeira mensagem, agente assume.
- Agente IA conhece contexto da campanha (objetivo) no system prompt.
- Regras de campanha podem exigir que agente peça ao humano em certos gatilhos (ex.: lead pediu desconto > X%).

## Estados da Campanha

```
draft → active → paused ↔ active → completed
                        ↓
                     archived
```

- `draft`: em construção, não dispara.
- `active`: em execução.
- `paused`: pausada manualmente ou automaticamente (ex.: detecção de bounce alto).
- `completed`: atingiu end_date ou objetivo.
- `archived`: oculta do dia-a-dia; dados preservados.

## Regras de Negócio

1. Campanha com `agent_id` e `target_filter` dinâmico: novos leads entrando no filtro são enrolados automaticamente se campanha ativa.
2. Lead opt-out global (tag `no-contact`) nunca é enrolado.
3. Respeita janela de negócio por default (mensagens agendadas pulam fins de semana).
4. Duas campanhas ativas para mesmo lead: sistema pode impedir ou permitir conforme config da campanha (`allow_parallel: bool`).
5. Rate limit por instância de canal evita spam (ex.: máx 100 mensagens/hora por número).
6. Pausa automática se taxa de bloqueio > threshold (proteção reputacional do número).

## Dashboard da Campanha

- Entries total, por stage, ativos, completed, exited.
- Taxa de resposta.
- Leads convertidos (atingiram exit = conversão).
- Performance por membro.
- Distribuição de motivos de exit.
- Custo por resposta (se ads relacionado).

## Eventos

- `CampaignCreated`, `CampaignActivated`, `CampaignPaused`, `CampaignCompleted`, `CampaignArchived`.
- `LeadEnrolledInCampaign`.
- `LeadCampaignStageChanged`.
- `LeadExitedCampaign(reason)`.

## Operações

### CreateCampaign (admin ou prospectador)
- Validar filter, stages, templates.
- Estado inicial `draft`.

### ActivateCampaign
- Validar: stages ≥ 1, target_filter válido, owner_id.
- Validar quota (campanhas ativas simultâneas por plano).
- Aplicar filter inicial, criar entries.
- Status = `active`.
- Job de dispatch começa a processar.

### PauseCampaign
- Não cria novos entries.
- Entries atuais aguardam na stage (timers congelam).

### EditCampaign em execução
- Edição parcial permitida: adicionar stages no fim, editar texto de stages não-executadas.
- Alteração no filtro de audiência só afeta entradas futuras.
- Remoção de stage não-executada permitida; stages com entries ativos requerem migração dos entries.

### CancelCampaign
- Stop imediato. Entries ativos marcados como `exited` com `reason=cancelled`.

## Integrações

- **Pipeline**: entry de campanha pode empurrar lead para pipeline específico ao final.
- **Agente IA**: conversação quando lead responde.
- **Webhooks de saída**: evento `LeadExitedCampaign` pode notificar endpoint externo.
- **Analytics**: métricas da campanha alimentam dashboards Outbound.
- **Janela de negócio**: respeitada em envios.

## Limites

- Campanhas ativas por org: limite do plano.
- Entries em uma campanha: limite alto (10.000+); para campanhas enormes, job de dispatch processa em lotes.
- Stages por campanha: 20 (soft limit — UI avisa em 10+).

## Segurança e Anti-spam

- Opt-out respeitado: lead com `no-contact` ou que respondeu "PARAR" nunca recebe.
- Rate limit por canal para proteger reputação.
- Monitoramento de bloqueio: se % de numbers bloqueando é alta, pausar e alertar admin.
- Mensagem inicial deve identificar quem fala (nome da empresa) para não parecer spam.
