---
tipo: arquitetura
---

# Padrões Transversais

Padrões aplicados em todas as features do Torque. Implementar qualquer feature sem respeitar estes padrões gera inconsistência e bugs.

## 1. Idempotência

Toda operação que cruza fronteira do sistema (webhook recebido, chamada externa, job consumido) é idempotente.

### Mecanismos
- **Chave de idempotência no input**: caller envia identificador; processamento consulta se já foi executado; se sim, devolve resultado antigo, não reexecuta.
- **Dedupe natural**: quando faz sentido (ex.: webhook de canal — mesmo `message_id` nunca gera dois registros).
- **Upsert consciente**: updates preferidos sobre inserts duplicados; chaves únicas em campos de negócio.
- **Check-then-act transacional**: bloquear linha, verificar estado, agir, commit. Evita race entre dois workers processando o mesmo job.

### Exemplos
- Webhook de ingestão de lead com `external_id`: repetido não duplica.
- Mensagem outbound com chave `(conversation_id, client_message_id)`: repetida não envia duas vezes.
- Workflow execution: cada passo tem identificador; reprocessamento não refaz passos já completos.

## 2. Validação em Camadas

### Layers
1. **UI**: validação imediata (required, format, length). Melhora UX. Nunca é suficiente.
2. **Aplicação**: schema estrito no input (tipos, limites, enums, formatos). Rejeita 400 com detalhe.
3. **Domínio**: invariantes de negócio. Retorna erro tipado quando violação.
4. **Persistência**: constraints finais (unique, foreign key, check). Última salvaguarda.

Qualquer bug em layer superior ainda é barrado em layer inferior.

## 3. Autorização Unificada

### Princípio
- Uma única engine de permissão decide se `usuário U`, papel `R`, na organização `O`, pode executar `ação A` sobre `entidade E`.
- Engine consulta: papel do usuário, feature permissions da organização, overrides por membro, regras de plano.
- Resultado booleano + motivo se negado.

### Uso
- UI consulta engine para saber se renderiza o botão.
- Aplicação consulta engine antes de executar.
- Em domínio, invariantes presumem que a autorização já foi checada (domínio só valida invariantes de negócio).

## 4. Observabilidade

### O que registrar
- **Logs estruturados** (chave-valor, não texto corrido): `timestamp`, `level`, `organization_id`, `user_id` (quando aplicável), `operation`, `duration_ms`, `result`, `context`.
- **Métricas**: counters (leads criados/hora), gauges (workers ativos), histograms (latência de endpoint).
- **Traces**: correlação de chamadas entre caso de uso → repositório → integração externa via `correlation_id`.
- **Exceções**: captura em ferramenta dedicada (Sentry ou equivalente) com contexto de tenant e usuário.

### O que NUNCA logar
- Senhas, tokens, API keys, secrets de webhook.
- Conteúdo completo de mensagens do lead em nível `info` em produção (usar `debug`).
- Números de cartão ou dados financeiros sensíveis.

### Logs operacionais por integração
- Cada chamada a serviço externo loga: serviço, endpoint, duração, resultado, código HTTP, tentativa.
- Falhas de integração geram alerta quando cruzam threshold.

## 5. Retry e Degradação Graciosa

### Para chamadas externas
- **Retry com backoff exponencial** (ex.: 1s, 2s, 4s, 8s, 16s). Máximo N tentativas.
- **Jitter** para evitar thundering herd.
- **Timeout** explícito em cada chamada (nunca depender do default do cliente HTTP).
- **Circuit breaker** quando um provedor específico está falhando em massa: paramos de tentar por X minutos.

### Degradação
- Modelo LLM falhou → agente IA pausa e notifica admin, não manda resposta ruim ao lead.
- Provedor de WhatsApp offline → mensagens enfileiram, UI mostra banner "canal indisponível".
- Analytics pesado falha → mostrar dados parciais + estado de erro no widget específico.

## 6. Realtime

### Como funciona
- Cliente se conecta ao backend via canal persistente (WebSocket ou equivalente).
- Subscreve a tópicos: `organization:<id>:leads`, `organization:<id>:conversations:<conv_id>`, etc.
- Backend, ao confirmar uma mutação, publica no tópico.
- Clientes conectados recebem e atualizam cache.

