---
tipo: feature
dominio: admin
---

# API Docs (API Pública)

## Propósito

Documentação da **API pública** que o Torque expõe para integrações externas. Admin da organização gera API keys, integra sistemas externos (n8n, Zapier, ERPs, BI, scripts próprios) e consome endpoints programaticamente.

## Atores e Permissões

- **Admin**: gera keys, consulta doc.
- **Integrador externo**: usa key para chamar endpoints.

Ações: `apikey.create`, `apikey.revoke`, `apikey.view`.

## Estrutura da API

### Convenções

- **Base URL**: `https://api.torque.com.br/v1` (versionada).
- **Autenticação**: `Authorization: Bearer <api_key>`.
- **Formato**: JSON entrada e saída.
- **Rate limit**: por key (ex.: 100 req/min).
- **Paginação**: cursor-based (`?cursor=...&limit=...`).
- **Versionamento**: `v1`, `v2` no path.
- **Errors**: payload estruturado com `code`, `message`, `details`.

### Endpoints Principais

#### Leads
- `POST /leads` — criar.
- `GET /leads/:id` — detalhe.
- `GET /leads` — listar (filtros em query).
- `PATCH /leads/:id` — editar.
- `DELETE /leads/:id` — soft-delete.
- `POST /leads/:id/tags` — adicionar tag.
- `DELETE /leads/:id/tags/:tag_id` — remover.
- `POST /leads/:id/assign` — atribuir responsável.

#### Lead Ingestion (webhook público)
- `POST /webhooks/leads` — entrada de lead (usado por orquestradores externos como n8n).
  - Formato descrito em [[05 - Integrações Externas/n8n (Orquestrador Externo)]] e detalhado abaixo.

#### Pipelines
- `GET /pipelines` — listar.
- `GET /pipelines/:id` — detalhe.
- `GET /pipelines/:id/entries` — entries com filtro.
- `POST /pipelines/:id/entries` — criar entry.
- `PATCH /pipelines/:id/entries/:entry_id` — mover stage / editar meta.

#### Conversas e Mensagens
- `GET /conversations` — listar.
- `GET /conversations/:id/messages` — histórico.
- `POST /conversations/:id/messages` — enviar mensagem (outbound).

#### Workflows
- `POST /workflows/:id/execute` — dispara manual (se workflow tem trigger=manual).

#### Meta
- `GET /me` — info da API key.
- `GET /me/permissions` — escopo.

#### Webhooks de Saída
- `GET /webhook-endpoints` — listar.
- `POST /webhook-endpoints` — criar.
- (Completo para CRUD.)

## Contrato do Webhook de Ingestão (principal)

Endpoint: `POST /webhooks/leads`

Autenticação: `Authorization: Bearer <api_key>` OU token em URL.

Payload:
```json
{
  "source": "meta_ads",
  "external_id": "trello-card-123",
  "fields": {
    "name": "Maria Silva",
    "phone": "+5511987654321",
    "email": "maria@empresa.com.br",
    "company": "Indústria XYZ",
    "position": "Gerente de Compras"
  },
  "custom_fields": {
    "faturamento_declarado": 500000,
    "canal_preferido": "whatsapp"
  },
  "tags": ["Ouro", "B2B-Industrial"],
  "place_in_pipe": {
    "pipe": "whatsapp",
    "stage": "novo"
  },
  "assigned_user_id": "uuid-do-membro",
  "update_existing_if_match": true,
  "utm": {
    "source": "facebook",
    "medium": "cpc",
    "campaign": "launch-2026",
    "term": "",
    "content": "variant_a"
  }
}
```

Resposta:
```json
{
  "lead_id": "uuid",
  "status": "created"    // ou "updated" se match
}
```

### Regras
- `tags` aceita array, string JSON, ou string simples.
- Busca case-insensitive.
- Idempotência via `external_id` recomendado.
- Todos os campos opcionais exceto `fields.name` + `fields.phone` OR `fields.email`.

### Códigos de Resposta
- 201 Created.
- 200 Updated.
- 400 Bad Request (validação).
- 401 Unauthorized (key inválida).
- 403 Forbidden (quota atingida, feature não disponível).
- 429 Too Many Requests (rate limit).
- 500 (erro interno).

## Fluxos do Usuário

### Criar API Key
1. `Configurações → API Keys → Nova`.
2. Nome (ex.: "n8n produção").
3. Escopos (checkboxes por ação ou preset).
4. Gera key. Exibida uma vez.
5. Admin copia + armazena externo.

### Consultar Doc
1. Menu → `API Docs`.
2. Navegador tipo Swagger/OpenAPI com endpoints.
3. Cada endpoint: descrição, params, body, responses, exemplo cURL.
4. Tab "Try it" com a própria key (admin pode testar no-op).

### Revogar
- Admin clica "Revogar".
- Key invalidada imediatamente.

### Ver Uso
- Por key: requests/dia, erros, rate limit hits.

## Automações

### Emite
- `ApiKeyCreated`, `ApiKeyRevoked`, `ApiKeyUsed` (logged, não evento).

### Reage
- Ao uso de endpoint: verifica key, aplica rate limit, executa.

## Integrações

- **Todos os endpoints** espelham casos de uso de domínio.
- **Audit**.
- **Quotas**.
- **Rate limit**.

## Edge Cases

- **Key vazada**: admin revoga, gera nova, atualiza integração.
- **Endpoint deprecated**: warning em response header `Deprecation: true`; sunset date em `Sunset`.
- **Rate limit atingido**: 429 com `Retry-After`.
- **Payload muito grande**: 413 Payload Too Large.
- **Campo custom inexistente** em ingestão: aceita (armazena mesmo assim).
- **Endpoint em manutenção**: 503 com mensagem.

## Validações

- API key válida e ativa.
- Scope da key contém ação necessária.
- Payload schema válido.
- Org tem feature habilitada.
- Quota OK.

## Métricas

- Requests/dia por org.
- Endpoints mais usados.
- Erros por tipo.
- Latência por endpoint (p50, p95, p99).
- Rate limit hits.

## Segurança

- HTTPS obrigatório.
- Keys nunca em URL, só em header.
- Secret storage no servidor (hash).
- Log não inclui key plaintext.
- Revogação imediata.

## Versionamento

- `v1`, `v2` paralelos quando breaking change.
- Deprecation notice com 6 meses de antecedência.
- Migração documentada.

## Documentação

- OpenAPI spec gerado e publicado.
- SDK / wrappers em linguagens comuns (futuro).
- Changelog visível.
