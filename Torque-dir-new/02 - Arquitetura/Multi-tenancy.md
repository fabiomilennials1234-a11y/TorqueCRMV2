---
tags: [arquitetura, multi-tenancy, isolamento, seguranca]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Multi-tenancy

Torque e um SaaS B2B. Cada organizacao cliente e um tenant isolado. Qualquer vazamento entre tenants e um incidente de severidade maxima. A arquitetura nao permite que vaze, por construcao.

## Invariante absoluta

**`organization_id` e extraido exclusivamente do JWT pelo servidor Go.**

Consequencias diretas:
- Nenhum endpoint aceita `organization_id` em body ou query de mutation. O middleware de tenant-scope rejeita com 400 se o campo aparecer.
- O frontend **nao envia** `organization_id` em body. O interceptor de `src/lib/fetch` faz strip preventivo em todo payload de POST/PATCH/PUT, como segunda linha de defesa.
- Componentes nao recebem `orgId` como prop em operacoes de escrita. Para leitura, `orgId` e sempre inferido do `useSession()`, nunca passado adiante por prop drilling.

Essa invariante e **validada em testes de arquitetura**: lint rule custom proibe literal `organization_id` em qualquer arquivo sob `src/api/` e `src/hooks/` em posicao de mutation body.

## Isolamento em persistencia

Toda tabela de dominio tem coluna `organization_id NOT NULL`. Sem excecoes:
- `leads`, `team_members`, `conversations`, `channel_messages`, `workflows`, `workflow_executions`, `campaigns`, `copilot_agents`, `followups`, `products`, `commissions`, `pipe_*`.
- Tabelas de sistema (`users`, `users_master`, `plans`, `feature_permissions`) nao sao tenant-scoped; sao globais.

Index composto `(organization_id, <coluna secundaria>)` em todas as queries frequentes. Nunca consultar por coluna secundaria sem prefixo de `organization_id` no WHERE.

## RLS-equivalente em Go

Sem depender de RLS do Postgres (substituido junto com o stack anterior), o isolamento vive em middleware + repository:

```
Handler chain:
  AuthMiddleware        -> extrai JWT, popula context com user_id
  TenantScopeMiddleware -> extrai organization_id do JWT, injeta em context
  RBACMiddleware        -> valida permissao para a rota
  Handler               -> usa repository
  Repository            -> toda query recebe ctx, aplica WHERE organization_id = ctx.OrgID automaticamente
```

Regras de review:
- Toda funcao de repositorio recebe `context.Context` como primeiro parametro.
- Proibido `SELECT` sem filtro por `organization_id` em tabelas tenant-scoped. CI tem linter SQL que falha PR.
- Repositorio nao exporta metodo que receba `organizationID` como parametro livre. O tenant vem sempre do contexto. Excecao unica: operacoes de Master Admin, ver abaixo.

## Master Admin cross-org

O Master Admin precisa operar em qualquer tenant (suporte, auditoria, migracao de dados).

- UI: `<OrgSwitcher>` no header, visivel apenas se `is_master === true`. Muda o tenant ativo da sessao.
- Wire: ao mudar org, cliente envia `POST /master/impersonate/:org_id`. Backend valida `is_master`, emite novo JWT com claim `impersonating_org_id`. Cookie e reescrito.
- A partir dai, todas as requisicoes levam o tenant novo no JWT, fluindo pelo mesmo middleware.
- **Bypass explicito em audit log:** cada requisicao feita com `impersonating_org_id` popula `audit_log.actor_type='master'`, `audit_log.actor_user_id=<master>`, `audit_log.target_org_id=<org_impersonada>`.
- Cliente mostra banner permanente "Visualizando como Master em Acme Inc" com botao sair.

Clientes comuns **nunca veem** a existencia do Master. Nao aparece em `/team`, nao aparece em mentions, nao aparece em audit log do cliente.

## Quotas: delta model

