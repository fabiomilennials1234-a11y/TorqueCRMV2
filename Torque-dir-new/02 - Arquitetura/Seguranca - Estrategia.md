---
tipo: arquitetura
---

# Segurança

Segurança é **não-negociável** e **transversal** — cada feature precisa considerar a seção abaixo desde o design. Esta página resume as práticas obrigatórias.

## Modelo de ameaça (resumido)

- **Adversário externo não autenticado**: tenta explorar endpoints públicos (webhook de ingestão, login, checkout), brute force, enumerar usuários.
- **Adversário autenticado de outra organização**: tenta acessar dados de Org B estando logado em Org A.
- **Adversário autenticado na própria organização (membro malicioso)**: tenta escalar privilégio, acessar recursos além de sua permissão.
- **Insider da empresa-dona (master admin comprometido)**: minimizado por auditoria imutável e MFA obrigatório.
- **Integração externa comprometida** (ex.: n8n do cliente comprometido): minimizado por secret de webhook rotacionável e rate limit.
- **Dependência comprometida** (supply chain): minimizado por bloqueio de updates automáticos e auditoria de dependências.

## Autenticação

### De usuário
- Login com email + senha. Senha em Argon2id ou bcrypt custo alto.
- Verificação de email na criação de conta.
- MFA opcional (TOTP). Obrigatório para master admin.
- Reset de senha via link temporário (válido curto, single-use, invalida sessões antigas).
- Sessões expiram por inatividade (ex.: 7 dias). Renovação via refresh token.
- Logout invalida token no servidor (não é só client-side).
- Bloqueio temporário após N tentativas falhas.
- Detecção de login suspeito (geolocalização incomum, user-agent desconhecido → notificação por email).

### De sistema
- **Integradores externos** (n8n): autenticam com API key por organização. Key rotacionável, revogável. Rate limit por key.
- **Webhooks externos recebidos** (canal, ERP, pagamento): autenticam via secret em header ou via assinatura HMAC do payload.
- **Cron jobs internos**: autenticam com secret interno, nunca exposto fora do ambiente.
- **Chamadas entre serviços internos**: mTLS ou token interno curto.

## Autorização

- Engine de permissão central (ver [[03 - Identidade e Permissões/Modelo de Permissões]]).
- Negação é default. Permissão é explícita.
- Master admin requer flag específica + auditoria.
- Toda ação sensível re-verifica autorização no backend (nunca confiar no cliente).

## Isolamento Multi-tenant

Ver [[Multi-tenancy]]. Pontos críticos:
- RLS ou equivalente na persistência é **camada obrigatória**, não opcional.
- Qualquer endpoint que aceita `organization_id` como input é suspeito e precisa justificativa forte (master admin apenas).
- Teste de isolamento automatizado por endpoint sensível.

## Dados Sensíveis

### Em repouso
- Senhas: hash (nunca reversível).
- Secrets de integração (API keys de terceiros, tokens OAuth): cifrados com chave gerenciada.
- Dados pessoais (PII) do lead: criptografia em nível de banco quando viável, mascarados em logs.
- Backups cifrados.

### Em trânsito
- TLS 1.2+ sempre.
- Certificados gerenciados automaticamente (renovação).

### Em log
- **Nunca logar**: senhas, tokens, conteúdo completo de mensagem em `info`.
- Mascarar: telefone parcial, email parcial, CPF parcial.
- IDs de entidades são OK.

## Input Validation

- Toda entrada validada por schema.
- Tipos estritos (não aceitar string em número).
- Limites de tamanho (payload máx, string máx, array máx).
- Formato explícito (email regex, telefone E.164, UUID).
- Sanitização de HTML em campos que podem ir a renderização.
- Proteção contra injection:
  - SQL: usar binding/parâmetro sempre.
  - NoSQL: igual.
  - Command: não construir comando de shell com input.
  - LDAP / XML / etc.: aplicar escape adequado.

## Output Escaping

- HTML: escape de strings de usuário em render.
- JSON: usar serializador padrão, não concatenação.
- URL: encode de parâmetros.

## Headers de Segurança

