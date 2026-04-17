---
tipo: fluxo
---

# Ingestão de Leads (Webhook)

Fluxo detalhado de um lead entrando via endpoint de webhook público — caminho principal usado por n8n, Zapier, formulários, integrações externas.

## Caminho Feliz

```
1. Fonte externa (ex.: n8n)
   - Usuário preenche formulário em landing, ou card criado em Trello.
   - Fluxo n8n captura, normaliza, envia POST.

2. POST /v1/webhooks/leads
   - Header: Authorization: Bearer <api_key>
   - Body: JSON descrito em API Docs
   - Org identificada via API key

3. Backend valida
   - API key válida e ativa.
   - Scope permite.
   - Rate limit OK.
   - Payload schema válido.
   - fields.name preenchido.
   - fields.phone ou fields.email preenchido.

4. Normalização
   - phone → E.164.
   - email → lowercase.
   - tags → array de strings normalizadas (string JSON, lista, ou string única aceitas).
   - custom_fields preservados.

5. Dedupe
   - Se external_id: busca por external_id.
     - Existe + update_existing_if_match=true → atualiza.
     - Existe + flag=false → rejeita com 409 Conflict.
     - Não existe → segue.
   - Sem external_id: busca por (phone OR email) na org.
     - Match + update_existing_if_match=true → atualiza.
     - Match + flag=false → cria duplicado (warn em log).
     - Sem match → cria.

6. Criação/Atualização
   - Transação atômica:
     a. Upsert Lead.
     b. Aplica tags (cria tags novas se não existem; case-insensitive).
     c. Cria/atualiza pipeline entry conforme place_in_pipe (ou default: pipe WhatsApp stage novo).
     d. Set assigned_user_id se informado E membro válido.
     e. Set UTM fields se informados.
   - Emite eventos:
     - LeadCreated ou LeadUpdated.
     - TagAdded (para cada tag nova no lead).
     - LeadEnteredPipe (se criou entry).

7. Pós-criação (assíncrono)
   - Distribuição automática se não tem assigned e pipe tem regra.
   - Calculadora de score.
   - Workflows com trigger lead_created iniciam.
   - Notificações ao responsável.
   - Webhook outbound para endpoints configurados da org.

8. Resposta
   - 201 Created (novo) ou 200 OK (atualizado).
   - Body: { lead_id, status: "created"|"updated" }.
```

## Latência-alvo

- p50: 300ms.
- p95: 1s.
- p99: 3s.

A maior parte do pós-criação é assíncrono — response do webhook não espera workflows nem distribuição.

## Códigos de Resposta

| Code | Significado |
|---|---|
| 201 | Lead criado. |
| 200 | Lead atualizado (match + update flag). |
| 400 | Payload inválido (detalhe no body). |
| 401 | API key inválida. |
| 403 | Quota excedida / feature não disponível / scope insuficiente. |
| 409 | Conflito (external_id existe + update flag false). |
| 429 | Rate limit. |
| 500 | Erro interno (retry seguro — idempotência via external_id). |

## Campos Aceitos

### Obrigatórios
- `fields.name` (mínimo 2 chars).
- `fields.phone` OU `fields.email`.

### Opcionais
- `source` (string livre para analytics).
- `external_id` (recomendado para dedupe).
- `fields.company`, `position`, `cnpj`, etc.
- `custom_fields` (objeto chave-valor; schema da org respeitado).
- `tags` (lista).
- `place_in_pipe` (pipe + stage).
- `assigned_user_id`.
- `update_existing_if_match` (bool, default true).
- `utm` (objeto com source, medium, campaign, term, content).

## Tratamento de Tags

Webhook aceita `tags` em múltiplos formatos:
```
"tags": ["Ouro", "B2B"]             # array
"tags": "Ouro,B2B"                  # string com separador
"tags": "[\"Ouro\",\"B2B\"]"        # JSON string
"tags": "Ouro"                      # single string
```

Todas normalizadas para array de strings internamente.

Match case-insensitive. Tag inexistente é criada.

## Tratamento de Telefone

- Formato recebido pode variar: `(11) 98765-4321`, `11 98765-4321`, `+55 11 98765-4321`, `5511987654321`.
- Torque tenta normalizar para E.164.
- Heurística: se começa com código de país, usa; senão, assume BR (+55).
- Se falha em normalizar: rejeita com erro explícito se phone é único meio de contato; aceita com warning se email preenchido.

## Tratamento de Campos Custom

- Sistema aceita chaves desconhecidas no `custom_fields`.
- Armazena como JSON (schema flexível).
- Se org tem schema configurado, valida tipos para chaves conhecidas.
- Admin pode retroativamente "promover" chave ad-hoc a campo configurado.

## Integrações Downstream

Após criação bem-sucedida, **event fanout**:

- **Workflow engine**: dispara todos workflows com trigger `lead_created` cujo filter matche.
- **Score engine**: calcula score inicial.
- **Distribution job**: atribui membro se configurado.
- **Notificação**: push ao responsável + email configurável.
- **Webhook outbound**: se org configurou endpoints subscritos a `LeadCreated`, entrega.
- **Analytics**: incrementa contador de lead criado.

## Idempotência

Garantida por:
- `external_id` (preferido).
- Dedupe por phone/email quando configurado.

Mesmo request enviado 2x resulta em 1 lead com as mesmas consequências.

## Observabilidade

Cada ingestão:
- Log: `request_id`, `api_key_hash`, `org_id`, `source`, `external_id`, `result` (created/updated/duplicated/rejected), `duration`.
- Métricas:
  - Ingestão por hora/dia/org.
  - Taxa de dedupe (updates / total).
  - Taxa de rejeição.
  - Latência p50/p95.

## Erros Comuns

| Erro | Causa | Correção |
|---|---|---|
| `name_required` | Nome vazio ou < 2 chars. | Enviar nome válido. |
| `contact_missing` | Sem phone nem email. | Enviar ao menos um. |
| `invalid_phone` | Phone não E.164-izável. | Corrigir fonte. |
| `invalid_email` | Email sintaxe inválida. | Corrigir. |
| `tag_quota_exceeded` | Máx tags por lead. | Limpar excesso. |
| `org_suspended` | Org inativa. | Contatar admin. |
| `quota_exceeded` | Quota de leads do mês. | Upgrade de plano. |
| `feature_not_available` | Pipeline customizado sem feature. | Upgrade. |
| `invalid_pipe_reference` | Pipe inexistente. | Corrigir payload. |
| `rate_limited` | Demasiadas requisições. | Backoff no cliente. |

## Segurança

- HTTPS obrigatório.
- API key em header.
- Validação estrita de schema (evita injeção).
- Sanitização de strings.
- Logs sem PII em produção info-level.
- Alertas de padrões suspeitos (ex.: 1000 ingestions/min de mesma key — possível abuso).

## SLA

- 99.9% disponibilidade.
- RTO 1h em caso de outage.
- Sem perda de dados: webhook idempotente + fila durável se saturado.
