---
tags: [adr, architecture]
created: 2026-04-15
last_updated: 2026-04-15
status: accepted
id: ADR-003
---

# ADR-003 — Autenticação via cookies httpOnly com SameSite=Strict

## Contexto

O legado carregava a CONCERN-S1: a service role key do Supabase era embarcada no bundle do front, o que tornava trivial para qualquer usuário do produto extrair credenciais com acesso total ao banco. A reescrita precisa eliminar essa classe de risco desde o primeiro commit, e a decisão de onde armazenar o token de sessão é o ponto mais alto de leverage nesse eixo.

Armazenar JWT em `localStorage` ou `sessionStorage` continua popular porque é simples, mas qualquer XSS — mesmo um que dure minutos até ser corrigido — permite exfiltração silenciosa do token. Uma única lib de terceiro comprometida, um único `innerHTML` com conteúdo de usuário, e todas as sessões ativas são comprometidas. O custo de um incidente assim, dado o perfil de dados do produto (leads, conversas, dados de CRM de clientes), é alto demais para ser aceito como trade-off.

Guardar o token apenas em memória resolve XSS mas quebra a experiência em refresh de aba, abre de novo o login em toda navegação dura, e cria pressão para encurtar a sessão. Nenhuma dessas opções resolve o problema real: o token não deveria estar acessível ao JavaScript em primeiro lugar.

CSRF precisa ser mitigado no mesmo passo. SameSite do cookie resolve a maior parte dos vetores; endpoints de mutação sensível (delete, payment, transferência de ownership) ganham defesa em profundidade com um CSRF token rotativo.

## Decisão

JWT de sessão em cookie httpOnly com SameSite=Strict, refresh token rotativo em cookie separado, e CSRF token para mutações sensíveis.

- Cookie de sessão: `__torque_session`, `HttpOnly`, `Secure`, `SameSite=Strict`, `Path=/`, TTL curto (15 min).
- Cookie de refresh: `__torque_refresh`, `HttpOnly`, `Secure`, `SameSite=Strict`, `Path=/auth/refresh`, TTL longo (30 dias), rotativo a cada uso (detecção de reuso força logout de todas as sessões).
- O front nunca toca, lê ou escreve tokens. Não existe `localStorage.setItem`, não existe header `Authorization` montado manualmente.
- `fetch` sempre com `credentials: 'include'`; wrapper em `src/lib/fetch.ts` adiciona isso por default.
- CSRF: header `X-CSRF-Token` obrigatório em `DELETE`, `POST` e `PATCH` de rotas marcadas como sensíveis (payment, delete de entidade, transfer); token rotativo emitido em `/auth/me` e guardado em cookie não-httpOnly `__torque_csrf` lido pelo front e espelhado no header (double-submit).
- Rotas: `/auth/login`, `/auth/logout`, `/auth/refresh`, `/auth/me`. `login` emite ambos cookies; `logout` limpa; `refresh` rota única que aceita apenas o cookie de refresh; `me` devolve usuário + novo CSRF token.

## Alternativas consideradas

- **JWT em localStorage** — descartado porque qualquer XSS exfiltra o token; a superfície de ataque inclui toda a cadeia de dependências do front.
- **JWT apenas em memória (React state)** — descartado porque quebra UX em refresh de aba e pressiona por tokens curtos sem solução elegante para refresh sem re-login.
- **OAuth2 com PKCE em SPA pública** — descartado porque adiciona complexidade (fluxo de redirect, state, nonce, code verifier) sem ganho no modelo de ameaça que importa aqui; PKCE protege contra interceptação de code, não contra XSS no cliente.
- **SameSite=Lax** — descartado porque permite que navegações top-level (`<a>` clique de um site externo) enviem o cookie, abrindo CSRF em fluxos que mudam estado via GET ou via forms top-level. Strict é mais duro e compatível com o modelo de uso (app com login).
- **Duplo token (access em memória + refresh em cookie)** — descartado porque ainda expõe access ao JS; ganho marginal sobre httpOnly puro não justifica a complexidade.

## Consequências

**Positivas**

- XSS não exfiltra token de sessão; o vetor mais comum de takeover de conta é eliminado.
- CSRF mitigado por SameSite=Strict; mutações sensíveis têm segunda camada via CSRF token.
- Front mais simples: zero código de gestão de token, zero risco de vazar token em logs ou erro reports.
- Rotação de refresh com detecção de reuso permite invalidar sessão inteira se um refresh for usado duas vezes (sinal de comprometimento).

**Negativas**

- Fluxos cross-origin (subdomínios, widgets embutidos) ficam mais delicados; qualquer embed futuro precisa desenho específico.
- Testes end-to-end precisam de browser context real (Playwright, não fetch node-side ingênuo) para cobrir cookies.
- Logout em todas as abas requer sinalização extra (BroadcastChannel ou evento via WS), já que o front não observa cookie diretamente.
- Debug é ligeiramente mais opaco em dev: tokens não são visíveis no console/storage.

## Impacto

- Backend Go: rotas `/auth/{login,logout,refresh,me}`, middleware de autenticação por cookie, emissão e validação de CSRF.
- `src/lib/fetch.ts`: wrapper com `credentials: 'include'`, injeção de `X-CSRF-Token` em mutations sensíveis, tratamento de 401 com retry via `/auth/refresh`.
- `src/app/AuthProvider.tsx`: estado de usuário derivado de `/auth/me`, sem manipulação de token.
- Deploy: domínio único ou cookies com `Domain` e `Path` bem escolhidos; HTTPS obrigatório em todos os ambientes (inclusive dev, via mkcert ou similar).

## Próximos passos

1. Implementar rotas de auth no Go com cookies configurados corretamente.
2. Escrever `src/lib/fetch.ts` com interceptors de 401 → refresh → retry e injeção de CSRF.
3. Implementar `AuthProvider` que hidrata a partir de `/auth/me` na montagem.
4. Configurar HTTPS em dev local (mkcert) e garantir cookies `Secure` funcionando.
5. Documentar a lista de rotas sensíveis que exigem `X-CSRF-Token`.

## Links

- [[Segurança]]
- [[Arquitetura do Front]]
- [[ADR-001-contratos-openapi-snake-camel]]
- [[CONCERN-S1 Service Role Key Exposta]]
