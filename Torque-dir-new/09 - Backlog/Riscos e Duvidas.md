---
tags: [riscos, duvidas, mitigacao, governanca]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Riscos e Dúvidas

Inventário vivo dos pontos de fragilidade do projeto. Revisar a cada fim de fase e atualizar conforme novas descobertas. Complementa [[Backlog Priorizado]] e [[Fluxo de Trabalho]].

## Riscos técnicos

### R1 — Descolamento semântico entre tipos do frontend e schema Go
Se os tipos TypeScript e o schema do backend divergem, o compilador deixa de ser barreira contra bugs. Em produto multi-tenant com 16 entidades e cascata de permissões, um campo fora de sincronia pode vazar dados entre orgs antes de qualquer teste pegar.
**Mitigação.** Geração automática via `openapi-typescript` desde o dia 1, com commit do `api.gen.ts` versionado. Script `pnpm api:gen` roda em pre-commit e em CI. Divergência manual entre `manual.ts` e `api.gen.ts` vira erro de build. Ver [[Contratos de API]].

### R2 — Mudança de modelo realtime (row completo para patch)
Enquanto o backend emite o row completo por evento WS, cache reconcilia trivialmente. Quando migrar para patch (delta apenas), a lógica de cache muda estrutura e pode gerar inconsistência temporária difícil de depurar.
**Mitigação.** [[ADR-002 Realtime Patch Model]] explícita com critérios de corte. Reprojetar `lib/ws` antes de migrar, com adapter que suporta ambos os modelos em paralelo durante transição. Teste de convergência forçada: após qualquer sequência de patches, estado final precisa bater com `GET` autoritativo em janela de 500ms.

### R3 — Batch de 8 segundos do Copilot causando UI inconsistente
Copilot agrupa mensagens em janelas de 8s antes de enviar. Usuário pode ver a própria mensagem "enviada" enquanto o servidor ainda não processou, ou o contrário. Abrir outra conversa no meio da janela complica.
**Mitigação.** Eventos WS `copilot.composing` e `copilot.sent` alimentam estado explícito na UI. Badge "Compondo em 6s" visível. Timeout duro de 15s: se nada chegou, estado é marcado como `failed` com CTA de retry. Teste de carga cobre janelas sobrepostas.

### R4 — Quota stale em concorrência administrativa
Dois admins criando usuários em paralelo podem ultrapassar quota porque cada um lê o snapshot antes da mutação do outro. Enforcement server-side pega, mas UI exibe estado inconsistente.
**Mitigação.** Invalidar cache de quota em React Query imediatamente após qualquer mutação de criação. Em paralelo, retornar quota atualizada no corpo da resposta de mutação (`quota_after`) para atualização otimista correta. Empty-state do formulário de criação já desabilita submit quando quota no limite.

### R5 — Reconexão WebSocket em redes móveis instáveis
Usuários em campo com 3G oscilante podem ficar conectados em nome mas sem tráfego real, causando dados stale sem alerta.
**Mitigação.** Backoff exponencial com jitter em `lib/ws/client`. Badge de status no TopBar (ver [[Observabilidade e Logs]]) com três estados visuais. Heartbeat de 30s: se três perdidos, força reconnect. Empty-state de lista quando desconectado há mais de 60s, com CTA "Recarregar".

## Riscos de produto

### R6 — Feature creep no Sistema Base
Tentação de incluir "só mais um primitivo" na Fase 0 atrasa o início da fatia vertical F01 e adia o momento em que o pipeline técnico é validado ponta a ponta.
**Mitigação.** [[Escopo do Sistema Base]] mantém lista explícita de "NÃO entra". Qualquer proposta de inclusão passa por revisão de escopo. Regra de ouro: se o item não é consumido por F01, fica de fora da Fase 0.

### R7 — Construir feature antes da anterior estar completa
Paralelismo desnecessário cria débito integrativo e retrabalho quando a feature A precisa ajustar contratos que a feature B já assumiu.
**Mitigação.** [[00 - Mapa de Features]] tem regra cardinal (sequencial por default) e checklist obrigatório de encerramento de feature. [[Fluxo de Trabalho]] passo 8: próxima feature só começa após checklist 100%.

### R8 — Microcopy genérico
Copy tipo "Item criado com sucesso" ou "Algo deu errado" rebaixa o produto ao padrão de template. Nada que uma referência como Linear ou Stripe aceitaria.
**Mitigação.** [[Vocabulario de UI]] define termos canônicos (Funil, Pipe, Lead, Card, Etapa, Canal, Toque). Revisão editorial explícita por feature antes do merge, como gate de qualidade. Tom de voz documentado em [[Identidade Verbal]].

## Dúvidas abertas

### D1 — Autenticação social (Google/Microsoft)
Ausência na base. Decisão: **posterga para F_futuro**. Base usa email e senha com confirmação obrigatória. Social login só quando houver sinal de demanda real de usuários empresariais, provavelmente após F13 (Onboarding) amadurecer.

### D2 — Multi-idioma
Infraestrutura `react-intl` está estruturada desde a Fase 0 para evitar refactor massivo. Idioma ativo: **apenas pt-BR**. Inglês e outros só entram se aparecer demanda comercial concreta. Risco baixo de postergar porque a infra já aceita novos catálogos sem mudança de código.

### D3 — Mobile app nativo
**Fora do roadmap atual.** PWA entra na consideração depois de F09 (Analytics). App nativo em React Native ou Swift/Kotlin exige dedicação de equipe que hoje não existe. Produto web responsivo é requisito não negociável em todas as features.

### D4 — Export de dados (CSV/PDF de analytics)
Entra como parte de F09. Não precisa de solução separada. CSV é obrigatório, PDF apenas para relatórios nomeados. Biblioteca: `exceljs` ou nativo via `Blob` para CSV; PDF avalia entre `react-pdf` e server-side rendering no Go.

### D5 — SLA de realtime
Target: **p95 abaixo de 500ms** entre evento no servidor e chegada no cliente. A validar após F01 quando houver tráfego de movimentação de cards para medir. Monitoramento via Sentry custom metrics e OpenTelemetry no Go. Se p95 estourar, revisar entre WebSocket vs Server-Sent Events vs long-polling com `etag`.
