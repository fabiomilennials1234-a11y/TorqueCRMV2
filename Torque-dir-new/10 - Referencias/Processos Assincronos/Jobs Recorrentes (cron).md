---
tipo: async
---

# Jobs Recorrentes (Cron)

Catálogo dos jobs agendados que rodam em intervalos fixos. Todos são disparados por um agendador (cron-like) e executados por workers.

## Princípios

- Cada job declara frequência, batch size e idempotência.
- Particionam por organização quando volume justifica.
- Log estruturado com `job_name`, `run_id`, `duration`, `result`.
- Dead letter em falha persistente.
- Alertas em atraso (se job não roda há mais de X tempo).

## Catálogo

### 1. Process Webhook Deliveries
- **Frequência**: cada minuto.
- **Batch**: 100 deliveries por run.
- **Função**: consome fila de webhook deliveries pendentes, entrega a endpoints externos, retry em falha.
- **Duração típica**: 5-30s.
- **Idempotente**: sim (via `delivery.attempts`).

### 2. Process Workflow Executions
- **Frequência**: cada minuto.
- **Batch**: 20.
- **Função**: avança workflows em `waiting` cujo resume_at expirou. Processa próximo node.
- **Idempotente**: sim (step tem `completed_at`).

### 3. Process Outbound Dispatches (Campanha)
- **Frequência**: cada 5 min.
- **Batch**: variável (respeita rate limit).
- **Função**: envia mensagens agendadas de campanhas.
- **Respeita**: rate limit, janela de negócio.

### 4. Process AI Actions
- **Frequência**: cada minuto.
- **Função**: processa AI Actions pendentes (actions do agente enfileiradas).
- **Exemplos**: aplicar tag, mover stage pedida pelo agente.

### 5. Process Copilot Followups
- **Frequência**: cada 5 min.
- **Função**: avalia follow-up rules do agente; dispara se satisfaz trigger (no_response_after_hours, etc.).

### 6. Process Followup Automations
- **Frequência**: cada 5 min.
- **Função**: avalia follow-ups manuais com `due_at` próximo; alerta atribuído.

### 7. Process Scheduled User Messages
- **Frequência**: cada minuto.
- **Função**: envia mensagens agendadas manualmente pelo usuário.

### 8. Campaign Rule Dispatch
- **Frequência**: cada minuto.
- **Função**: avalia regras de campanha ativas; avança entries por stage.

### 9. Pipe Rule Dispatch
- **Frequência**: cada minuto.
- **Função**: avalia dispatch rules de pipe (ex.: SLA expirado em stage).

### 10. Retry Dead Letter Jobs
- **Frequência**: cada 5 min.
- **Função**: re-tenta jobs em dead letter com política de recovery.

### 11. Refresh OAuth Tokens
- **Frequência**: diário 2h AM.
- **Função**: renova tokens Meta, Google Calendar próximos de expirar.

### 12. Pipeline SLA Check
- **Frequência**: cada 15 min.
- **Função**: escaneia pipeline entries; aciona `sla_expired` trigger nos que excederam tempo.

### 13. Confirmação Temporal (Pipe Confirmação)
- **Frequência**: cada 15 min.
- **Função**: avança entries em Pipeline Confirmação pelas stages D-5/D-3/D-1/dia.

### 14. Score Recalculation
- **Frequência**: horário.
- **Função**: aplica decay temporal, recalcula scores estagnados.

### 15. Embeddings Indexing
- **Frequência**: cada minuto.
- **Função**: processa FAQs recém-criadas/editadas sem embedding.

### 16. Sync Produtos (ERP)
- **Frequência**: cada 1h (por org com integração).
- **Função**: puxa delta do ERP e upsert local.

### 17. Upsell Opportunity Generator
- **Frequência**: diário 9h AM.
- **Função**: avalia regras de upsell; cria oportunidades.

### 18. Quota Reset
- **Frequência**: diário 0h (UTC ou fuso da org — mensal reset mês).
- **Função**: reseta contadores de quota diária/mensal.

### 19. Analytics Materialization
- **Frequência**: diário 3h AM.
- **Função**: refresh de views materializadas para analytics histórico.

### 20. Cleanup
- **Frequência**: diário.
- **Função**: limpa dados expirados (tokens antigos, URLs de mídia expiradas, soft-deletes muito antigos em janela de carência).

### 21. Backup Database
- **Frequência**: diário + incremental cada hora.
- **Função**: backup automático com retenção configurada.

### 22. Monitoring Healthchecks
- **Frequência**: cada minuto.
- **Função**: pinga endpoints internos + integrações; marca saúde.

## Autenticação

- Jobs chamam endpoints internos.
- Header `X-Cron-Secret: <secret>`.
- Secret rotacionável em vault.
- Nunca expor a cliente.

## Particionamento

Para orgs grandes, jobs particionam:
- Lista todas orgs ativas.
- Itera paralelo (com limite).
- Cada org processa seus dados.
- Alternativa: fila de "org_to_process" e workers puxam.

## Observabilidade

- Cada run:
  - `job_name`, `started_at`, `completed_at`, `duration_ms`.
  - `items_processed`, `items_failed`.
  - `result`: success / partial / failed.
- Métricas:
  - Runs por hora.
  - Duração média.
  - Backlog (se fila).
- Alertas:
  - Job não rodou há > X tempo.
  - Duração cresce (sinal de gargalo).
  - Taxa de falha > threshold.

## Edge Cases

- **Job demora demais**: próximo agendamento sobrepõe. Solução: lock por nome de job; novo run só se anterior terminou.
- **Servidor reinicia no meio**: item marcado como em-progresso; tem timeout; próximo run retoma.
- **Dependency down** (ex.: LLM): job falha gracefully; retry; dead letter após N.
- **Org suspensa**: jobs pulam orgs inativas.

## Deploy

- Crontab ou equivalente (pg_cron, cloud scheduler, k8s cronjobs).
- Mudança de frequência versionada.
- Pausar job global: flag no config.

## Segurança

- Secret do cron não em código.
- Worker valida header antes de executar.
- Audit log: operação automática com actor=`system`.
