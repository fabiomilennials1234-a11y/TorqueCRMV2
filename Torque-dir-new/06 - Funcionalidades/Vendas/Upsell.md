---
tipo: feature
dominio: vendas
---

# Upsell

## Propósito

Módulo dedicado à **identificação e conversão de oportunidades de upsell/cross-sell** em clientes existentes (leads vendidos). Operacionaliza o processo pós-venda: quando um cliente atinge marcos (uso, tempo, eventos), cria oportunidade de vender mais.

## Atores e Permissões

- **Admin**: configura regras de upsell, vê dashboard.
- **Closer/SDR específico**: trabalha oportunidades.
- **Copilot específico** (agente IA de upsell): pode conduzir a conversa inicial.

Ações: derivadas de `pipeline.*` (Upsell geralmente é implementado como pipeline customizado) + flags próprias.

## Dados Envolvidos

Implementação típica: **pipeline customizado "Upsell"** com stages:
1. `oportunidade_identificada`.
2. `contato_iniciado`.
3. `proposta_upsell_enviada`.
4. `negociando`.
5. `ganho_upsell` (positivo).
6. `rejeitado` (negativo).
7. `postergado` (lead pediu voltar depois).

Cada entry carrega `meta`:
- `trigger`: o que disparou a oportunidade (regra aplicada — ex.: "90 dias desde compra", "uso atingiu 80% da quota").
- `current_products`: produtos que cliente já tem.
- `proposed_products`: o que queremos adicionar.
- `potential_value`: valor da oportunidade.
- `related_original_lead_id` (apenas para clareza — geralmente é o próprio lead).

## Regras de Negócio

1. Oportunidade só é gerada para leads com status `vendido` em algum momento (pipe Propostas).
2. Triggers configuráveis pelo admin:
   - **Temporal**: N dias após compra.
   - **Uso**: métrica de uso do cliente cruza threshold (requer integração com sistema do cliente).
   - **Evento**: renovação próxima, aniversário da venda, produto complementar lançado.
   - **Manual**: admin/closer identifica.
3. Cliente pode ter múltiplas oportunidades ativas simultaneamente (produtos diferentes).
4. Opt-out respeitado: cliente que pediu "não recebo ofertas" não gera nova oportunidade automática.
5. Ganho upsell dispara:
   - Nova proposta em Pipeline Propostas (ou atualização da existente se política assim).
   - Cálculo de comissão.
   - Event `UpsellWon`.

## Fluxos do Usuário

### Configurar Regra de Upsell
1. Admin abre `Configurações → Upsell → Regras`.
2. Cria regra:
   - Nome.
   - Trigger (temporal / uso / evento / manual).
   - Filtros: tag, produto comprado, segmento, ticket mínimo, última compra.
   - Output: stage inicial do Upsell, agente IA opcional, template de mensagem.
3. Ativa.

### Geração Automática de Oportunidades
- Job periódico (ex.: diário às 9h) avalia leads elegíveis.
- Para cada lead que satisfaz regra e não tem oportunidade ativa: cria entry no pipe Upsell stage `oportunidade_identificada`.
- Atribui conforme distribuição configurada.

### Trabalhar Oportunidade
1. Membro abre pipeline Upsell.
2. Drag lead para `contato_iniciado` + dispara mensagem (template).
3. Lead responde → move conforme interesse.
4. Negociação → proposta → ganho ou rejeitado.

### Ganho
1. Marca `ganho_upsell`.
2. Sistema:
   - Cria/atualiza entry em Pipeline Propostas com novo item.
   - Calcula comissão (possivelmente com regra diferente — comissão de upsell costuma ser menor).
   - Emite `UpsellWon`.
   - Pode enviar mensagem automatizada de agradecimento.

### Rejeitado / Postergado
- `rejeitado`: motivo obrigatório; lead mantém segmento; pode re-entrar em nova regra no futuro.
- `postergado`: entry pausa, follow-up automático em N meses.

## Automações e Eventos

### Emite
- `UpsellOpportunityCreated`, `UpsellWon`, `UpsellRejected`, `UpsellPostponed`.

### Reage
- `LeadSold` em Pipeline Propostas → cliente vira "base ativa" → avaliado por regras.
- Cron diário → avalia regras temporais.

## Integrações

- **Pipeline Propostas**: destino do ganho upsell.
- **Comissões**: regra específica para upsell (geralmente menor %).
- **Copilot**: agente dedicado pode conduzir.
- **Campanhas**: upsell pode ser uma campanha se for broadcast (ex.: "lançamento do Produto Y para base").
- **Analytics**: receita de upsell vs primeira venda (expansion revenue).

## Edge Cases

- **Cliente em inadimplência**: não gera oportunidade (filtro padrão).
- **Cliente pediu cancelamento da conta**: filtro bloqueia.
- **Produto complementar ainda não lançado**: regra em estado `draft`.
- **Múltiplas regras matcham mesmo lead**: prioridade configurável (ex.: maior potential_value primeiro, ou round-robin).
- **Re-venda** (cliente renova contrato): distinguir upsell (novo produto) vs renovação (mesmo produto) — campos diferentes.

## Validações

- Trigger config válido.
- Filtros coerentes.
- Agente IA referenciado ativo.
- Template existe.

## Métricas

- **Expansion revenue**: receita gerada por upsell / receita total.
- **Taxa de ganho upsell**: ganho / oportunidade criada.
- **Tempo médio de ciclo**: identificação → ganho.
- **Por regra**: performance (quais regras convertem mais).
- **Ticket médio de upsell**.
- **LTV aumentado**: lead sold + upsells acumulados.

## Observações

- Upsell é **pipeline customizado especializado** — implementado em cima da infraestrutura de pipelines, não como entidade nova.
- Regras de upsell são semelhantes a triggers de workflow, mas com foco específico em oportunidade comercial.
- Integração com sistema do cliente (para detectar uso/thresholds) é opcional e depende de disponibilidade de API do cliente.
