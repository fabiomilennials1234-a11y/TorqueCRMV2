---
tipo: arquitetura
---

# Camadas do Sistema

Detalhamento das responsabilidades e dos contratos entre as cinco camadas lógicas introduzidas em [[Visão Geral]].

## 1. Camada de Apresentação

### Responsabilidades
- Interface gráfica dark-first, mobile-friendly.
- Estado de UI transiente (form, modal, drawer, tabs).
- Cache de dados do servidor com invalidação por chave.
- Subscriptions em tempo real filtradas pela organização ativa.
- Autenticação: login, renovação de token, logout.
- Primeira camada de validação de forms.
- Roteamento SPA com lazy-loading de módulos.
- Feature flags: UI respeita o que o backend declara como disponível para a organização.

### Contratos
- **Para backend**: HTTPS com token no header; inputs validados mas não confiáveis.
- **Do backend**: JSON estruturado, erros tipados (código + mensagem + detalhes), eventos via subscription persistente.
- **Com usuário**: UX responsiva, feedback instantâneo, loading states explícitos, erros humanos.

### Não-responsabilidades
- Não decide permissões (só reflete).
- Não armazena secrets.
- Não executa regras de negócio sozinha.

## 2. Camada de Aplicação

### Responsabilidades
- Endpoints HTTP (REST/RPC) para o cliente.
- Endpoints públicos de webhook para ingestão (leads, canais, pagamento, calendário).
- Endpoints públicos de API para integradores (com API key).
- Casos de uso orquestrados: receber input → validar → invocar domínio → persistir → emitir evento → responder.
- Tradução entre DTO de transporte e entidades de domínio.
- Autorização pontual (além do tenant): "este usuário pode executar esta ação sobre esta entidade?".
- Rate limit por endpoint e por organização.

### Contratos de entrada
- Todo endpoint declara:
  - Método HTTP.
  - Path.
  - Autenticação exigida (token de usuário, API key, secret de webhook, secret de cron).
  - Schema de input.
  - Schema de output (sucesso e erro).
  - Status codes possíveis.
  - Rate limit.
  - Idempotência (se aplicável, qual chave).

### Organização interna
- Cada caso de uso é uma unidade com um ponto de entrada, um output, e lista de side effects (persistir, emitir, chamar externo, enfileirar).
- Side effects são declarados, não difusos.
- Transações: múltiplas escritas relacionadas em persistência transacional ocorrem atomicamente ou não ocorrem.

### Não-responsabilidades
- Não implementa regras de negócio (delega ao domínio).
- Não conhece schema de persistência (delega a repositórios).

## 3. Camada de Domínio

### Responsabilidades
- Representar entidades e seus invariantes.
- Regras de transição de estado.
- Cálculos derivados (score, comissão, métricas instantâneas).
- Validações cross-field e cross-entidade.
- Emitir eventos de domínio quando estado muda.
- Serviços stateless (política de distribuição, engine de permissões, engine de workflow).

### Organização
- Entidades: classes/structs com estado e métodos que enforcem invariantes.
- Value Objects: pequenos objetos imutáveis (valor monetário, telefone, email, identificador).
- Serviços de domínio: operações que não pertencem naturalmente a uma entidade.
- Eventos de domínio: registros imutáveis representando "algo aconteceu" (ver [[09 - Referências/Catálogo de Eventos]]).

### Não-responsabilidades
- Não conhece HTTP, banco, fila, worker, LLM, WhatsApp, calendário.
- Não logs em formato específico — emite eventos/observações que infraestrutura formata.
- Não inicia transações — apenas reporta o que precisa ser persistido.

## 4. Camada de Persistência

### Responsabilidades
- Persistir entidades do domínio.
- Aplicar isolamento multi-tenant na fonte.
- Garantir consistência transacional.
- Fornecer consultas otimizadas (índices, views materializadas) para analytics e listagens.
- Append-only para auditoria, mensagens e histórico.
- Índice vetorial para RAG.
- Retenção e arquivamento de dados antigos.

### Particionamento lógico
- **Store transacional**: entidades mutáveis (lead, pipeline entry, team member, workflow definition).
- **Store append-only**: mensagens, eventos de domínio, lead_history, execution_steps, webhook_deliveries.
- **Store vetorial**: embeddings de FAQ + metadados.
- **Store de objetos/blob**: áudios, imagens, documentos, avatares.
- **Cache**: resultados de agregação pesada, tokens de sessão, rate limits.

### Contrato com domínio
- Repositórios expõem métodos como "encontrar por id na organização X", "salvar", "remover", "listar com filtros".
- Domínio não sabe como são armazenadas as coisas; persistência não decide regras.

## 5. Camada de Processamento Assíncrono

### Componentes
- **Agendador (scheduler)**: configurado com cron-like expressions. Dispara jobs em intervalos.
- **Fila**: recebe mensagens a processar. Suporta prioridade, delay, retry, dead letter.
- **Worker**: consome fila e executa handler. Escala horizontalmente. Stateless.
- **Event bus**: propaga eventos de domínio a múltiplos consumidores independentes.

### Tipos de processamento
- **Jobs recorrentes**: agendados (ex.: a cada minuto, a cada 5 minutos, diário 2h). Exemplos: processar fila de webhook delivery, processar execuções de workflow em espera, processar follow-ups agendados, refresh de tokens Meta, cálculo de lead score em lote.
- **Jobs disparados por evento**: consumidos do event bus (ex.: lead_created → start workflows aplicáveis → calcular score inicial).
- **Jobs pontuais enfileirados**: submetidos por um caso de uso (ex.: gerar embedding para FAQ nova, sincronizar calendário).

### Garantias
- **At-least-once delivery**: mensagem pode ser entregue mais de uma vez → handler deve ser idempotente.
- **Dead letter após N retries** (típico: 5 com backoff exponencial).
- **Visibilidade**: job em progresso bloqueia retentativa até timeout de visibilidade.
- **Observabilidade**: cada processamento loga início, fim, resultado, duração, input hash (não input completo se sensível).

### Contrato com domínio
- Worker recebe payload estruturado.
- Hidrata contexto de organização.
- Invoca caso de uso de domínio.
- Emite eventos se o caso de uso emitir.
- Ack em caso de sucesso. Nack/retry em caso de erro transiente. Dead letter em erro permanente.

## Fluxo de uma requisição típica

1. Usuário clica em "Criar lead" na UI.
2. UI valida form localmente, envia POST `/leads` com token.
3. Aplicação autentica, extrai `organization_id`, valida schema.
4. Aplicação chama caso de uso `CreateLead` do domínio.
5. Domínio valida invariantes (ex.: nome obrigatório, telefone válido), cria entidade Lead, emite evento `LeadCreated`.
6. Aplicação persiste em transação: lead, entrada em pipeline default, tags iniciais.
7. Aplicação publica `LeadCreated` no event bus.
8. Workers consumidores reagem: engine de workflow verifica se há workflow com trigger `lead_created`, calculador de score agenda cálculo, notificador pode enviar notificação ao responsável.
9. Aplicação responde 201 ao cliente com o lead criado.
10. UI atualiza cache; subscription entrega evento para outras abas/outros usuários da mesma organização que estão olhando para a lista.

Todo este fluxo ocorre em milissegundos de caminho síncrono. Workers rodam em background sem bloquear a resposta.
