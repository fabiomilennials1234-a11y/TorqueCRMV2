---
tags:
  - sistema-base
  - plano
  - execucao
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Plano de Execução — Sistema Base

Ordem de execução do Sistema Base. Blocos explicitam o que paraleliza e o que depende.

**Critério de saída geral:** [[Checklist Sistema Base]] 100%. Só então [[00 - Mapa de Features]] começa em F01.

---

## Bloco 1 — Fundação visual

**Paralelizável. ~1 dia.**

Objetivo: a base visual já está em pé antes de qualquer lógica de produto ser tocada.

1. Novos tokens CSS (calor, canal, countdown, job) em `globals.css`
2. Self-hosting de Fraunces + Instrument Sans + JetBrains Mono via `@fontsource`
3. Remover referência à Google Fonts CDN
4. Renomear sidebar de "Pipelines" para "Funis"
5. Grupos de navegação "Automação" e "Equipe" com placeholders (vazios)

---

## Bloco 2 — Estrutura

**Sequencial.**

Objetivo: esqueleto de código pronto para receber qualquer feature.

6. Criar pastas: `src/api/`, `src/hooks/`, `src/contracts/`, `src/providers/`, `src/lib/domain/`, `src/i18n/`
7. Configurar script `generate:types` com `openapi-typescript`
8. Rascunhar `src/contracts/manual.ts` com as 16 entidades fundamentais
9. Implementar transformers snake↔camel + paginação cursor-based

---

## Bloco 3 — HTTP + Auth + Tenancy

**Requer backend Go skeleton com `/auth/login`, `/auth/me`, `/auth/refresh`.**

Objetivo: autenticação real, com tenancy respeitada desde o primeiro request.

10. `src/lib/fetch.ts` com interceptor 401 → refresh
11. `AuthProvider` + `GET /auth/me` bundle (user + permissões + org)
12. Hooks `useSession`, `useCanPerformAction`, `usePermission`
13. `ProtectedRoute`, `MasterRoute`, `<PermissionGate>`
14. Página `/login` real (form validado, loading, error states)

---

## Bloco 4 — Realtime + Observabilidade

**Pode rodar em paralelo com Bloco 3 depois que `/auth/me` existe.**

Objetivo: o usuário sente o sistema vivo e nós vemos o que está acontecendo.

15. `src/lib/ws.ts` singleton com reconexão exponencial
16. `useWSStatus` + badge no `TopBar`
17. Sentry inicializado com scrubbing de PII
18. Logger cliente estruturado

---

## Bloco 5 — Segurança e qualidade

**Sequencial no final. Bloqueia "pronto para features".**

Objetivo: auditoria externa não encontra nada para ter vergonha.

19. Headers no servidor Go: CSP nonce-based, HSTS, XFO, XCTO, Referrer-Policy, Permissions-Policy
20. `GET /api/bootstrap` (config sensível via backend, sem `VITE_*` com secret)
21. ESLint rules: `no-restricted-imports`, proibição de `dangerouslySetInnerHTML` sem allowlist
22. Testes dos primitivos (Vitest + Testing Library + snapshot)
23. Pipeline CI: lint + typecheck + test + build

---

## Critério de saída

[[Checklist Sistema Base]] **100%**.

Só então [[00 - Mapa de Features]] começa em **F01**.

## Referências

- [[Escopo do Sistema Base]]
- [[Estrutura de Pastas]]
- [[Checklist Sistema Base]]
- [[Pendencias e Lacunas]]
- [[00 - Mapa de Features]]
