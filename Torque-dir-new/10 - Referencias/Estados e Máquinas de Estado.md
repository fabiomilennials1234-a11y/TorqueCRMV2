---
tipo: referencia
---

# Estados e Máquinas de Estado

Catálogo de máquinas de estado do sistema. Cada entidade com estado explícito tem seu diagrama.

## Organização

```
[provisioning] → [onboarding] → [active] ─┬─► [suspended]  ↔ [active]
                                          ├─► [expired]    ↔ [active]
                                          └─► [deactivated] → [archived]
                                                              ↓
                                                        [hard_deleted]
```

Transições:
- `provisioned → onboarding`: após criação.
- `onboarding → active`: wizard completo OU admin acessou X vezes.
- `active → suspended`: admin master ação, OR X falhas de pagamento consecutivas.
- `active → expired`: plano expirou sem renovação.
- `suspended → active`: master reativa OR pagamento confirmado.
- `expired → active`: pagamento confirmado.
- `deactivated → archived`: após janela de carência sem reativação.
- `archived → hard_deleted`: master + confirmação + janela adicional.

## Time Member

```
[invited] → [pending] → [active] ─┬─► [suspended] ↔ [active]
                                  └─► [removed]
```

- `invited → pending`: convite enviado.
- `pending → active`: aceite.
- `active → suspended`: admin toggle.
- `active → removed`: admin remove (soft).

## Lead

Lead não tem estado singular; é **derivado** de pipeline entries:

```
Derivação:
  Se qualquer entry em stage.is_final='positive' em pipe Propostas → status="vendido"
  Senão se qualquer entry em stage.is_final='negative' sem positivo → status="perdido"
  Senão se entry ativo em Pipe Propostas → "em_proposta"
  Senão se entry ativo em Pipe Confirmação → "em_reuniao"
  Senão se entry ativo em Pipe WhatsApp → "em_qualificacao"
  Senão → "novo"
```

Lead também tem estados:
- `active` | `deleted` (soft).

## Pipeline Entry

```
[ativo no stage X] → [ativo no stage Y] → ...
                ↓
       [finalizado com reason]

  reason: won | lost | abandoned | cancelled | campaign_ended | other
```

Entry não tem `status` singular; tem `current_stage_id` + `finished_at`.

## Pipe WhatsApp (estágios)

```
[novo] → [abordado] → [respondeu] → [agendado] (=> cria entry Confirmação)
                          │              
                          └─► [esfriou] → [abordado] (reabre) OR [final]
```

## Pipe Confirmação (estágios)

```
[reuniao_marcada] → [confirmar_d5] → [confirmar_d3] → [confirmar_d1] → [confirmar_mesmo_dia]
        │                │                │                │                  │
        │                │                │                │                  ▼
        │                ▼                ▼                ▼           [compareceu] (positivo)
        │           (confirmação        (idem)          (idem)         [nao_compareceu] (negativo)
        │            flag set;                                          [cancelou]
        │            continua)
        │
        └──► [reagendado] → [reuniao_marcada] (nova data)
        └──► [cancelado] (negativo)
```

Transições temporais automáticas:
- `meeting_date - now <= 5d → move para confirmar_d5`.
- `<= 3d → d3`, `<= 1d → d1`, mesmo dia → `confirmar_mesmo_dia`.

## Pipe Propostas (estágios)

```
[preparando_proposta] → [proposta_enviada] ↔ [negociando]
                             │                    │
                             │                    │
                             └─► [aguardando_decisao]
                                      │
                             ┌────────┴────────┐
                             ▼                 ▼
                         [vendido]         [perdido]
                        (positivo)        (negativo)
```

## Conversa

```
[active] ↔ [muted]
    ↓
[archived] ↔ [active]
```

- Takeover substate: conversa tem `human_takeover_until` que expira por tempo.

## Mensagem

```
[pending] → [sending] ─┬─► [sent] → [delivered] → [read]
                       │
                       └─► [failed] (pode re-queue)
```

## Workflow Execution

```
[pending] → [running] ─┬─► [completed]
                       ├─► [failed]
                       └─► [waiting] → [running] (resume)
                             ↓
                         [cancelled]
```

## Mensagem Agendada

```
[scheduled] → [sent]
     ↓
[cancelled]
     ↓
[failed]
```

## Follow-up

```
[pending] ─┬─► [done]
           ├─► [missed] (prazo passou) → [pending] (reabrir)
           └─► [cancelled]
```

## Campanha

```
[draft] → [active] ↔ [paused] → [completed]
                         ↓
                     [archived]
```

## Lead Entry em Campanha

```
[active] ─┬─► [completed] (percorreu todos os stages)
          ├─► [exited] (exit_condition atingida; reason: scheduled_meeting, converted, opted_out, blocked, etc.)
          └─► [failed] (erro técnico)
```

## Proposta (dentro de Pipe Propostas Entry)

Estado é derivado do stage da entry + campos de meta.

- Em `preparando_proposta`: items em montagem.
- Em `proposta_enviada`: sent_at registrado.
- Em `negociando`: ajustes.
- Em `vendido`: won_at preenchido.
- Em `perdido`: lost_reason preenchido.

## Agente IA

```
[draft] → [active] ↔ [paused] → [archived]
             ↓
         [deleted]
```

## Pagamento / Fatura

```
[open] ─┬─► [paid]
        ├─► [failed] → [retrying] → [paid] | [cancelled]
        ├─► [refunded]
        └─► [cancelled]
```

## Assinatura do Torque

```
[trialing] → [active] ↔ [past_due]
                 ↓
             [cancelled]
```

## Webhook Delivery

```
[pending] → [retrying] ─┬─► [success]
                        └─► [failed] (dead letter)
```

## Commission Entry

```
[pending] → [approved] → [paid]
            ↓              ↓
        [cancelled]   [reversed] (novo entry negativo)
```

## Regras Gerais de Transições

- Transições ilegais: rejeitadas com erro tipado.
- Transições críticas logadas em audit.
- Transições automáticas triggered por jobs/workflows.
- Transições manuais triggered por usuário via UI.

## Validação de Estado

- Setter de `status` valida transição permitida.
- Estado inicial gerado em factory.
- Estado final (vendido, perdido, completed, etc.) implica campos adicionais obrigatórios.
