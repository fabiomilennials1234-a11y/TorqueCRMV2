---
tipo: arquitetura
---

# Arquitetura Conceitual — Visão Geral

Esta seção descreve o **formato lógico** do sistema — as camadas, as responsabilidades, os padrões que atravessam features. Nenhuma tecnologia específica é citada: a intenção é que o sistema possa ser implementado em qualquer stack moderna.

## Camadas lógicas

O Torque se organiza em cinco camadas distintas:

### 1. Camada de Apresentação (Cliente)

Aplicação interativa que o usuário acessa via navegador (desktop e mobile). Responsabilidades:

- Renderizar UI dark-first, responsiva.
- Gerenciar estado de interface (seleção, filtros, modais, wizards).
- Manter cache de dados do servidor com invalidação granular.
- Assinar eventos em tempo real do backend (novo lead, mensagem nova, stage atualizado).
- Executar validações de formulário localmente (primeira camada, não a única).
- Autenticar via token de sessão e enviar em toda requisição.
- Detectar o escopo de organização a partir do token (nunca envia `organization_id` manualmente).

### 2. Camada de Aplicação (API / Orquestração)

Endpoints HTTP e funções de orquestração. Responsabilidades:

- Receber chamadas do cliente e de sistemas externos.
- Validar autenticação e autorização (incluindo isolamento de tenant).
- Executar casos de uso (criar lead, mover stage, enviar mensagem, disparar workflow).
- Coordenar acesso a múltiplas entidades em uma única transação quando necessário.
- Emitir eventos de domínio para consumo assíncrono.
- Responder com resultado estruturado e códigos de status apropriados.

### 3. Camada de Domínio (Regras de Negócio)

Núcleo do sistema. Independente de qualquer tecnologia:

- Entidades do domínio (Lead, Organização, Pipeline, Workflow, Agente IA, etc.) com invariantes.
- Máquinas de estado (stages, execução de workflow, conversação).
- Serviços de domínio (scoring, distribuição, quotas, permissões).
- Regras cross-entidade (ex.: ao mover lead para "vendido", calcular comissão).

### 4. Camada de Persistência

Armazenamento durável. Dividida em:

- **Store transacional**: entidades mutáveis do domínio (leads, organizações, pipelines, workflows).
- **Store append-only**: audit logs, mensagens, execuções de workflow, histórico.
- **Store vetorial**: FAQs embedadas para RAG.
- **Store de objetos**: áudios gerados por TTS, anexos, avatares.
- **Store de cache**: resultados agregados frequentes, sessões.

Isolamento por tenant é invariante obrigatório **na camada de persistência**, não apenas na aplicação.

### 5. Camada de Processamento Assíncrono

- **Agendador**: dispara jobs recorrentes em intervalos fixos (minutos).
- **Worker de fila**: consome tarefas, aplica retries, move para dead letter em caso de falha persistente.
- **Handler de webhook**: recebe eventos externos, normaliza, enfileira.
- **Event bus**: propaga eventos de domínio para múltiplos consumidores (workflow engine, analytics, agente IA, webhooks externos).

## Integrações externas

O sistema se comunica com serviços externos em dois sentidos:

- **Saída**: envia mensagens a canais de mensagem (WhatsApp, Messenger), consome modelo LLM, consome serviço de embeddings, gera áudios em TTS, faz chamadas ao ERP, ao calendário, ao provedor de pagamento.
- **Entrada**: recebe webhooks de canais (nova mensagem), de orquestradores externos (novo lead), de Meta (novo formulário preenchido), do provedor de pagamento (evento financeiro), do calendário (agendamento modificado).

Toda integração é abstraída atrás de um **adaptador** — trocar o provedor de WhatsApp, por exemplo, deve ser trocar um adaptador, não reescrever features.

## Diagrama lógico (texto)

```
                    ┌──────────────────────────┐
                    │      Usuário (Admin /    │
                    │       Membro / Lead)     │
                    └─────────────┬────────────┘
                                  │
                    ┌─────────────▼────────────┐
                    │  Camada de Apresentação  │
                    └─────────────┬────────────┘
                                  │ HTTPS (token)
                    ┌─────────────▼────────────┐
                    │   Camada de Aplicação    │
                    │    (endpoints, RPCs)     │
                    └───┬──────────┬──────┬────┘
                        │          │      │
              ┌─────────▼──┐ ┌─────▼──┐ ┌─▼────────────┐
              │  Domínio    │ │ Event │ │  Integrações │
              │ (regras +   │ │  Bus  │ │   externas   │
              │  entidades) │ └───┬───┘ └──────────────┘
              └──────┬──────┘     │
                     │            │
        ┌────────────▼───┐    ┌───▼─────────────────┐
        │  Persistência  │    │  Workers / Jobs /   │
        │ (transacional, │    │   Cron / Filas      │
        │  append, vetor)│    └─────────────────────┘
        └────────────────┘
```

## Padrões transversais aplicados

- **Isolamento multi-tenant** em todas as camadas (detalhado em [[Multi-tenancy]]).
- **Autenticação e autorização** centralizadas (ver [[03 - Identidade e Permissões/Autenticação]]).
- **Idempotência** em toda operação externa e em webhooks.
- **Observabilidade** por padrão em handlers, workers e integrações.
- **Segurança** por default — headers de segurança, validação estrita, secrets fora do código.
- **Tempo real** via subscriptions filtradas por organização.

Cada um desses padrões tem doc próprio em `01 - Arquitetura Conceitual/`.

## Fronteiras e responsabilidades

- **UI nunca** toma decisão de permissão — só reflete decisão do backend.
- **Cliente nunca** conhece secrets ou chaves de serviços externos.
- **Backend sempre** revalida o escopo de organização ao receber requisição.
- **Worker sempre** opera em nome de uma organização específica; não há workers "globais" sobre dados de tenants.
- **Domínio nunca** conhece detalhes de transporte (HTTP, WebSocket, fila) — recebe inputs, produz outputs e emite eventos.

## Deploy e operação (conceitual)

- **Aplicação cliente**: artefato estático distribuído (SPA) via CDN.
- **Aplicação backend**: função/container sem estado local; escala horizontalmente.
- **Persistência**: banco relacional com capacidade de índice vetorial, mais store de objetos.
- **Workers**: processos escaláveis que consomem de filas/agendador.
- **Agendador**: serviço que dispara jobs recorrentes.
- **Observabilidade**: coletor de logs estruturados + captura de exceções + métricas.

Nenhum desses componentes exige tecnologia específica. As escolhas atuais do Torque refletem preferência pela plataforma adotada, não requisitos de arquitetura.
