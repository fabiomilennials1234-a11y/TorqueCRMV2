---
tipo: dominio
entidade: Lead
---

# Lead

Entidade central do Torque. Toda operação de negócio gira em torno do Lead.

## Definição

Pessoa física ou jurídica identificada como oportunidade comercial para a organização. Entra no sistema por qualquer origem (webhook externo, formulário, cadastro manual, conversa inbound), percorre pipelines, recebe interações de humanos e agentes IA, e termina em estado positivo (vendido, cliente) ou negativo (perdido, descarte).

## Atributos Conceituais

### Identificação
- `id`: identificador único (UUID).
- `organization_id`: tenant.
- `external_id`: identificador externo para dedupe entre ingestões (opcional, único dentro da org quando presente).

### Dados Pessoais / Comerciais
- `name`: nome (obrigatório, mínimo 2 chars).
- `company`: nome da empresa do lead (opcional em B2C, forte em B2B).
- `phone`: telefone em formato E.164 normalizado. Ao menos um de phone/email obrigatório.
- `email`: email validado por formato. Ao menos um de phone/email obrigatório.
- `position`: cargo (opcional).
- `cnpj_cpf`: documento (opcional, validado quando informado).

### Atribuição
- `responsible_id`: responsável geral (membro).
- `sdr_id`: SDR atribuído.
- `closer_id`: closer atribuído.
- As três referências podem apontar para o mesmo membro ou para membros diferentes. Todas opcionais inicialmente.

### Classificação
- `rating`: 1 a 5 (manual). Representa percepção humana da qualidade.
- `qualification_score`: 0 a 100 (automático). Calculado por engine de scoring (ver [[04 - Funcionalidades/IA/Lead Score]]).
- `segment`: rótulo livre de segmentação (opcional, além de tags).
- `origin`: origem do lead (enum livre). Exemplos: `meta_ads`, `google_ads`, `organic`, `referral`, `whatsapp_inbound`, `manual`, `outbound`, `form_site`.

### UTM (rastreamento de atribuição)
- `utm_source`, `utm_medium`, `utm_campaign`, `utm_term`, `utm_content`.
- Todos opcionais. Preenchidos na ingestão se origem trouxer.

### Campos Customizados
- `custom_fields`: mapa chave-valor configurável pela organização (ex.: "faturamento_declarado", "preferencia_contato", "evento_origem"). Schema dinâmico.

### Timestamps
- `created_at`: criação.
- `updated_at`: última modificação.
- `last_interaction_at`: última mensagem ou ação.
- `first_response_at`: timestamp da primeira resposta do lead (métrica de engajamento).

### Metadados
- `tags`: relação N:N (lista de tag ids).
- `active_pipelines`: lista derivada de Pipeline Entries ativos.

## Invariantes

1. Nome obrigatório e não-vazio após trim.
2. Ao menos um de `phone` ou `email` preenchido.
3. `phone` normalizado para E.164 (ex.: `+5511987654321`) antes de gravar.
4. `email` lowercased e validado por regex.
5. `qualification_score` em `[0, 100]`.
6. `rating` em `[1, 5]`.
7. `organization_id` imutável após criação.
8. Lead não pode ser reatribuído para outra organização (criar novo em vez disso).
9. Deleção é soft-delete por default; hard-delete apenas por admin com confirmação.

## Estados Derivados

Um Lead não tem "status" unificado. O status é derivado dos pipeline entries ativos:

- **Novo** (sinônimo): entrou no sistema, ainda não abordado.
- **Em qualificação**: está em pipeline WhatsApp em stage inicial.
- **Em reunião**: está em pipeline Confirmação.
- **Em proposta**: está em pipeline Propostas.
- **Vendido**: chegou em stage final positivo de Propostas.
- **Perdido**: chegou em stage final negativo.
- **Descartado**: marcado manualmente como não-oportunidade.

Derivação:
```
if lead has pipeline_entry in any pipe with stage.is_final == "positivo":
    status = "vendido"
elif lead has pipeline_entry with stage.is_final == "negativo":
    status = "perdido" (se não houver entry ativo positivo)
elif lead has active entry in pipe Propostas:
    status = "em_proposta"
elif lead has active entry in pipe Confirmacao:
    status = "em_reuniao"
elif lead has active entry in pipe WhatsApp:
    status = "em_qualificacao"
else:
    status = "novo"
```

## Operações

### CreateLead (caso de uso)
1. Validar input (nome, phone ou email, organization_id).
2. Normalizar phone e email.
3. Dedupe: se `external_id` informado e já existe → retornar lead existente (ou atualizar conforme flag).
4. Dedupe por phone/email: se `update_existing_if_match` ativo e há match → atualizar.
5. Criar Lead com `id`, `created_at`.
6. Aplicar tags iniciais.
7. Criar Pipeline Entry default (tipicamente em pipeline WhatsApp stage `novo`).
8. Emitir evento `LeadCreated`.
9. Retornar Lead completo.