### Garantias
- **Debounce** para evitar flood (ex.: 2s agrega múltiplas mudanças rápidas em uma atualização).
- **Payloads deltas**: só o que mudou. Campos aninhados (tags do lead, responsável) vêm do cache — o delta carrega só IDs.
- **Escopo de tenant rígido**: impossível assinar tópico de outra organização (servidor valida).

## 7. Feature Gating e Quotas

### Dimensões
- **Feature on/off**: funcionalidade disponível no plano da organização?
- **Quota quantitativa**: quantos objetos pode criar? quantas chamadas pode fazer no período?

### Enforcement
- **UI**: esconde / desabilita botões, mostra upsell quando falta.
- **Aplicação**: valida antes de executar. Retorna 402/403 com motivo explícito (`quota_exceeded`, `feature_not_available`).
- **Domínio**: invariante "não criar Nth agente se plano permite N-1".

### Contadores
- Contadores de quota são incrementados na mesma transação da criação do recurso.
- Resetam em ciclo mensal para métricas de volume (leads/mês).

## 8. Eventos de Domínio

### Formato padrão
```
{
  "event_name": "LeadCreated",
  "event_id": "<uuid>",
  "organization_id": "<uuid>",
  "actor_id": "<uuid ou null se sistema>",
  "entity_type": "lead",
  "entity_id": "<uuid>",
  "timestamp": "<iso8601>",
  "version": 1,
  "payload": { ...campos relevantes... }
}
```

### Catálogo completo em [[09 - Referências/Catálogo de Eventos]].

### Consumo
- **Workflow engine** assina eventos e decide quais workflows disparar.
- **Analytics** assina e agrega em tempo quase real.
- **Notificações** assinam para push in-app, email, in-UI toast.
- **Webhooks externos** configurados pela organização assinam para emitir para endpoints de terceiros.
- **Audit log** assina tudo para história imutável.

## 9. Concorrência e Consistência

### Otimista vs Pessimista
- **Otimista**: maioria dos casos. `version` ou `updated_at` no record. Update só se versão bate. Conflito → recarregar.
- **Pessimista (lock)**: quando há contenção real (ex.: distribuição de lead — dois workers processando mesma fila). Lock por tempo limitado, liberado em qualquer caminho de saída.

### Eventual Consistency Aceitável
- Contadores agregados (ex.: "X leads hoje") podem ter alguns segundos de atraso.
- Score recalculado pode demorar minutos depois de evento que o afeta.
- Search index/vetorial pode ficar atrás alguns segundos.

### Forte Consistência Exigida
- Movimento de lead entre pipes.
- Estado de pagamento.
- Aplicação de quota.
- Qualquer mutação em permissão.

## 10. Segurança por Default

### Headers
- HTTPS obrigatório em toda origem.
- HSTS ligado.
- CSP restritivo (só scripts do próprio domínio + CDN conhecida).
- X-Frame-Options: DENY.
- X-Content-Type-Options: nosniff.

### Input
- Toda entrada validada por schema (tipo, tamanho, padrão).
- Nada é concatenado em query sem binding/parâmetro.
- Uploads: tipo MIME validado, tamanho limitado, varredura antivírus.

### Output
- Escape de HTML ao renderizar strings de usuário.
- CORS estrito — domínios conhecidos apenas.

### Secrets
- Nunca em código ou log.
- Em vault dedicado (env variables gerenciadas, secret manager).
- Rotação documentada.

### Autenticação
- Senha: hash strong (bcrypt/argon2), salt, não-reversível.
- Token de sessão: curto (minutos a horas), refresh token separado.
- MFA opcional para admin e master.

## 11. Internacionalização / Localização

- **Idioma primário**: português brasileiro.
- **Inglês**: possível mas não prioridade imediata — sistema preparado via chave de tradução.
- **Formatos**: moeda BRL, data DD/MM/AAAA, fuso horário por organização (default America/Sao_Paulo).

## 12. Acessibilidade

- WCAG 2.1 AA como alvo.
- Contraste adequado em ambos dark e light.
- Navegação por teclado em toda feature crítica.
- Leitores de tela: roles ARIA em componentes custom.
- Foco visual sempre presente.
