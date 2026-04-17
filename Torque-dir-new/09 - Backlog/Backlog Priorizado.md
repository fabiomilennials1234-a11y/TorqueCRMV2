---
tags: [backlog, priorizacao, roadmap, fases]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Backlog Priorizado

Backlog completo do Torque-v2 organizado em 6 fases. A ordenação respeita dependências técnicas e maximiza ROI: a fatia vertical F01 vem imediatamente após o skeleton de auth porque é ela que prova, de ponta a ponta, que o pipeline técnico (contratos tipados, auth em cookie httpOnly, realtime WS, DnD otimista, permissões RBAC em quatro camadas) se sustenta sob carga real. Qualquer feature subsequente herda essa confiança.

A regra cardinal permanece: nenhuma feature começa antes da anterior estar completa, revisada e documentada. Ver [[00 - Mapa de Features]] e [[Fluxo de Trabalho]].

## Fase 0 — Sistema Base fundação (Torque-v2 sem Go)

Objetivo: deixar o frontend pronto para receber o backend real sem retrabalho. Tudo que não depende do Go entra aqui. Ver [[Escopo do Sistema Base]].

| Item | Por quê | Esforço (S/M/L) | Dependência | Risco |
|---|---|---|---|---|
| Tokens CSS calor/canal/countdown/job em `globals.css` | Fundação visual de todo componente de pipe, inbox e workflow; muda a chave semântica do design system | S | Nenhuma | Baixo |
| Self-host fontes via `@fontsource` e remoção do Google Fonts CDN | Elimina dependência externa, melhora LCP, remove vazamento de dados para terceiros, compatível com CSP estrita | S | Nenhuma | Baixo |
| Renomear label "Pipelines" para "Funis" na sidebar | Alinha vocabulário ao domínio ([[Vocabulario de UI]]); a rota `/pipes/whatsapp` será criada em F01 | S | Nenhuma | Baixo |
| Grupos de navegação "Automação" e "Equipe" com placeholders | Arquitetura de informação correta desde o início; usuário vê o esqueleto do produto final | S | Item anterior | Baixo |
| Estrutura de pastas `src/{api,hooks,contracts,providers,lib/{domain,fetch,ws}}` | Convenção de projeto consolidada; evita refactor em massa na Fase 1 | S | Nenhuma | Baixo |
| Script `openapi-typescript` com rascunho `manual.ts` para 16 entidades | Destrava trabalho paralelo front/back; tipos viram fonte única de verdade | M | Estrutura de pastas | Médio — manter sync manual até backend existir |
| Transformers snake_case ↔ camelCase | Isola convenção do banco Go da convenção idiomática de TS | S | Contratos | Baixo |
| `src/lib/fetch.ts` estrutura base (sem auth real ainda) | Interceptor pattern pronto para receber 401→refresh na Fase 1 | S | Estrutura de pastas | Baixo |
| Primitivos novos: `StepProgress`, `QuotaGauge`, `ChannelBadge` | Reaproveitados por 5+ features; construir cedo evita divergência visual | M | Tokens CSS | Baixo |
| i18n estruturado com `react-intl` (só pt-BR ativo) | Custo marginal enorme se postergar; PT-BR hoje, outros idiomas sem refactor | S | Nenhuma | Baixo |

## Fase 1 — Backend skeleton, Auth e Tenancy

Depende do backend Go estar inicializado. Entrega o mínimo para sair de mock e ter sessão real, multi-tenant, segura.

| Item | Por quê | Esforço (S/M/L) | Dependência | Risco |
|---|---|---|---|---|
| Endpoints `/auth/{login,logout,refresh,me}` no Go | Base de toda sessão; sem isso nenhuma feature avança | M | Backend Go | Alto — erros aqui se propagam |
| Cookies httpOnly SameSite=Strict | Fecha XSS no token; padrão world-class não negociável | S | Endpoints auth | Baixo |
| `AuthProvider` + `useSession` + `ProtectedRoute` + `MasterRoute` | Contrato único de sessão no front; evita redirect loops | M | Endpoints auth | Médio |
| Interceptor 401 → refresh | UX contínua sem reautenticar; window de race coberto por mutex | S | `fetch.ts` base | Médio — race condition |
| `useCanPerformAction` + `usePermission` + `PermissionGate` | RBAC 4 camadas aplicado na UI; complementa enforcement server-side | M | Sessão | Médio |
| CSP estrita, HSTS e demais headers no Go | Postura de segurança desde o dia 1; evita débito técnico tipo CONCERN-S do legado | M | Backend Go | Baixo |
| `GET /api/bootstrap` para config runtime | Sentry DSN, feature flags, versão; sem rebuild para trocar config | S | Backend Go | Baixo |
| Sentry com scrubbing PII | Observabilidade desde o primeiro erro; ver [[Observabilidade e Logs]] | S | Bootstrap | Baixo |
| `OrgSwitcher` real | Multi-tenancy visível; testa isolamento de dados | S | Sessão, RBAC | Médio |
| Gerar `api.gen.ts` do schema Go | Elimina drift front/back (risco R1 em [[Riscos e Duvidas]]) | S | Schema Go | Baixo |

