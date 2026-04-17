---
tipo: feature
dominio: analytics
---

# Performance Individual

## Propósito

Visão **focada em um membro específico** (próprio ou, se admin, de qualquer membro do time). Responde: "como está meu desempenho vs meta, vs time, vs histórico?"

## Atores e Permissões

- **Membro**: vê a própria.
- **Admin**: vê de qualquer membro.

Ações: `analytics.view:own`, `analytics.view:team`.

## Widgets

### Header
- Avatar, nome, especialização, tempo de casa.
- Status da meta principal (se houver): % atingida, projeção.

### KPIs do membro
- Leads trabalhados.
- Leads qualificados.
- Reuniões agendadas.
- Reuniões realizadas.
- Propostas enviadas.
- Vendas fechadas.
- Receita gerada.
- Ticket médio.
- Taxa de conversão pessoal (leads → vendas).
- Comissão do período (próxima / paga).
- Tempo médio de primeira resposta.
- Follow-ups concluídos no prazo.

### Comparativo vs Média do Time
- Para cada KPI: barra mostrando membro vs média do time.
- Verde se acima, vermelho se abaixo.

### Série Temporal
- Gráfico de atividade (leads trabalhados/dia, vendas/dia).
- Identifica padrões (dias mais produtivos).

### Ranking
- Posição do membro em cada métrica relevante.

### Follow-ups
- Concluídos no prazo vs atrasados.
- Pendentes atuais.

### Atividade Recente
- Feed: últimas N ações do membro.

### Metas
- Metas vinculadas ao membro: progresso visual.

## Métricas

Ver [[Dashboard Principal]] e [[Metas]] para fórmulas — aplicadas com escopo do membro.

## Filtros

- Período.
- Comparar com período anterior.
- Comparar com média do time (toggle).

## Fluxos do Usuário

### Membro vê própria
1. Menu → `Performance` (ou perfil).
2. Filtro default: mês corrente.
3. Vê tudo seu.

### Admin vê de membro específico
1. Menu → `Equipe → [Membro] → Performance`.
2. Ou dashboard principal → click em membro → performance.

### Comparar membros (admin)
1. Seleciona 2-3 membros.
2. View lado a lado.

## Automações

- Sem emissão própria (read-only).

## Integrações

- Leads, Pipelines, Follow-ups, Comissões, Metas, Mensagens.
- Ranking vem do dashboard de Ranking.

## Edge Cases

- **Membro novo** (dados insuficientes): mostra aviso, compara mesmo assim.
- **Membro inativo**: snapshot congelado no dia da inativação.
- **Mudança de specialty**: histórico preservado, novos eventos classificados com nova specialty.

## Performance

- Cache de 5 min.
- Queries indexadas por `member_id`.

## UX

- Layout limpo tipo "scorecard".
- Comparativos visuais claros.
- Link para leads específicos quando relevante.
- Dark-first.

## Motivacional vs Avaliativo

- Autovisão do membro é motivacional (foco em progresso, metas).
- Visão do admin é avaliativa (comparar, identificar desafios).
- UI/tom ajustado por contexto.
