---
tags: [adr, architecture, ui, rbac, domain]
created: 2026-04-17
last_updated: 2026-04-17
status: accepted
id: ADR-007
supersedes: null
---

# ADR-007 — Modos de UI Vendedor e Gerente, Task como agregado unificado

## Contexto

O produto ganha dois modos de visualização selecionáveis pelo próprio usuário no canto superior direito do AppShell:

- **Gerente** — visão completa atual do Torque (sidebar, todas as features, dashboards, configuração).
- **Vendedor** — visão simplificada, focada em execução, layout de 4 painéis: fila priorizada à esquerda, task em foco no centro, kanban enxuto e backlog no lado direito. Design Claymorphism premium, escopo isolado.

A mudança não é cosmética. Ela (1) introduz roteamento paralelo com layout próprio, (2) redesenha o que hoje é `Follow-up` para suportar um ciclo de execução explícito ("em execução agora"), (3) adiciona preferência de UI persistida por usuário no `/auth/me`, e (4) afeta a regra cardinal do roadmap (o que é Sistema Base versus feature).

A decisão precisa travar, antes de qualquer código ou schema:

1. Se "Vendedor" é papel de RBAC novo ou modo de UI.
2. Se `Task` é nova entidade, rename de `Follow-up`, ou apenas uma view.
3. Se o modo é toggle de UI global ou rota separada.
4. Onde a preferência vive.
5. Como a segurança se mantém íntegra quando a superfície de UI muda.
6. Como o Claymorphism entra no design system sem contaminar o canvas atual.

## Decisão

### 1. Vendedor e Gerente são **modos de UI**, não papéis

A invariante de roles permanece inalterada: o sistema tem `admin`, `membro`, `master` e nada mais (conforme [[Papeis]] e [[Autenticacao e Autorizacao]]). "Vendedor" é vocabulário de produto, não coluna de banco, não enum de role.

O modo é uma **preferência pessoal do usuário**, ortogonal ao papel. Um `admin` pode entrar em modo Vendedor para validar o que o time vê. Um `membro` pode alternar para Gerente apenas se tiver a permissão `ui.view_manager_mode` (default `false` para `membro`, `true` para `admin`). `master` sempre permanece em Gerente — impersonação não muda isso.

### 2. `Task` substitui `Follow-up` — rename com consolidação

O domínio ganha uma única entidade de trabalho atribuível, chamada `Task`. Ela absorve e estende o que hoje está descrito em [[Follow-ups]] (que passa a ser `task.kind = 'followup'`).

**Motivos operacionais:**
- Backend é 0% hoje. Zero tabelas, zero migrations. O custo de rename é zero material.
- "Task" é o vocabulário natural do usuário final (aparece três vezes no print que originou esta decisão).
- Unificar evita duas máquinas de estado, duas UIs de atribuição, dois conjuntos de regras de permissão, dois caminhos WS.
- O modo Vendedor precisa de um estado `in_progress` explícito (task em execução ativa) que `Follow-up` não tem. Criar uma segunda entidade para isso duplicaria conceitos.

**Motivos de produto:**
- "Follow-up" descreve um subset de tasks (retomar contato). Existem outras: "ligar para qualificar", "enviar proposta", "confirmar presença", "responder objeção". Todas são tasks atribuíveis com prazo; `kind` as especializa.
- UI pode continuar falando "Follow-up" em contexto de lead se isso for pedagógico, mas o domínio é um só.

**Consequência:** o documento [[Follow-ups]] em `06 - Funcionalidades/Vendas/` torna-se subsidiária de `Tasks`, renomeada e reescrita quando o domínio for implementado.

### 3. Máquina de estados da Task

```
                  ┌───────────────────────┐
                  │        pending        │
                  │ (backlog ou em fila)  │
                  └────────┬──────────────┘
                           │
              in_queue=false │ in_queue=true
              (Em aberto)    │ (A fazer, com queue_position)
                             │
                             ▼
                  ┌───────────────────────┐
         ┌────────│     in_progress       │────────┐
         │        │ (unique por assignee) │        │
         │        └───────────┬───────────┘        │
         │                    │                    │
         │     concluir       │     pausar         │
         │                    │                    │
         ▼                    ▼                    ▼
       done                cancelled            pending
                                            (volta à fila)

cron de SLA: pending com due_at < now → missed
missed pode voltar a pending via reabrir.
done e cancelled são terminais.
```

**Invariantes:**
- No máximo **uma** task `in_progress` por `assigned_to` por vez. Constraint parcial no banco: `UNIQUE (assigned_to) WHERE status = 'in_progress'`.
- `pending + in_queue=true` exige `queue_position` não nulo e único por `assigned_to`.
- Iniciar task do backlog ("Em aberto") promove automaticamente para fim da fila e dispara transição para `in_progress`.
- Iniciar nova task enquanto já existe `in_progress` pausa a anterior (volta para `pending` + `in_queue=true` na primeira posição, não no fim).

