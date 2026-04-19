---
tags: [feature, sistema-base, cockpit, tasks, claymorphism]
created: 2026-04-17
last_updated: 2026-04-17
status: spec
feature_id: F17
classification: sistema-base-extension
decided_by: ADR-007
---

# F17 — Modo Vendedor (Task Cockpit)

> **Aviso de classificação.** Na avaliação do Architect ([[ADR-007-modo-vendedor-gerente]]), esta mudança é **extensão do Sistema Base**, não feature vertical no sentido F01-F16. O identificador F17 é preservado apenas como apontador de backlog — o documento operacional de implementação é [[UI Modes - Vendedor e Gerente]] em `05 - Sistema Base/`. O Mapa de Features referencia ambos.

---

## 1. Objetivo

Entregar ao vendedor (qualquer usuário em modo Vendedor) um cockpit de execução focado: sabe em cinco segundos o que fazer agora, executa sem friccão, vê o próximo da fila e o contexto de funil do lead relacionado. O gerente (qualquer usuário em modo Gerente) mantém a visão completa do Torque intacta. O alternador vive no canto superior direito do AppShell e persiste a preferência por usuário.

Critério de sucesso qualitativo: o vendedor abre o cockpit de segunda de manhã, vê sua fila priorizada, inicia a primeira task em um clique, conclui no máximo em dois cliques. Zero navegação até entregar valor.

## 2. Personas

### Primária — Vendedor (membro em modo `salesperson`)

- **Rotina**: inbox + execução de tasks. Não configura nada, não analisa nada fora do essencial.
- **Dor**: hoje teria que navegar entre Funis, Inbox, Follow-ups, Lead drawer para executar uma única ação. O cockpit consolida.
- **Expectativa**: visual tátil, confortável para ficar 8 horas. Atalhos de teclado. Realtime claro quando nova task entra.

### Secundária — Gerente (admin em modo `manager`)

- **Rotina**: dashboard, configuração, analytics, atribuição em massa de tasks ao time.
- **Necessidade relacionada ao cockpit**: precisa atribuir tasks com prioridade explícita, acompanhar fila de cada vendedor, reatribuir quando um vendedor sai.
- **Expectativa**: conseguir entrar em modo Vendedor para validar a experiência do time. UI do modo Gerente cobre atribuição em `/tasks` (admin view) e dentro do Lead drawer.

### Observador — Master

- Permanece em modo Gerente sempre. Operações de impersonação continuam via `OrgSwitcher`. O modo Vendedor é por-org, não cross-tenant.

## 3. Domínio — entidade Task

Decisão em [[ADR-007-modo-vendedor-gerente]]: `Task` substitui e unifica o que hoje é `Follow-up`.

### 3.1 Estrutura

```
Task {
  id                uuid
  organization_id   uuid                 NOT NULL
  lead_id           uuid                 NULL       (task pode não ter lead)
  assigned_to       uuid                 NOT NULL   (team_member.id)
  created_by        uuid                 NOT NULL   (team_member.id ou 'system')
  kind              enum                 NOT NULL   ('followup' | 'call' | 'qualification' |
                                                    'send_proposal' | 'confirm_meeting' |
                                                    'objection' | 'generic')
  title             text                 NOT NULL   (3-200 chars)
  description       text                 NULL       (até 2000 chars)
  priority          enum                 NOT NULL   ('low' | 'normal' | 'high' | 'urgent')
                                                    default 'normal'
  status            enum                 NOT NULL   ('pending' | 'in_progress' | 'done' |
                                                    'cancelled' | 'missed')
                                                    default 'pending'
  in_queue          bool                 NOT NULL   default false
  queue_position    int                  NULL       (only when in_queue=true)
  due_at            timestamptz          NULL
  started_at        timestamptz          NULL       (quando foi para in_progress)
  completed_at      timestamptz          NULL
  completed_by      uuid                 NULL
  cancelled_at      timestamptz          NULL
  cancelled_reason  text                 NULL
  missed_reason     text                 NULL
  origin            enum                 NOT NULL   ('manual' | 'workflow' | 'agent' |
                                                    'rule' | 'system')
  context           jsonb                NULL       (workflow_execution_id, etc.)
  result_note       text                 NULL       (preenchido ao concluir)
  created_at        timestamptz          NOT NULL
  updated_at        timestamptz          NOT NULL
}
```

