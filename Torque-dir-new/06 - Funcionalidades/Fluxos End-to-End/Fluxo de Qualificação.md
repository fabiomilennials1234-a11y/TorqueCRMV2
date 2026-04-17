---
tipo: fluxo
---

# Fluxo de Qualificação

Do lead recém-criado ao agendamento de reunião — o caminho mais comum e crítico. Combina Pipeline WhatsApp, Chat, Copilot (se ativo), Follow-ups, Workflow Builder.

## Diagrama

```
Lead criado em "novo"
       │
       ▼
[Distribuição SDR]
       │
       ▼
SDR/Agente IA envia abordagem
       │
       ▼
Stage: "abordado"
       │
       ├──── Lead responde ──────────┐
       │                              ▼
       └──── Sem resposta 48h        Stage: "respondeu"
                  │                    │
                  ▼                    │
               Stage: "esfriou"        │
                  │                    │
                  ▼                    │
            Workflow retenção         │
            (mensagem automática)     │
                  │                    │
                  ▼                    │
          Resposta? ──Sim──►Volta para "respondeu"◄┘
                  │
                 Não
                  ▼
              Stage final negativa
                 OU
              Campanha reengajamento
                                      Diálogo qualificando (BANT):
                                      - Necessidade
                                      - Orçamento
                                      - Timing
                                      - Decisor

                                      Se qualificado → propõe reunião
                                      Lead aceita → Stage: "agendado"
                                      Regra cria entry em Confirmação
```

## Passo a Passo

### 1. Distribuição
Lead criado em `novo`. Job de distribuição executa:
- Avalia regra do pipe (round_robin, load_based, tag_based).
- Atribui a SDR ativo.
- Se ninguém: lead fica `unassigned` — aparece em view "pending-attribution" para admin.

### 2. Abordagem
SDR (ou agente) abre chat. Usa template "Abordagem inicial":
```
Olá {{lead.name}}, tudo bem? Sou da {{org.name}}. Vi que você demonstrou interesse em {{lead.interest_topic or "nossos produtos"}}...
```
- Envia mensagem.
- Sistema move entry para `abordado` (auto ou manual).
- `first_approached_at` registrado.

### 3. Bateria de Qualificação
Lead responde:
- Agente IA (se ativo) executa batch 8s → responde.
- Ou SDR humano responde.
- Tags são adicionadas conforme respostas (ex.: "orçamento_10k", "decisor", "urgente").
- Stage vai para `respondeu`.
- Score recalcula com novos sinais.

### 4. Critério BANT (exemplo)
Agente/SDR tentar entender:
- **B**udget: tem orçamento alinhado?
- **A**uthority: é decisor?
- **N**eed: tem necessidade real?
- **T**iming: quando pretende decidir?

Depende de cada operação qual é o critério mínimo.

### 5. Proposta de Reunião
Qualificado:
- Agente/SDR sugere horário.
- Lead aceita, oferece horário, ou pede reagendar.
- Confirmado: agendamento criado.

### 6. Agendamento
- Form de confirmação: data, hora, duração, local.
- Integra Google Calendar se conectado.
- Entry move para `agendado`.
- Regra de pipe cria entry em Confirmação.
- `meeting_date` armazenada.

### 7. Cenário "Esfriou"
Se lead abordado mas sem resposta em 48h:
- SLA dispara workflow.
- Tentativa #1: mensagem de "lembrete suave" (template).
- Sem resposta em +48h: tentativa #2 (possivelmente áudio humanizado).
- Sem resposta em +72h: stage `esfriou`, tag `esfriado`.
- Workflow de retenção (mensagem especial, oferta, áudio).
- Se ainda nada: campanha de reengajamento mensal.

## Atores e Responsabilidades

| Ator | Papel neste fluxo |
|---|---|
| Lead | Responde, qualifica-se (ou não). |
| SDR | Aborda, qualifica, agenda. |
| Agente IA | Escala operação: responde quando SDR não está disponível, padroniza qualificação, aplica BANT. |
| Admin | Configura templates, agentes, workflows, regras. |
| Sistema | Distribuição, SLA, scoring, automações. |

## Métricas-chave

- **Tempo de primeira resposta**: abordagem → primeira resposta do lead.
- **Taxa de resposta**: `respondeu / abordados`.
- **Taxa de agendamento**: `agendados / respondidos`.
- **Taxa de esfriamento**: `esfriados / abordados`.
- **Produtividade SDR**: agendamentos por dia.
- **Eficácia da abordagem**: qual template tem maior taxa de resposta.

## Eventos Emitidos

Ordem típica:
1. `LeadCreated`.
2. `LeadAssigned(sdr_id)`.
3. `MessageSent(outbound, abordagem)`.
4. `LeadStageChanged(whatsapp, novo→abordado)`.
5. `MessageReceived` (quando lead responde).
6. `LeadStageChanged(whatsapp, abordado→respondeu)`.
7. `LeadTagAdded(qualificado)`.
8. `LeadScoreRecalculated`.
9. `LeadStageChanged(whatsapp, respondeu→agendado)`.
10. `MeetingScheduled`.
11. `LeadEnteredPipe(confirmacao)`.

## Pontos de Decisão

- **Usar agente IA ou humano?** Volume alto e respostas repetitivas → agente. Conta sensível → humano.
- **Tags aplicar manualmente ou automaticamente?** Automático via AI Actions sempre que possível (consistência).
- **SLA de 48h é fixo?** Configurável por org.
- **Esfriado volta para abordado ou fica esfriado?** Regra: envio/resposta move para abordado/respondeu.

## Pitfalls Comuns

- **Agente IA mal configurado** (sem business_context) dá respostas genéricas → conversão cai.
- **Template de abordagem igual para todos** → taxa de resposta baixa.
- **Sem follow-up de esfriamento** → leads morrem em esfriou.
- **Distribuição desbalanceada** → alguns SDRs sobrecarregados, outros ociosos.
- **Admin não monitora taxa de agendamento** → problema persiste invisível.

## Integração com Outras Features

- **Copilot**: agente IA opera neste fluxo.
- **Chat Multi-canal**: comunicação visível.
- **Workflows**: automações temporais (SLA, follow-up).
- **Follow-ups**: tarefas de follow-up criadas pelos dois.
- **Templates**: abordagem padronizada.
- **Campanhas**: reengajamento para esfriados.
- **Analytics Comercial**: monitoria.
- **Lead Score**: priorização.
