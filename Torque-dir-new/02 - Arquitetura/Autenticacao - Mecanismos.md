---
tipo: identidade
---

# Autenticação

Verificação da identidade de usuários e sistemas que interagem com o Torque.

## Tipos de Identidade

1. **Usuário humano** (admin, membro, master) — faz login com email/senha.
2. **Integrador externo** — autentica com API key (gerada por organização).
3. **Canal externo (webhook recebido)** — autentica com secret por organização + opcionalmente HMAC.
4. **Sistema interno** (cron, worker) — autentica com secret interno não exposto.

## Login de Usuário

### Fluxo
1. Usuário envia `email + password + (opt) mfa_code` para endpoint de login.
2. Backend valida credenciais:
   - Busca usuário por email (case-insensitive).
   - Valida senha via hash (Argon2id ou bcrypt).
   - Se MFA habilitado: valida `mfa_code` contra TOTP.
   - Checa se usuário está ativo.
   - Checa bloqueio por tentativas falhas (lockout).
3. Se sucesso: emite `access_token` (curto, 15-60 min) + `refresh_token` (longo, dias).
4. Registra login em audit log com IP, user-agent.
5. Detecta login suspeito (geolocalização incomum) → notifica por email.

### Campos do token
Access token contém (como claims):
- `user_id`: usuário global.
- `organization_id`: organização ativa no momento (usuário pode trocar).
- `team_member_id`: membro desta org.
- `role`: `admin` | `membro` | `master`.
- `permissions_version`: incrementa quando permissões mudam (força invalidação).
- `issued_at`, `expires_at`.

### Refresh
- Cliente envia refresh_token para endpoint `/refresh`.
- Backend valida, emite novo access_token.
- Refresh_token é invalidado ao logout.
- Rotação de refresh_token em cada uso (opcional, ver segurança).

### Logout
- Backend invalida tokens (lista de revogação ou expiração curta + refresh revogado).
- Cliente limpa estado local.

## Senha

### Criação
- Mínimo 10 caracteres. Mix recomendado (mas não obrigatório) de maiúscula, número, símbolo.
- Check contra lista de senhas vazadas (haveibeenpwned-style).
- Armazenada em hash Argon2id (preferido) ou bcrypt custo ≥ 12.

### Reset
1. Usuário informa email.
2. Sistema sempre responde "se o email existir, enviaremos link" (não vaza existência).
3. Se existir: emite token curto (1 hora), single-use.
4. Link por email.
5. Usuário clica, informa nova senha.
6. Token consumido; sessões antigas invalidadas.

### Mudança
- Usuário autenticado informa senha atual + nova.
- Validação: senha atual correta, nova atende critérios.
- Hash atualizado.
- Sessões ativas em outros devices: opcional invalidar (recomendado).

## MFA (Two-Factor Authentication)

### Tipos suportados
- TOTP (Google Authenticator, Authy, 1Password, etc.) — padrão.
- SMS — não recomendado como único (SIM swap).
- Backup codes — 10 códigos de uso único para recuperação.

### Ciclo
1. Usuário ativa MFA em configurações.
2. Sistema gera secret, exibe QR code para TOTP.
3. Usuário escaneia, informa código de confirmação.
4. Se válido, MFA ligado. Backup codes gerados.
5. Logins subsequentes pedem código TOTP além de senha.

### Obrigatoriedade
- **Master admin**: obrigatório.
- **Admin de org**: altamente recomendado, pode ser obrigatório conforme plano.
- **Membro**: opcional.

## Sessão

### Duração
- Access token: 60 min (ajustável).
- Refresh token: 14 dias de inatividade.
- Session completa: 90 dias máximo (força relogin).

### Invalidação
- Logout explícito.
- Mudança de senha.
- Mudança de permissão crítica.
- Admin remove membro.
- Token comprometido (ação manual master).

### Multi-device
- Usuário pode ter múltiplas sessões ativas.
- Lista de "sessões ativas" na UI mostra device + IP + último acesso.
- "Logout de todos os devices" invalida tudo.

## API Keys (Integradores Externos)

### Criação
- Admin da org cria em Configurações → API Keys.
- Informa nome (ex.: "n8n produção") e escopo (permissões que a key tem).
- Sistema gera key (prefixo + corpo aleatório longo).
- **Exibida uma vez** — depois só prefixo fica visível.

### Uso
- Header `Authorization: Bearer <key>` em toda chamada.
- Backend resolve key → organization_id + scopes.
- Rate limit por key.

### Revogação
- Admin revoga key com um clique.
- Key fica inválida imediatamente.
- Audit log registra.

### Rotação
- Recomendado a cada 90 dias.
- Admin pode criar nova, deixar ambas ativas em janela de transição, revogar antiga.

## Webhook Inbound Authentication

Provedores externos (canal, pagamento, ERP) enviam webhooks. Autenticação:

- **Token embutido na URL**: `https://api.torque/webhooks/canal/<token_da_org>`. Backend resolve token → organization.
- **HMAC no header**: provider assina payload com shared secret; backend valida assinatura.
- **IP allowlist**: complementar, não substituto.

Se falhar: 401 e ignorar payload.

## Autenticação entre componentes internos

- Cron ou worker chamando endpoint interno: usa `X-Cron-Secret` header com secret rotacionável.
- Chamadas entre funções backend: mTLS ou token compartilhado interno.
- Secret nunca exposto para cliente ou logs.

## Segurança de Tokens

### Em trânsito
- Sempre HTTPS.
- Token no header `Authorization`. Nunca em query string.

### Em armazenamento
- Cliente: storage seguro (HttpOnly cookie preferido para access; refresh token também).
- Backend: não persistir access token; persistir refresh token em tabela `sessions` com hash.

### Revogação
- Lista de refresh_tokens ativos por usuário.
- Access token curto minimiza janela se comprometido.

## Anti-brute force

- Lockout temporário após N tentativas falhas (ex.: 5 em 15 min → bloqueio 30 min).
- CAPTCHA opcional após M falhas.
- Notifica usuário por email em tentativa suspeita.

## Detecção de Sessão Anômala

- Login de geolocalização significativamente diferente → email de alerta.
- Login de user-agent novo → registra, opcional email.
- Muitas ações em curto período → rate limit + alerta.

## Troca de Organização (usuários em múltiplas orgs)

- Master ou usuário em N orgs pode trocar contexto sem logout.
- Endpoint `/switch-organization` valida que usuário pertence à org destino.
- Emite novo access_token com `organization_id` atualizado.
- Cliente limpa cache de org anterior e recarrega.

## Impersonação (Master Only)

- Master pode "atuar como" admin de uma organização cliente.
- Flag `impersonation_of: <original_user_id>` no token.
- UI do cliente mostra banner de warning ao master.
- Toda ação fica como feita pelo admin real mas com flag `via_master_impersonation`.
- Audit log dedicado.

## Recuperação de Acesso

- Admin da org perdeu senha: reset padrão.
- Único admin perdeu acesso + reset falhou: suporte via master (identity verification fora do sistema).
- Org inteira sem acesso: master provisiona reset.

## LGPD / Direitos do Titular

- Usuário pode solicitar exportação dos seus dados.
- Usuário pode solicitar exclusão (anonimização — preserva audit com identificadores removidos).
- Solicitações ficam em audit log.
