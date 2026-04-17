---
tipo: fluxo
---

# Atendimento via Copilot

Fluxo detalhado de uma conversa atendida por agente IA, desde o inbound do lead até resposta + eventuais ações de domínio. Visão operacional de como o Copilot funciona na prática.

## Diagrama

```
Lead escreve mensagem em canal (WhatsApp)
      │
      ▼
Provedor de canal → webhook inbound
      │
      ▼
Backend persiste mensagem + atualiza conversa
      │
      ▼
Detecta: conversa tem agente ativo E não está em takeover?
   ├── Não → mensagem fica para atendente humano
   └── Sim ▼
           │
           ▼
   Inicia (ou adiciona a) Batch Window (8s)
           │
           ▼ (expira)
   Hidrata contexto:
     - Lead + pipeline + stage
     - Kanban Rule ativa
     - Últimas 10-30 mensagens
     - Embedding da(s) última(s) mensagem(ns)
     - Top-5 FAQs por similaridade
     - System prompt montado
           │
           ▼
   Chama LLM
           │
           ▼
   Resposta JSON: {message, actions}
           │
           ▼
   Valida:
     - JSON válido?
     - Actions permitidas?
   ├── Não → retry 1x; se falha, pausa + alerta
   └── Sim ▼
           │
           ▼
   Executa AI Actions em ordem
     - move_stage, add_tag, assign, create_followup, etc.
           │
           ▼
   Envia mensagem:
     - Se > max_length: Smart Split em chunks
     - Se TTS: gera áudio + envia
     - Simula delay entre chunks (2-4s)
           │
           ▼
   Persiste mensagem outbound
           │
           ▼
   Realtime publica para UI do time
           │
           ▼
   Aguarda próxima inbound do lead (ou pausa quando objetivo cumprido)
```

## Componentes em Jogo

- **Webhook inbound** do canal.
- **Deduplication** por external_id da mensagem.
- **Conversation + Message** entities.
- **Batch Window** (fila com debounce).
- **Copilot Agent** + Kanban Rule da stage atual.
- **Vector DB** para RAG.
- **LLM Provider** via adaptador.
- **Action Executor** que valida e aplica AI Actions.
- **Smart Split** para chunking.
- **TTS Provider** (opcional).
- **Canal** para envio outbound.
- **Audit log** granular.

## Detalhes Críticos

### Batch Window
- Timer de 8s (configurável por agente).
- Primeira mensagem do lead dispara timer.
- Cada mensagem subsequente dentro da janela: reseta ou acumula.
- Ao expirar: todas mensagens processadas em 1 chamada LLM.
- Objetivo: não responder a cada keystroke; humanizar.

### Human Takeover
- Detectado quando `MessageSent` vem de `sender_type=member` na conversa.
- Seta `conversation.human_takeover_until = now + 10min` (configurável).
- Novas mensagens inbound durante takeover: persistidas mas NÃO acionam agente.
- Takeover expira → agente retoma.
- UI mostra banner.
- Usuário pode "retomar agente" antes de expirar.

### Context Hydration
- Lead com todos os campos (nome, empresa, phone, custom fields).
- Entries ativos em pipes + stages + `last_moved_at`.
- Kanban Rule da stage atual: goal, behavior, allowed/forbidden actions.
- Últimas N mensagens da conversa (N=20 tipicamente).
- Resumo de mensagens antigas (se conversa > N).
- Business context do agente (texto).
- FAQs relevantes: top-5 por similaridade à última msg do lead.

### System Prompt
Montado dinamicamente com template (ver [[04 - Funcionalidades/IA/Copilot (Agentes IA)]]#system-prompt-geração-automática).

### Chamada LLM
- Timeout 30s.
- Retry 1x em erro transiente.
- Response format JSON.
- Máx tokens configurável.

### Action Execution
- Cada action é validada contra Kanban Rule.
- Actions não permitidas: descartadas + log.
- Actions permitidas: executadas em ordem.
- Falha de uma não impede as outras.

### Smart Split
- Texto > max_response_length: split em 1-3 parágrafos por chunk.
- Break em finais de frase/parágrafo.
- Delay 2-4s entre chunks (simula digitação).
- Máx N chunks (evita interminável).

### TTS
- Se agente tem `tts_config` e texto apropriado:
  - Gera áudio via TTS provider.
  - Upload storage.
  - Envia como mensagem tipo áudio.
- Alternativo: áudio pré-gerado da biblioteca.

### Envio
- Cada chunk é mensagem outbound individual.
- Via adaptador de canal.
- Status tracking: pending → sent → delivered → read.

## Cenários Especiais

### Lead pede humano
- Agente detecta via heurística ou LLM explicit ("quero falar com um atendente").
- Executa `pause_conversation` action.
- Conversa fica pendente atribuição; membro é notificado.

### Lead em stage sem Kanban Rule
- Fallback para rule genérica da pipe.
- Warning no log para admin configurar.

### Lead muda de assunto
- Agente detecta via `allowed_topics` / `forbidden_topics`.
- Se tópico proibido: redireciona ou pausa.

### Off-hours
- Se org tem janela de negócio e agente configurado para respeitar: agente agrega mensagens fora de horário, responde só no início do próximo horário (ou envia auto-resposta "Recebi, responderei em horário comercial").

### Lead muda idioma
- Agente detecta (LLM) e responde no idioma do lead (se permitido).

### Múltiplas mensagens rápidas
- Batch window agrupa.
- LLM vê todas como contexto unificado.

## Edge Cases Graves

- **LLM retorna JSON inválido mesmo após retry**: pausa agente, notifica admin, deixa mensagem pendente para humano.
- **Rate limit LLM**: backoff, pode atrasar resposta; UI mostra agente como "processando".
- **Action com referência inexistente** (stage id deletada): skip com log; envia só a mensagem.
- **Canal desconectado no momento do envio**: mensagem fica em pending, retry; UI alerta admin.
- **FAQ sem embedding** (recém-criada): exclui da busca RAG; agente funciona mas possivelmente menos informado.
- **Conversa com histórico gigante** (1000+ msgs): trunca para N mais recentes + resumo.
- **Orçamento LLM da org estourado**: agente pausa automático; admin alertado.

## Métricas do Fluxo

- Latência total (msg inbound → msg outbound): p50 < 15s.
- % conversas que exigem takeover humano.
- Custo LLM médio por conversa.
- Taxa de qualificação pelo agente.
- NPS implícito (avaliação automática das conversas).

## Observabilidade

- Cada passo do fluxo loga:
  - `conversation_id`, `message_in_id`, `message_out_ids`.
  - Latência por passo (batch wait, RAG, LLM, execução, envio).
  - Tokens, custo.
  - Actions executadas.
- Dashboard de health: agente em X% de takeover, latência acima de Y → alerta.

## Pitfalls de Config

- **Business context vazio**: agente genérico, conversão ruim.
- **FAQs escassas/irrelevantes**: RAG não ajuda.
- **Kanban Rules mal desenhadas**: agente não sabe o que fazer em cada stage.
- **Permissões amplas demais**: agente aplica tag errada, move stage inadequado.
- **Personalidade incoerente** (formal + energético=alto + casual): agente esquizofrênico.

## Otimizações

- **Prompt caching**: system prompt estável cacheado no provider → economia grande.
- **Model tiering**: FAQ simples com modelo mais barato; negociação com premium.
- **Pre-generated audios**: áudios comuns (cumprimento, despedida) pré-gerados → latência menor.
- **Continuous evaluation**: avaliador auto pontua; admin revisa abaixo de 3/5.
