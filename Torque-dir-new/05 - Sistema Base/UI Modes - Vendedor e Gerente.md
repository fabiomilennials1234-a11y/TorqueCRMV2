---
tags: [sistema-base, ui-modes, cockpit, tasks, schema, openapi]
created: 2026-04-17
last_updated: 2026-04-17
status: spec
decided_by: ADR-007
references: [F17 - Modo Vendedor (Task Cockpit)/Spec]
---

# UI Modes — Vendedor e Gerente

Documento operacional da extensão do Sistema Base decidida em [[ADR-007-modo-vendedor-gerente]]. Contém: persistência de preferência, schema SQL da entidade `Task` e coluna `ui_mode_preference`, excerto OpenAPI dos endpoints, contrato WS, e checklist de integração.

Spec detalhada de produto/UX: ver [[F17 - Modo Vendedor (Task Cockpit)/Spec]].

---

## 1. Schema SQL — migration

Localização futura: `torque-api/migrations/0003_tasks_and_ui_preferences.{up,down}.sql`. Dependências: migrations prévias `0001_organizations_users` e `0002_team_members_leads_pipes` (ainda inexistentes — documentadas apenas como intenção).

### 1.1 UP

```sql
-- =====================================================================
-- 0003_tasks_and_ui_preferences.up.sql
-- Extensão do Sistema Base: entidade Task + preferência de modo de UI.
-- Absorve e substitui a entidade legada 'followup' (nunca implementada).
-- Decidido em ADR-007.
-- =====================================================================

-- ---------- Enums ----------------------------------------------------

CREATE TYPE task_kind AS ENUM (
  'followup',
  'call',
  'qualification',
  'send_proposal',
  'confirm_meeting',
  'objection',
  'generic'
);

CREATE TYPE task_priority AS ENUM ('low', 'normal', 'high', 'urgent');

CREATE TYPE task_status AS ENUM (
  'pending',
  'in_progress',
  'done',
  'cancelled',
  'missed'
);

CREATE TYPE task_origin AS ENUM (
  'manual',
  'workflow',
  'agent',
  'rule',
  'system'
);

CREATE TYPE ui_mode AS ENUM ('manager', 'salesperson');

-- ---------- users: preferência de modo -------------------------------

ALTER TABLE users
  ADD COLUMN ui_mode_preference ui_mode NOT NULL DEFAULT 'manager';

-- Observação: o default 'manager' é de banco; o backend recalcula o
-- default real com base no role do usuário ao criar a linha.
-- Membros recebem 'salesperson' no handler de signup/invite.

-- ---------- tabela tasks ---------------------------------------------

CREATE TABLE tasks (
  id                uuid           PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid           NOT NULL
                                   REFERENCES organizations(id) ON DELETE CASCADE,
  lead_id           uuid           NULL
                                   REFERENCES leads(id) ON DELETE SET NULL,
  assigned_to       uuid           NOT NULL
                                   REFERENCES team_members(id) ON DELETE RESTRICT,
  created_by        uuid           NULL
                                   REFERENCES team_members(id) ON DELETE SET NULL,
  kind              task_kind      NOT NULL DEFAULT 'generic',
  title             text           NOT NULL CHECK (char_length(title) BETWEEN 3 AND 200),
  description       text           NULL      CHECK (char_length(description) <= 2000),
  priority          task_priority  NOT NULL DEFAULT 'normal',
  status            task_status    NOT NULL DEFAULT 'pending',
  in_queue          bool           NOT NULL DEFAULT false,
  queue_position    int            NULL,
  due_at            timestamptz    NULL,
  started_at        timestamptz    NULL,
  completed_at      timestamptz    NULL,
  completed_by      uuid           NULL
                                   REFERENCES team_members(id) ON DELETE SET NULL,
  cancelled_at      timestamptz    NULL,
  cancelled_reason  text           NULL,
  missed_reason     text           NULL,
  origin            task_origin    NOT NULL DEFAULT 'manual',
  context           jsonb          NULL,
  result_note       text           NULL      CHECK (char_length(result_note) <= 4000),
  created_at        timestamptz    NOT NULL DEFAULT now(),
  updated_at        timestamptz    NOT NULL DEFAULT now(),

  -- Integridade de estado --------------------------------------------
  CONSTRAINT tasks_queue_consistency CHECK (
    (in_queue = true  AND status = 'pending' AND queue_position IS NOT NULL)
    OR
    (in_queue = false AND queue_position IS NULL)
  ),
  CONSTRAINT tasks_in_progress_requires_started CHECK (
    status <> 'in_progress' OR started_at IS NOT NULL
  ),
  CONSTRAINT tasks_done_requires_completion CHECK (
    status <> 'done'
    OR (completed_at IS NOT NULL AND completed_by IS NOT NULL)
  ),
  CONSTRAINT tasks_cancelled_requires_reason CHECK (
    status <> 'cancelled'
    OR (cancelled_at IS NOT NULL AND cancelled_reason IS NOT NULL)
  )
);

-- ---------- Índices --------------------------------------------------

-- Multi-tenancy: tudo começa por organization_id.
CREATE INDEX idx_tasks_org_assignee_status
  ON tasks (organization_id, assigned_to, status);

CREATE INDEX idx_tasks_org_lead
  ON tasks (organization_id, lead_id)
  WHERE lead_id IS NOT NULL;

CREATE INDEX idx_tasks_org_created_at
  ON tasks (organization_id, created_at DESC);

-- Fila priorizada: ordenar por queue_position em pending+in_queue.
CREATE INDEX idx_tasks_queue
  ON tasks (organization_id, assigned_to, queue_position)
  WHERE status = 'pending' AND in_queue = true;

-- SLA cron: encontrar pending com due_at vencido.
CREATE INDEX idx_tasks_due_pending
  ON tasks (organization_id, due_at)
  WHERE status = 'pending' AND due_at IS NOT NULL;

-- ---------- Constraints parciais únicas ------------------------------

-- Uma única task in_progress por (org, assignee).
CREATE UNIQUE INDEX uq_tasks_one_in_progress_per_assignee
  ON tasks (organization_id, assigned_to)
  WHERE status = 'in_progress';

-- queue_position único dentro da fila de cada assignee.
CREATE UNIQUE INDEX uq_tasks_queue_position_per_assignee
  ON tasks (organization_id, assigned_to, queue_position)
  WHERE status = 'pending' AND in_queue = true;

-- ---------- Trigger updated_at ---------------------------------------

CREATE TRIGGER trg_tasks_updated_at
  BEFORE UPDATE ON tasks
  FOR EACH ROW
  EXECUTE FUNCTION set_updated_at();
-- (set_updated_at() é função utilitária global criada em 0001.)

-- ---------- Comentários ----------------------------------------------

COMMENT ON TABLE  tasks IS 'Unidade de trabalho atribuível; unifica follow-ups e demais tasks (ADR-007).';
COMMENT ON COLUMN tasks.in_queue IS 'True = aparece em "A fazer"; false = em "Em aberto" (backlog).';
COMMENT ON COLUMN tasks.queue_position IS 'Posição 1..N dentro da fila do assignee.';
COMMENT ON COLUMN tasks.origin IS 'Quem criou: manual (humano), workflow, agent (IA), rule (pipe rule), system.';
```

