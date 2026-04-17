---
tipo: feature
dominio: ia
---

# Lead Score

## Propósito

Pontuação automática `0-100` atribuída a cada lead para priorização. O time trabalha primeiro os leads com maior potencial, sem depender de intuição individual. Algoritmo determinístico (ou hybrid com LLM) baseado em sinais objetivos.

## Atores e Permissões

- **Sistema**: calcula automaticamente.
- **Admin**: configura pesos e sinais.
- **Membros**: veem score; não modificam (exceto o `rating` manual, que é 1-5).

Ações: `lead.view_score`, `lead.edit_score_weights` (admin).

## Dados Envolvidos

- `qualification_score` no Lead (0-100).
- `score_last_calculated_at`.
- `score_breakdown` (opcional, JSON explicativo: quais sinais contribuíram quanto).
- Configuração da org: `lead_score_weights` (JSON em `custom_settings`).

## Sinais Considerados (categorias)

### Dados do Lead
- **Completude do cadastro**: tem nome? empresa? email? telefone? custom fields chave?
- **Segmento fiscal** (se org usa): Ouro/Prata/Bronze via tag.
- **Porte de empresa**: tamanho (funcionários, faturamento se disponível).
- **Posição/cargo**: decisor vs executor.
- **Região**: se org foca em região específica.

### Origem e Atribuição
- **Origem** (meta_ads_ICP vs formulario_geral): pesos por canal.
- **UTMs campaign**: campanhas com histórico de conversão têm peso maior.
- **Referral** (indicado por cliente atual): peso alto.

### Comportamento (interação)
- **Respondeu em <X min** à primeira msg.
- **Fez pergunta específica sobre produto**.
- **Mencionou interesse urgente** (palavras-chave).
- **Engajou com link enviado**.
- **Já consumiu conteúdo gratuito da empresa**.

### Tempo
- **Freshness**: lead novo (< 24h) tem score boost.
- **Decay**: lead sem interação há > 7 dias perde score.

### Dados Externos (opcional)
- **Pesquisa de mercado**: se integração com Clearbit/serviço de enrichment.
- **Histórico de compra**: repeat customer (de base de clientes atual).

## Algoritmo Conceitual

```
score = soma_ponderada(sinais, pesos) normalizada para [0, 100]

Cada sinal:
- Extrai valor observado (boolean, numérico, categoria).
- Mapeia para contribuição (-20 a +30, por exemplo).
- Multiplicado pelo peso configurado.

Clamp a [0, 100].
Arredonda.
```

Exemplo simples:

```
score_base = 50
+ 10 se origem in [meta_ads_ICP, referral]
+ 15 se respondeu em < 10 min
+ 10 se empresa preenchida
+ 5 por cada FAQ relevante consultada
+ 15 se tag "Ouro"
- 10 se sem resposta em 24h
- 5 se sem email
- 10 se sumiu (>7 dias sem msg)
clamp 0-100
```

Admin configura via UI: cada sinal tem slider de peso (0-10x).

## Fluxo de Cálculo

### Recálculo
- **Na criação do lead**: score inicial calculado (só dados estáticos).
- **A cada evento relevante**: score recalculado:
  - Mensagem recebida.
  - Tag adicionada/removida.
  - Stage alterado.
  - Custom field mudado.
  - Job periódico (a cada hora): recalcula decay temporal.
- Emite `LeadScoreRecalculated(before, after, breakdown)` se mudou.

### Priorização na UI
- Lista de leads ordena por score DESC por default (em visão "próximos a trabalhar").
- Kanban mostra badge de score no card.
- Filtros "score > 70" disponíveis.

## Variação: Score com LLM

Versão avançada:
- LLM consome resumo do lead + histórico de conversa.
- Retorna score com justificativa.
- Custo maior; usado para leads ambíguos.

Implementação híbrida:
- Algoritmo determinístico calcula base.
- LLM ajusta ± 10-20 pontos para leads em zona cinzenta (40-70).

## Regras de Negócio

1. Score inicial: default 50 se sem sinais.
2. Clamp estrito [0, 100].
3. `rating` manual (1-5) é SEPARADO e não afeta score automático.
4. Admin pode override manualmente (ação explícita + audit).
5. Breakdown disponível para debugging (UI expansível).
6. Configuração de pesos requer admin + aprovação.
7. Alterar pesos recalcula todos os leads em background.

## Fluxos do Usuário

### Ver Score
- Card do lead mostra badge: cor (verde alto / amarelo médio / vermelho baixo) + número.
- Click no badge → breakdown.

### Configurar Pesos (admin)
1. `Configurações → Lead Score`.
2. Lista de sinais com slider de peso.
3. Preview de "simulação" com alguns leads reais.
4. Salvar → dispara recálculo em background.

### Override Manual
- Admin em drawer do lead → "Ajustar score manualmente".
- Form: novo valor + motivo.
- Registra como ajuste manual; algoritmo não sobrescreve.
- Flag `score_locked` para impedir recálculo.

## Automações e Eventos

### Emite
- `LeadScoreRecalculated(lead_id, before, after, breakdown)`.

### Reage
- Qualquer evento que pode afetar score: `LeadCreated`, `LeadUpdated`, `LeadTagAdded/Removed`, `LeadStageChanged`, `MessageReceived`.
- Cron horário: decay.

### Workflow
- Trigger `lead_score_crossed(threshold)`: dispara quando score cruza valor (ex.: > 80 → atribui a sênior).

## Integrações

- **Todos os pontos que modificam lead** emitem eventos consumidos.
- **Distribuição**: pode usar score para priorizar leads para seniors.
- **Analytics**: score distribution, correlação com fechamento.

## Edge Cases

- **Lead novo sem nenhum sinal**: score default (50).
- **Lead com dados conflitantes** (ex.: origem A=pouco qualificado mas cargo=CEO): soma ponderada resolve; cada sinal adiciona independente.
- **Mudança de pesos**: recálculo em background não é instantâneo (minutos a horas em org grande).
- **Score manual + recálculo**: flag `score_locked` respeitada; admin pode destravar.
- **Tag deletada**: sinal some; lead perde contribuição; recálculo.

## Validações

- Pesos em [0, 10].
- Score final em [0, 100].
- Override manual: justificativa mínima 10 chars.

## Métricas

- Distribuição de score (histograma).
- Correlação score × conversão (leads score > 80 fecham quanto %?).
- Média de score por origem, por campanha, por SDR.
- Lift: leads trabalhados em ordem de score vs FIFO → diferença de conversão.
- Accuracy do modelo (se temos LLM validando).

## Performance

- Cálculo determinístico é barato (< 10ms).
- Batch recálculo: job paralelo por partition de org.
- Breakdown armazenado (opcional) para evitar recomputar sob demand.

## UX

- Badge de score com cor clara.
- Tooltip com breakdown.
- Filtros "score >= X".
- Sort por score em qualquer lista.
- Widget no dashboard "Leads top 10 por score".