Quotas nao sao um numero fixo. Sao compostas:

```
effective_limit = plan_base + purchased_addons + admin_adjustment
```

- `plan_base`: definido pelo plano contratado (vem da tabela `plans`).
- `purchased_addons`: add-ons comprados avulso (ex: +5 membros).
- `admin_adjustment`: ajuste manual por Master (positivo ou negativo, grava audit log).

Fontes separadas e composicao explicita tem tres vantagens:
1. Historico claro: ninguem precisa calcular retroativamente o motivo de um limite.
2. Billing simples: upgrade de plano nao destroi add-ons comprados.
3. Override de suporte: Master pode conceder limite extra sem mexer em plano nem billing.

Tabela `org_quotas`:
```
organization_id  resource         plan_base  purchased_addons  admin_adjustment  current_usage
org_acme         leads            5000       1000              0                 3421
org_acme         team_members     10         3                 1                 11
org_acme         workflows        20         0                 0                 14
```

`current_usage` e incrementado/decrementado transacionalmente junto com a criacao/delecao do recurso. Nunca calculado por `COUNT(*)` em request-time.

Frontend consome apenas `effective_limit` e `current_usage` (ja calculados no `/auth/me`) mais `can_add` (boolean derivado).

## Uploads

- Path scheme no object storage: `org/<org_id>/<entidade>/<yyyy>/<mm>/<uuid>.<ext>`.
- Bucket policy rejeita reads que nao venham com URL pre-assinada.
- URL pre-assinada e gerada pelo backend apos validar que o `entity_id` referenciado pertence ao tenant do usuario.

## Logs e metricas

- Toda linha de log de aplicacao inclui `organization_id` quando existir contexto de tenant.
- Dashboards agregam por tenant. Alertas de abuso sao por tenant, nao globais.
- PII em log e proibido. `organization_id` e um UUID, nao e PII; nome de empresa e.

## Referencias

- [[Arquitetura do Sistema]]
- [[Autenticacao e Autorizacao]]
- [[Contratos e Boundaries]]
- [[Seguranca Web]]
- [[Realtime e Jobs]]
- [[Glossario e Vocabulario]]
---
tipo: arquitetura
---

# Multi-tenancy

Isolamento entre organizações é **o invariante mais crítico** do sistema. Esta página descreve como o isolamento é modelado e garantido em todas as camadas.

## Modelo

- Cada organização é um **tenant**.
- Toda entidade de domínio (exceto entidades verdadeiramente globais como planos de assinatura, catálogo de integrações disponíveis) tem um campo conceitual `organization_id` (ou equivalente) apontando para o tenant proprietário.
- Usuários pertencem a uma ou mais organizações. Em regime normal, um usuário opera sobre uma organização ativa por vez (o token carrega o contexto).
- Master admins (usuários da empresa-dona do produto) podem atuar cross-org. Toda ação deles deixa rastro de auditoria explícito.

## Invariantes

1. **Nenhum dado de uma organização deve ser observável por outra.** Isso inclui: listas, contagens, IDs, sugestões em autocomplete, mensagens de erro que vazem existência.
2. **Se o escopo de organização não está disponível, a operação falha.** Não há "padrão global" — na ausência de tenant, a resposta é 401/403.
3. **Webhooks externos** não pulam o isolamento: o handler precisa identificar a organização antes de qualquer gravação (via token/secret embutido na URL do webhook ou no payload).
4. **Jobs recorrentes** operam em lote, mas cada iteração escolhe uma organização por vez ou particiona claramente.
5. **Busca vetorial (RAG)** filtra por `organization_id` antes de calcular similaridade — nunca depois.

## Isolamento em cada camada

### Apresentação
- UI recebe do token, no login, o `organization_id` ativo.
- Nunca envia `organization_id` manualmente em chamadas — o backend deduz.
- Ao trocar de organização (usuário master ou usuário com múltiplas orgs), faz logout do estado de cache anterior.