### 1.2 DOWN

```sql
-- =====================================================================
-- 0003_tasks_and_ui_preferences.down.sql
-- =====================================================================

DROP TRIGGER IF EXISTS trg_tasks_updated_at ON tasks;
DROP INDEX IF EXISTS uq_tasks_queue_position_per_assignee;
DROP INDEX IF EXISTS uq_tasks_one_in_progress_per_assignee;
DROP INDEX IF EXISTS idx_tasks_due_pending;
DROP INDEX IF EXISTS idx_tasks_queue;
DROP INDEX IF EXISTS idx_tasks_org_created_at;
DROP INDEX IF EXISTS idx_tasks_org_lead;
DROP INDEX IF EXISTS idx_tasks_org_assignee_status;
DROP TABLE IF EXISTS tasks;

ALTER TABLE users DROP COLUMN IF EXISTS ui_mode_preference;

DROP TYPE IF EXISTS ui_mode;
DROP TYPE IF EXISTS task_origin;
DROP TYPE IF EXISTS task_status;
DROP TYPE IF EXISTS task_priority;
DROP TYPE IF EXISTS task_kind;
```

### 1.3 Observações de DBA

- **Sem FK para `users`** em `assigned_to`/`created_by`: a relação é com `team_members` (associação usuário↔org). Preserva multi-tenancy: um user pode estar em múltiplas orgs, mas a task pertence à membership.
- **`ON DELETE SET NULL` em `lead_id`**: task sobrevive ao soft/hard delete de lead; UI mostra tag "arquivado".
- **`ON DELETE RESTRICT` em `assigned_to`**: banco protege contra delete acidental de membership com tasks pendentes. Admin precisa reatribuir antes (enforçado em API).
- **`ON DELETE CASCADE` em `organization_id`**: hard delete de org limpa tudo em cascata, conforme política de hard delete em [[Multi-tenancy]].
- **Constraints parciais únicas** são a garantia final da invariante "uma task in_progress por assignee" — independente de bug em código Go, o banco barra.
- **Reordenação da fila** usa transação: decrementar/incrementar `queue_position` em bulk via `UPDATE tasks SET queue_position = new_pos FROM (VALUES ...) v WHERE ...`. Não tentar update por update em loop (race).
- **Performance esperada**: até 10k tasks ativas por org, 200 assignees. Fila típica por assignee: 5-30 itens. Índices cobrem todos os acessos do cockpit bundle.

