---
tags: [arquitetura, seguranca, csp, headers, web]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Seguranca Web

Headers e politicas nao sao configuracao de build. Sao contrato operacional. Quem responde e o servidor Go em producao, nao o dev server do Vite. Em dev, um reverse proxy espelha os headers para evitar surpresas em prod.

## Content Security Policy

CSP **nonce-based estrita**. Sem `unsafe-inline`, sem `unsafe-eval`. O servidor gera um nonce criptograficamente aleatorio por request e injeta nos tags `<script>` e `<style>` do HTML de entrada.

```
Content-Security-Policy:
  default-src 'none';
  script-src 'nonce-{RANDOM}' 'strict-dynamic';
  style-src 'nonce-{RANDOM}' 'unsafe-hashes';
  font-src 'self';
  img-src 'self' data: https://storage.torque.app;
  connect-src 'self' wss://api.torque.app https://api.torque.app;
  media-src 'self' blob:;
  frame-src 'none';
  frame-ancestors 'none';
  base-uri 'self';
  form-action 'self';
```

Notas:
- `strict-dynamic` libera scripts carregados dinamicamente por um script com nonce valido (precisa para chunks Vite).
- `font-src 'self'`: Google Fonts e **self-hosted** (baixamos os `.woff2` no build). CDN e proibido porque invalida CSP estrita e adiciona dependencia de terceiro.
- `img-src` libera `data:` para avatares gerados e `https://storage.torque.app` para uploads.
- `connect-src` enumera exatamente os endpoints HTTP/WS. Zero wildcards.
- `frame-ancestors 'none'`: a aplicacao nao pode ser embutida em iframe, anti-clickjacking.

Reporting: `Content-Security-Policy-Report-Only` em ambientes de staging com `report-to` apontando para endpoint interno, para detectar regressoes antes de enforcement em prod.

## Outros headers

| Header | Valor | Razao |
|---|---|---|
| `Strict-Transport-Security` | `max-age=63072000; includeSubDomains; preload` | 2 anos, HSTS preload |
| `X-Frame-Options` | `DENY` | Redundancia com `frame-ancestors` para browsers antigos |
| `X-Content-Type-Options` | `nosniff` | Bloqueia MIME sniffing |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Vaza apenas origin em cross-site |
| `Permissions-Policy` | `camera=(), microphone=(), geolocation=(), payment=()` | Nega features que nao usamos |
| `Cross-Origin-Opener-Policy` | `same-origin` | Isola browsing context |
| `Cross-Origin-Resource-Policy` | `same-site` | Bloqueia cross-origin reads |

## CORS

- Whitelist estrita baseada em `VITE_APP_ORIGIN` (convertido em lista server-side).
- `Access-Control-Allow-Credentials: true` **apenas** em rotas `/auth/*`. Nao e sempre ligado; reduz superficie.
- `Access-Control-Allow-Methods` por rota (nao curinga).
- Preflights (`OPTIONS`) respondidos em 204 pelo middleware CORS, sem atingir handler.

## XSS

- `dangerouslySetInnerHTML` e **banido por lint rule**. ESLint rule custom no repo do front.
- Markdown e renderizado via `marked` com sanitizacao `DOMPurify` configurada em allowlist:
  - Tags: `p, h1-h6, ul, ol, li, strong, em, code, pre, blockquote, a, img, br, hr, table, thead, tbody, tr, td, th`.
  - Atributos: `href` (com protocol allowlist `http`, `https`, `mailto`), `src` (so `https://storage.torque.app` e `data:`), `alt`, `title`.
- Quando um caso legitimo exigir HTML raw (template de email em preview), sempre via componente `<SafeHtml>` que passa por DOMPurify com politica explicita.

## Secrets

- Zero `VITE_*` com credencial, chave privada ou secret. `VITE_*` e publico por design.
- Configuracao runtime (feature flags, URLs de parceiro) vem de `GET /api/bootstrap` chamado no boot, cached com `staleTime: Infinity`.
- Env vars publicas permitidas: `VITE_APP_ORIGIN`, `VITE_API_BASE`, `VITE_WS_BASE`, `VITE_SENTRY_DSN_PUBLIC`. Nada alem.

## Rate limiting

- Servidor responde `429 Too Many Requests` com header `Retry-After` em segundos.
- Cliente (`src/lib/fetch`) detecta 429: mostra toast com countdown, desabilita o CTA originador, reabilita quando `Retry-After` expira.
- Mutations crticas (envio de campanha, disparo em massa) tem circuit breaker client-side adicional para evitar spam acidental por clique duplo.

## Sentry

- `beforeSend` scrub obrigatorio: regex em chaves `token|password|authorization|cookie|csrf|secret|api_key` de qualquer profundidade; valor substituido por `[REDACTED]`.
- Sampling: 10% em producao (`tracesSampleRate: 0.1`, `replaysSessionSampleRate: 0.1`), 100% em erro (`replaysOnErrorSampleRate: 1`).
- PII: nao enviar `email`, `phone`, `document` em tags ou extras. Usar apenas `user.id` opaco.
- Breadcrumbs de fetch com `sanitizeKeys` sobre headers.

## Upload

- URLs pre-assinadas geradas pelo backend (`POST /uploads/presign` retorna `{ url, fields, key }`).
- Cliente faz PUT direto no object storage. Frontend nunca ve credenciais de storage.
- Validacao dupla: MIME e tamanho no cliente (UX), revalidacao e scan (antivirus, EXIF strip) no backend apos upload.

## Referencias

- [[Arquitetura do Sistema]]
- [[Autenticacao e Autorizacao]]
- [[Multi-tenancy]]
- [[Realtime e Jobs]]
- [[Contratos e Boundaries]]
- [[Glossario e Vocabulario]]
