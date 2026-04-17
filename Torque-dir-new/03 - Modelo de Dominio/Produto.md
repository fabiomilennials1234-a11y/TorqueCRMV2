---
tipo: dominio
entidade: Produto
---

# Produto

Item do catálogo vendido pela organização. Usado primariamente em propostas comerciais no Pipeline Propostas, em cálculo de comissão, e em relatórios financeiros.

## Atributos

- `id`: UUID.
- `organization_id`.
- `external_id`: id do produto no ERP externo (ex.: TinyERP). Null se criado manualmente.
- `name`: nome do produto.
- `sku`: código único (opcional em manual, geralmente sincronizado do ERP).
- `description`: descrição longa.
- `price`: preço base.
- `currency`: moeda (default `BRL`).
- `category`: categoria livre.
- `unit`: unidade (`un`, `kg`, `m`, `hora`, etc.).
- `tax_rate`: alíquota (opcional, default da org).
- `is_active`: ativo/inativo.
- `stock`: estoque (se sincronizado do ERP; pode ser null se não aplicável).
- `metadata`: JSON com campos livres e dados do ERP.
- `commission_config`: override de comissão específico (opcional).
- `created_at`, `updated_at`.
- `synced_at`: última sincronização com ERP (se aplicável).

## Invariantes

1. `sku` único dentro da organização quando preenchido.
2. `price >= 0`.
3. Produto em uso em entry ativo de Propostas não pode ser hard-deletado (soft-delete com flag `is_active=false`).

## Relações

- Pertence a **Organização**.
- Usado em **Pipeline Entry** de Propostas (via `meta.items` — lista de `{product_id, quantity, unit_price, discount}`).
- Pode ter **Commission Config** específico.

## Operações

### CreateProduct (manual)
- Validar nome, preço.
- Validar SKU único se preenchido.

### UpsertProductFromErp (sync)
- Executado por job de sincronização.
- Match por `external_id`.
- Campos protegidos de sobrescrita: `commission_config` (se admin customizou), `metadata.local_notes`.

### UpdateProduct
- Editar nome, preço, descrição, categoria, is_active.
- Alertar quando altera preço de produto em proposta ativa (proposta mantém preço vigente no momento da criação).

### DeactivateProduct
- Toggle `is_active=false`.
- Esconde de seletores; não afeta propostas já abertas.

## Sincronização com ERP

Ver [[05 - Integrações Externas/TinyERP]]. Fluxo:

1. ERP dispara webhook de evento `product.created` / `product.updated` / `product.deleted`.
2. Handler autentica, extrai dados, executa `UpsertProductFromErp`.
3. Campo `synced_at` atualizado.
4. Evento `ProductSyncCompleted` emitido (para analytics).

Alternativa: job de **pull** periódico que sincroniza lote (fallback quando webhook falha).

Conflict resolution: ERP é fonte de verdade para preço, estoque, dados fiscais. UI do Torque mostra esses como read-only quando produto tem `external_id`.

## Eventos

- `ProductCreated`, `ProductUpdated`, `ProductDeactivated`, `ProductDeleted`.
- `ProductSyncStarted`, `ProductSyncCompleted`, `ProductSyncFailed`.
- `ProductAddedToProposal(entry_id, product_id, quantity, price)`.

## Uso em Propostas

Pipeline Entry de Propostas carrega `meta`:

```
{
  "items": [
    {"product_id": "...", "name_snapshot": "...", "quantity": 2, "unit_price": 1500.00, "discount": 0.10},
    ...
  ],
  "subtotal": 2700.00,
  "discount_total": 300.00,
  "total": 2700.00,
  "currency": "BRL",
  "valid_until": "2026-05-15",
  "notes": "..."
}
```

- `name_snapshot`: nome do produto no momento da adição (caso nome mude depois).
- `unit_price`: pode ser diferente do `price` do produto (closer pode dar desconto).
- `discount`: percentual (0.0-1.0).

## Catálogo público (opcional)

- Organização pode expor catálogo via link público com produtos ativos.
- Lead pode clicar → vira lead novo via form + interesse no produto X.

## Regras de Negócio

- **Soft delete** obrigatório se houver propostas históricas: preservar nome/preço snapshot.
- **Alteração de preço** em produto não retroage a propostas já abertas — nova proposta usa novo preço.
- **Moeda** diferente da org default é suportada mas propostas múltiplas-moedas exigem confirmação explícita.
- **Desconto** em item é do closer; admin pode limitar via permissão (ex.: SDR sem permissão de desconto acima de 10%).

## Métricas

- Produto mais vendido (qtd, receita).
- Ticket médio por produto.
- Margem (se custo informado).
- Taxa de ganho quando produto está em proposta.

## Integrações com Agentes IA

- Agente IA pode citar produtos no contexto da conversa.
- Catálogo simplificado (nome + preço + descrição curta) é injetado no prompt do agente como `business_context` opcional.
- Agente nunca propõe preço sem confirmação humana (permissão `agent.propose_price` desabilitada por default).

## Internacionalização

- Nomes e descrições podem ter traduções se org operar multi-idioma (`translations: {en: ..., es: ...}` opcional).
- Moeda por produto é preservada em propostas.