---

## 2. Excerto OpenAPI

Futuramente em `torque-api/api/openapi.yaml`. Excerto abaixo é a fonte de verdade até lá. Usa convenções de [[ADR-001-contratos-openapi-snake-camel]] (snake_case wire).

```yaml
openapi: 3.1.0
info:
  title: Torque API — Tasks & UI Preferences
  version: 0.1.0
  description: Excerto referente a F17 / ADR-007.

components:
  schemas:
    UiMode:
      type: string
      enum: [manager, salesperson]

    TaskKind:
      type: string
      enum: [followup, call, qualification, send_proposal, confirm_meeting, objection, generic]

    TaskPriority:
      type: string
      enum: [low, normal, high, urgent]

    TaskStatus:
      type: string
      enum: [pending, in_progress, done, cancelled, missed]

    TaskOrigin:
      type: string
      enum: [manual, workflow, agent, rule, system]

    Task:
      type: object
      required:
        - id
        - assigned_to
        - kind
        - title
        - priority
        - status
        - in_queue
        - origin
        - created_at
        - updated_at
      properties:
        id:               { type: string, format: uuid }
        lead_id:          { type: string, format: uuid, nullable: true }
        assigned_to:      { type: string, format: uuid }
        created_by:       { type: string, format: uuid, nullable: true }
        kind:             { $ref: '#/components/schemas/TaskKind' }
        title:            { type: string, minLength: 3, maxLength: 200 }
        description:      { type: string, maxLength: 2000, nullable: true }
        priority:         { $ref: '#/components/schemas/TaskPriority' }
        status:           { $ref: '#/components/schemas/TaskStatus' }
        in_queue:         { type: boolean }
        queue_position:   { type: integer, nullable: true, minimum: 1 }
        due_at:           { type: string, format: date-time, nullable: true }
        started_at:       { type: string, format: date-time, nullable: true }
        completed_at:     { type: string, format: date-time, nullable: true }
        completed_by:     { type: string, format: uuid, nullable: true }
        cancelled_at:     { type: string, format: date-time, nullable: true }
        cancelled_reason: { type: string, nullable: true }
        missed_reason:    { type: string, nullable: true }
        origin:           { $ref: '#/components/schemas/TaskOrigin' }
        context:          { type: object, additionalProperties: true, nullable: true }
        result_note:      { type: string, maxLength: 4000, nullable: true }
        created_at:       { type: string, format: date-time }
        updated_at:       { type: string, format: date-time }

    LeadSummary:
      type: object
      required: [id, name, pipe_id, stage_id, heat]
      properties:
        id:       { type: string, format: uuid }
        name:     { type: string }
        phone:    { type: string, nullable: true }
        pipe_id:  { type: string, format: uuid }
        stage_id: { type: string, format: uuid }
        heat:     { type: integer, minimum: 1, maximum: 5 }
        channel:  { type: string, enum: [whatsapp, messenger, instagram, sz_chat] }

    PipeStage:
      type: object
      required: [id, name, position, lead_count]
      properties:
        id:         { type: string, format: uuid }
        name:       { type: string }
        position:   { type: integer }
        lead_count: { type: integer }
        color_token: { type: string, example: stage-3 }

    TaskCockpitBundle:
      type: object
      required: [in_progress, queue, backlog, missed, counts]
      properties:
        in_progress:  { $ref: '#/components/schemas/Task', nullable: true }
        queue:        { type: array, items: { $ref: '#/components/schemas/Task' } }
        backlog:      { type: array, items: { $ref: '#/components/schemas/Task' } }
        missed:       { type: array, items: { $ref: '#/components/schemas/Task' } }
        active_lead:  { $ref: '#/components/schemas/LeadSummary', nullable: true }
        pipe_stages:  { type: array, items: { $ref: '#/components/schemas/PipeStage' } }
        counts:
          type: object
          required: [queue, backlog, missed, completed_today]
          properties:
            queue:            { type: integer }
            backlog:          { type: integer }
            missed:           { type: integer }
            completed_today:  { type: integer }

    UiPreferences:
      type: object
      required: [mode]
      properties:
        mode: { $ref: '#/components/schemas/UiMode' }

    ErrorResponse:
      type: object
      required: [code, message]
      properties:
        code:    { type: string }
        message: { type: string }
        details: { type: object, additionalProperties: true }

paths:
  /tasks/me/cockpit:
    get:
      summary: Bundle de hidratação do cockpit do usuário autenticado.
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema: { $ref: '#/components/schemas/TaskCockpitBundle' }

  /tasks:
    get:
      summary: Listar tasks (com filtros e paginação por cursor).
      parameters:
        - { in: query, name: status,      schema: { $ref: '#/components/schemas/TaskStatus' } }
        - { in: query, name: assigned_to, schema: { type: string, format: uuid } }
        - { in: query, name: lead_id,     schema: { type: string, format: uuid } }
        - { in: query, name: kind,        schema: { $ref: '#/components/schemas/TaskKind' } }
        - { in: query, name: in_queue,    schema: { type: boolean } }
        - { in: query, name: cursor,      schema: { type: string } }
        - { in: query, name: limit,       schema: { type: integer, minimum: 1, maximum: 100, default: 25 } }
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                required: [data]
                properties:
                  data:        { type: array, items: { $ref: '#/components/schemas/Task' } }
                  next_cursor: { type: string, nullable: true }

    post:
      summary: Criar task.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [assigned_to, kind, title]
              properties:
                lead_id:     { type: string, format: uuid, nullable: true }
                assigned_to: { type: string, format: uuid }
                kind:        { $ref: '#/components/schemas/TaskKind' }
                title:       { type: string, minLength: 3, maxLength: 200 }
                description: { type: string, maxLength: 2000, nullable: true }
                priority:    { $ref: '#/components/schemas/TaskPriority' }
                due_at:      { type: string, format: date-time, nullable: true }
                in_queue:    { type: boolean, default: false }
                context:     { type: object, additionalProperties: true, nullable: true }
      responses:
        '201': { description: Created, content: { application/json: { schema: { $ref: '#/components/schemas/Task' } } } }
        '422': { description: Validation error, content: { application/json: { schema: { $ref: '#/components/schemas/ErrorResponse' } } } }

  /tasks/{id}:
    parameters: [{ in: path, name: id, required: true, schema: { type: string, format: uuid } }]
    get:
      summary: Detalhe.
      responses:
        '200': { description: OK, content: { application/json: { schema: { $ref: '#/components/schemas/Task' } } } }
        '404': { description: Not found }
    patch:
      summary: Atualização parcial (title, description, priority, due_at, kind).
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                title:       { type: string }
                description: { type: string, nullable: true }
                priority:    { $ref: '#/components/schemas/TaskPriority' }
                due_at:      { type: string, format: date-time, nullable: true }
                kind:        { $ref: '#/components/schemas/TaskKind' }
      responses:
        '200': { description: OK, content: { application/json: { schema: { $ref: '#/components/schemas/Task' } } } }
        '409': { description: Invalid transition }

  /tasks/{id}/start:
    post:
      parameters: [{ in: path, name: id, required: true, schema: { type: string, format: uuid } }]
      responses: { '200': { description: OK, content: { application/json: { schema: { $ref: '#/components/schemas/Task' } } } } }

  /tasks/{id}/pause:
    post:
      parameters: [{ in: path, name: id, required: true, schema: { type: string, format: uuid } }]
      responses: { '200': { description: OK } }

  /tasks/{id}/complete:
    post:
      parameters: [{ in: path, name: id, required: true, schema: { type: string, format: uuid } }]
      requestBody:
        required: false
        content:
          application/json:
            schema:
              type: object
              properties:
                result_note: { type: string, maxLength: 4000 }
      responses: { '200': { description: OK } }

  /tasks/{id}/cancel:
    post:
      parameters: [{ in: path, name: id, required: true, schema: { type: string, format: uuid } }]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [reason]
              properties: { reason: { type: string, minLength: 3 } }
      responses:
        '200': { description: OK }
        '403': { description: Forbidden (need tasks.cancel) }

  /tasks/{id}/enqueue:
    post:
      parameters: [{ in: path, name: id, required: true, schema: { type: string, format: uuid } }]
      responses: { '200': { description: OK } }

  /tasks/{id}/dequeue:
    post:
      parameters: [{ in: path, name: id, required: true, schema: { type: string, format: uuid } }]
      responses: { '200': { description: OK } }

  /tasks/{id}/reopen:
    post:
      parameters: [{ in: path, name: id, required: true, schema: { type: string, format: uuid } }]
      responses: { '200': { description: OK } }

  /tasks/{id}/reassign:
    post:
      parameters: [{ in: path, name: id, required: true, schema: { type: string, format: uuid } }]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [assigned_to]
              properties: { assigned_to: { type: string, format: uuid } }
      responses:
        '200': { description: OK }
        '403': { description: Forbidden (need tasks.reassign) }

  /tasks/queue/reorder:
    post:
      summary: Reordenação atômica da fila de um assignee.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [assignee_id, ordered_ids]
              properties:
                assignee_id: { type: string, format: uuid }
                ordered_ids: { type: array, items: { type: string, format: uuid }, minItems: 1 }
      responses: { '200': { description: OK } }

  /tasks/bulk/assign:
    post:
      summary: Atribuição em lote (assíncrona, ADR-006).
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [tasks, assigned_to]
              properties:
                assigned_to: { type: string, format: uuid }
                tasks:
                  type: array
                  items:
                    type: object
                    required: [kind, title]
                    properties:
                      lead_id:  { type: string, format: uuid, nullable: true }
                      kind:     { $ref: '#/components/schemas/TaskKind' }
                      title:    { type: string }
                      priority: { $ref: '#/components/schemas/TaskPriority' }
                      due_at:   { type: string, format: date-time, nullable: true }
      responses:
        '202':
          description: Accepted
          content:
            application/json:
              schema:
                type: object
                required: [operation_id]
                properties: { operation_id: { type: string, format: uuid } }

  /me/preferences:
    patch:
      summary: Atualizar preferências de UI do usuário autenticado.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                ui_mode: { $ref: '#/components/schemas/UiMode' }
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                required: [ui_preferences]
                properties:
                  ui_preferences: { $ref: '#/components/schemas/UiPreferences' }
        '403':
          description: Forbidden (e.g., membro sem ui.view_manager_mode tentando setar 'manager')
```

