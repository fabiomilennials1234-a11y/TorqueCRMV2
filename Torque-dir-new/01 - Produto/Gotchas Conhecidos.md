---
tipo: requisitos
---

# Gotchas Conhecidos

Armadilhas e pontos de atenção descobertos na operação do Torque. Quem reimplementar deve ter ciência.

## Canais de Mensagem

### Rate limit não-oficial do WhatsApp
- WhatsApp pode **banir números** por comportamento agressivo (envios em massa, muitos opt-outs).
- Sem limite oficial exposto; cautela obrigatória.
- Recomendação: ≤ 100 msg/h para número recém-conectado; escalar gradualmente.

### 24h window do Messenger
- Messenger permite mensagens livres apenas nas primeiras 24h após última interação do lead.
- Depois: só templates pré-aprovados (tagged messages).
- Cuidar em campanhas outbound via Messenger.

### Reconexão QR frequente
- Evolution API e similares perdem conexão eventualmente (número usado em WhatsApp Web fora do provider).
- Admin precisa re-escanear QR.
- Notificar proativamente quando detectado.

### Webhook duplicado
- Provedores podem enviar mesmo evento 2x (rede instável, retry deles).
- Dedupe obrigatório por `external_id` da mensagem.

## Pipelines

### Stage deletada com entries ativos
- Falha se não migrar entries.
- UI deve exigir escolha de destino.

### Lead em múltiplos pipes simultâneos
- Feature, não bug.
- Analytics precisa decidir: como apresentar? Convenção: contagem distinta (lead aparece 1x) e simultaneidade (em quantos pipes, média?).

### "Status" do lead é derivado
- Não há `lead.status` singular.
- UI calcula a partir dos entries.
- Workflows que query "leads status=X" precisam traduzir para query sobre entries.

## Agente IA

### Business context vazio = agente genérico
- Agente sem contexto fala em abstrações.
- Warning na ativação se vazio.
- Taxa de qualificação cai dramaticamente.

### Cota de LLM estourada em bursts
- Campanha grande pode saturar LLM quota em minutos.
- Controlar budget por org/dia.
- Alerta antecipado.

### Parse JSON inválido do LLM
- Modelos podem retornar JSON malformado ocasionalmente.
- Retry 1x com instrução reforçada.
- Se falha: pausar agente, alerta.

### LLM hallucinates action
- Modelo pode inventar stage inexistente ou tag não-permitida.
- Validador descarta actions inválidas silenciosamente.
- Log para análise.

### Tamanho do contexto
- Conversa com 500+ mensagens: contexto explode.
- Truncamento + resumo necessário.
- Pode perder informação histórica relevante.

## Workflows

### Loop acidental
- Workflow A dispara workflow B que dispara A.
- Contador de hops (max 10) bloqueia.
- Detectar no editor se possível.

### Delay longo (meses) + mudança de config
- Workflow pausado há 2 meses com `context` vem antigo.
- Lead pode ter mudado muito.
- Admin precisa ter ciência; UI avisa.

### Trigger cron em timezone errado
- Expression `0 9 * * *` interpretada em qual fuso?
- Default: fuso da org.
- DST pode quebrar timing.

### Execuções órfãs
- Worker crasha no meio → lock expira → outro worker pega.
- Duplicação de step evitada por idempotência.

## Ingestão

### Tags em múltiplos formatos
- n8n envia tags como string com vírgula, ou JSON string, ou array.
- Torque aceita todos.
- Normalização crítica.

### Telefone em formatos variados
- `(11) 9...`, `11 9...`, `+55 11 9...`.
- Torque normaliza para E.164.
- Detecção de país via heurística (assume BR se sem código).

### Dedupe vs Criação
- `update_existing_if_match=true` pode levar a atualização indesejada.
- Documentar bem no API Docs.
- Cliente deve decidir conscientemente.

## Permissões

### Last admin rebaixado
- Sistema deve bloquear rebaixar último admin.
- Cuidar de race condition (dois rebaixamentos simultâneos).

