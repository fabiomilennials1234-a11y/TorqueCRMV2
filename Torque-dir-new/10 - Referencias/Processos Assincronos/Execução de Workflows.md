---
tipo: async
---

# Execução de Workflows

Motor que consome `WorkflowExecution` entities e processa node a node. Gerencia estado, retry, concorrência, persistência de steps.

## Arquitetura

```
Evento de domínio (LeadCreated, StageChanged, etc.)
        │
        ▼
Engine consulta workflows ativos cujo trigger matche
        │
        ▼
Cria WorkflowExecution (status=pending, current_node=trigger)
        │
        ▼
Fila de execução
        │
        ▼
Worker (job process-workflow-executions, cada minuto, batch 20):
  - Pega executions status=pending ou status=waiting com resume_at<=now
  - Para cada:
    - status = running
    - Hidrata contexto (entity data, variáveis)
    - Processa node corrente
      - trigger: no-op, captura data, move para próximo
      - action.*: chama domínio, executa side effect
      - condition: avalia expressão, segue edge true/false
      - delay: status=waiting, resume_at = now + delay
      - wait_response: status=waiting, listener em conversa
      - split_ab: sorteia edge
      - parallel: cria sub-executions
      - join: espera condição
      - end: status=completed
    - Persiste ExecutionStep
    - Se próximo node: segue; se não, status=completed
  - Commit
```

## Hidratação de Contexto

No início de cada processamento:
- Carrega entidade principal (lead).
- Carrega campos referenciados em variáveis (`{{ lead.custom_fields.budget }}`).
- Carrega resultado de steps anteriores (output disponível como `{{ context.step_X.result }}`).

## Estados da Execution

- `pending`: aguardando primeiro processamento.
- `running`: em execução síncrona no worker.
- `waiting`: pausado (delay, wait_response).
- `completed`: chegou a `end`.
- `failed`: erro irrecuperável.
- `cancelled`: admin ou evento cancelou.

## Timer para Waiting

- `delay`: `resume_at = now + delay_duration`.
- `wait_response`: `resume_at = now + timeout` (se timeout configurado).
- Worker `scan` para `resume_at <= now`: retoma.
- `wait_business_window`: calcula próxima janela, seta `resume_at`.

## Listener para Wait Response

- Worker adiciona "listener" em memória (ou persistente por conversa).
- Quando mensagem inbound chega nessa conversa: listener notifica a execution para retomar.
- Se `timeout_hours` expira sem mensagem: execution retoma por timeout, segue edge `timeout` se existir.

## Concorrência

### Worker Concurrency
- Múltiplos workers em paralelo.
- Lock por `execution_id` durante processamento evita double-processing.
- Tempo máximo de lock (ex.: 5 min); se worker cai, outro pega.

### Execuções por Lead
- Mesmo lead pode ter múltiplas executions simultâneas (de workflows diferentes).
- Workflow com `mutex_per_lead=true`: nova execution cancela anterior (ou rejeita, config).

## Retry

- Action falha:
  - Erro transiente (timeout, 5xx de integração): retry até 5x com backoff.
  - Erro permanente (validação, 4xx): sem retry, step=failed.
- Execution com step failed:
  - Se edge `on_failure` existe: segue por ela.
  - Senão: status=failed.

## Dead Letter

- Execution failed fica no banco.
- UI lista executions failed, admin pode inspecionar e, se aplicável, manualmente reprocessar.

## Parallel e Join

### Parallel
- Node `parallel` cria N sub-executions, uma por branch.
- Main execution status=waiting.
- Cada sub roda independentemente.

### Join
- Após parallel, join espera:
  - `wait_for: "all"`: todas as branches completam.
  - `wait_for: "any"`: primeira que completa libera.
- Quando condição satisfeita: main execution retoma.

## Variáveis e Templating

- Cada step tem `context` acumulado.
- Variáveis referenciáveis em nodes via `{{ ... }}`.
- Set explícito: action `set_variable(key, value)` adiciona a `context`.
- Context limite (ex.: 1MB JSON) para evitar explosão.

## Versionamento

- Workflow tem `version`.
- Execution snapshot: ao criar, `workflow_version` registrado.
- Edits ao workflow não retroagem a executions em curso.
- Workflow desativado: executions em andamento completam; novas não disparam.

## Garbage Collection

- Executions em `waiting` há > max_execution_time (default 7d): cancelled automaticamente.
- Executions completed com > 90 dias: arquivadas (mantidas acessíveis mas fora do path quente).

## Observabilidade

### Logs
- Cada step: `execution_id`, `step_id`, `node_type`, `duration`, `result`, `error`.

### Métricas
- Executions/hora por workflow.
- Taxa de sucesso.
- Duração média (execution completa).
- Gargalo: node type/step com mais tempo ou falha.

### UI
- Lista de executions filtráveis (status, workflow, período, lead).
- Detalhe: grafo com highlight do path.
- Step detail: input, output, duração, erro.

## Edge Cases

- **Workflow editado durante execução**: execution usa versão snapshot.
- **Lead deletado**: execution falha no próximo step que referencia (validação); termina como failed.
- **LLM indisponível em node `copilot`**: retry; se falha, step=failed; execution segue edge `on_failure` ou fail.
- **Loop sem saída**: contador `max_iterations` (100 default).
- **Context explode em tamanho**: truncamento ou erro explícito.
- **Delay de meses**: aceita mas UI avisa.
- **Executions em massa** (10k disparando do mesmo evento): worker escala horizontal.

## Segurança

- Context pode conter PII: não logado em plaintext em info.
- Actions que manipulam dados sensíveis respeitam permissões.
- Webhook outbound de node `call_webhook` valida URL (SSRF).

## Performance

- Worker escala horizontalmente.
- Batch size ajustável.
- Particionamento por org em orgs grandes.
- Índices em `(status, resume_at)`, `(organization_id, workflow_id, status)`.

## Auditoria

- Cada step é audit ponto em tempo.
- Permite reconstituir exatamente o que aconteceu.
- Debug post-mortem de issues "por que workflow X não fez Y para lead Z".