`/auth/me` (não repetido aqui) é estendido para retornar:

```json
{
  "user": {
    "...": "...",
    "ui_preferences": { "mode": "salesperson" }
  }
}
```

---

## 3. Contrato WebSocket

Reuso do protocolo de [[ADR-002-realtime-websocket-go-hub]]. Novos `type`:

```json
{ "type": "task.created",         "tenant_id": "...", "entity_id": "task_...", "patch": { ... }, "version": 1, "occurred_at": "..." }
{ "type": "task.updated",         "tenant_id": "...", "entity_id": "task_...", "patch": { "priority": "urgent" }, "version": 2, "occurred_at": "..." }
{ "type": "task.started",         "tenant_id": "...", "entity_id": "task_...", "patch": { "status": "in_progress", "started_at": "...", "paused_task_id": "task_..." }, "version": 3 }
{ "type": "task.completed",       "tenant_id": "...", "entity_id": "task_...", "patch": { "status": "done", "completed_at": "...", "completed_by": "mbr_..." }, "version": 4 }
{ "type": "task.cancelled",       "tenant_id": "...", "entity_id": "task_...", "patch": { "status": "cancelled", "cancelled_reason": "..." }, "version": 5 }
{ "type": "task.missed",          "tenant_id": "...", "entity_id": "task_...", "patch": { "status": "missed" }, "version": 6 }
{ "type": "task.queue_reordered", "tenant_id": "...", "entity_id": "mbr_<assignee>", "patch": { "ordered_ids": [...] }, "version": 7 }
{ "type": "task.reassigned",      "tenant_id": "...", "entity_id": "task_...", "patch": { "assigned_to": "mbr_new", "from_assigned_to": "mbr_old" }, "version": 8 }
```