### 3.2 Invariantes

1. `organization_id` sempre do JWT do servidor. Nunca do cliente.
2. `assigned_to` pertence à mesma org (`team_members.organization_id = tasks.organization_id`).
3. Se `lead_id != NULL`, o lead pertence à mesma org.
4. **Constraint parcial UNIQUE**: no máximo uma task `in_progress` por `(organization_id, assigned_to)`.
5. `pending + in_queue=true` ⇒ `queue_position IS NOT NULL`; único por `(organization_id, assigned_to)`.
6. `in_progress` ⇒ `started_at IS NOT NULL`.
7. `done` ⇒ `completed_at IS NOT NULL AND completed_by IS NOT NULL`.
8. Transições válidas: tabela abaixo. Qualquer outra é 409.

### 3.3 Máquina de estados

| De | Para | Gatilho | Regras |
|---|---|---|---|
| `pending (in_queue=false)` | `pending (in_queue=true)` | `enqueue` (vendedor ou admin) | Adiciona ao fim da fila, `queue_position = max+1` |
| `pending (in_queue=true)` | `pending (in_queue=false)` | `dequeue` | Remove `queue_position`, compacta fila |
| `pending` | `in_progress` | `start` | Se outra `in_progress` existir para o mesmo assignee, pausa (volta a `pending` in_queue=true, `queue_position=1`). `started_at=now` |
| `in_progress` | `pending (in_queue=true)` | `pause` | `queue_position=1`, `started_at` preservado (metadado de início tentado) |
| `in_progress` | `done` | `complete` | Exige `result_note` opcional. `completed_at=now`, `completed_by=user` |
| `pending`, `in_progress` | `cancelled` | `cancel` | Apenas admin ou `created_by`. `cancelled_reason` obrigatório |
| `pending` (due_at passou) | `missed` | cron SLA | Automático |
| `missed` | `pending` | `reopen` | Admin ou assignee. Novo `due_at` opcional |
| `done`, `cancelled` | — | — | Estados terminais |

### 3.4 Buckets da UI do Vendedor

A UI expõe três buckets derivados do estado:

- **Tasks a fazer** (painel esquerdo): `status='pending' AND in_queue=true AND assigned_to=me`, ordenado por `queue_position ASC`. É a fila priorizada.
- **Task fazendo agora** (painel centro): `status='in_progress' AND assigned_to=me`. Zero ou uma.
- **Tasks em aberto** (painel direita-inferior): `status='pending' AND in_queue=false AND assigned_to=me`, ordenado por `priority DESC, due_at ASC NULLS LAST`.

Missed: aparece no topo de "Em aberto" com badge vermelho; clique ou ação rápida joga para fila. Done/cancelled: não aparecem no cockpit; visíveis em `/cockpit/history` (rota secundária).

### 3.5 Kind e UI

`kind` especializa o ícone, o título sugerido e o verbo de ação na UI, mas não muda a máquina de estados. Lista inicial:

| kind | Ícone | Ação primária | Sugestão de título |
|---|---|---|---|
| `followup` | MessageCircle | Abrir conversa | "Retomar contato com {lead}" |
| `call` | Phone | Iniciar ligação | "Ligar para {lead}" |
| `qualification` | Compass | Qualificar | "Qualificar {lead}" |
| `send_proposal` | FileText | Gerar proposta | "Enviar proposta para {lead}" |
| `confirm_meeting` | CalendarCheck | Confirmar | "Confirmar presença de {lead}" |
| `objection` | ShieldAlert | Tratar objeção | "Responder objeção de {lead}" |
| `generic` | CheckSquare | Abrir | título livre |

## 4. Contratos de API (snake_case)

Convenções em [[ADR-001-contratos-openapi-snake-camel]]: wire snake_case, frontend camelCase. Cursor pagination ([[ADR-004-paginacao-cursor-based]]). Jobs assíncronos no padrão 202 ([[ADR-006-jobs-assincronos-202-poll]]) quando aplicável.

