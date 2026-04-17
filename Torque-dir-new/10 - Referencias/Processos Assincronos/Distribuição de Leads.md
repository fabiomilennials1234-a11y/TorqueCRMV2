---
tipo: async
---

# Distribuição de Leads

Processo que **atribui automaticamente** leads novos aos membros do time conforme regra configurada. Crítico para equilíbrio de carga, rapidez de resposta, e justiça na distribuição.

## Quando Dispara

- Lead criado sem `responsible_id`/`sdr_id` informado.
- Regra de pipe configurada para distribuição em stage específica.
- Regra de campanha.

## Fluxo

```
Lead criado (ou entrou em stage)
        │
        ▼
Regra de distribuição ativa para este pipe/stage?
   ├── Não → lead fica unassigned (visível em view dedicada)
   └── Sim ▼
           │
           ▼
   Tipo da regra:
     ├── round_robin
     ├── load_based
     ├── weighted
     ├── tag_based (leads com tag X → grupo específico)
     └── manual (admin distribui depois)

           │
           ▼
   Seleciona membros elegíveis:
     - is_active=true
     - specialty compatível
     - não em vacation/off
     - capacidade disponível (se load_based)

           │
           ▼
   Aplica algoritmo:
     - Escolhe próximo membro
     - Atualiza state persistente (para round_robin)

           │
           ▼
   Atribui:
     - lead.sdr_id (ou closer_id, responsible_id)
     - Emite LeadAssigned(member_id)
     - Notifica membro (push/email)
```

## Algoritmos

### Round-Robin
```
Estado: último membro atribuído (persistente).
Próximo = membros_elegiveis[(index_anterior + 1) % len].
Atualiza estado.
```
Simples, justo, previsível.

### Load-Based
```
Para cada membro elegível:
  contagem = count(leads ativos atribuídos nos pipes relevantes)
Próximo = membro com menor contagem.
Empate: round-robin entre empatados.
```
Leva em conta carga real; mais justo quando membros estão desbalanceados.

### Weighted
```
Pesos configurados (ex.: A:50%, B:30%, C:20%).
Sorteio ponderado.
```
Útil quando membros têm capacidade diferente (sênior vs júnior).

### Tag-Based
```
Regras: se lead tem tag X → grupo A;
         se tag Y → grupo B;
         senão → default.
Aplica round-robin dentro do grupo escolhido.
```
Permite priorizar leads premium para seniors.

### Manual
- Sem distribuição automática.
- Leads ficam `unassigned`.
- Admin distribui via UI.

## Configuração por Pipe

Admin define em `Configurações → Pipeline → [Pipe] → Distribuição`:
- Tipo.
- Members elegíveis (por specialty + whitelist/blacklist específico).
- Trigger: `on_enter_pipe` ou `on_stage_change:stage_X`.
- Ativo/inativo.

## Regras de Negócio

1. Membros inativos não recebem.
2. Lead com `assigned_user_id` vindo no payload de ingestão: respeitado (skip distribuição).
3. Round-robin state é per-tenant per-regra.
4. Se nenhum membro elegível online: lead fica unassigned; admin é notificado.
5. Reatribuir lead existente: caso de uso separado (não é distribuição automática).

## Observabilidade

- Log por atribuição: `lead_id`, `member_id`, `rule_id`, `algorithm`.
- Métricas:
  - Atribuições/hora por membro.
  - Balance (% de desbalanceamento).
  - Tempo de primeira resposta após atribuição (proxy de qualidade).
- Alertas:
  - Membro recebendo muito mais que outros (sinal de regra ruim).
  - Leads unassigned crescendo (sinal de equipe insuficiente ou off).

## Edge Cases

- **Todos os membros offline** (fim de semana sem plantão): unassigned. Workflow pode enviar mensagem automática "Retornaremos em breve".
- **Membro volta de férias**: se disponibilidade configurada, entra de novo na distribuição.
- **Novo membro**: deve ser adicionado à regra elegível; alternativamente, distribuição automática detecta novos membros ativos.
- **Alteração de regra durante lead ativo**: afeta apenas atribuições futuras.
- **Lead com telefone errado** é atribuído mesmo assim (sem avaliação de qualidade).

## Reatribuição

Caso distinto da atribuição inicial:
- Admin altera `responsible_id` manualmente.
- Workflow action `assign_responsible` reatribui.
- Regra de pipe pode reatribuir em transições (ex.: `on_enter(confirmacao) → assign_closer`).

Eventos: `LeadReassigned(from, to, actor)`.

## Capacidade e Disponibilidade

### Disponibilidade
- Membro pode ter `availability_mode`: always_available, business_hours_only, manual_on/off.
- Status manual: "Disponível" / "Ausente" (botão na UI).
- Distribuição respeita.

### Capacidade
- Membro pode ter `max_active_leads`: limite de leads atribuídos ativos.
- Load-based respeita; round-robin pode pular se atingido.

## Integração com Outras Features

- **Lead**: atualização de campos.
- **Time**: members elegíveis.
- **Pipeline**: regra por pipe.
- **Notificações**: membro atribuído.
- **Analytics**: balance, performance por membro.

## Justiça

- Round-robin é justo em tempo.
- Load-based é justo em carga real.
- Weighted reflete diferença objetiva de capacidade.
- Admin deve escolher o que faz sentido para a operação.

## Pitfalls

- **Round-robin sem check de availability**: atribui a membro offline → atraso.
- **Weighted mal-calibrado**: júnior sobrecarregado.
- **Tag-based confuso**: leads escapam.
- **Sem fallback**: lead unassigned perpetuamente.