**Filtro no servidor** (aplicado antes do broadcast):

```go
// pseudocódigo
if evt.TenantID != client.TenantID { return }
if !client.IsAdmin && !eventConcernsClient(evt, client.MemberID) { return }
// eventConcernsClient: assignee atual OU anterior (para reassigned).
```

Membro nunca vê task de outro membro via WS. Admin recebe tudo da org.

---

## 4. Hidratação pós-login

```
POST /auth/login
  → 204 + Set-Cookie: __torque_session; __torque_refresh

Client:
  GET /auth/me
  → user.ui_preferences.mode = 'salesperson' | 'manager'

Client decide rota inicial:
  if role == 'master'              -> '/'          (Gerente)
  elif ui_mode == 'salesperson'    -> '/cockpit'   (Vendedor)
  else                             -> '/'          (Gerente)
```

Fallback localStorage (pré-backend): lê `torque.ui_mode` antes da primeira chamada `/auth/me`, usa como hint para render inicial. Uma vez que `/auth/me` retornar, o backend é a fonte da verdade.

---

## 5. RBAC — tabela de feature_permissions

Linhas a seed em `feature_permissions` (tabela global, [[Autenticacao e Autorizacao]] §RBAC):

```sql
INSERT INTO feature_permissions (feature_key, is_admin_only, default_value, description) VALUES
  ('tasks.view_own',              false, true,  'Ver próprias tasks.'),
  ('tasks.view_team',             true,  true,  'Ver tasks do time (admin).'),
  ('tasks.create',                false, true,  'Criar task para si.'),
  ('tasks.create_for_others',     true,  true,  'Criar task para outro membro (admin).'),
  ('tasks.update',                false, true,  'Editar task própria.'),
  ('tasks.complete',              false, true,  'Concluir task própria.'),
  ('tasks.cancel',                false, true,  'Cancelar task criada por si.'),
  ('tasks.reassign',              true,  true,  'Reatribuir task a outro membro (admin).'),
  ('tasks.bulk_assign',           true,  true,  'Atribuir tasks em lote (admin).'),
  ('tasks.reopen',                true,  true,  'Reabrir task missed/cancelled.'),
  ('ui.view_manager_mode',        false, false, 'Pode alternar para modo Gerente.'),
  ('ui.view_salesperson_mode',    false, true,  'Pode alternar para modo Vendedor.');
```

