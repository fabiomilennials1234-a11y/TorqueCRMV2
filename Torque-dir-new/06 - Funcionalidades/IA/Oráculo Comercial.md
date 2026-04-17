---
tipo: feature
dominio: ia
---

# Oráculo Comercial

## Propósito

Assistente IA destinado ao **admin / usuário da plataforma** (não ao lead). Responde perguntas em linguagem natural sobre o **estado da operação**: "quantos leads entraram hoje?", "quem é o melhor SDR do mês?", "qual a taxa de fechamento do produto X?", "temos leads paradoz há mais de uma semana em qual stage?". Permite consultas analíticas sem navegar por múltiplos dashboards.

## Atores e Permissões

- **Admin**: uso primário, pode consultar qualquer dado da org.
- **Membros**: uso limitado aos dados que teriam acesso por permissão (veria apenas próprios).
- **Master**: pode consultar cross-org em contexto de suporte.

Ações: `oraculo.query` (granular por escopo — own_only, team, company).

## Dados Envolvidos

- Entidade temporária `oraculo_query`:
  - `id`, `organization_id`, `user_id`.
  - `question`: texto do usuário.
  - `generated_query` (interno): query estruturada traduzida.
  - `result`: resposta + dados.
  - `duration_ms`.
  - `cost_estimate`.
  - `created_at`.
- Histórico de conversas com Oráculo (opcional — útil para contexto sequencial).

## Regras de Negócio

1. Oráculo tem acesso aos **mesmos dados** que o usuário teria via UI. Respeita permissões.
2. Respostas são **data-driven** — cita números reais, não inventa.
3. Quando a pergunta é ambígua: Oráculo faz perguntas de clarificação.
4. Quando dado não existe: responde "não encontrei dado" explicitamente (não alucina).
5. Consultas são **somente leitura** — Oráculo não altera dados.
6. Auditoria: toda query fica em audit com usuário, pergunta, tipo de dado consultado.
7. Quota por plano: número de queries/mês.

## Arquitetura Conceitual

### Text-to-Query
LLM interpreta pergunta em linguagem natural e gera:
1. Query estruturada (SQL-like ou DSL interna).
2. Fontes de dados a consultar (leads, pipes, analytics views).
3. Agregações (count, sum, avg, group_by).
4. Filtros (org_id sempre implícito, period, user, stage).

Query é executada com escopo automático de tenant/user.

### Formato de Resposta
- Texto narrativo curto que responde diretamente.
- Tabela ou gráfico quando relevante (Oráculo pode propor visualização).
- Link/deeplink para tela onde o usuário pode explorar mais.
- Fonte citada: "Baseado em X leads criados nos últimos 7 dias".

### Segurança
- Query gerada pelo LLM passa por **sanitização estrita** antes de execução.
- Whitelist de tabelas/colunas acessíveis.
- Parâmetros vinculados (nunca concatenação).
- Timeout hard (5-10s) para queries caras.

## Fluxos do Usuário

### Fazer Pergunta
1. Usuário abre Oráculo (ícone no header ou atalho `Ctrl+K`).
2. Campo de texto; placeholder com exemplos ("Quantos leads entraram esta semana?").
3. Enter → envia.
4. Loading (animação sutil).
5. Resposta aparece:
   - Texto.
   - Tabela se dados tabulares.
   - Gráfico se série temporal.
   - Botão "Explorar" → leva a página equivalente.

### Conversa Sequencial
- Pode fazer follow-up: "E comparando com o mês passado?"
- Oráculo mantém contexto da última pergunta.
- Reset com botão "Nova conversa".

### Sugestões Proativas
- Em dashboard, Oráculo pode sugerir insights: "Sua taxa de resposta caiu 15% esta semana comparada à anterior. Quer investigar?"
- Usuário clica → Oráculo gera análise.

### Feedback
- Cada resposta tem 👍/👎 para melhoria contínua.
- Feedback agregado alimenta tuning futuro.

## Perguntas Típicas

- "Quantos leads novos hoje?"
- "Qual o ticket médio deste mês?"
- "Qual SDR teve maior conversão na última semana?"
- "Mostre leads parados no pipe Confirmação há mais de 3 dias."
- "Qual campanha trouxe mais leads qualificados?"
- "Compare performance de Marcos e Ana neste trimestre."
- "Quais produtos apareceram mais em propostas ganhas?"
- "Estou perdendo muitos leads por qual motivo?"
- "Quais leads tem score > 80 mas nenhum responsável atribuído?"
- "Me dê 5 insights sobre minha operação esta semana."

## Edge Cases

- **Pergunta fora do escopo de dados** (ex.: "Por que o Brasil está em recessão?"): Oráculo declina gentilmente, explica escopo.
- **Pergunta ambígua** ("meu melhor vendedor"): pergunta "por qual métrica?" antes.
- **Dado não existe**: "Não encontrei nenhum lead nesta situação".
- **Período inválido** ("ontem que vem"): pede clarificação.
- **Query muito cara** (cross-join imenso): recusa com motivo; sugere filtros.
- **Privacidade**: membro perguntando "leads de outros SDRs" → Oráculo responde apenas os próprios, explica.
- **LLM halucina query**: sanitização bloqueia, retry com instrução mais estrita.

## Integrações

- **Analytics** (views agregadas).
- **Banco de dados do domínio** com escopo restrito.
- **LLM** (provedor para text-to-query + resposta).
- **Audit log** consumível (Oráculo pode responder "quem editou lead X?").
- **UI** (deeplinks para contexto).

## Automações e Eventos

### Emite
- `OraculoQueryExecuted(user_id, question_hash, duration, result_count)`.
- `OraculoQueryFailed(reason)`.

### Reage
- Sugestões proativas: job diário gera 1-3 insights e envia via notificação.

## Performance

- Queries simples: p95 < 3s (LLM + execução).
- Queries complexas: limit 10s; timeout após.
- Cache por `(user_id, question_hash)` de curta duração (1 min) para evitar recomputar em scroll/refresh.

## Validações

- Question: 3-500 chars.
- Rate limit por usuário: X queries/min (evita abuso).
- Escopo de dados: automaticamente limitado por permissão.

## Métricas

- Queries/dia por org.
- Taxa de sucesso (resposta útil vs decline).
- Thumbs up/down.
- Queries top (perguntas mais feitas).
- Latência.
- Custo LLM por query.
- Correlação uso Oráculo ↔ engagement (admins que usam Oráculo fazem outras ações mais).

## Observabilidade

- Log de cada query: pergunta, query gerada, duração, resultado (metadados, não conteúdo sensível).
- Alerta se taxa de erro sobe.
- Dashboard de uso do Oráculo por org (master).

## Ética e Segurança

- Oráculo não emite julgamento subjetivo sobre membros ("Fulano é ruim") — apenas dados.
- Dados pessoais mascarados em respostas amplas (ex.: "10 leads perdidos" em vez de listar nomes a menos que solicitado).
- Rate limit evita custo descontrolado.
- Auditoria completa para compliance.

## Roadmap Conceitual

- V1: Text-to-query básico.
- V2: Conversa sequencial com contexto.
- V3: Proatividade (insights diários).
- V4: Ações (Oráculo pode executar algumas ações com confirmação — "quer atribuir esses leads ao João?").
- V5: Integração com outras features (gerar workflow a partir de prompt).