### 4.1 Leitura

```
GET /tasks/me/cockpit
  → 200 TaskCockpitBundle

TaskCockpitBundle {
  in_progress:  Task | null
  queue:        Task[]        (pending in_queue=true, ordenado)
  backlog:      Task[]        (pending in_queue=false)
  missed:       Task[]        (status=missed)
  active_lead:  LeadSummary | null  (lead da in_progress, inclui pipe + stage atual)
  pipe_stages:  PipeStage[]   (stages do pipe do active_lead, para o painel direita-superior)
  counts: { queue: int, backlog: int, missed: int, completed_today: int }
}
```

Uma única chamada hidrata o cockpit inteiro (evita waterfall, mesmo princípio do `/auth/me`).

```
GET /tasks?status=&assigned_to=&lead_id=&cursor=&limit=
  → 200 { data: Task[], next_cursor: string | null }

GET /tasks/:id → 200 Task
```

### 4.2 Mutações

```
POST /tasks
  body: { lead_id?, assigned_to, kind, title, description?, priority?, due_at?, in_queue?, context? }
  → 201 Task | 422 validation | 429 rate

PATCH /tasks/:id
  body: partial Task (exceto id, organization_id, created_by, created_at)
  → 200 Task | 409 invalid_transition | 404

POST /tasks/:id/start      → 200 Task (pausa outra in_progress do assignee, se houver)
POST /tasks/:id/pause      → 200 Task
POST /tasks/:id/complete
  body: { result_note? }    → 200 Task
POST /tasks/:id/cancel
  body: { reason }          → 200 Task   (admin ou created_by)
POST /tasks/:id/enqueue    → 200 Task
POST /tasks/:id/dequeue    → 200 Task
POST /tasks/:id/reopen     → 200 Task   (de missed|cancelled → pending)
POST /tasks/:id/reassign
  body: { assigned_to }     → 200 Task   (admin only)

POST /tasks/queue/reorder
  body: { assignee_id, ordered_ids: [uuid] }
  → 200 { queue: Task[] }   (atômico; admin ou próprio assignee)

POST /tasks/bulk/assign
  body: { tasks: [{ lead_id, kind, title, due_at, priority }], assigned_to }
  → 202 { operation_id }    (poll em GET /operations/:id; admin only)
```

Rotas marcadas como "admin only" no servidor são via `feature_permissions` e check de role; frontend esconde CTAs mas servidor é a fonte da verdade.

Mutations sensíveis (`cancel`, `reassign`, `bulk/assign`, `queue/reorder`) exigem `X-CSRF-Token` conforme [[ADR-003-auth-httponly-cookies-samesite]].

### 4.3 Preferência de UI

```
GET /auth/me
  → resposta estendida: user.ui_preferences.mode: 'manager' | 'salesperson'

PATCH /me/preferences
  body: { ui_mode: 'manager' | 'salesperson' }
  → 200 { ui_preferences: { mode } }

  - Rate limit: 10 req/min por usuário
  - Membro sem feature_permission ui.view_manager_mode que tentar setar 'manager' → 403
```

## 5. WebSocket (reuso do hub ADR-002)

Novos `type` no protocolo existente:

| type | Quem recebe | Payload |
|---|---|---|
| `task.created` | tenant + (assigned_to OR admin) | `{ task }` |
| `task.updated` | tenant + (assigned_to OR admin) | `{ task_id, patch, version }` |
| `task.started` | tenant + (assigned_to OR admin) | `{ task_id, started_at, paused_task_id? }` |
| `task.completed` | tenant + (assigned_to OR admin) | `{ task_id, completed_at, completed_by }` |
| `task.cancelled` | tenant + (assigned_to OR admin) | `{ task_id, cancelled_reason }` |
| `task.missed` | tenant + (assigned_to OR admin) | `{ task_id }` |
| `task.queue_reordered` | assignee + admin | `{ assignee_id, ordered_ids }` |
| `task.reassigned` | assignee antigo, novo, admin | `{ task_id, from_assigned_to, to_assigned_to }` |

Frontend aplica patches sobre cache React Query do bundle `/tasks/me/cockpit` e recalcula buckets.