### 4. Rota separada, layout próprio

O modo Vendedor é uma rota em paralelo, não um toggle global do AppShell:

- **`/` e descendentes** — modo Gerente, AppShell atual (Sidebar + TopBar + Outlet).
- **`/cockpit`** — modo Vendedor, layout próprio (`CockpitShell`), sem sidebar, header mínimo com o toggle e menu do usuário, área de conteúdo em grid de 4 painéis em claymorphism.

Path técnico `/cockpit` escolhido em vez de `/vendedor` para:
- Manter consistência com as rotas atuais (todas em inglês: `/pipeline`, `/inbox`, `/workflows`).
- Desassociar a URL do vocabulário de negócio ("Vendedor" é rótulo da UI, o endpoint é o cockpit de tasks).
- Preparar para expansão: podem existir outros cockpits focados (ex.: cockpit de prospecção) sem confusão semântica.

O toggle no canto superior direito é um segmented control com dois estados — **Gerente** e **Vendedor** — e navega entre `/` e `/cockpit` preservando contexto quando aplicável.

### 5. Preferência persistida no usuário

Campo novo no domínio de `users`:

```
users.ui_mode_preference  enum('manager', 'salesperson')  default conforme role
```

- `admin` default → `manager`
- `membro` default → `salesperson`
- `master` default → `manager` (fixo, não alterável pelo próprio)

Contrato:

- `GET /auth/me` passa a incluir `user.ui_preferences.mode`.
- `PATCH /me/preferences` body `{ ui_mode: 'manager' | 'salesperson' }` — 200 retorna preferência atualizada, dispara invalidação do cache de sessão do próprio usuário.
- Rate-limit leve (evita flood do toggle).

Antes do backend existir, fallback via `localStorage.torque.ui_mode`. Ao entrar o backend, o valor do servidor vence; localStorage é apenas hint de render antes do `/auth/me` completar.

Rota padrão pós-login: respeita `ui_mode`. Se `salesperson`, `navigate('/cockpit', { replace: true })` após `/auth/me`. Se `manager`, `navigate('/')`.

### 6. Segurança — modo é apresentação, nada mais

**Invariante explícita:** o modo de UI não concede nenhuma permissão. Toda chamada de API continua passando por:

```
AuthMiddleware -> TenantScopeMiddleware -> RBACMiddleware -> Handler -> Repository (filtro tenant)
```

- Um `membro` em modo Gerente não acessa nada que não acessaria em modo Vendedor — a UI só mostra mais superfícies, cada uma gate-ada pela mesma `feature_permission` ou `role`.
- Um `admin` em modo Vendedor continua sendo admin no servidor; o que muda é a superfície visível.
- `organization_id` continua extraído **exclusivamente do JWT** pelo servidor. `ui_mode` nunca é input para decisões de autorização.
- CSRF e cookies httpOnly permanecem idênticos (ADR-003 inalterado).

Consequência: nenhum endpoint novo "só para cockpit" é necessário. Os endpoints de `Task` são os mesmos para ambos os modos; a UI filtra e apresenta diferente.

### 7. Claymorphism — tokens isolados por escopo

Claymorphism só existe dentro de `/cockpit`. Entra no design system como **família de tokens opt-in** (`--clay-*`), ativada exclusivamente pelo layout do cockpit. O resto do app (modo Gerente) permanece com o design atual — chapado, editorial, hairline — sem contaminação.

Catálogo de tokens (declarado em `globals.css`, aplicado via classe raiz `.cockpit-theme` no `CockpitShell`):

```css
/* Claymorphism — escopo: /cockpit apenas */
--clay-surface:       222 16% 10%;   /* fundo de peça padrão */
--clay-surface-up:    222 15% 13%;   /* peça em foco/hover/ativa */
--clay-surface-down:  222 20%  5%;   /* slot vazio, depressão */
--clay-highlight:     220 12% 24%;   /* rim light interno */
--clay-shadow-hsl:      0  0%  0%;   /* oclusão/ambient */
--clay-accent-rim:     44 93% 54%;   /* rim gold (estado ativo) */

--clay-radius-sm:     14px;
--clay-radius:        20px;
--clay-radius-lg:     28px;

--clay-shadow-raised:
  inset 0 1.5px 0 0 hsl(var(--clay-highlight) / 0.55),
  inset 0 -1px  0 0 hsl(var(--clay-shadow-hsl) / 0.40),
  0 2px  1px 0 hsl(var(--clay-shadow-hsl) / 0.25),
  0 18px 32px -12px hsl(var(--clay-shadow-hsl) / 0.60),
  0 8px  16px -8px  hsl(var(--clay-shadow-hsl) / 0.40);

--clay-shadow-sunken:
  inset 0 2px 4px 0 hsl(var(--clay-shadow-hsl) / 0.65),
  inset 0 -1px 0 0 hsl(var(--clay-highlight) / 0.10);

--clay-shadow-floating:
  inset 0 2px 0 0 hsl(var(--clay-highlight) / 0.60),
  0 4px  2px 0 hsl(var(--clay-shadow-hsl) / 0.30),
  0 32px 48px -12px hsl(var(--clay-shadow-hsl) / 0.70),
  0 16px 24px -8px  hsl(var(--clay-shadow-hsl) / 0.50);

--clay-ring-focus:
  0 0 0 2px hsl(var(--accent) / 0.55),
  0 0 16px 0 hsl(var(--accent) / 0.35);

--clay-transition: 380ms cubic-bezier(0.22, 1, 0.36, 1);
```

