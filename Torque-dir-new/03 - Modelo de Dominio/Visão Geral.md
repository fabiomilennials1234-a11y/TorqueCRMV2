---
tipo: dominio
---

# Modelo de Domínio — Visão Geral

Esta seção descreve o **modelo conceitual** das entidades do Torque — não o schema físico. Cada entidade é listada com seu propósito, atributos conceituais, invariantes e principais relações. A implementação pode escolher livremente como persistir (relacional, documento, grafo), desde que respeite as invariantes.

## Mapa de Entidades

```
                       ┌────────────────┐
                       │  Organização   │
                       │    (Tenant)    │
                       └────┬───────────┘
                            │ (1..N)
            ┌───────────────┼────────────────┬──────────────────┬───────────────────┐
            │               │                │                  │                   │
    ┌───────▼───────┐ ┌─────▼────┐   ┌───────▼──────┐   ┌──────▼──────┐   ┌────────▼───────┐
    │  Time Member  │ │   Lead   │   │  Pipeline    │   │  Workflow   │   │  Campanha      │
    │  (user + role)│ │          │   │  (+ Stages)  │   │  (DAG)      │   │                │
    └───────┬───────┘ └─┬──┬─┬─┬─┘   └───┬──────────┘   └──────┬──────┘   └────────┬───────┘
            │           │  │ │ │         │                     │                   │
            │           │  │ │ │         │                     │                   │
            │           │  │ │ │   ┌─────▼──────┐       ┌──────▼───────┐   ┌───────▼─────┐
            │           │  │ │ │   │   Entry    │       │  Execution   │   │  Distribution│
            │           │  │ │ │   │(LeadInPipe)│       │              │   │              │
            │           │  │ │ │   └────────────┘       └──────┬───────┘   └──────────────┘
            │           │  │ │ │                                │
            │           │  │ │ │                         ┌──────▼──────┐
            │           │  │ │ │                         │Execution    │
            │           │  │ │ │                         │   Step      │
            │           │  │ │ │                         └─────────────┘
            │           │  │ │ │
            │           │  │ │ └───────────────┐
            │           │  │ │                 │
            │           │  │ └──────────┐      │
            │           │  │            │      │
            │           │  ▼            ▼      ▼
            │      ┌─────────┐   ┌───────────┐ ┌──────────────┐
            │      │   Tag   │   │ Conversa  │ │  FollowUp    │
            │      │  (N:N)  │   │           │ │              │
            │      └─────────┘   └─────┬─────┘ └──────────────┘
            │                          │
            │                    ┌─────▼──────┐
            │                    │  Mensagem  │
            │                    └────────────┘
            │
     ┌──────▼────────┐         ┌──────────────┐
     │ Agente IA     │◄────────┤  FAQ Embed   │
     │  (+ rules)    │         └──────────────┘
     └───────────────┘
```

Cada seta representa uma associação pertencente-a (composition) ou referência. Lead é o **centro funcional** do sistema.

## Entidades principais

| Entidade | Propósito | Doc |
|---|---|---|
| Organização (Tenant) | Unidade de isolamento. Cliente que usa o sistema. | [[Organização]] |
| Lead | Pessoa/empresa prospect. Centro de tudo. | [[Lead]] |
| Time Member | Usuário da org com papel e especialização. | [[Time de Vendas]] |
| Pipeline + Stage + Entry | Funis configuráveis onde leads avançam. | [[Pipeline]] |
| Tag | Rótulo de segmentação N:N com leads. | [[Tag]] |
| Produto | Item do catálogo vendido pela org. | [[Produto]] |
| Conversa + Mensagem | Histórico persistente de comunicação com lead. | [[Conversa e Mensagem]] |
| Workflow + Execution | Automação em grafo. | [[Workflow]] |
| Campanha | Processo outbound paralelo aos pipes. | [[Campanha]] |
| Agente IA | Bot conversacional configurado. | [[Agente IA]] |
| Audit Log / Lead History | Registro imutável. | [[Auditoria]] |

## Entidades de apoio

- **Follow-up**: tarefa de acompanhamento atribuída a membro, com prazo.
- **Mensagem Agendada**: mensagem com envio programado para o futuro.
- **Template de Mensagem**: texto parametrizado reutilizável.
- **Webhook Endpoint**: configuração de webhook de saída da org.
- **Webhook Delivery**: tentativa de entrega de evento externo.
- **Plano de Assinatura**: definição de plano (features habilitadas, quotas).
- **Quota Counter**: contador consumível por org/período.
- **Nota Interna**: comentário do time numa conversa, invisível ao lead.
- **Calendário Conectado**: integração OAuth com calendário externo.
- **FAQ Embedado**: par pergunta-resposta com vetor.
- **Instância de Canal**: conexão concreta a canal (ex.: um número WhatsApp da org).

## Invariantes globais do modelo

1. **Toda entidade pertence a uma organização** (exceto metadados verdadeiramente globais: planos de assinatura, catálogo de templates de agente, tipos de node de workflow).
2. **Identificadores são UUIDs**. Nunca números sequenciais expostos externamente.
3. **Nomes de display** são editáveis pelo usuário; identificadores nunca.
4. **Timestamps**: toda entidade tem `created_at`. Entidades mutáveis têm `updated_at`. Entidades logáveis têm `timestamp` de ação.
5. **Soft delete** preferido: entidades importantes têm `deleted_at`/`is_active` em vez de remoção física. Master pode hard-delete.
6. **Referências cruzadas** são sempre dentro da mesma organização (exceto master operando).
7. **Criação de entidade** dispara evento de domínio.

## Relações N:N

- Lead ↔ Tag (via `lead_tags`)
- Lead ↔ Pipeline (via Entry — um lead pode estar em múltiplos pipes simultaneamente)
- Workflow ↔ Stage (associação opcional — workflow pode ter stages associadas para badges)
- Agente IA ↔ FAQ (um agente tem N FAQs)
- Time Member ↔ Role (via coluna simples — cada member tem UM papel)
- Campanha ↔ Lead (via entry de campanha)
- Conversa ↔ Mensagem (1 conversa, N mensagens; composição)

## Padrões de modelagem

- **Entity**: tem identidade e ciclo de vida (Lead, Organização).
- **Value Object**: imutável, definido por valores (Telefone, Email, Money).
- **Aggregate**: conjunto sempre transacionado junto (Workflow + seus Nodes + Edges).
- **Event**: fato imutável (LeadCreated, StageChanged).
- **Snapshot**: estado agregado num momento (para analytics materializados).

## Vocabulário

- **"Pipe"** e **"Pipeline"** são intercambiáveis — "pipe" é uso cotidiano, "pipeline" é o termo formal.
- **"Lead"** refere-se sempre à **entidade** — não ao "lead novo" nem ao status.
- **"Agente"** sem qualificador refere-se ao **Agente IA**, não a membro humano.
- **"Workflow"** é a **definição**; **"Execution"** é a **instância executando**.

## Onde ler mais

- Cada entidade tem doc próprio em `02 - Modelo de Domínio/`.
- Estados e transições detalhados em [[09 - Referências/Estados e Máquinas de Estado]].
- Eventos emitidos por cada entidade em [[09 - Referências/Catálogo de Eventos]].