## 6. Layout — 4 painéis (descrição detalhada)

### 6.1 Grid

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ CockpitHeader (64px) — logo + UiModeToggle + atalho busca + avatar        │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌────────────────┐  ┌────────────────────────────┐  ┌──────────────────┐   │
│  │                │  │                            │  │ Pipe snapshot     │   │
│  │  Tasks a fazer │  │   Task fazendo agora       │  │  (stages)         │   │
│  │  (fila)        │  │   (foco zen, claymorphism) │  │                  │   │
│  │                │  │                            │  │  [stage 1]        │   │
│  │                │  │                            │  │  [stage 2] ●      │   │
│  │                │  │                            │  │  [stage 3]        │   │
│  │                │  │                            │  │  [stage 4]        │   │
│  │                │  │                            │  │                  │   │
│  │                │  │                            │  ├──────────────────┤   │
│  │                │  │                            │  │                  │   │
│  │                │  │                            │  │  Tasks em aberto │   │
│  │                │  │                            │  │  (backlog)       │   │
│  │                │  │                            │  │                  │   │
│  └────────────────┘  └────────────────────────────┘  └──────────────────┘   │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

### 6.2 Proporções e responsividade

- Desktop ≥1280px: grid 320px | 1fr | 360px. Gaps 16px. Padding do canvas 24px.
- Tablet 1024-1279px: 280px | 1fr | 320px.
- ≤1023px: stack vertical — centro primeiro, fila acima, backlog/kanban abaixo em tabs.
- Altura: `100vh - header`. Painéis internos têm scroll próprio; canvas nunca scrolla.

### 6.3 Painel — Tasks a fazer (fila)

- Header: "Tasks a fazer · N", ação inline "Reordenar" (toggle para drag).
- Lista vertical de cards claymorphism pequenos. Ordem = `queue_position`.
- Cada card: ícone do `kind`, título (2 linhas máx), lead (se houver), badge de `priority` (accent se urgent/high), countdown de `due_at`.
- Interações:
  - **Clique**: abre drawer com detalhe da task (sem sair do cockpit).
  - **Duplo clique / enter**: inicia task (vai para in_progress, pausa atual se houver).
  - **Drag**: reordena fila (persiste via `POST /tasks/queue/reorder`; otimista no frontend).
  - **Menu contextual**: tirar da fila, cancelar, alterar prioridade, reatribuir (se admin).
- Vazio: ilustração mínima + CTA "Pegar da lista em aberto" (move do bucket direita-inferior).

### 6.4 Painel — Task fazendo agora (foco)

Desenho central, peça claymorphism maior e ligeiramente "levantada" (shadow-floating), com accent rim sutil.

- Header interno: `kind` em tag pequena + chronômetro desde `started_at`.
- Conteúdo:
  - Título da task (Fraunces display, 24-28px).
  - Descrição (Instrument Sans, 15px).
  - Chips: lead (clicável, abre drawer do lead), stage atual do pipe, due_at (countdown).
  - Contexto embutido: 3 últimas mensagens com o lead (se `lead_id`), botão "Abrir conversa".
- Área de ação (fim do painel, sempre visível):
  - Campo livre "Resultado" (textarea, opcional).
  - CTA primária "Concluir" (gold accent + glow).
  - CTAs secundárias: "Pausar", "Abrir lead completo", "Menu".
- Vazio (sem task in_progress): card com CTA "Iniciar próxima" (clique → `start` na primeira da fila) e contagem de fila.

### 6.5 Painel — Kanban snapshot (direita-superior)

Visualização ultra-compacta das stages do pipe do lead da task ativa:

- Header: nome do pipe (ex: "Pipe WhatsApp"), link discreto "Ver pipe completo" (abre em nova aba ou modal do cockpit).
- Lista vertical de stages. Cada stage: nome + contagem de leads do stage + indicador (●) na stage do lead atual.
- Arrastar o lead entre stages a partir daqui: **não**. Leitura apenas neste primeiro release. Mudança de stage via drawer do lead.
- Se não há task in_progress: painel mostra o pipe padrão do vendedor (definido em user_preferences, fallback: Pipe WhatsApp se existir) em estado muted.