Admin recebe overrides automáticos via resolução da cascata em [[Autenticacao e Autorizacao]] §RBAC em 4 camadas.

---

## 6. Frontend — integração com Sistema Base

### 6.1 Arquivos novos / alterados

```
src/
├── providers/
│   └── UiModeProvider.tsx          (novo)
├── shell/
│   ├── TopBar.tsx                  (adicionar UiModeToggle)
│   └── UiModeToggle.tsx            (novo)
├── features/
│   └── cockpit/
│       ├── CockpitShell.tsx        (novo, layout próprio)
│       ├── CockpitHeader.tsx       (novo)
│       ├── QueuePanel.tsx          (novo — Tasks a fazer)
│       ├── ActiveTaskPanel.tsx     (novo — Task fazendo agora)
│       ├── PipeSnapshotPanel.tsx   (novo — Kanban snapshot)
│       ├── BacklogPanel.tsx        (novo — Tasks em aberto)
│       ├── clay/
│       │   ├── ClayPanel.tsx
│       │   ├── ClayButton.tsx
│       │   ├── ClayCard.tsx
│       │   └── ClayChip.tsx
│       └── __fixtures__/
│           └── cockpit-mock.ts     (bundle mock enquanto backend é 0%)
├── api/
│   └── tasks.ts                    (novo, transform snake↔camel)
├── hooks/
│   ├── useCockpit.ts               (novo)
│   ├── useTaskActions.ts           (novo)
│   └── useUiMode.ts                (novo)
├── contracts/
│   └── manual.ts                   (adicionar Task, TaskStatus, UiMode, TaskCockpitBundle)
├── routes.tsx                      (adicionar rota /cockpit com CockpitShell)
└── styles/
    └── globals.css                 (adicionar bloco clay tokens, escopo .cockpit-theme)
```