## Fase 2 — Fatia vertical F01 (Lead + Pipe WhatsApp)

Vale como marco porque exercita, em um só recorte, todos os pilares técnicos do produto. Depois dela, qualquer feature é incremental.

Ver [[F01 - Funis Hub e Pipe WhatsApp/Tasks]] para decomposição completa. Esforço agregado: **L**.

O escopo cobre: CRUD de leads, hub de funis, Pipe WhatsApp com DnD, realtime de movimentação de cards, permissões por coluna, quota de leads por plano, estados vazios, skeletons, empty-states, erros.

## Fase 3 — Comunicação e IA (F02 a F06)

| Item | Por quê | Esforço | Dependência | Risco |
|---|---|---|---|---|
| F02 Pipe Confirmação | Segundo funil confirma abstração de pipe genérico; sem isso F01 pode estar acoplado ao WhatsApp | M | F01 | Baixo |
| F03 Pipe Propostas | Fecha triângulo Lead→Confirmação→Proposta | M | F02 | Baixo |
| F04 Inbox Multi-canal | Unificação de WhatsApp, Instagram, Messenger, Webchat via `contact_key`; desbloqueia Copilot | L | F01 | Alto — modelo de mensagem precisa estar certo |
| F05 Follow-ups | Automação de toque; depende de inbox para registrar envios | M | F04 | Médio |
| F06 Copilot com wizard 20+ steps, RAG, TTS | Diferencial de IA; batch de 8s exige UX cuidadoso de pending state | L | F04, F05 | Alto — [[Riscos e Duvidas]] R3 |

## Fase 4 — Automação e Analytics (F07 a F09)

| Item | Por quê | Esforço | Dependência | Risco |
|---|---|---|---|---|
| F07 Workflow Builder (12 node types) | Plataforma de automação visual; valor de lock-in alto | L | F04, F05 | Alto |
| F08 Campanhas (disparos segmentados) | Monetização por consumo; depende de workflow | L | F07 | Médio |
| F09 Analytics com Visx | Valor analítico para gestão; consome dados das fases anteriores | L | F01 a F08 | Médio |

## Fase 5 — Governança e polimento (F10 a F16)

| Item | Por quê | Esforço | Dependência | Risco |
|---|---|---|---|---|
| F10 Equipe (comissões, metas, premiações + confetti) | Gamificação e gestão de equipe | M | F09 | Baixo |
| F11 Produtos (catálogo, import XLSX, materiais) | Alimenta Pipe Propostas | S | F03 | Baixo |
| F12 Upsell + Pipelines Customizados | Generaliza modelo de pipe | M | F01 | Médio |
| F13 Onboarding Wizard + Gate | Ativação inicial; ver doc do legado | M | Sistema Base | Baixo |
| F14 Checkout + PIX + Provisioning | Monetização direta | L | Sistema Base | Alto — integração pagamento |
| F15 Configurações completas (8 tabs) | Transversal, consolidada ao longo do roadmap | M | Transversal | Médio |
| F16 Master Admin (5 views + Operations Center) | Ferramenta interna; depende do produto inteiro | L | Todas anteriores | Médio |

## Justificativa da ordenação por ROI

A fatia vertical F01 é o maior ROI do cronograma inteiro. Ela destrava simultaneamente a confiança em autenticação real, realtime sobre WebSocket, drag-and-drop otimista com reconciliação, permissões em quatro camadas e contratos tipados ponta a ponta. Qualquer tentativa de construir F02+ sem essa prova sujeita o produto a refatoração dolorosa quando o primeiro edge case aparecer. Uma vez validada F01, as fases seguintes herdam a infraestrutura pronta e passam a custar apenas a complexidade de domínio da feature em si.
