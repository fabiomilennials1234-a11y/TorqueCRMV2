---
tipo: fluxo
---

# Campanha Outbound

Fluxo completo de uma campanha outbound: configuração → ativação → distribuição → envio de sequência → monitoramento → conclusão. Combina Campanhas, Chat, Copilot, Pipelines, Analytics Outbound.

## Objetivo Típico

Prospectar ou reengajar lista de leads com sequência de mensagens outbound em horizonte determinado — ex.: "Reativar leads perdidos há > 60 dias com oferta especial" em 15 dias.

## Fase 1: Configuração

### 1.1 Criar Campanha
Admin/prospectador abre wizard:
- Nome: "Reativação B2B Industrial Abril".
- Objetivo: reativar leads perdidos com oferta.
- Período: 2026-04-15 a 2026-04-30.
- Público-alvo (filtro):
  - Tags: `B2B-Industrial` AND NOT `opt-out`.
  - Stage em Propostas = `perdido`.
  - `lost_at` há mais de 60 dias.
- Snapshot vs dinâmico: dinâmico (novos perdidos qualifying entram também).
- Distribuição: `round_robin` entre prospectadores.
- Agente IA: "Prospectador Q2-2026" (template prospectador + business context).

### 1.2 Sequência de Stages
```
Stage 1: Abordagem inicial (imediato)
  Template: "Olá {{lead.name}}! Aqui é da {{org.name}}. Notei que conversamos há um tempo. Temos novidade..."
  Exit: lead respondeu.

Stage 2: Seguir silêncio (24h sem resposta)
  Template: "Só quero confirmar que recebeu..."
  Exit: lead respondeu.

Stage 3: Oferta (48h)
  Template: "Preparei uma oferta especial válida até X..."
  Exit: lead respondeu.

Stage 4: Última tentativa (96h)
  Template: "Última oportunidade..."
  Exit: lead respondeu.

Stage 5: Encerramento (168h=7d)
  Ação: adicionar tag `campanha-2026-04-sem-resposta`.
  Ação: mover para pipe custom `Reciclagem`.
  Exit: encerramento forçado.
```

### 1.3 Exit Conditions Globais
- Lead responde → sai + volta para pipe WhatsApp `respondeu`.
- Lead aceita reunião → sai + vai para Confirmação.
- Lead compra → tag `convertido-campanha-Q2` + sai.
- Lead opta por STOP → tag `opt-out` + exit.
- Lead bloqueia canal → exit.

### 1.4 Metas
- Team goal: 50 respostas.
- Per-member: 10 respostas cada.

### 1.5 Salvar como draft

## Fase 2: Ativação

### 2.1 Admin valida
- Preview: "347 leads matcham o filtro".
- Simulação de dispatch: 100 leads/dia × 4 dias = cobertura inicial.
- Rate limit confirmado (50 msg/h por canal).

### 2.2 Ativar
- Click "Ativar".
- Sistema:
  - Aplica filter → cria entries em stage 1 para 347 leads.
  - Distribui: round_robin entre 3 prospectadores = ~115 leads cada.
  - Status = `active`.
  - Job de dispatch começa a consumir entries.

## Fase 3: Execução

### 3.1 Dispatcher Roda
Job a cada minuto:
- Lê entries com `next_dispatch_at <= now` e `status=active`.
- Respeita:
  - Rate limit por canal (max 50/h).
  - Janela de negócio (não envia à noite se configurado).
  - Distribuição (cada prospectador processa seus).
- Para cada entry devido:
  - Executa stage (envia mensagem ou ação).
  - Atualiza `next_dispatch_at` para próxima stage.
  - Se exit condition: finaliza com reason.

### 3.2 Rate Limiting
- Instância de canal tem limite.
- Dispatcher distribui envios ao longo do tempo.
- Se limite atingido: entries aguardam próxima janela.

### 3.3 Lead Responde
- Inbound chega.
- Sistema detecta que lead está em campanha ativa.
- Agente IA assume conversa (se configurado).
- Agente segue Kanban Rules da campanha + business context.
- Objetivos: qualificar interesse, agendar reunião, ou identificar desinteresse.