### Aplicação
- Toda rota autenticada extrai `organization_id` do token.
- Todo caso de uso recebe `organization_id` como primeiro parâmetro implícito.
- Toda consulta à persistência adiciona filtro por `organization_id` — mesmo quando feita via helper.
- Nenhum endpoint aceita `organization_id` como input do cliente.

### Domínio
- Entidades com referência a `organization_id` validam em construção que o valor é consistente (não pode criar Lead apontando para Org X enquanto o contexto é Org Y).
- Serviços de domínio que atuam em múltiplas entidades validam que todas pertencem à mesma organização antes de operar.

### Persistência
- Política de acesso no próprio banco (row-level security ou equivalente) garante que **mesmo uma query sem cláusula `WHERE organization_id = ...`** devolve apenas linhas da organização ativa.
- Isso é a **última linha de defesa** contra vazamento por bug em camada superior.
- Índice composto `(organization_id, ...)` em todas as tabelas grandes para performance.

### Processamento assíncrono
- Cada job carrega `organization_id` no seu payload.
- Worker ao despachar sempre configura o contexto de tenant antes de executar a lógica.
- Agendador de cron particiona por organização quando aplicável (não processa todas as orgs na mesma iteração se isso for caro).

### Integrações externas
- Cada organização tem credenciais próprias para provedores (instância de WhatsApp, token Meta, API key Asaas, etc.).
- Credenciais ficam em store isolado, indexadas por `organization_id`.
- Webhooks externos: a URL inclui token específico da organização, ou o payload contém identificador que o backend resolve para `organization_id`.

## Master admin

- Pertence à organização especial do operador do produto.
- Recebe permissão para bypass de tenant em endpoints específicos.
- Toda ação master registra:
  - Identidade do master.
  - Organização alvo.
  - Ação realizada.
  - Justificativa (quando configurado pedir).
  - Timestamp.
- Cliente nunca vê que um master olhou/alterou dado dele (exceto se a ação modifica algo visível).
- Impersonação de organização é sessão temporária com badge visível para o master e auditoria reforçada.

## Provisionamento de nova organização

Quando uma nova organização é criada:

1. Gera-se novo identificador de tenant.
2. Usuário inicial é promovido a `admin` dessa organização.
3. São criadas entidades default: pipelines estruturais (WhatsApp, Confirmação, Propostas), tags básicas, planos default, templates iniciais.
4. Quota padrão do plano é aplicada.
5. Eventos de bootstrap são disparados (ex.: disparar onboarding wizard no primeiro acesso).

## Remoção / arquivamento de organização

- **Soft delete** por padrão: organização marcada como inativa, acesso bloqueado, dados preservados por período legal.
- **Hard delete** só por master admin e apenas após janela de carência.
- Quando hard delete ocorre: deleção em cascata de todas as entidades da organização, incluindo objetos em storage e vetores em banco vetorial.

## Anti-padrões a evitar

- ❌ Buscar lista de organizações para o usuário e então filtrar — sempre começar do contexto.
- ❌ Endpoints tipo `/api/admin/all-leads` que retornam cross-org sem checar master.
- ❌ Jobs que "pegam os últimos X eventos" sem particionar por organização quando há variação brutal de volume entre orgs.
- ❌ Compartilhar cache de resposta entre organizações.
- ❌ Gerar URLs de webhook sem token específico da organização.
- ❌ Logar `organization_id` como string crua em produção sem mascaramento quando sensível.

## Teste de isolamento

Deve existir um conjunto de testes automatizados que, para cada endpoint sensível:

1. Autentica como usuário da Org A.
2. Tenta acessar/modificar recurso da Org B.
3. Espera negação.
4. Verifica que a operação não deixou efeito colateral (nenhum log de acesso bem-sucedido).

Sem este suite, alterações em qualquer camada são arriscadas.