Regras:
- Accent canônico preservado (`44 93% 54%`). Claymorphism não introduz nova cor.
- Tipografia idêntica ao resto do app (Fraunces display, Instrument Sans UI, JetBrains Mono para métricas).
- Motion mais suave (380ms em vez dos 150-200ms do AppShell) — reforça o toque tátil.
- Componentes claymorphism vivem em `torque-web/src/features/cockpit/clay/` — não são reaproveitados fora do escopo.

### 8. Roadmap — é Sistema Base, não feature F17

Esta mudança modifica o AppShell, introduz roteamento paralelo, preferência transversal do usuário e nova invariante semântica ("todo feature futuro precisa decidir sua aparição em cada modo"). Classificação correta: **extensão do Sistema Base**, não uma feature vertical.

O documento operacional da extensão vive em `05 - Sistema Base/UI Modes - Vendedor e Gerente.md` (a ser criado) e é referenciado pelo Checklist. Para manter rastreabilidade com o backlog existente, um apontador é deixado em `07 - Features/F17 - Modo Vendedor (Task Cockpit)/Spec.md` com aviso explícito no topo.

A regra cardinal "uma feature por vez" permanece intacta: esta mudança precede F01 porque expande a fundação. F01 (Funis Hub + Pipe WhatsApp), quando executada, já nasce sabendo em qual modo aparece.

### 9. WebSocket — reutiliza ADR-002

Sem novo hub. Sem novo protocolo. Novos `type` no protocolo JSON existente:

- `task.created` — nova task atribuída.
- `task.updated` — patch (status, priority, queue_position, due_at, etc.).
- `task.started` — transição para `in_progress`.
- `task.completed` — transição para `done`.
- `task.cancelled` — transição para `cancelled`.
- `task.queue_reordered` — fila do `assigned_to` foi reordenada em lote (bulk do admin).

Filtro no servidor: tenant match + `(assigned_to == user_id OR user.role == 'admin')`. Membro nunca recebe evento de task de outro membro; admin recebe todos da org.

## Alternativas consideradas

- **Toggle global de UI sem rota separada.** Mesma URL, AppShell muda conforme preferência. Descartado porque (a) compartilhar link entre usuários se torna ambíguo (cada um vê diferente), (b) o layout Vendedor é radicalmente distinto — DOM completamente diferente, não uma variante — e montar condicionalmente o AppShell cria um híbrido difícil de manter, (c) testes E2E ficam cheios de branches por modo.

- **Papel novo `vendedor` em RBAC.** Infla `role` para 4 valores, quebra a cascata atual, obriga refatorar `Papeis.md`, `feature_permissions`, hooks `useSession`. Descartado por inconsistência com invariante declarada em [[Autenticacao e Autorizacao]] e CLAUDE.md.

- **Task como entidade nova, Follow-up preservado.** Considerado porque preserva o doc atual de `Follow-ups`. Descartado: duplica conceitos idênticos (atribuição, prazo, estados, permissões), aumenta superfície de testes, gera confusão no produto ("por que é follow-up aqui e task ali?").

- **`/vendedor` como path.** Descartado em favor de `/cockpit`. URLs do app são em inglês; "vendedor" evoca papel; "cockpit" é técnico e genérico para suportar futuras visões focadas.

- **Preferência em `team_members` (per-org).** Descartado: se o usuário pertence a mais de uma org no futuro (multi-org), faz sentido que a preferência o acompanhe, não a membership. Fica em `users`.

## Consequências

**Positivas**

- Uma única entidade de trabalho (`Task`) simplifica produto, banco, testes e UI.
- Modo de UI como preferência pessoal respeita invariante de RBAC e evita inflação de roles.
- Rota separada torna o layout Vendedor auditável isoladamente (acessibilidade, performance, design).
- Claymorphism escopado por classe raiz evita regressão visual em outras áreas.
- Sistema Base absorve a mudança; nenhuma feature futura precisa retrofittar suporte a modos.

