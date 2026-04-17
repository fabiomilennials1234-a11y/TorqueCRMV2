---
tipo: integracao
direcao: out
criticidade: baixa
---

# Observabilidade (Captura de Erros via Sentry)

Integração com serviço externo de captura de exceções e monitoramento — atualmente Sentry. Usado internamente pelo Torque; não é feature exposta ao cliente.

## Propósito

- Capturar exceções não-tratadas em produção.
- Fornecer contexto (org_id, user_id, trace) ao time de engenharia.
- Monitorar releases (regressões após deploy).
- Alertas em picos de erros.
- Trace de performance (latência por endpoint, cold start).

## Provedor

Sentry (SaaS) — principal.
Alternativas: Rollbar, Bugsnag, Datadog APM, New Relic.

## Contrato

- Cada serviço (backend, frontend, worker) tem DSN de Sentry.
- SDK envia exceções + contexto.
- Release tracking.
- Performance transactions.

## Autenticação

- DSN configurado como secret.
- Não tem "usuário" — serviço identifica o projeto Sentry pelo DSN.

## Captura de Exception

Quando erro não-tratado ocorre:
1. SDK captura stack trace.
2. Adiciona contexto enriquecido:
   - `organization_id` (se disponível).
   - `user_id`.
   - `environment` (production, staging, dev).
   - `release` (versão do deploy).
   - `request_id` para correlação.
   - Tags custom (ex.: feature="copilot").
3. Envia ao Sentry.

## Contexto Adicionado

- Breadcrumbs: últimas N operações antes do erro.
- HTTP context: método, path, status.
- User context: id, email (mascarado).
- Tenant context: organization_id.
- Release info: SHA, deploy time.

## Regras de Negócio

1. PII mascarado antes de enviar.
2. Senhas, tokens, keys nunca em erros enviados.
3. Taxa de sampling configurável (100% de errors; 10% de performance).
4. Rate limit pelo provider — protege custo.
5. Alertas configurados no Sentry para events críticos.

## Edge Cases

- **Sentry offline**: SDK tem queue local; retry quando volta.
- **Quota Sentry excedida**: events descartados; alerta ao time.
- **Exception recorrente** (mesma stack 1000x): Sentry agrupa; não inflaciona quota.

## Observabilidade do Próprio Sistema

- Além de Sentry: logs estruturados (stdout em worker/backend), métricas (Prometheus / similar), traces (OpenTelemetry).
- Sentry é ponto focal para erros.

## Segurança / Privacidade

- PII: nome, telefone, email de lead mascarados em Sentry.
- Conteúdo de mensagem nunca em Sentry.
- Acesso ao Sentry restrito (poucos engenheiros).
- Compliance: contrato com Sentry inclui DPA (data processing agreement).

## Métricas

- Exceptions/dia.
- Top errors (para priorização).
- Release health: % de sessions com erro após novo deploy.

## Alertas

- Taxa de erro > X% em 5 min.
- Novo tipo de erro detectado após deploy.
- Erro em feature crítica (copilot, checkout).

## Evolução

- Integração com OpenTelemetry para traces mais ricas.
- Self-hosted alternativa para clientes enterprise que exigem on-premise.
