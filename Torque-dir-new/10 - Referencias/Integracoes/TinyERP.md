---
tipo: integracao
direcao: bidirectional
criticidade: media
---

# TinyERP

Integração com TinyERP (ERP brasileiro popular entre distribuidores e fábricas) para sincronizar catálogo de produtos e registrar pedidos gerados por vendas.

## Propósito

- Puxar catálogo de produtos do ERP para o Torque.
- Manter preços e estoque atualizados.
- Registrar pedido no ERP quando proposta é ganha (evita re-trabalho).
- Fonte de verdade financeira/fiscal continua no ERP.

## Contrato

- API REST do TinyERP.
- Webhook de eventos (se disponível no plano).
- Alternativa: pull periódico.

## Autenticação

- API token do TinyERP (gerado pelo cliente na própria plataforma).
- Admin insere no Torque em `Configurações → Integrações → TinyERP`.
- Armazenado em vault cifrado.

## Endpoints Consumidos

### Produtos
- `GET /produtos.pesquisa.php` — lista.
- `GET /produto.obter.php?id=...` — detalhe.
- `POST /produto.incluir.php` — criar (raro, Torque não cria).

### Pedidos
- `POST /pedido.incluir.php` — criar pedido a partir de proposta ganha.
- `GET /pedido.obter.php?id=...` — detalhe.

### Clientes / Contatos (opcional)
- Sincronização de leads → contatos no TinyERP.

## Webhooks Recebidos

TinyERP (quando plano permite):
- `produto.alterado`, `produto.incluido`, `produto.excluido`.
- `estoque.alterado`.
- `pedido.alterado`.

Payload contém ID; Torque faz GET para detalhe.

## Fluxos

### Sync de Produtos (pull)
1. Cron a cada 1h (ou menos).
2. Torque chama `produtos.pesquisa.php` com filtros (ativos, modificados desde X).
3. Para cada produto, upsert no Torque via `UpsertProductFromErp(external_id, data)`.
4. Status `synced_at` atualizado.
5. Diff log: criados, atualizados, erros.

### Sync de Produtos (webhook push)
1. TinyERP → webhook.
2. Torque recebe, chama GET para detalhe, upsert.

### Criar Pedido (após venda)
1. Pipeline Propostas stage `vendido`.
2. Sistema dispara ação `create_order_in_erp`.
3. Torque monta payload conforme schema TinyERP (items, cliente, valor).
4. POST para TinyERP.
5. Recebe pedido_id → armazena em `pipeline_entry.meta.erp_order_id`.
6. Se falha: admin alerta, pode retry.

## Regras de Negócio

1. Produto com `external_id` tem campos "protegidos" de edição (preço, estoque) no Torque — fonte é ERP.
2. Produto criado manualmente no Torque (sem external_id) não é enviado de volta ao ERP.
3. Pedido criado no ERP a partir de venda: campos mínimos (items, cliente com CNPJ, valor, forma de pagamento).
4. Se cliente (lead) não tem CNPJ: Torque tenta criar contato no TinyERP só com nome + telefone (ou alerta).
5. Sync respeita timezone.
6. Dedupe por `external_id`.

## Edge Cases

- **ERP offline**: sync job falha, retry; alerta após > 3 falhas.
- **Produto deletado no ERP**: recebe webhook ou detecta em pull; marca `is_active=false` no Torque (não deleta — propostas históricas dependem).
- **Lead sem dados fiscais completos**: cria pedido pendente de validação.
- **Múltiplos Torques conectados ao mesmo TinyERP**: não-suportado oficialmente; dedupe via `external_id` deve ajudar mas cuidado.
- **Schema do TinyERP muda**: adaptador versionado; admin atualiza.
- **Grande volume** (10k produtos): pull em lotes; background job.

## Rate Limits

- TinyERP tem limite por token (respeitar).
- Backoff quando retorna 429.

## Fallback

- Sem TinyERP: admin cadastra produtos manualmente.
- Sem criação de pedido automática: exporta CSV com vendas para admin carregar no ERP.

## Segurança

- Token em vault.
- HTTPS.
- Nunca expor token ao cliente.
- Log de chamadas sem conteúdo sensível.

## Observabilidade

- Log de sync.
- Métricas: produtos sincronizados, pedidos criados, failures.
- UI de integração mostra último sync e status.

## LGPD

- Dados de lead enviados ao ERP (nome, telefone, eventualmente CNPJ): consentimento ao virar cliente.
- Organização é o controller; TinyERP o operator (terceirizado).

## Evolução

- Integração com outros ERPs brasileiros (Bling, Omie) seguindo mesmo padrão de adaptador.
- Sync de notas fiscais para analytics financeiros mais ricos.
