---
tipo: integracao
direcao: in
criticidade: alta
---

# n8n (Orquestrador Externo)

Muitos clientes do Torque usam n8n para construir fluxos de ingestão de leads: monitoram board Trello, formulários externos, captura de Meta Ads via Zapier → normalizam → enviam ao webhook de ingestão do Torque.

n8n é "do cliente" — não da Milennials. O contrato relevante é o **webhook de ingestão** do Torque, que n8n consome.

## Propósito

- Flexibilizar ingestão: qualquer fonte externa pode conectar ao Torque via n8n.
- Cliente tem controle: usa nodes/triggers familiares, monta fluxo próprio.
- Desacopla Torque de fontes externas específicas (no lugar de adaptadores para 20 fontes, tem 1 webhook bem documentado).

## Contrato (webhook de ingestão)

Endpoint: `POST https://api.torque.com.br/v1/webhooks/leads`

### Autenticação
- Header `Authorization: Bearer <api_key_da_org>`.
- OU token específico de webhook em URL: `/webhooks/leads/<token>`.

### Payload
Ver [[04 - Funcionalidades/Admin/API Docs]] para spec completo. Resumo:

```json
{
  "source": "meta_ads" | "trello" | "forms_site" | "zapier" | "custom",
  "external_id": "trello-card-123",
  "fields": {
    "name": "...",
    "phone": "+55...",
    "email": "...",
    "company": "...",
    "position": "..."
  },
  "custom_fields": {
    "campo_x": "valor"
  },
  "tags": ["Ouro", "B2B-Industrial"],
  "place_in_pipe": {
    "pipe": "whatsapp",
    "stage": "novo"
  },
  "assigned_user_id": "uuid",
  "update_existing_if_match": true,
  "utm": { "source": "...", "medium": "...", ... }
}
```

### Resposta
- 201 Created — novo lead.
- 200 OK — lead atualizado (se dedupe match).
- 400 — payload inválido.
- 401 — auth.
- 403 — quota/permissão.
- 429 — rate limit.

### Idempotência
- `external_id` é recomendado; repetidos não duplicam.

## Padrão típico em n8n (exemplo)

Fluxo comum em n8n:
1. **Trigger**: `Trello Trigger` — monitora board do cliente.
2. **Extract**: `Function Node` que lê `desc` do card com regex (faturamento, CNPJ).
3. **Classify**: IF/Switch baseado em faturamento → tag (Latão/Prata/Ouro/Diamante).
4. **Normalize**: mapeia para payload Torque.
5. **HTTP Request**: POST para webhook Torque.
6. **Error Handler**: retry + notifica responsável em caso de falha.

Cada cliente tem seu próprio workflow n8n com board Trello + `assigned_user_id` específico.

## Variações

- Meta Ads → Zapier → Trello → n8n → Torque.
- Google Forms → Zapier → n8n → Torque.
- Landing Page → Webhook direto → n8n → Torque.
- Planilha Google → n8n cron → Torque.

## Regras de Negócio do Contrato

1. Tags aceitas como array, JSON string, ou string simples.
2. Busca case-insensitive para matching.
3. Nomenclatura: `pipe: "whatsapp"` aceita (resolvido para pipe estrutural).
4. `assigned_user_id` valida que membro existe e está ativo.
5. `place_in_pipe` vs default: se omitido, Torque usa pipe default da org.
6. Campos custom desconhecidos armazenados mesmo assim.

## Edge Cases

- **Payload mal formado**: 400 com detalhes.
- **Nome com só espaços**: rejeitado.
- **Telefone em formato local** (ex.: `(11) 98765-4321`): Torque normaliza para E.164 (adicionar +55 se BR detectado).
- **Duplicata enviada duas vezes**: idempotência via external_id previne.
- **Tag nova desconhecida**: criada automaticamente.
- **Stage inexistente no pipe**: ignora `place_in_pipe`, usa default; warn em audit.
- **Volume em burst** (100 leads/s): rate limit; n8n deve fazer retry.

## Performance

- Webhook processa em < 1s em ingestões normais.
- Sob carga: 200ms p50, < 2s p95.
- Rate limit por API key: 100 req/min default; ajustável por plano.

## Segurança

- HTTPS obrigatório.
- API key em header (não em query).
- Rate limit.
- Payload validado estritamente.
- Dados sensíveis não logados em info.

## Observabilidade

- Cada request no audit log (origem, external_id, resultado).
- Métricas de ingestão por origem.
- Dashboard de n8n health (não está no escopo do Torque, mas admin pode configurar alertas externos).

## Valor para o Cliente

- Flexibilidade: conectar qualquer fonte.
- Sem vendor lock-in em fontes — muda n8n sem tocar Torque.
- Curva de aprendizado baixa (n8n é visual).

## Limitações

- Cliente gerencia a infra de n8n (ou usa n8n cloud).
- Falha em n8n = leads perdidos → Torque não tem como saber se houve falta de ingestão (monitora métricas de ingestão como proxy).

## Documentação ao Cliente

- Em `API Docs` do Torque há seção com exemplos de n8n configurado para cenários comuns.
- Templates de workflow n8n exportáveis para o cliente importar.
