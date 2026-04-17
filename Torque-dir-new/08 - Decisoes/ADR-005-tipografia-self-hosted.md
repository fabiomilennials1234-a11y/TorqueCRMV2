---
tags: [adr, architecture]
created: 2026-04-15
last_updated: 2026-04-15
status: accepted
id: ADR-005
---

# ADR-005 — Tipografia self-hosted via @fontsource

## Contexto

A estratégia de segurança define uma CSP estrita como padrão (nenhum `unsafe-inline`, nenhum host externo desnecessário, nenhum wildcard). Nesse regime, `font-src 'self'` é a diretiva correta, e qualquer dependência de fonte carregada de um CDN terceiro obriga uma whitelist explícita que enfraquece a CSP e cria um ponto de confiança externo permanente. Google Fonts é o caso mais comum, e carregar `fonts.googleapis.com` mais `fonts.gstatic.com` vira concessão estrutural, não exceção pontual.

Há também um eixo de performance. O `<link>` para Google Fonts envolve DNS, TLS handshake, request do CSS, e só depois o request dos WOFF2 — tudo isso na crítica do LCP, já que a fonte da H1 é inevitavelmente o gargalo de render inicial. Self-hosting com preload corta DNS e handshake extras, permite HTTP/2 push com o resto do bundle, e permite subsetting para reduzir bytes.

Um terceiro eixo é privacidade. Google Fonts CDN, historicamente e ainda hoje em certas jurisdições, levanta questões de LGPD/GDPR por expor IP do usuário a um terceiro em toda visita. Self-hosting elimina essa exposição de origem.

Por fim, controle de versão: uma mudança silenciosa no serviço externo (um glyph, um hinting, um formato) não deveria alterar a renderização do produto. Self-hosting fixa a versão exata das fontes no `package-lock.json`.

## Decisão

Self-host das três famílias via pacotes `@fontsource`, com preload das variantes críticas e remoção completa do `<link>` Google Fonts.

- Famílias: `Fraunces` (display editorial, eixo `opsz` até 144), `Instrument Sans` (UI), `JetBrains Mono` (código, tabular, valores financeiros).
- Pacotes: `@fontsource-variable/fraunces`, `@fontsource/instrument-sans`, `@fontsource/jetbrains-mono` — formato WOFF2, subset `latin-ext` como padrão.
- Preload no `index.html` para as duas variantes mais críticas no render inicial: Fraunces variable (ou a peso específico usado na H1) e Instrument Sans 400 e 500.
- `font-display: swap` por default; Fraunces display na H1 pode usar `optional` se FOIT visual for preferível em redes lentas.
- CSS importa via `@fontsource/.../index.css` em `src/styles/globals.css`; nada de `<link rel="stylesheet" href="fonts.googleapis.com">`.
- CSP passa a ser `font-src 'self'; style-src 'self'` sem exceções para fonts externos.

## Alternativas consideradas

- **Google Fonts CDN via `<link>`** — descartado por quatro motivos cumulativos: CSP fraca, LCP pior, privacidade, falta de controle de versão.
- **CDN próprio (Cloudflare, S3+CloudFront) para as fontes** — descartado porque adiciona complexidade operacional (cache, invalidation, headers) sem ganho sobre servir do mesmo origin do app; com HTTP/2 e bundle do app já sendo servido, o ganho de CDN para fontes é marginal.
- **Fontes do sistema (`system-ui`, `ui-sans-serif`, `ui-monospace`)** — descartado porque o produto tem sensibilidade tipográfica editorial (Fraunces com `opsz` é uma escolha ativa), e cair para fontes do sistema mata a identidade visual em qualquer plataforma que não seja iOS recente.
- **Adobe Fonts / Fontshare via CDN** — mesma categoria de Google Fonts em termos de CSP e privacidade, sem vantagem.

## Consequências

**Positivas**

- CSP estrita viabilizada sem exceções para domínios externos.
- LCP melhora pela eliminação de DNS, TLS e request CSS extra na crítica.
- Privacidade: zero exposição de IP do usuário a terceiros por conta de fontes.
- Renderização reproduzível — versão da fonte está fixada no `package-lock.json`.

**Negativas**

- Aumento leve do bundle total (WOFF2 de variável Fraunces + Instrument Sans + JetBrains Mono); mitigado por subsetting `latin-ext` e preload seletivo.
- Atualizar uma fonte exige bump de dependência e novo deploy, em vez de silenciosamente puxar do CDN.
- Precisa cuidado em `<link rel="preload">`: preload demais piora performance em vez de melhorar.
- Primeira instalação do projeto puxa mais MB no `node_modules`.

## Impacto

- `package.json` — adiciona `@fontsource-variable/fraunces`, `@fontsource/instrument-sans`, `@fontsource/jetbrains-mono`.
- `src/styles/globals.css` — `@import` das fontes e definição de `--font-display`, `--font-sans`, `--font-mono`.
- `index.html` — tags `<link rel="preload" as="font" type="font/woff2" crossorigin>` para as variantes críticas; remoção do `<link>` Google Fonts.
- CSP em headers — atualização para `font-src 'self'; style-src 'self'`.
- Build pipeline — garantir que os arquivos WOFF2 entrem no output com cache headers imutáveis.

## Próximos passos

1. Instalar os três pacotes `@fontsource`.
2. Importar em `globals.css` e definir tokens CSS de família.
3. Adicionar preload das variantes críticas no `index.html`.
4. Remover qualquer `<link>` de Google Fonts remanescente e atualizar o header CSP.
5. Medir LCP antes e depois para validar o ganho previsto.

## Links

- [[Design System]]
- [[Segurança]]
- [[Performance]]
- [[ADR-003-auth-httponly-cookies-samesite]]
