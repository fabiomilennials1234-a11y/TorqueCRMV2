---
tags:
  - sistema-base
  - arquitetura
  - estrutura
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Estrutura de Pastas

A organização de `src/` é canônica e enforçada. Não é sugestão — é contrato. ESLint bloqueia imports fora do padrão.

## Árvore

```
src/
├── main.tsx
├── App.tsx
├── routes.tsx
├── providers/              (AuthProvider, OrgProvider, ThemeProvider, QueryProvider, WSProvider)
├── shell/                  (AppShell, Sidebar, TopBar, CommandPalette, OrgSwitcher, TorqueMark)
├── ui/                     (17 primitivos + StepProgress, QuotaGauge, ChannelBadge)
├── features/               (uma pasta por feature — vazio no sistema base)
├── hooks/                  (useSession, useCanPerformAction, usePermission, useWSStatus)
├── api/                    (client fetch + endpoints por entidade)
├── contracts/              (api.gen.ts + manual.ts + transformers snake↔camel)
├── lib/
│   ├── fetch.ts            (HTTP client + interceptor)
│   ├── ws.ts               (WebSocket singleton)
│   ├── domain/             (regras puras de domínio — sem React)
│   ├── vocabulary.ts       (termos canônicos)
│   └── utils.ts            (cn, formatters)
├── styles/
│   └── globals.css
└── i18n/                   (react-intl + pt-BR apenas)
```

## Responsabilidades por pasta

### `providers/`
Context providers de alto nível. Um arquivo por provider. Sem lógica de domínio — apenas wiring de contextos globais.

### `shell/`
Estrutura de aplicação. Layout, navegação, chrome. Sem conhecimento de features específicas.

### `ui/`
Primitivos puros, livres de domínio. Se tiver regra de negócio, não pertence aqui — vai para `features/<feature>/components/`.

### `features/`
Uma pasta por feature. Padrão interno de cada feature:

```
features/<feature>/
├── routes/             (páginas)
├── components/         (componentes de domínio)
├── hooks/              (hooks específicos da feature)
├── api.ts              (endpoints da feature — reexportando api/)
└── index.ts            (barrel público controlado)
```

### `hooks/`
Hooks transversais (sessão, permissões, WS). Hooks de feature vivem em `features/<feature>/hooks/`.

### `api/`
Camada de acesso a endpoints. Funções puras (`async`) retornando dados tipados. Sem React.

### `contracts/`
- `api.gen.ts` — gerado por `openapi-typescript`
- `manual.ts` — tipos manuais até o OpenAPI cobrir
- `transformers.ts` — conversões snake↔camel, paginação cursor

### `lib/`
Utilitários puros. `fetch.ts` e `ws.ts` são os únicos "clientes" globais. `domain/` contém regras de negócio desacopladas de React.

### `styles/`
Um único `globals.css` com tokens, reset, utilities. Nada de CSS espalhado por feature.

### `i18n/`
Mensagens organizadas por feature. Hoje PT-BR. Estrutura preparada para multi-idioma futuro.

## Regras de import

- **Sempre** via alias `@/` configurado no `tsconfig.json` + `vite.config.ts`
- **Nunca** `../../..`
- **Proibido** `features/<a>` importar de `features/<b>` (use contratos via `lib/domain/` ou eventos)
- **Proibido** `ui/` importar de `features/`
- **Proibido** `lib/domain/` importar React

ESLint `no-restricted-imports` enforçando:

```js
{
  patterns: [
    { group: ['../*', '../../*'], message: 'Use alias @/ em vez de caminho relativo.' },
    { group: ['@/features/*/!(index)'], message: 'Importe apenas via barrel @/features/<feature>.' }
  ]
}
```

## Referências

- [[Escopo do Sistema Base]]
- [[Plano de Execucao]]