### 3.4 Conversão
- Agente qualifica → agenda reunião.
- Entry da campanha `exited` com reason `scheduled_meeting`.
- Lead volta para pipe WhatsApp stage `agendado`.
- Segue fluxo padrão.

### 3.5 Desistência
- Lead responde "PARAR" ou similar.
- Agente detecta → aplica tag `opt-out` + `pause_conversation`.
- Entry `exited` com reason `opted_out`.

### 3.6 Bloqueio
- Mensagens sucessivas falham (lead bloqueou canal).
- Entry `exited` com reason `blocked`.
- Canal tem métrica; se % de bloqueio sobe acima de threshold: pausa automática da campanha.

## Fase 4: Monitoramento

### 4.1 Dashboard
Admin/prospectador monitora:
- Total enrolados: 347.
- Ativos: 287 (60 já saíram).
- Respostas: 42 (12%).
- Convertidos: 8 (agendaram reunião).
- Vendas até agora: 2 (ticket médio R$ 10k).
- Taxa de bloqueio: 1.2% (dentro do aceitável).
- Metas: time em 84% da meta.

### 4.2 Performance por Membro
- Prospectador A: 45 msg enviadas, 18 respostas.
- Prospectador B: 42 msg, 15 respostas.
- Prospectador C: 38 msg, 9 respostas.

### 4.3 Performance por Stage
- Stage 1: 12% de resposta.
- Stage 2: 7% adicional.
- Stage 3: 3% adicional.
- Stage 4: 1%.
- Maior parte em stage 1 — bom sinal.

### 4.4 Ajustes
- Admin identifica: Stage 3 (oferta) pouco efetiva.
- Edita: novo template com urgência maior.
- Aplica só a novos entries.

## Fase 5: Conclusão

### 5.1 Natural
- `end_date` atingida.
- Entries ativos finalizam com reason `campaign_ended`.
- Status = `completed`.

### 5.2 Resultados Finais
- 347 enrolados.
- 68 respostas (19.5%).
- 22 reuniões agendadas.
- 7 vendas (R$ 85k total).
- Custo: 2 horas do prospectador × 3 = 6 horas de setup + monitoramento.
- ROI: R$ 85k / (6h × R$150/h = R$900) = 94× (incrível, mas realístico para campanhas bem-orientadas).

### 5.3 Lições
- Admin documenta em Obsidian interno (post-mortem da campanha).
- Templates que funcionaram bem viram biblioteca.
- Agente IA é melhorado com FAQs adicionadas.

## Regras Importantes

1. Lead opt-out globalmente: NUNCA enrolado.
2. Respeitar rate limit: canal banindo = perda reputacional catastrófica.
3. Janela de negócio respeitada (lead não receber às 23h).
4. Duplicata de enrolamento: impedida.
5. Agente IA identifica opt-out implícito ("me deixa em paz") e age.

## Métricas-chave

- Taxa de resposta.
- Taxa de conversão (agendamentos).
- ROI (receita / esforço).
- Taxa de bloqueio (reputacional).
- Performance por template (para iteração).

## Pitfalls Comuns

- **Filter muito amplo**: manda para leads desinteressados → bloqueios.
- **Template genérico**: taxa de resposta baixa.
- **Rate limit ignorado**: número banido.
- **Sem follow-up humano em respostas complexas**: perde oportunidade.
- **Sem monitoramento**: campanha ruim passa despercebida.

## Otimizações

- **Teste A/B de templates**: split_ab em stage 1.
- **Áudio em stage 2**: humaniza.
- **Personalização**: usar custom fields ({{lead.budget}}) quando disponível.
- **Timing**: enviar em horário específico (ex.: 11h terça costuma ser melhor).
- **Agente sofisticado**: responde com qualidade, não robótico.

## Eventos Emitidos

- `CampaignActivated`.
- `LeadEnrolledInCampaign` (×347).
- `LeadCampaignStageChanged` (progressão).
- `MessageSent` (outbound, centenas).
- `MessageReceived` (inbounds).
- `LeadExitedCampaign(reason)` (para cada).
- `CampaignCompleted`.