### UpdateLead
1. Validar permissão (membro responsável, admin, ou quem tem ação `lead.update`).
2. Validar invariantes nos campos alterados.
3. Gravar Lead History: action=`updated`, before/after dos campos mudados.
4. Persistir.
5. Emitir evento `LeadUpdated` com campos mudados.

### AssignLead (atribuir responsável/SDR/closer)
1. Validar permissão e que o membro alvo pertence à org.
2. Atualizar Lead.
3. Registrar Lead History.
4. Emitir evento `LeadAssigned`.
5. Notificar membro alvo.

### AddTag / RemoveTag
1. Validar que tag pertence à org.
2. Criar/remover associação.
3. Emitir `LeadTagAdded` / `LeadTagRemoved`.
4. Pode disparar workflows subscritos a esses eventos.

### DeleteLead (soft)
1. Validar permissão (admin only por default).
2. Marcar `deleted_at`.
3. Cancelar todas execuções de workflow desse lead.
4. Arquivar conversas.
5. Emitir `LeadDeleted`.

## Eventos Emitidos

- `LeadCreated`
- `LeadUpdated` (com delta de campos)
- `LeadAssigned` (responsible/sdr/closer)
- `LeadTagAdded`, `LeadTagRemoved`
- `LeadStageChanged` (quando muda de stage em algum pipe)
- `LeadEnteredPipe` (quando entra em novo pipe)
- `LeadLeftPipe` (quando finaliza em um pipe)
- `LeadScoreRecalculated`
- `LeadDeleted` (soft)
- `LeadMessageReceived` / `LeadMessageSent` (passam por conversa, duplicados para conveniência)

## Fluxos de Ingestão

Ver [[06 - Fluxos End-to-End/Ingestão de Leads (Webhook)]] para fluxo completo. Resumo:

1. **Webhook externo**: `/webhooks/leads` recebe payload normalizado, autentica via org key, valida, cria.
2. **Orquestrador externo (n8n)**: monta payload e POST para o webhook acima.
3. **Formulário público**: página hospedada, captura → chama endpoint de aplicação autenticado por token público da org.
4. **Cadastro manual**: UI de admin ou SDR.
5. **Conversa inbound**: mensagem nova em canal cujo número é desconhecido → cria lead automaticamente + conversa vinculada.

## Validações especiais

- Telefone inválido após normalização → rejeita se for o único meio de contato; aceita com warning se houver email.
- Email com domínio descartável (lista) → flag de suspeita, não rejeita.
- Nome idêntico a outro lead + telefone idêntico → aplica dedupe automático.
- Campo custom com valor maior que limite (ex.: 1000 chars) → rejeita.

## Fluxo de Dedupe

```
received input with external_id?
  yes → buscar por external_id. existe? atualizar ou retornar. não? criar.
  no  → buscar por (phone normalizado OR email).
         match? update_existing_if_match flag? sim → atualizar. não → criar duplicado (warning em log).
         no match → criar.
```

## Custom Fields (schema dinâmico)

- A organização define schema de custom fields em configurações: `{key, label, type, options, required}`.
- Tipos aceitos: `text`, `number`, `date`, `enum`, `boolean`, `long_text`.
- Ingestão aceita custom fields desconhecidos e os armazena — admin pode retroativamente promover a campo configurado.

## Relações com outras entidades

- **1:N** com Pipeline Entry.
- **1:N** com Conversa.
- **1:N** com Follow-up.
- **1:N** com Lead History.
- **N:N** com Tag.
- **N:1** com Time Member (três papéis).

## Regras de Negócio Derivadas

- **Lead duplicado** é inaceitável em condições normais — sempre preferir atualizar.
- **Lead sem nenhum pipeline entry** só existe transitoriamente; sistema tenta colocar em pipeline default ao criar.
- **Lead ganho** não pode "voltar" a estado anterior sem ação explícita + auditoria (ex.: reabrir venda).
- **Lead perdido** pode ser reabrir — cria novo entry em pipeline apropriado, mantém histórico.
- **Campo de contato alterado** invalida cache de normalização e dispara nova validação.

## Métricas de Lead (agregadas)

- Tempo médio em cada stage.
- Taxa de conversão entre stages.
- Origem → taxa de fechamento.
- SDR → taxa de qualificação.
- Closer → taxa de fechamento.
- Tempo da criação à primeira resposta.
- Tempo da criação ao fechamento (lead time de venda).