### Override persist após specialty change
- Admin muda specialty do membro → overrides continuam.
- Pode virar combinação confusa.
- UI deve mostrar claramente.

### Feature gating vs override
- Override concede mas feature não está no plano → ação negada.
- UI precisa deixar claro qual camada bloqueia.

## Integrações Externas

### Token OAuth expirado sem detecção
- Se cron de refresh falha silenciosamente, token vence.
- Cobertura: alerta em cada chamada externa que falha por `token_expired`.

### ERP mudança de schema
- TinyERP pode mudar contrato sem aviso.
- Adaptador versionado.
- Monitor de saúde detecta mudança.

### Meta rate limit
- Graph API tem rate limit por app + por usuário.
- Campanhas grandes podem bater.
- Backoff + retry.

## Analytics

### Queries lentas em período longo
- 1 ano de dados em org grande: 100k+ leads.
- Materialização necessária.
- Sem cache, frontend trava.

### Múltiplas moedas em agregação
- Se org tem vendas em BRL e USD, "total do mês" precisa converter.
- Taxa de conversão: do dia da venda ou do momento do relatório?
- Convenção: fixa do dia da venda; acurado.

### Timezone em agregações
- "Hoje" depende do fuso da org.
- Agregações em UTC e convertidas na apresentação podem ter edge cases (ex.: venda às 00:30 do dia 15 pode aparecer no dia 14 ou 15 dependendo do fuso).

## Quotas

### Quota reset em meio de ciclo
- Admin muda plano no dia 15.
- Quota do mês era 2000, agora 5000.
- Regra: upgrade aplica novo limite prorata ou zera? Documentar.

### Múltiplas quotas intertwined
- Mensagens/mês, agentes ativos, FAQs por agente.
- Uma feature pode estar bloqueada por várias quotas.
- UI mostra qual quota bloqueia.

## Realtime

### Handler recebe delta, não row completo
- Update de lead emite só campos mudados.
- UI precisa mesclar com cache.
- Dados aninhados (tags, responsible) podem ficar stale se não vierem.

### Debounce agrega múltiplas mudanças
- 2s de debounce.
- Mudanças rápidas mescladas em um render.
- Geralmente bom; em cenários raros pode perder intermediate.

## Segurança

### Impersonação acidental
- Master termina sessão de impersonação mas esqueceu?
- Banner forte + expiração automática são críticos.

### Secret em commit
- Nunca deve acontecer; scanner automatizado ajuda.
- Se acontece: rotacionar IMEDIATAMENTE + audit.

### PII em log
- Log de nível INFO não deve ter PII.
- DEBUG é ok em desenvolvimento, NUNCA produção.

## Performance

### N+1 queries
- Lista de leads com tags: 1 query para leads + N queries para tags de cada = lento.
- JOIN ou batch load obrigatório.

### Contagens em tempo real
- "Quantos leads em stage X?" em tempo real é caro.
- Cache + invalidação em evento.

### Índices
- Queries sem índice adequado: lentas em volume.
- Ver sessão de Performance em cada entity doc.

## Deploy

### Migration destrutiva
- Remover coluna sem backup: perda de dados.
- Sempre migração em fases (add → migrate data → remove).

### Cache stale após deploy
- Config em memória do worker: reiniciar após mudança de env.
- Mudança de feature flag: invalidação global necessária.

### Rollback de schema
- Migration complexa pode ser difícil de reverter.
- Plano de rollback documentado antes do deploy.

## LGPD / Privacidade

### Export
- Pedido do titular: exportar dados em formato legível.
- Inclui mensagens, lead_history, follow-ups associados ao titular.

### Exclusão
- Anonimizar (não deletar) para preservar integridade de audit.
- Dados estatísticos agregados preservados.

### Consentimento
- Lead em ingestão externa: consentimento foi dado na fonte (form de Meta, Trello, etc.).
- Opt-out respeitado.

## Observação

Esta lista vai crescer com a operação. Documentar novos gotchas conforme descobertos.
