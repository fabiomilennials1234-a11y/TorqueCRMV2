---
tags: [migracao, legado, exclusao, decisoes]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Itens Não Migrados

Lista explícita de itens do vault legado que **deliberadamente ficam fora** do Torque-v2. Cada exclusão tem justificativa. Este documento serve de guardrail para evitar que decisões ruins do passado voltem por inércia ou por referência inconsciente ao legado.

Complementa [[Mapa do Vault Legado]] e [[Top 15 Docs Prioritarios]].

## D5 do ADR inicial — Supabase BaaS

A decisão D5 do legado optava por Supabase como Backend-as-a-Service: edge functions Deno, auth via GoTrue, realtime via Supabase Realtime, storage integrado.

**Por que não migra.** Backend Go próprio substitui completamente. Supabase introduzia coupling com infraestrutura proprietária, limitava controle de auth (cookies httpOnly não suportados nativamente), e tornava o modelo de permissões dependente de RLS via SQL em vez de middleware com visibilidade de contexto completo. Ver [[ADR-002 Realtime Patch Model]] para a decisão de realtime que substitui Supabase Realtime.

## Todo `_shared/` de Edge Functions Deno

O legado continha funções de edge (Deno Deploy) com pasta compartilhada `_shared/` entre funções para utilitários como CORS, logging, validation.

**Por que não migra.** Backend Go usa módulos Go com contratos de interface, não funções soltas com compartilhamento por import de pasta. A abstração muda de "funções stateless invocadas por HTTP" para "serviço com ciclo de vida, middleware chain e DI via constructors".

## pg_cron + pg_net

O legado usava `pg_cron` para agendamento de jobs e `pg_net` para fazer requisições HTTP diretamente do PostgreSQL.

**Por que não migra.** Substituído por worker pool Go com scheduler próprio. Mover lógica de aplicação para dentro do banco viola separação de camadas e torna observabilidade (tracing, correlation ID, structured logging) impraticável. Scheduler Go permite backoff, retry com dead-letter, métricas e cancelamento graceful.

## Nomes de edge functions

Funções como `agent-message`, `evolution-webhook`, `broadcast-send`, `copilot-response` eram endpoints Deno com nomes ad hoc.

**Por que não migra.** Viram endpoints REST ou RPC convencionais no Go, seguindo padrão de rotas `POST /api/v1/{recurso}/{ação}`. Naming do legado era descritivo mas inconsistente (mix de resource-based e action-based sem padrão).

## Arquivos `supabase/` — config.toml, migrations SQL com RLS

Toda a pasta `supabase/` do legado, incluindo `config.toml`, `seed.sql` e migrations com políticas RLS em SQL.

**Por que não migra.** Banco gerenciado externamente (Postgres puro, sem acoplamento Supabase). Migrations via `golang-migrate` com arquivos `.sql` versionados em `backend/migrations/`. RLS via SQL é substituído por middleware Go que aplica `tenant_id` em toda query via contexto de request, sem depender de features proprietárias do Postgres.

## lovable-tagger (CONCERN-D1)

Artefato do framework Lovable injetado automaticamente no build. Adicionava metadados e tracking ao DOM.

**Por que não migra.** Removido. Torque-v2 não usa Lovable. Qualquer instrumentação de DOM é explícita via Sentry Replay (e apenas em erros, ver [[Observabilidade e Logs]]).

## `lib/seed.ts` do frontend atual

Arquivo de seed usado para alimentar a UI com dados fictícios durante o desenvolvimento com Supabase ausente.

**Por que não migra.** Congelado como referência de mockup. Em F01 será substituído por dados reais via API Go. Seeds de desenvolvimento vivem em `backend/seed/` como SQL inserível, não como constantes TypeScript no bundle do front.

## `verify_jwt = false` (CONCERN-S3)

O legado desabilitava verificação de JWT em certas edge functions para simplificar desenvolvimento.

**Por que não migra.** Go faz verificação JWT nativa em middleware, obrigatória em todo endpoint autenticado. Nenhuma rota protegida aceita request sem token válido. O anti-padrão de desabilitar verificação não existe como opção na arquitetura nova.

## Service role key no bundle (CONCERN-S1)

O legado expunha a service role key do Supabase no bundle do frontend, dando acesso irrestrito ao banco.

**Por que não migra.** Eliminado por design. Secrets existem apenas server-side. Frontend nunca recebe credenciais que deem acesso direto a banco ou a serviços internos. Chave de API pública é limitada por rate limiting e por scoping de tenant.

## `enable_confirmations = false` (CONCERN-S5)

O legado desabilitava confirmação de email no fluxo de registro para simplificar onboarding em dev.

**Por que não migra.** Email de confirmação é obrigatório no Go, em todos os ambientes incluindo dev (usando Mailpit local). Contas sem email confirmado não acessam nenhum recurso. Esta decisão evita contas fantasmas e reduz superfície de abuso.

## Motivo geral

O Sistema Base do Torque-v2 existe como oportunidade de **não reproduzir os débitos do legado**. Cada item nesta lista representa uma decisão do passado que facilitou desenvolvimento imediato ao custo de segurança, qualidade ou manutenibilidade. A nova base começa sem esses débitos, e este documento garante que eles não voltem por acidente.

Quando alguma necessidade futura parecer "resolver" com um destes itens, a resposta correta é encontrar a solução equivalente na arquitetura nova, não ressuscitar o padrão antigo. Se nenhuma alternativa satisfizer, o assunto vira item em [[Riscos e Duvidas]] para decisão explícita.
