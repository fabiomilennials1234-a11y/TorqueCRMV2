---
tipo: feature
dominio: automacao
---

# Campanhas

## Propósito

Esforço de comunicação **pontual e direcionado** com objetivo, prazo, público-alvo e sequência de mensagens. Paralelo aos pipelines — lead pode estar em campanha E em pipe ao mesmo tempo. Suporta outbound proativo (disparo para lista) e reengajamento de inativos.

## Atores e Permissões

- **Admin** e **Prospectador**: criam, ativam, gerenciam.
- **Membros gerais**: veem dashboard da campanha se atribuídos leads.
- **Copilot** (se campanha tem agente): atua em conversas quando lead responde.

Ações: `campaign.view`, `campaign.create`, `campaign.edit`, `campaign.activate`, `campaign.pause`, `campaign.delete`.

## Modelo Conceitual

Ver [[02 - Modelo de Domínio/Campanha]]. Resumo:

- **Campanha**: nome, objetivo, período, agente, filtro de público-alvo, distribuição, metas.
- **Stage de Campanha**: etapa na sequência (ex.: "Abordagem inicial", "Follow-up 1", "Última tentativa").
- **Entry de Campanha**: lead específico em stage específica da campanha.

## Fluxos do Usuário

### Criar Campanha
1. Menu → `Campanhas → Nova`.
2. Wizard:
   - **Nome e Objetivo** (texto).
   - **Período**: data início, data fim (opcional).
   - **Público-alvo**: filtros (tags, pipeline, stage, origem, período de criação, inatividade, score, custom fields).
   - **Snapshot vs Dinâmico**: snapshot = leads atuais; dinâmico = novos que passam a satisfazer entram.
   - **Distribuição**: round_robin entre SDRs, load_based, weighted, manual, none (agente).
   - **Agente IA** (opcional): qual agente atende respostas.
   - **Sequência de Stages**: adiciona N stages com `wait_hours`, template de msg, exit_conditions.
   - **Metas**: team goal, per-member goal.
3. Preview: quantos leads match filter? Simulação de dispatch.
4. Salvar como `draft`.

### Ativar
1. Click "Ativar".
2. Sistema:
   - Aplica filter ao universo de leads atual.
   - Cria entries na stage inicial.
   - Distribui entre membros (se aplicável).
   - Status=`active`.
   - Inicia job de dispatch.

### Pausar / Retomar
- Pausar não cria novos entries; entries ativos congelam (timers param).
- Retomar continua.

### Editar Ativa
- Pode: adicionar stages no fim, editar conteúdo de stages futuras, adicionar exclusions.
- Não pode: remover stages com entries em progresso sem migrar.

### Encerrar
- Manual: admin marca `completed`.
- Automático: `end_date` atingida + nenhum entry ativo.
- Entries em progresso: finaliza com exit_reason=`campaign_ended`.

### Dashboard da Campanha
- Visão geral com:
  - Total enrolados, ativos, completed, exited.
  - Funil de stages (quantos em cada).
  - Taxa de resposta.
  - Leads convertidos (exit por responder = conversão).
  - Performance por membro.
  - Distribuição de motivos de exit.

## Regras de Negócio

1. Lead opt-out (tag `no-contact`) nunca enrolado.
2. Lead já enrolado em outra campanha ativa: sistema pode bloquear (default) ou permitir (config).
3. Respeita janela de negócio em envios (se configurado).
4. Rate limit por instância de canal (ex.: 100 msgs/hora por número WhatsApp).
5. Pausa automática se taxa de bloqueio do canal excede threshold (proteção reputacional).
6. Distribuição respeita membros ativos; se todos offline, entries esperam (config).
7. Duplicata de enrolamento: impedida — se lead já em campanha, não recria entry.
8. Agente IA só atende se lead responde; disparo inicial não é interativo.

## Sequência de Stages (exemplo)