### 6.6 Painel — Tasks em aberto (backlog, direita-inferior)

- Header: "Em aberto · N", filtro rápido (Todas | Hoje | Atrasadas | Alta prioridade).
- Lista densa de cards pequenos. Ordenação default: `priority DESC, due_at ASC`.
- Cada card: ícone, título curto, priority dot, due_at relativo.
- Interações:
  - **Clique**: drawer de detalhe.
  - **Arrastar para painel esquerdo**: enqueue (vai para fila, `in_queue=true`).
  - **Shift+clique / atalho `E`**: enqueue rápido.
- Vazio: mensagem sucinta, sem emoji.

### 6.7 Header do cockpit

Altura 64px. Layout:

```
[Logo Torque · 28px]          [Busca global (⌘K) · 320px]       [UiModeToggle]  [Bell]  [Avatar]
```

- **UiModeToggle**: segmented control com dois estados "Vendedor" / "Gerente". Pill claymorphism. Clique alterna e navega. Disabled com tooltip para `membro` sem `ui.view_manager_mode`.
- **Busca**: mesma CommandPalette do AppShell, com escopo filtrado para "Tasks, Leads, Conversas" quando aberta pelo cockpit.
- **Bell**: mesmo do AppShell.

### 6.8 Atalhos de teclado

| Atalho | Ação |
|---|---|
| `J` / `K` | Navegar na fila (esquerda) |
| `Enter` | Iniciar task selecionada |
| `E` | Enqueue (backlog → fila) |
| `C` | Concluir task ativa |
| `P` | Pausar task ativa |
| `⌘K` | Busca global |
| `G M` | Ir para modo Gerente |
| `G V` | Ir para modo Vendedor |
| `?` | Overlay de atalhos |

## 7. Fluxo de atribuição (gerente → vendedor)

### 7.1 UI do admin (modo Gerente)

Nova rota `/tasks` (admin only) com três abas:

- **Por vendedor**: lista de membros ativos, cada um com contador de fila + backlog + concluídas hoje. Clique expande a fila com drag-drop para reordenar.
- **Não atribuídas**: tasks criadas por workflow/regra de pipe sem assignee. Admin atribui em bulk.
- **Auditoria**: tasks missed/cancelled do time na última semana.

Atribuir individualmente: do Lead drawer → aba "Tasks" → botão "+ Nova task" → form (título, kind, priority, due_at, assigned_to, in_queue).

Atribuir em lote: tela "Não atribuídas" → selecionar → "Atribuir a X" → aparece 202 + toast "processando" → WS notifica conclusão.

### 7.2 Ordem e prioridade

- `priority` é sugestão visual (cor, urgência).
- `queue_position` é ordem real de execução. Admin pode empurrar uma task para o topo da fila do vendedor via drag ou ação "Subir para o topo".
- Vendedor pode reordenar a própria fila. Admin pode bloquear (feature_permission futura `tasks.pin_by_admin` — fora deste release).

## 8. RBAC

Novas `feature_keys`:

| Key | Default (membro) | Default (admin) | Master | Observação |
|---|---|---|---|---|
| `tasks.view_own` | true | true | true | vê próprias |
| `tasks.view_team` | false | true | true | admin vê todos |
| `tasks.create` | true | true | true | criar para si |
| `tasks.create_for_others` | false | true | true | admin atribui |
| `tasks.update` | true (own) | true (all) | true | patch |
| `tasks.complete` | true (own) | true (all) | true | |
| `tasks.cancel` | true (own, criadas por si) | true (all) | true | |
| `tasks.reassign` | false | true | true | |
| `tasks.bulk_assign` | false | true | true | |
| `tasks.reopen` | false | true | true | |
| `ui.view_manager_mode` | false | true | true | gate do toggle |
| `ui.view_salesperson_mode` | true | true | true | sempre |

Resolução server-side por [[Autenticacao e Autorizacao]] (4 camadas). Frontend esconde CTAs via `useCanPerformAction`, servidor enforce com 403.

## 9. Design — Claymorphism (escopo cockpit)

Tokens em [[ADR-007-modo-vendedor-gerente]] §7. Aplicação via classe raiz `.cockpit-theme` no `CockpitShell`.

