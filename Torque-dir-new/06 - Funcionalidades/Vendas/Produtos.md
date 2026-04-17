---
tipo: feature
dominio: vendas
---

# Produtos

## Propósito

Catálogo de produtos/serviços vendidos pela organização. Primariamente usado em montagem de propostas no Pipeline Propostas, em cálculo de comissão, e em relatórios. Pode ser sincronizado com ERP externo (TinyERP ou equivalente).

## Atores e Permissões

- **Admin**: CRUD completo.
- **Closer**: visualiza, adiciona a propostas.
- **ERP externo**: sincroniza via webhook/pull.
- **Copilot**: pode consultar (catálogo injetado como contexto) mas não modificar.

Ações: `product.view`, `product.create`, `product.edit`, `product.delete`, `product.add_to_proposal`, `product.import_from_erp`.

## Dados Envolvidos

Ver [[02 - Modelo de Domínio/Produto]] para atributos completos. Resumo:

- Nome, SKU, descrição, preço, moeda, categoria, unidade, tax_rate, estoque, metadata.
- `external_id` (ERP) + `synced_at`.
- `is_active`.
- `commission_config` (override por produto).

## Regras de Negócio

1. SKU único dentro da org (quando preenchido).
2. Preço ≥ 0.
3. Produto em uso em propostas ativas não pode ser hard-deletado (apenas desativado).
4. Produto com `external_id` tem campos "protegidos" de edição pela UI (preço, estoque, tax) quando sync ativo — fonte é o ERP.
5. Alteração de preço não retroage a propostas já criadas.
6. Moedas múltiplas suportadas; default da org.

## Fluxos do Usuário

### Listar Produtos
1. Menu → `Produtos`.
2. Tabela com filtros: categoria, ativo/inativo, busca por nome/SKU, sincronizado ou manual.
3. Ordenação por qualquer coluna.
4. Admin vê métricas rápidas: produto mais vendido, total ativo.

### Criar Manual
1. Botão "Novo Produto".
2. Form: nome, descrição, preço, moeda, SKU, categoria, unidade, imagem (opcional).
3. Salvar.

### Editar
1. Abrir produto → editar campos permitidos.
2. Se `external_id`: avisar que sync pode sobrescrever.

### Desativar / Deletar
1. Toggle `is_active=false` = escondido em seletores mas propostas antigas mantêm referência.
2. Hard-delete: só admin, exige confirmação, bloqueado se em uso.

### Importar de ERP
1. Admin conecta integração (ver [[05 - Integrações Externas/TinyERP]]).
2. Click "Sincronizar agora" → job puxa catálogo.
3. Produtos upsert por `external_id`.
4. Admin vê log de sync (criados, atualizados, erros).

### Adicionar a Proposta
1. Closer edita proposta.
2. Busca produto por nome/SKU.
3. Seleciona, define quantidade e preço (pode diferir do default se permissão de desconto).
4. Item adicionado com snapshot de nome/SKU/preço.

## Automações e Eventos

### Emite
- `ProductCreated`, `ProductUpdated`, `ProductDeactivated`, `ProductDeleted`.
- `ProductSyncStarted`, `ProductSyncCompleted`, `ProductSyncFailed`.

### Reage
- Webhook do ERP: `product.created/updated/deleted` → upsert local.
- Cron de sync: job de pull periódico (fallback).

## Integrações

- **ERP externo** (TinyERP): fonte de verdade quando conectado.
- **Pipeline Propostas**: uso primário.
- **Comissões**: `commission_config` por produto.
- **Copilot**: contexto para agente (opcional — produtos simplificados).
- **Catálogo público** (futuro): expor lista com preços.
- **Analytics**: relatórios de venda por produto.

## Edge Cases

- **Produto com preço 0**: aceito (ex.: serviço grátis como upsell); proposta com total 0 exige confirmação ao marcar vendido.
- **ERP offline durante sync**: job falha, retry com backoff, alerta em dashboard de integração.
- **Produto deletado no ERP**: webhook marca `is_active=false` localmente; não deleta (preserva histórico).
- **Conflito (manual + sync)**: sync vence em campos fonte-ERP; campos locais (commission_config, metadata.local_notes) preservados.
- **Estoque zerado**: aceito em proposta mas UI alerta "sem estoque".
- **Produto com >1000 variações**: importar em lote em background; UI avisa.

## Validações

- Nome: 2-200 chars.
- SKU: quando informado, único, formato livre.
- Preço: decimal com 2-4 casas, ≥ 0.
- Moeda: enum (BRL, USD, EUR, ...).
- Imagem: <5MB, JPG/PNG.
- Tax rate: 0-100%.

## Métricas

- Produto mais vendido (qtd e receita).
- Produto com maior margem (se custo informado).
- Taxa de fechamento quando produto em proposta.
- Categoria mais vendida.
- Tempo médio do ERP sync.

## Catálogo Público (opcional/futuro)

- Organização pode expor URL pública com produtos ativos.
- Lead acessa, seleciona produto de interesse → preenche form → vira lead novo com tag "interesse:produto-X".
- Útil para SEO e geração de leads orgânicos.

## Observações

- Produtos podem representar **serviços recorrentes** (ex.: assinatura mensal R$ 500/mês) — usar metadata para duração.
- Preço público vs preço negociado: preço base é default; closer pode alterar (com permissão).
- Para venda consultiva complexa, proposta pode ter items customizados (sem product_id) — closer cria "item ad-hoc" com nome, descrição, preço digitados.