```
Campanha: "Reativar leads perdidos com oferta"
Período: 2026-04-15 a 2026-04-30
Público-alvo: leads com stage=perdido há > 60 dias, tag "B2B Industrial"
Distribuição: round_robin entre prospectadores

Stage 1 "Abordagem" (dispara imediatamente):
  Template: "Olá {{lead.name}}, faz um tempo que conversamos. Temos uma novidade..."
  Exit: lead responde.

Stage 2 "Follow-up A" (24h após stage 1 sem resposta):
  Template: "Só queria confirmar que chegou a mensagem anterior..."
  Exit: lead responde.

Stage 3 "Oferta" (48h após stage 2 sem resposta):
  Template: "Temos uma oferta especial válida até X..."
  Exit: lead responde.

Stage 4 "Desistência" (120h depois):
  Ação: adicionar tag "Campanha-X-sem-resposta"; mover para pipe "Reciclagem"; encerrar entry.
```

## Exit Conditions (por stage)

- Lead respondeu → sai + volta para pipe natural (ex.: pipe WhatsApp `respondeu`).
- Lead aceitou reunião → sai + vai para Confirmação.
- Lead pediu PARE/STOP → sai + tag `opt-out`.
- Lead converteu (comprou) → sai + tag `convertido-via-campanha-X`.
- Campanha encerrou → sai com reason `campaign_ended`.

## Distribuição entre Membros

### Algoritmos
- **round_robin**: estado persistente, próximo lead vai para próximo da lista de membros ativos com specialty X.
- **load_based**: membro com menos entries ativos.
- **weighted**: pesos configurados (ex.: A:50%, B:30%, C:20%).
- **tag_based**: leads com tag X → membros sênior.
- **manual**: admin atribui manualmente um a um (entries ficam em "aguardando atribuição").
- **none**: agente IA assume; sem membro humano dedicado.

## Agente IA em Campanhas

- Ao configurar, seleciona agente existente.
- Agente conhece campanha no system_prompt (objetivo, oferta, etc.).
- Responde às respostas dos leads.
- Se atinge exit_condition, move lead adequadamente.

## Automações e Eventos

### Emite
- `CampaignCreated`, `CampaignActivated`, `CampaignPaused`, `CampaignCompleted`, `CampaignArchived`.
- `LeadEnrolledInCampaign`.
- `LeadCampaignStageChanged`.
- `LeadExitedCampaign(reason)`.

### Dispatcher
- Job assíncrono a cada minuto:
  - Lê entries com `next_dispatch_at <= now`.
  - Executa stage (envia mensagem ou ação).
  - Avança para próxima stage ou encerra.
  - Respeita rate limit e janela de negócio.

## Integrações

- **Chat**: envio de mensagens.
- **Agente IA**: atendimento.
- **Pipeline**: destino em exit.
- **Analytics**: métricas de outbound (Dashboard Outbound).
- **Janela de negócio**.
- **Tags**.
- **Webhooks externos**: evento de exit pode notificar sistemas externos.

## Edge Cases

- **Lead muda de número durante campanha**: mensagens subsequentes podem falhar; sistema detecta via webhook, alerta admin.
- **Canal bloqueado pelo lead**: detecta via webhook, marca entry como `exited` com reason `blocked`.
- **Rate limit excedido**: entries esperam próxima janela sem falhar.
- **Campanha muito grande** (50k entries): dispatch em lotes controlados.
- **Edit em stage com entries ativos**: sistema pede confirmação; nova versão aplica aos novos entries.
- **Filtro dinâmico + lead deixa de satisfazer filtro durante campanha**: entry ativo permanece (já enrolado); entradas novas não ocorrem.
- **Template com placeholder inexistente no lead**: fallback vazio + audit warning.

## Validações

- Período válido.
- Filtro produz > 0 leads.
- Ao menos 1 stage.
- Templates existem.
- Agente IA ativo (se referenciado).
- Distribuição referencia membros existentes e ativos.

## Métricas

- Leads enrolados.
- Taxa de resposta (entries que responderam / total).
- Taxa de conversão (exit por conversão / total).
- Tempo médio até primeira resposta.
- Performance por stage: taxa de resposta por stage (onde lead responde mais).
- Performance por membro (distribuídos).
- Custo por resposta (se custo de canal + LLM for rastreado).
- Bloqueios recebidos (reputação do canal).

## Segurança e Anti-spam

- Opt-out global respeitado em todos os canais.
- Mensagem inicial identifica empresa (não "robo enviando spam").
- Rate limit impede disparo em massa agressivo.
- Monitoramento de bloqueio: se > 2% dos leads bloqueiam/reportam, pausa automática + alerta.
- Cliente pode configurar máx N mensagens/dia por campanha.
