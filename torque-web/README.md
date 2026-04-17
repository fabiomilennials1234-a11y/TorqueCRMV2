# Torque — Web (Sprint 1)

Cliente do Torque CRM. Skeleton de frontend **sem backend**, sem API, sem integrações. Só as telas-âncora, o design system, o shell e uma paleta de dados estáticos (`src/lib/seed.ts`).

O próximo sprint substituirá o seed por hooks TanStack Query contra a API Go.

## Stack

- Vite 6 + React 18 + TypeScript strict
- Tailwind 3.4 + Radix primitives + Motion + cmdk
- React Router v6
- Fontes: **Fraunces** (display), **Instrument Sans** (UI), **JetBrains Mono** (dados) — Google Fonts CDN
- Dark-first, sem light mode no Sprint 1

## Direção estética

Minimalismo editorial-industrial. Dark cinematográfico com ouro quente `#F5C518` (não amarelo, não amber). Tipografia serif variável + sans humanista + mono numérico — rejeita Inter/Geist/Space Grotesk. Command palette (⌘K) como metáfora primária de navegação. Hairlines por toda parte, sombras comedidas, grain sutil onde cabe.

## Rodar localmente

```bash
cd torque-web
npm install
npm run dev
```

Abre em `http://localhost:5173`. Login → `/` entra direto na dashboard (não há auth real ainda).

### Scripts

- `npm run dev` — dev server com HMR
- `npm run build` — build produção (tsc + vite)
- `npm run preview` — preview do build
- `npm run typecheck` — somente TS
- `npm run lint` — eslint

## Estrutura

```
torque-web/
├── index.html
├── package.json · tsconfig.json · vite.config.ts · tailwind.config.ts
├── public/
└── src/
    ├── main.tsx · App.tsx · routes.tsx
    ├── styles/
    │   └── globals.css              ← tokens (HSL) + tipografia + utilidades
    ├── lib/
    │   ├── utils.ts                 ← cn(), formatCurrency, formatRelative
    │   └── seed.ts                  ← leads, conversas, campanhas, agentes, atividade
    ├── ui/                          ← primitivos (custom, não shadcn default)
    │   ├── button · input · badge · avatar · card · separator · skeleton
    │   ├── dropdown · tooltip · sheet · tabs
    │   ├── pill · kbd · page-header · empty-state
    │   └── score-meter · spark      ← componentes-assinatura
    ├── shell/
    │   ├── AppShell.tsx             ← layout root
    │   ├── Sidebar.tsx · TopBar.tsx
    │   ├── TorqueMark.tsx           ← brand glyph
    │   ├── OrgSwitcher.tsx
    │   └── CommandPalette.tsx       ← ⌘K
    └── features/
        ├── auth/LoginPage.tsx
        ├── dashboard/DashboardPage.tsx      ← KPIs + timeline ao vivo
        ├── pipeline/
        │   ├── KanbanPage.tsx · KanbanColumn · LeadCard
        │   └── LeadDrawer.tsx                ← sheet com 4 abas
        ├── inbox/InboxPage.tsx              ← 3 colunas · compositor IA
        ├── workflows/WorkflowBuilderPage    ← canvas + paleta + inspector
        ├── campaigns/CampaignsPage.tsx
        ├── copilot/AgentsPage.tsx            ← roster + playground
        ├── analytics/AnalyticsPage.tsx
        └── settings/SettingsPage.tsx
```

## Telas entregues

| Rota | Propósito |
|---|---|
| `/login` | Login editorial com citação + medidor de status |
| `/` | Dashboard — KPIs com sparkline, pipeline snapshot, atividade ao vivo |
| `/pipeline` | Kanban WhatsApp · drawer do lead com 4 abas |
| `/inbox` | Chat multi-canal · lista + thread + painel de contexto |
| `/workflows` | Workflow builder · canvas, paleta, inspector |
| `/campaigns` | Cadências outbound · progresso por campanha |
| `/copilot` | Agentes IA · roster + playground |
| `/analytics` | Funil, ranking do time, atribuição UTM |
| `/settings` | Organização, time, integrações |

## Decisões de design (log curto)

- **Tokens HSL em CSS vars** → Tailwind usa `<alpha-value>` nativamente; light mode futuro é trocar um `:root`.
- **Fonts editorial trio**: Fraunces (display, `opsz` variable) + Instrument Sans (body) + JetBrains Mono (dados). Rejeitamos Inter/Geist/Space Grotesk por serem o "default AI".
- **Sem shadcn default**: primitivos escritos à mão sobre Radix, com nossa linguagem (hairline shadows, size scale sm/md/lg).
- **Focus ring** em 2 camadas (bg-offset + accent) em vez do típico outline azul.
- **Command Palette** é cidadão de primeira — atalho ⌘K global, recente/ações/navegação juntos, animação scale-in.
- **Score Meter** como componente-assinatura: gauge 270°, cor muda por banda (dim/warn/accent/success).
- **Hairlines** (`shadow-hairline`) são usados em vez de border sempre que possível — compõem melhor com dark backgrounds.

## Limitações do Sprint 1

- Dados 100% estáticos em `seed.ts`. Nenhuma chamada de rede.
- Login é visual apenas — submit redireciona para `/`.
- Drag & drop do kanban ainda não implementado (visual-only).
- WebSocket/realtime não plumbed.
- Sem dark/light toggle — dark-first é default e única.
- Acessibilidade WCAG AA foi considerada mas não auditada formalmente.
- Mobile usável mas não otimizado — alvo primário é desktop (operação).

Tudo isso será endereçado no Sprint 2 e posteriores, conforme o backend Go for habilitando cada capability.

## Próximos sprints (roadmap resumido)

- **Sprint 2** — Skeleton do backend Go (sem features).
- **Sprint 3** — Identity + tenancy + autenticação real.
- **Sprint 4** — Vertical slice `CreateLead` end-to-end (HTTP → outbox → realtime).
- **Sprint 5+** — features por prioridade (Inbox realtime → Workflow runtime → Copilot → Campanhas → Analytics).