### 9.1 Primitivos claymorphism

- **ClayPanel**: peça base, `shadow-raised`, radius `lg`, padding variável. Variantes: `surface`, `raised` (foco), `sunken` (slot vazio).
- **ClayButton**: mesma mecânica, CTA primária com accent gold + `clay-accent-glow`. Estados: idle, hover (lift 1px), pressed (sunken), disabled.
- **ClayCard**: variante menor de panel para itens de lista. Reage a hover com lift + rim highlight sutil.
- **ClayInput**: sunken por default, focus → raised + ring accent.
- **ClayChip**: radius full, padding reduzido, sombra leve.

### 9.2 Motion

- Transições 380ms `cubic-bezier(0.22, 1, 0.36, 1)` (easeOutQuart).
- Hover lift: translateY(-1px) + shadow-floating. Pressed: translateY(0.5px) + shadow-sunken.
- Mudança de bucket (enqueue/dequeue): FLIP animation nos cards (300ms, preserve layout).
- Iniciar task: card voa do painel esquerdo para o centro (400ms, spring ease). Acessibilidade: respeita `prefers-reduced-motion`.

### 9.3 Tipografia

Sem alteração vs resto do app:
- Display (título da task ativa, Fraunces 24-28px).
- UI (Instrument Sans 13-15px).
- Métricas (JetBrains Mono, tabular-nums, para chronômetro e contadores).

### 9.4 Dark-first, light-second

Claymorphism é nativo de dark. Em modo light (se ativado), tokens `--clay-*` trocam para variante clara: `clay-surface 220 20% 92%`, `clay-highlight 0 0% 100%`, shadows mais claras. Peso dark permanece canônico; light é subordinado.

### 9.5 Regras de reprovação ([[Criterios de Reprovacao]])

- Shadow sem rim highlight interno → reprovado (deixa de ser clay).
- Radius < 14px → reprovado (deixa de ser tátil, vira card plano).
- Accent gold fora da CTA primária e estado ativo → reprovado (accent restraint).
- Claymorphism aplicado em qualquer área fora de `.cockpit-theme` → reprovado.

## 10. Critérios de aceite

### 10.1 Funcionais

- [ ] Toggle no header navega entre `/` e `/cockpit`, respeitando permissão.
- [ ] Ao logar, usuário com `ui_mode=salesperson` é redirecionado para `/cockpit`.
- [ ] Preferência persiste entre sessões (backend + localStorage fallback).
- [ ] `GET /tasks/me/cockpit` hidrata os 4 painéis em uma única chamada.
- [ ] Iniciar task pausa a atual automaticamente.
- [ ] Constraint UNIQUE de `in_progress` impede corrida (teste concorrente).
- [ ] Reordenar fila persiste e propaga via WS para outras sessões do mesmo usuário.
- [ ] Admin atribui task individual (Lead drawer) e em bulk (/tasks) com sucesso.
- [ ] Task criada por outro usuário/workflow aparece no cockpit do assignee em tempo real.
- [ ] Atalhos de teclado funcionam em todos os painéis.
- [ ] Membro sem `ui.view_manager_mode` não consegue navegar para `/` (redirect).

### 10.2 Segurança

- [ ] `organization_id` sempre do JWT; teste de tenant isolation passa.
- [ ] Chamada direta a `PATCH /tasks/:id` em task de outra org → 404.
- [ ] Chamada direta a `POST /tasks/:id/reassign` por membro → 403.
- [ ] CSRF obrigatório em cancel, reassign, bulk, reorder.
- [ ] Modo de UI não concede permissão alguma no servidor (teste: membro em modo Gerente simulado frontend não consegue `bulk_assign`).

### 10.3 Design

- [ ] Claymorphism passa em [[Criterios de Reprovacao]] com aprovação do Designer.
- [ ] Nenhum token `--clay-*` referenciado fora de `.cockpit-theme` (ESLint rule ou grep em review).
- [ ] Accent gold `44 93% 54%` em todos os usos.
- [ ] `prefers-reduced-motion` respeitado.
- [ ] Contraste WCAG AA em todos os textos (ink vs clay-surface). Título e body AAA.

