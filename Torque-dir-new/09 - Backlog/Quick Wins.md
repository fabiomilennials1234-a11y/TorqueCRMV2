---
tags: [backlog, quick-wins, fase-0, execucao]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Quick Wins

Cinco entregas independentes que podem ser executadas hoje, sem bloqueio do backend Go. Cada uma por si só eleva a base do produto e evita retrabalho posterior. Todas cabem na [[Backlog Priorizado|Fase 0 do backlog]].

## 1. Tokens CSS de calor, canal, countdown e job em `globals.css`

**Descrição.** Adicionar as variáveis CSS de semântica de domínio diretamente no arquivo `globals.css`. Quatro escalas principais: `--heat-{cold,warm,hot,burning}` para temperatura de lead, `--channel-{whatsapp,instagram,messenger,webchat}` para canal de comunicação, `--countdown-{safe,alert,critical}` para urgência de follow-up, `--job-{queued,running,done,failed}` para estado de processamento assíncrono. Cada token tem variante foreground, background e border. Dark-first, com valores suaves e contraste AAA.

**Arquivos afetados.** `src/styles/globals.css`. Opcionalmente `tailwind.config.ts` se os tokens forem expostos como classes utilitárias (`bg-heat-hot`, `text-channel-whatsapp`, etc.).

**Critério de aceite.** Página de referência visual (pode ser rota interna `/_dev/tokens`) renderiza todos os tokens em swatches com label. Contraste testado. Snapshot visual aprovado. Nenhuma classe hard-coded `bg-red-500` ou similar sobrevive em código de domínio a partir deste ponto.

**Estimativa.** 2 horas.

## 2. Componente `ChannelBadge` com quatro variantes

**Descrição.** Primitivo reutilizável que recebe `channel: "whatsapp" | "instagram" | "messenger" | "webchat"` e renderiza um badge com ícone e cor semântica do canal. Suporta props `size: "sm" | "md" | "lg"`, `variant: "solid" | "soft" | "outline"` e estado `muted` para canais desativados. Consome os tokens `--channel-*` do quick win 1. Acessível: `aria-label` com o nome do canal, ícone com `role="img"`.

**Arquivos afetados.** `src/components/primitives/ChannelBadge.tsx`, `src/components/primitives/ChannelBadge.stories.tsx` se houver Storybook, teste unitário básico em `ChannelBadge.test.tsx`.

**Critério de aceite.** Componente renderiza sem erro para os quatro canais. Ícones alinhados com o peso visual da marca de cada canal (WhatsApp verde, Instagram gradiente sutil, Messenger azul, Webchat neutro). Sem uso direto fora do primitivo: busca por `text-green-500` em contextos de canal deve retornar zero resultados.

**Estimativa.** 3 horas.

## 3. Renomear label "Pipelines" para "Funis" na sidebar

**Descrição.** Alinha o vocabulário do produto ao domínio definido em [[Vocabulario de UI]]. "Funil" e "Funis" são os termos de negócio. Atualizar apenas o label na sidebar e qualquer referência textual interna visível. A rota `/pipes/whatsapp` será criada em F01 — aqui apenas o vocabulário muda.

**Arquivos afetados.** `src/shell/Sidebar.tsx` atualiza o label. Constantes de navegação se existirem.

**Critério de aceite.** Nenhuma ocorrência de "Pipeline" ou "Pipelines" visível na sidebar. Rota interna não muda nesta etapa (a rota existente `/pipeline` permanece até F01 a substituir).

**Estimativa.** 30 minutos.

## 4. Grupos de navegação "Automação" e "Equipe" com placeholders

**Descrição.** Sidebar ganha dois grupos novos com a arquitetura de informação do produto final, mesmo que os itens dentro ainda sejam placeholders. "Automação" contém Workflows, Campanhas, Follow-ups. "Equipe" contém Usuários, Papéis, Performance. Cada placeholder aponta para uma rota que renderiza estado vazio com copy "Em breve" e um link de volta. Assim o usuário já vê o mapa mental do produto.

**Arquivos afetados.** `src/components/AppShell/Sidebar.tsx`, `src/app/automacao/{workflows,campanhas,followups}/page.tsx`, `src/app/equipe/{usuarios,papeis,performance}/page.tsx`, componente compartilhado `ComingSoonPlaceholder` em primitivos.

**Critério de aceite.** Sidebar mostra os dois grupos com ícones e labels finais. Clicar em qualquer placeholder leva para página `Em breve` consistente. Grupos colapsáveis, estado persistido em localStorage.

**Estimativa.** 2 horas.

## 5. `StepProgress` primitivo para wizards futuros

**Descrição.** Primitivo reutilizável que renderiza um stepper horizontal editorial. Recebe `steps: string[]`, `currentStep: number`, opcionalmente `completedSteps: number[]`. Usado futuramente por OnboardingWizard (F13), CheckoutWizard (F14), CopilotWizard (F06). Hairline connectors com animação de progresso via `ease-out-soft`. Step ativo com accent ring.

**Arquivos afetados.** `src/ui/step-progress.tsx`, teste unitário básico.

**Critério de aceite.** Componente renderiza N steps com label, estado ativo/completo/pendente. Acessível (aria-current, aria-label). Sem dependência de feature específica. Zero `border` Tailwind (shadow-hairline only).

**Estimativa.** 3 horas.

---

Total agregado: 11.5 horas. Quick wins 2 e 5 dependem dos tokens do quick win 1. Demais são independentes.