- `Strict-Transport-Security: max-age=63072000; includeSubDomains; preload`.
- `Content-Security-Policy`: restritivo, só scripts de origem conhecida.
- `X-Frame-Options: DENY`.
- `X-Content-Type-Options: nosniff`.
- `Referrer-Policy: strict-origin-when-cross-origin`.
- `Permissions-Policy`: negar APIs não usadas (microphone, camera, geolocation) por default.

## CORS

- Lista branca explícita de origens.
- Credenciais apenas em origens confiáveis.
- Preflight respeitado.

## Rate Limit

- Por IP em endpoints públicos (login, webhook público).
- Por organização em endpoints de API.
- Por usuário em endpoints autenticados.
- Limites visíveis via header (X-RateLimit-*).
- 429 com `Retry-After` quando excedido.

## Secrets Management

- Secrets em gerenciador dedicado (vault, secret manager do provedor de cloud, Doppler, etc.).
- Nunca em repositório, nunca em config versionada.
- Rotação documentada e executada periodicamente.
- Acesso a secrets em produção auditado.
- Secrets diferentes por ambiente (dev / staging / produção).

## Dependências

- Lock files versionados (garante reproduzibilidade).
- Varredura automatizada de vulnerabilidades em cada merge.
- Atualização de dependências com CVE crítico em < 7 dias.
- Dependências com manutenção parada ou autor único marcadas como risco.

## Supply Chain

- Artefatos de build assinados quando possível.
- Docker images de fontes confiáveis; pin por digest, não apenas tag.
- CI/CD com permissões mínimas (principle of least privilege).

## Auditoria

- Log imutável de ações sensíveis:
  - Login e logout de usuário.
  - Mudança de papel / permissão.
  - Ação de master admin em organização de cliente.
  - Criação/deleção de organização.
  - Rotação de API key.
  - Export de dados.
  - Consumo de Oráculo Comercial (consulta IA sobre dados).
- Retenção do audit log ≥ 1 ano (compatível com LGPD).
- Auditoria consultável pelo admin da própria organização (tudo dentro dela) e pelo master.

## Resposta a incidente

- Playbook documentado.
- Contato 24/7 do time de segurança.
- Capacidade de revogar tokens em massa, bloquear organização, desabilitar integração em minutos.
- Comunicação ao cliente em incidente que o afete (LGPD: 72h após descoberta).

## Testes de Segurança

- **SAST** (análise estática) em CI.
- **DAST** (teste dinâmico) pré-release em endpoints sensíveis.
- **Pentest externo** anual.
- **Teste de isolamento multi-tenant** automatizado por endpoint.
- **Teste de permissão** para toda combinação papel × ação crítica.
- **Revisão de código** obrigatória em mudanças que afetem auth, autorização, ou isolamento.

## OWASP Top 10 — Mapeamento

| Risco | Mitigação Torque |
|---|---|
| Broken Access Control | Engine de permissão central + RLS + testes por endpoint |
| Cryptographic Failures | TLS + hash forte + criptografia de secrets + backup cifrado |
| Injection | Binding/parâmetro obrigatório; validação de input estrita |
| Insecure Design | Revisão de spec de segurança em toda feature sensível |
| Security Misconfiguration | Headers de segurança, CORS restrito, defaults seguros |
| Vulnerable Components | Varredura automática + SLA de patch |
| Auth Failures | Argon2id, MFA opcional, bloqueio após tentativas, sessões com expiry |
| Software & Data Integrity Failures | Dependências pinadas, artefatos assinados |
| Logging & Monitoring Failures | Log estruturado + captura de exceção + alertas |
| Server-Side Request Forgery | URLs externas validadas antes de fetch; allowlist de destinos |

## Princípios

1. **Fail closed**: ausência de autorização = negação.
2. **Least privilege**: sempre menor escopo possível.
3. **Defense in depth**: múltiplas camadas, não confiar em uma só.
4. **No secrets in client**: cliente nunca vê chave de serviço externo.
5. **Audit everything sensitive**: quem fez, quando, sobre o quê.
6. **Validate all the things**: input, auth, permissão, escopo de tenant.
