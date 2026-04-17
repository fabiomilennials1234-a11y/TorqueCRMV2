---
tipo: feature
dominio: analytics
---

# Ranking

## Propósito

Lista ordenada dos membros por **métrica selecionada**. Instrumento de gamificação e transparência (quando visibilidade permite). Alimenta premiações automáticas do tipo "top X".

## Atores e Permissões

- **Admin**: vê tudo.
- **Membros**: veem ranking conforme `visibility` da organização (transparency toggle).

Ações: `analytics.view:team` (admin) ou herdada de `analytics.view:own` quando toda organização opta por transparência.

## Dados / Métricas Ranking

Métricas suportadas (selecionáveis):
- Leads qualificados.
- Reuniões agendadas.
- Reuniões realizadas.
- Propostas enviadas.
- Vendas fechadas.
- Receita gerada.
- Ticket médio.
- Taxa de conversão.
- Taxa de resposta.
- Follow-ups concluídos no prazo.
- Pontos (soma ponderada configurável para gamificação).

## Filtros

- Período (dia, semana, mês, trimestre, ano, custom).
- Métrica.
- Specialty (ranking apenas de SDRs, apenas de Closers).

## Visualizações

### Ranking Clássico (lista)
- Posição (🥇🥈🥉), avatar, nome, valor da métrica, delta vs semana anterior.

### Top N Card
- Destaque em pódio visual para top 3.

### Comparação Temporal
- Gráfico de linha: posição no ranking de cada membro ao longo do tempo.
- Mostra ascenções e quedas.

## Regras de Negócio

1. Membros inativos: excluídos por default (toggle para incluir com snapshot).
2. Empate: regra de desempate (ex.: menor tempo de primeira resposta, mais antigo na empresa).
3. Transparency config: admin decide se membros veem ranking completo, apenas top 5, apenas própria posição, ou zero.
4. Rankings informais (não vinculados a reward) podem ser divulgados livremente.
5. Rankings vinculados a reward: anunciados oficialmente no fim do período.

## Fluxos do Usuário

### Ver Ranking
1. Menu → `Ranking` (ou widget no dashboard).
2. Filtros: métrica, período.
3. Visualiza lista.
4. Click em membro → Performance Individual.

### Configurar (admin)
1. Admin em `Configurações → Ranking`.
2. Define:
   - Transparência (quem vê o quê).
   - Métricas visíveis.
   - Períodos padrão.
   - Pontos (se gamificação ativa).

## Automações

### Emite
- `RankingPeriodClosed` (fim do período).

### Reage
- Reward trigger `rank_top_x` consome ranking fechado.

## Integrações

- **Metas**: podem incluir "ser top N no ranking".
- **Premiações**: reward automático para top.
- **Dashboard**: widget.
- **TV Dashboard**: pode exibir ao vivo.
- **Performance Individual**: deep link.

## Edge Cases

- **Membro sem atividade**: aparece com 0 ou não aparece (config).
- **Mudança de specialty durante período**: atividade conta conforme specialty no momento do evento.
- **Mudança de hiring** (entrou no meio do período): ajuste pro-rata opcional.
- **Manipulação de dados** (ex.: lead criado artificialmente): prevenção via audit + sanity checks.

## Validações

- Métrica válida.
- Período válido.

## Métricas Secundárias

- Variação de posição semana a semana.
- Consistência (quem está sempre no top — merece destaque especial).

## UX

- Confete ou animação para top 3.
- Dark-first.
- Opção "ver meu progresso" para quem não está no top.
- Export do ranking para anúncio interno.
