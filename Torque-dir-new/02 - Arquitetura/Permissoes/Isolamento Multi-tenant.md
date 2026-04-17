---
tipo: identidade
---

# Isolamento Multi-tenant

Este documento complementa [[01 - Arquitetura Conceitual/Multi-tenancy]] do ponto de vista de identidade e acesso. Explica como identidade de usuário e contexto de organização se combinam para garantir isolamento estrito.

## Contexto de Identidade

Cada requisição autenticada carrega:

- `user_id` (identidade global do usuário).
- `organization_id` (organização ativa).
- `team_member_id` (vínculo do usuário na organização).
- `role` + flags (master, impersonation).

## Origem do Contexto

- **Token de sessão**: access_token carrega claims com `user_id`, `organization_id`, `role`. Derivado no login/switch.
- **API key**: resolve para `organization_id` + scopes. Não há `user_id` (salvo se key for em nome de um membro).
- **Webhook inbound**: token na URL ou HMAC → resolve para `organization_id`.
- **Cron / worker interno**: payload carrega `organization_id` explicitamente. Validado contra `X-Cron-Secret`.

## Regras Obrigatórias

### R1 — Nenhum endpoint aceita `organization_id` como input do cliente

- Exceção: endpoints master (`/master/*`) que por definição operam cross-org. Nestes, master explicita `organization_id` e a engine registra auditoria.

### R2 — Toda consulta à persistência inclui filtro por `organization_id`

- Mesmo quando feita por helper ou ORM, o filtro é obrigatório.
- Último-bastião: política de row-level security (RLS) do banco rejeita linhas fora da org.

### R3 — Nenhuma entidade filha cruza organização

- Não é possível: Lead da org A ser adicionado a Pipeline da org B.
- Validação em domínio: ao associar, verificar `entity_a.organization_id == entity_b.organization_id`.

### R4 — Subscriptions realtime filtram por organização na origem

- Cliente tenta assinar `organization:Y:*` quando token é da org X → servidor nega.
- Não é "servidor envia tudo e cliente filtra" — é "servidor só envia o que é devido".

### R5 — Jobs assíncronos são per-org

- Cada job carrega `organization_id` no payload.
- Worker hidrata contexto antes de executar lógica.
- Agendador que opera em várias orgs faz loop explícito.

### R6 — Integrações externas têm credenciais per-org

- Tokens OAuth (Google, Meta) isolados.
- API keys de provedores (Asaas, TinyERP) por org.
- Falha de credencial em org A não contamina org B.

### R7 — Secrets de infraestrutura não são por-org

- Secret do cron job interno, secret HMAC da plataforma, etc. são globais.
- Esses secrets nunca são expostos ao cliente.

## Master Admin: Bypass Auditado

### Como master opera
1. Master loga em `/master/*`.
2. Para operar em org cliente, explicita `organization_id` na requisição.
3. Engine aprova com flag `master_action`.
4. Audit log crítico registra: quem, qual org, qual ação, quando, justificativa.
5. Cliente da org tem log próprio (dentro dele) de que master fez X em Y (visibilidade parcial ou total, configurável por caso).

### Impersonação
- Master pode "atuar como admin" da org cliente temporariamente.
- Emitido token com `impersonation: {original_user_id, target_organization_id, target_member_id, expires_at}`.
- UI mostra banner forte.
- Ações ficam registradas como "feitas por admin X via master Y".
- Duração limitada (ex.: 1h) com extensão explícita.

### Limites
- Master não pode alterar dados sensíveis de cliente sem justificativa.
- Algumas ações nunca são permitidas mesmo a master (ex.: ler conteúdo de conversa específico, salvo em investigação autorizada pelo cliente).

## Trocas de Organização

Usuário pertence a múltiplas orgs (ex.: consultor com acesso a 3 clientes):

- Login autentica identidade global.
- UI oferece menu de orgs. Usuário escolhe.
- Sistema emite token com `organization_id` atual.
- Ao trocar: front limpa cache, backend emite token novo.
- Subscriptions realtime são refeitas para o novo escopo.

## Dados Globais vs Per-org

### Globais (não têm `organization_id`)
- Catálogo de planos de assinatura.
- Catálogo de templates de agente IA padrão.
- Tipos de node de workflow.
- Catálogo de integrações suportadas.
- Usuários (entidade de identidade global); vínculo a org é separado.

### Per-org (têm `organization_id`)
- Todo o resto.

## Teste de Isolamento

Deve existir conjunto de testes automatizados que, para cada endpoint sensível:

1. Cria dois tenants A e B com dados distintos.
2. Autentica como usuário de A.
3. Tenta ler/escrever/deletar recurso de B.
4. Verifica que resposta é negativa (404 / 403 sem vazar detalhes).
5. Verifica que não houve efeito colateral em B.

Complementar: testes que validam RLS do banco (queries construídas "sem" filtro de org devolvem dados apenas da org ativa).

## Vazamento de Informação por Erro

- Mensagem de erro não deve **confirmar existência** de recurso em outra org.
  - ❌ "Lead 123 não pertence à sua organização".
  - ✔ "Lead não encontrado".
- Autocomplete, contagens, sugestões: nunca ultrapassam escopo de tenant.
- Stack traces em produção não expõem caminhos internos.

## Anti-padrões

- ❌ Helper `getAllLeads()` sem `organization_id` — até em código "administrativo" deve receber escopo.
- ❌ Endpoint `/api/internal/leads-by-phone?phone=...` que busca cross-tenant.
- ❌ Cache de resposta compartilhado entre tenants (key sem `organization_id`).
- ❌ Webhook URL sem token — permite adivinhar e injetar em qualquer tenant.
- ❌ Log que contém dado de tenant A emitido em contexto de tenant B.

## Auditoria de Isolamento

- Logs operacionais sempre carregam `organization_id`.
- Alertas: qualquer operação sem `organization_id` em log de nível produção é bug (investigar).
- Dashboards de operação separados por organização quando possível (evitar mistura visual).

## Processos Especiais

### Export em lote (cross-org para analytics da empresa-dona)

- Feito por master apenas.
- Dados anonimizados ou agregados (sem PII cross-org).
- Audit log.
- Consentimento dos clientes previsto em contrato/ToS.

### Ferramentas internas de suporte

- Acesso somente master.
- Preferir view resumida sem conteúdo sensível (sem conversas em plaintext).
- Se precisar ver conteúdo: workflow com justificativa + aprovação + auditoria reforçada.

## Recuperação de Desastre e Isolamento

- Backup é per-tenant quando possível (restaurar uma org sem afetar outras).
- Alternativa: backup global com ferramenta de restauração seletiva por org.

## Tokens e Claims

- Access token carrega `organization_id` atual.
- Claim `allowed_organizations`: lista de orgs em que o usuário tem acesso. Usado para validar switch.
- `permissions_version`: incrementa quando permissão muda; força re-check.
- Assinatura criptográfica do token não pode ser forjada.

## Compartilhamento Consentido (futuro)

Futuras features podem permitir compartilhar dado pontual com org parceira (ex.: integração entre empresas):

- Compartilhamento sempre explícito.
- Visível em audit.
- Revogável a qualquer momento.
- Nunca quebra invariante de "org A não vê dados de org B" — o que B vê é **cópia** consentida, não acesso direto.