### 10.4 Qualidade

- [ ] Lighthouse ≥ 95 em performance no cockpit.
- [ ] Sem layout shift ao hidratar do `/tasks/me/cockpit`.
- [ ] Testes de componente para cada clay primitivo.
- [ ] Testes de integração do fluxo fila→in_progress→done.
- [ ] Teste de concorrência: dois clients tentam start simultaneamente → apenas um vence, outro recebe 409.

## 11. Riscos e edge cases

| Risco | Mitigação |
|---|---|
| Vendedor abre duas abas e inicia tasks diferentes | Constraint UNIQUE no banco + WS propaga pause → UI se auto-sincroniza |
| Fila reordenada simultaneamente por admin e vendedor | Último write vence + evento `queue_reordered` notifica; UI mostra toast "fila atualizada pelo gerente" |
| Task atribuída a vendedor desligado | Admin reatribui via /tasks (aba auditoria tem alerta) |
| Lead deletado (soft) com task pending | Task mantém referência, UI exibe lead com tag "arquivado"; ao concluir, pede confirmação |
| Muitas tasks missed geram ruído | Backlog filtro default esconde missed com >7 dias; admin vê tudo |
| Claymorphism impacta performance em máquinas fracas | Shadows complexas atrás de `@media (prefers-reduced-transparency)` fallback com shadow-elev-2 |
| Preferência vence entre device → user troca aparelho | Servidor é fonte da verdade; localStorage é apenas hint de render inicial |
| Deep link para `/cockpit` com usuário sem permissão | Middleware de rota: sem auth → `/login`; sem `salesperson` nem `manager` viável → `/403` |

## 12. Dependências

- Sistema Base frontend: `AuthProvider`, `useSession`, `UiModeProvider` (novo), `fetch.ts`.
- Backend (futuro): endpoints §4, tabela `tasks`, migration `users.ui_mode_preference`.
- ADRs: 001, 002, 003, 004, 006, **007**.
- Docs: [[Tasks]] (a ser criado a partir do rename de [[Follow-ups]]), [[UI Modes - Vendedor e Gerente]] em `05 - Sistema Base/`.

## 13. Fora de escopo (deste release)

- Admin "pin" de task na fila do vendedor (evita reordenar).
- Métricas de produtividade no cockpit (taxa de conclusão, tempo médio).
- Notificações push desktop para novas tasks.
- Integração com Google Calendar (task com `due_at` gerar evento).
- Kanban drag-and-drop direto no painel snapshot.
- Temas além de dark/light (ex: high-contrast premium).

Esses itens entram em iterações futuras mediante validação de uso real.

## 14. Ordem de execução sugerida

1. DBA: schema `tasks` + migration `users.ui_mode_preference` (spec) — Sprint S01 (Go backend skeleton).
2. Backend: endpoints §4 atrás de flag até integrados pelo frontend — Sprint S02.
3. Frontend (agora, mesmo sem backend): `UiModeProvider`, toggle no TopBar, rota `/cockpit`, `CockpitShell`, 4 painéis com mock fixture, tokens clay, primitivos clay. Fallback para localStorage até `/auth/me` retornar `ui_mode`.
4. Designer: auditoria do claymorphism.
5. QA: plano de testes §10.
6. Documentação: atualizar [[Follow-ups]] → [[Tasks]], [[Design System Base]] com seção claymorphism, [[Checklist Sistema Base]], [[Plano Mestre de Finalizacao do SaaS CRM]], [[00 - Mapa de Features]].

## 15. Referências

- [[ADR-007-modo-vendedor-gerente]] (fonte desta spec)
- [[ADR-001-contratos-openapi-snake-camel]]
- [[ADR-002-realtime-websocket-go-hub]]
- [[ADR-003-auth-httponly-cookies-samesite]]
- [[Papeis]]
- [[Multi-tenancy]]
- [[Autenticacao e Autorizacao]]
- [[Design System Base]]
- [[Principios de Identidade Visual]]
- [[Criterios de Reprovacao]]
- [[Follow-ups]] (a converter em [[Tasks]])
- [[Escopo do Sistema Base]]
- [[Plano Mestre de Finalizacao do SaaS CRM]]