**Negativas**

- Rename de `Follow-up → Task` exige atualização de várias notas do vault (`06 - Funcionalidades/Vendas/Follow-ups.md`, regras de pipe, workflow actions) — custo pontual, mas mapeado.
- Claymorphism adiciona família de tokens nova; disciplina de uso é necessária (revisão em code review).
- Dois shells (AppShell + CockpitShell) aumentam superfície de testes de layout.
- Deep-links entre modos exigem decisão explícita: clicar em "ver lead completo" dentro do cockpit deve abrir em modo Gerente? Em modal do cockpit? Proposta: modal no cockpit para contexto rápido; navegação para `/` só via ação explícita. Documentado na spec.

## Impacto

### Documentação

- `06 - Funcionalidades/Vendas/Follow-ups.md` → renomear para `Tasks.md`, reescrever com `kind`, estados estendidos e relação com Cockpit. Tarefa do agente AI/Automation quando revisarem workflow actions.
- `03 - Modelo de Dominio/` → nova nota `Task.md` (substitui referências a Follow-up).
- `04 - Design/Design System Base.md` → adicionar seção "Claymorphism (escopo cockpit)".
- `02 - Arquitetura/Autenticacao e Autorizacao.md` → reforçar invariante "modo de UI não é autorização".
- `05 - Sistema Base/UI Modes - Vendedor e Gerente.md` (novo).
- `05 - Sistema Base/Checklist Sistema Base.md` → itens de cockpit.
- `07 - Features/00 - Mapa de Features.md` → nota sobre Sistema Base estendido.
- `09 - Backlog/Plano Mestre de Finalizacao do SaaS CRM.md` → Sprint S00 recebe bloco Cockpit; S01 (Go backend) ganha `tasks` no schema.

### Frontend

- `src/features/cockpit/` — novo diretório (shell, painéis, clay primitives, hooks).
- `src/shell/TopBar.tsx` — adicionar `UiModeToggle` à direita.
- `src/providers/UiModeProvider.tsx` — estado global, hidrata do `/auth/me`, persiste em `PATCH /me/preferences` + localStorage.
- `src/routes.tsx` — nova rota `/cockpit` com `CockpitShell` (não usa `AppShell`).
- `src/styles/globals.css` — família `--clay-*` declarada no `:root`, aplicada por classe `.cockpit-theme`.
- `src/contracts/manual.ts` — tipos `Task`, `TaskStatus`, `TaskKind`, `UiMode` até o OpenAPI existir.
- `src/api/tasks.ts` — cliente HTTP com transform snake/camel.
- `src/hooks/useTasks.ts`, `useActiveTask.ts`, `useTaskQueue.ts` — React Query hooks.

### Backend (futuro)

- Domínio `internal/domain/task/` — entidade, estados, invariantes.
- `internal/repository/task_repository.go` — CRUD com filtro tenant.
- `internal/handler/task_handler.go`, `me_handler.go` (preferências).
- `internal/ws/protocol.go` — novos `type`.
- OpenAPI spec em `api/openapi.yaml` → `tasks`, `/me/preferences`.
- Middleware `UniqueInProgressAssignee` no repository (constraint transacional).

### Banco

- Nova tabela `tasks` (substitui planejamento de `followups`).
- Migration `users` → coluna `ui_mode_preference`.
- Índices: `(organization_id, assigned_to, status)`, `(organization_id, lead_id)`, partial unique `(assigned_to) WHERE status='in_progress'`, `(organization_id, assigned_to, queue_position) WHERE status='pending' AND in_queue=true`.

## Invariantes recapituladas

1. Roles do sistema são `admin`, `membro`, `master`. Nada mais. Nunca.
2. `ui_mode` nunca é input de autorização. Servidor o ignora para decisões de acesso.
3. `organization_id` sempre do JWT. Cockpit não muda isso.
4. No máximo uma task `in_progress` por `assigned_to`. Enforced no banco.
5. Claymorphism vive apenas dentro de `.cockpit-theme`. Fora, o design canônico do Torque prevalece.
6. WS reutiliza hub por tenant; eventos de task filtrados por tenant + destinatário.

## Referências cruzadas

- [[Papeis]]
- [[Follow-ups]] (será convertida em `Tasks.md`)
- [[Multi-tenancy]]
- [[Autenticacao e Autorizacao]]
- [[Seguranca Web]]
- [[Design System Base]]
- [[Principios de Identidade Visual]]
- [[ADR-001-contratos-openapi-snake-camel]]
- [[ADR-002-realtime-websocket-go-hub]]
- [[ADR-003-auth-httponly-cookies-samesite]]
- [[Escopo do Sistema Base]]
- [[Checklist Sistema Base]]
- [[00 - Mapa de Features]]
- [[Plano Mestre de Finalizacao do SaaS CRM]]
