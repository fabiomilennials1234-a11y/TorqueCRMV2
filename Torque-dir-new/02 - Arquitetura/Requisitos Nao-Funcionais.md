---
tipo: arquitetura
---

# Requisitos Não-Funcionais

Definições quantitativas de qualidade que a implementação deve atingir. Estes números são **metas**, validadas continuamente.

## Performance

### Caminho crítico (síncrono)
- **Login**: p95 < 800 ms.
- **Listar leads da página atual** (até 50 cards): p95 < 500 ms.
- **Abrir kanban de pipe** (até 500 cards distribuídos em 5-8 colunas): p95 < 1.2 s.
- **Mover card entre stages**: p95 < 400 ms (do clique à confirmação).
- **Enviar mensagem no chat**: p95 < 600 ms do envio à confirmação de fila.
- **Criar lead manual**: p95 < 500 ms.
- **Dashboard com 90 dias**: p95 < 2 s.

### Caminho assíncrono
- **Webhook externo de ingestão → lead criado**: p95 < 3 s do POST ao lead listável.
- **Mensagem recebida em canal → visível no chat**: p95 < 2 s via realtime.
- **Agente IA responde ao lead**: 8 s (batch window) + p95 < 5 s de processamento LLM + envio. Total p95 < 15 s.
- **Workflow disparado por evento → primeira ação executada**: p95 < 10 s.
- **Dispatch de campanha → mensagem entregue**: depende de janela de negócio e rate limit.

## Escala

- **Organizações ativas**: projetar para 500 (atualmente ~30, margem para crescimento 15x sem reestruturação).
- **Usuários ativos simultâneos**: 5.000 em pico.
- **Leads por organização**: até 1.000.000 total, até 10.000 criados por mês na maior.
- **Mensagens por dia no sistema inteiro**: até 1.000.000.
- **Workflows ativos por organização**: até 200.
- **Agentes IA por organização**: até 10.
- **FAQs por agente**: até 500.

## Disponibilidade

- **Uptime meta**: 99.5% mensal (~3.6 h indisponível/mês tolerado). Caminho alvo 99.9% quando volume e criticidade justificarem.
- **RTO** (Recovery Time Objective): 1 hora após incidente crítico.
- **RPO** (Recovery Point Objective): 5 minutos — perda de dados aceitável em desastre extremo.
- **Backup**: diário com retenção de 30 dias. Backup semanal retido por 1 ano. Restore testado mensalmente.

## Latência de integrações externas

- Timeout padrão para chamada externa: 10 s.
- LLM: timeout 30 s, retry 2x com backoff.
- Envio em canal de mensagem: timeout 15 s, retry via fila.
- Calendário: timeout 20 s.
- Provedor de pagamento: timeout 30 s.

## Consistência

- Escritas em persistência transacional: consistência forte.
- Index de busca / vetorial: eventualmente consistente, lag < 30 s.
- Contadores agregados: lag < 10 s.
- Realtime subscriptions: lag < 3 s.

## Observabilidade

- 100% dos endpoints de aplicação emitem log estruturado com duração.
- 100% dos jobs assíncronos emitem log de início/fim + resultado.
- 100% das exceções em produção são capturadas com contexto.
- **Alertas**:
  - Taxa de erro 5xx > 1% em 5 min → alerta.
  - Fila com mais de N mensagens por mais de M minutos → alerta.
  - Worker parado → alerta.
  - Integração externa com > X% de falha → alerta.

## Segurança

- TLS 1.2+ em toda origem externa.
- Senhas: Argon2id ou bcrypt custo ≥ 12.
- Token de sessão: curta duração (15-60 min), renovável.
- Secrets em vault com rotação ≤ 90 dias.
- Patch de CVE crítico aplicado em < 7 dias.
- Varredura de dependências automatizada a cada merge.
- Pentest externo anual.

## Privacidade / Compliance

- LGPD (Brasil): conformidade plena.
  - Base legal de tratamento documentada.
  - Titular pode solicitar exportação e exclusão.
  - DPO designado.
- Dados sensíveis em trânsito sempre cifrados.
- Logs retêm no máximo 180 dias.
- Auditoria de acesso a dado pessoal por master admin.

## Manutenibilidade

- **Tempo para adicionar nova integração**: < 2 semanas com um engenheiro sênior.
- **Tempo para adicionar novo tipo de node em workflow**: < 3 dias.
- **Cobertura de testes**: meta 70% em código crítico (domínio, engines, integrações).
- **Dependências**: atualizadas mensalmente em segurança, trimestralmente em geral.
- **Deploy**: automatizado via pipeline. Zero-downtime. Rollback em < 10 min.

## Usabilidade

- **Onboarding**: admin novo chega ao primeiro valor em < 15 minutos.
- **Help inline** em todos os wizards complexos.
- **Mensagens de erro** acionáveis: dizem o que deu errado e o que o usuário pode fazer.
- **Mobile**: uso essencial (ver kanban, responder chat, criar lead) funcional em telas pequenas.

## Acessibilidade

- WCAG 2.1 AA em todas as telas.
- Teste com leitor de tela em features críticas.
- Atalhos de teclado documentados em `?`.

## Internacionalização

- Texto separado do código (chaves de tradução).
- Formatos regionais (moeda, data, telefone) configuráveis por organização.
- Fuso horário por organização.

## Custos

- **Margem operacional** por organização positiva mesmo no plano mais barato, considerando custos de LLM, armazenamento, mensagens, infra.
- Mecanismos de throttle para prevenir abuso em plano free/baixo (se existir).
- Monitoramento de consumo por organização (telemetria de custo) para evoluir planos.

## Critérios de aprovação para release

Feature nova não entra em produção sem:

1. Spec funcional aprovada.
2. Testes unitários e de integração passando.
3. Validação de segurança (revisão de permissões, secrets, input).
4. Métricas de performance coletadas em staging.
5. Documentação atualizada (este vault + CLAUDE.md).
6. Plano de rollback definido.
7. Monitor / alerta configurados para a feature.