### 6.2 Rotas

```tsx
// routes.tsx (resumo)
[
  { path: '/login', element: <LoginPage /> },
  {
    element: <AuthRequired />,   // gate: /auth/me OK
    children: [
      {
        path: '/cockpit',
        element: <CockpitShell />,        // sem AppShell
        children: [
          { index: true, element: <CockpitView /> },
          { path: 'history', element: <TaskHistory /> },
        ],
      },
      {
        element: <ManagerModeGate><AppShell /></ManagerModeGate>,
        children: [
          { path: '/',         element: <Dashboard /> },
          { path: 'pipeline',  element: <PipelinePage /> },
          // ...demais rotas atuais
          { path: 'tasks',     element: <TasksAdminPage /> },  // admin only
        ],
      },
    ],
  },
  { path: '*', element: <NotFound /> },
]
```

`ManagerModeGate` redireciona para `/cockpit` quando membro sem `ui.view_manager_mode` tenta acessar `/` — preserva invariante de segurança (gate é apenas UX; servidor mantém enforce por RBAC).

### 6.3 Estado do UiMode

```ts
// UiModeProvider.tsx (resumo)
type UiMode = 'manager' | 'salesperson';

const STORAGE_KEY = 'torque.ui_mode';

function deriveInitial(user: Session | null): UiMode {
  // 1) backend: user.ui_preferences.mode
  if (user?.ui_preferences?.mode) return user.ui_preferences.mode;
  // 2) localStorage (pré-hidratação)
  const cached = localStorage.getItem(STORAGE_KEY) as UiMode | null;
  if (cached === 'manager' || cached === 'salesperson') return cached;
  // 3) fallback por role
  return user?.role === 'membro' ? 'salesperson' : 'manager';
}

// PATCH /me/preferences + optimistic update + invalida ['session']
// Se 403, reverte e toast.
```

### 6.4 Checklist de integração

- [ ] Adicionar UiModeToggle ao TopBar.
- [ ] Adicionar UiModeProvider ao árvore de providers (após AuthProvider).
- [ ] Criar rota `/cockpit` + `CockpitShell`.
- [ ] Tokens `--clay-*` em globals.css (dentro de `.cockpit-theme { ... }`).
- [ ] Primitivos clay.
- [ ] 4 painéis com fixture mock.
- [ ] Hook `useCockpit` com React Query consumindo fixture (placeholder para futuro `/tasks/me/cockpit`).
- [ ] Tipos `Task`, `TaskStatus`, `UiMode`, `TaskCockpitBundle` em `src/contracts/manual.ts`.
- [ ] `ManagerModeGate` redireciona corretamente.
- [ ] Atalhos de teclado.
- [ ] `prefers-reduced-motion` respeitado.
- [ ] Pós-login: se `ui_mode='salesperson'`, redireciona para `/cockpit`.
- [ ] i18n PT-BR em todas as strings novas.

---

## 7. Referências

- [[ADR-007-modo-vendedor-gerente]]
- [[F17 - Modo Vendedor (Task Cockpit)/Spec]]
- [[ADR-001-contratos-openapi-snake-camel]]
- [[ADR-002-realtime-websocket-go-hub]]
- [[ADR-003-auth-httponly-cookies-samesite]]
- [[ADR-004-paginacao-cursor-based]]
- [[ADR-006-jobs-assincronos-202-poll]]
- [[Multi-tenancy]]
- [[Autenticacao e Autorizacao]]
- [[Design System Base]]
