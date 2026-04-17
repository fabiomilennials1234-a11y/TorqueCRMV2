---
tipo: feature
dominio: comunicacao
---

# Chat Multi-canal

## Propósito

Central unificada de atendimento que exibe **conversas de todos os canais** em uma interface única: inbox com lista de conversas + thread da conversa selecionada + envio de mensagens. Substitui o WhatsApp Web e plataformas similares com visão integrada ao CRM (lead, pipeline, histórico).

## Atores e Permissões

- **SDR/Closer/Prospectador**: atende conversas atribuídas ou em aberto; envia, recebe, anexa mídia.
- **Admin**: vê todas as conversas da org; pode atribuir/arquivar.
- **Agente IA**: atua em conversas configuradas; pausa quando humano envia (takeover).

Ações: `conversation.view`, `conversation.send_message`, `conversation.take_over`, `conversation.assign`, `conversation.archive`.

## Dados Envolvidos

Ver [[02 - Modelo de Domínio/Conversa e Mensagem]]. Resumo:

- Conversa: lead, canal, instância, status, agente, takeover_until, unread_count, assigned_to.
- Mensagem: direção, sender_type, content_type, content, media, status, timestamp.
- Notas internas (invisíveis ao lead).

## Canais Suportados

- Canal WhatsApp (principal via provedor primário).
- Canal WhatsApp (alternativo — segundo provedor para redundância ou multi-número).
- Canal Messenger (Meta).
- Canais futuros: SMS, email, web chat embed.

Cada canal tem uma abstração de adaptador → UI trata igual.

## Fluxos do Usuário

### Inbox (lista de conversas)
1. Menu → `Chat`.
2. Lista de conversas ordenada por `last_message_at DESC`.
3. Cada item mostra: avatar do lead, nome, preview da última msg, tempo, badge de canal, contador não-lida, status (agente/humano/arquivada).
4. Filtros:
   - Por canal.
   - Atribuído a mim / todos (admin).
   - Não-lidas.
   - Apenas com agente IA.
   - Apenas takeover humano.
   - Período.
5. Busca: texto em mensagens (server-side, full-text).

### Thread da Conversa
1. Clica na conversa → abre área de mensagens à direita.
2. Exibe histórico paginado, mensagens mais novas embaixo.
3. Status de cada mensagem: pending/sent/delivered/read/failed (ícones).
4. Separação visual entre inbound e outbound.
5. Mostra quem enviou (membro X / agente / lead / sistema).
6. Header:
   - Nome + empresa do lead.
   - Canal ativo.
   - Botão "Abrir drawer do lead" (tudo sobre o lead).
   - Botão "Assumir" (takeover do agente).
   - Botão "Atribuir a" (reatribuir conversa).
   - Status do agente (ativo / pausado por takeover).

### Enviar Mensagem
1. Campo de texto embaixo.
2. Atalhos:
   - `/template` para inserir template.
   - Emoji picker.
   - Anexar mídia (imagem, vídeo, áudio, documento).
   - Gravar áudio in-app.
3. Enter envia; Shift+Enter nova linha.
4. Mensagem aparece otimista (status=pending), atualiza para sent em seg.
5. Agente IA (se ativo): takeover_until set automaticamente.

### Receber Mensagem (inbound)
1. Lead escreve no canal externo.
2. Provedor envia webhook → sistema cria mensagem → publica realtime.
3. UI do atendente: inbox atualiza (topo), badge de não-lida incrementa, som/notificação.
4. Se conversa aberta, mensagem aparece inline.

### Takeover Humano
- Ao enviar uma mensagem em conversa com agente ativo → `human_takeover_until = now + 10min`.
- UI mostra banner "Humano no controle até HH:MM".
- Agente não responde; novas mensagens do lead chegam mas ficam para humano.
- Expira automaticamente → agente retoma.
- Usuário pode clicar "Retomar agente" para encerrar takeover antes.

### Anexar Mídia
1. Click no ícone ou arrasta arquivo.
2. Preview antes de enviar.
3. Upload para storage → gera URL + metadata.
4. Envia como mensagem do tipo adequado (image, document, etc.).
5. Limites por canal respeitados.

### Áudio
1. Clica ícone de microfone → grava.
2. Preview → envia.
3. Áudio é mensagem tipo `audio` com waveform.
4. Agente IA (se ativo com TTS) pode enviar áudios gerados.

### Notas Internas
- Toggle "Nota interna" ao compor mensagem.
- Aparece na thread com fundo diferente (ex.: amarelo) e tag "Interno".
- NUNCA enviada ao lead.
- Ver [[Notas Internas]].

### Atribuir Conversa
1. Botão "Atribuir a" → lista de membros.
2. Select → conversa marcada com `assigned_to`.
3. Membro é notificado.

### Arquivar
- Botão "Arquivar" na conversa.
- Move status=`archived`.
- Some do inbox default; acessível por filtro "Arquivadas".

### Busca
- Campo no topo.
- Busca full-text em mensagens da org.
- Respeita escopo (membro só vê suas conversas; admin vê tudo).

## Automações e Eventos

### Emite
- `MessageSent`, `MessageReceived`, `MessageFailed`, `MessageRead`.
- `HumanTakeoverStarted`, `HumanTakeoverExpired`.
- `ConversationAssigned`, `ConversationArchived`.

### Reage
- Webhook de canal: `MessageReceived`.
- Workflow action `send_message`: dispara envio.
- Agente IA: consome `MessageReceived` após batch window.

## Integrações

- Adaptadores de canal (ver [[05 - Integrações Externas]]).
- Agentes IA (copilot).
- Templates de mensagem.
- Lead e Pipeline Entries.
- Storage de mídia.
- Busca full-text.
- Notificações push.

## Edge Cases

- **Canal desconectado**: envio falha, mensagem fica em `pending` e UI mostra erro. Admin é alertado no dashboard de integrações.
- **Lead trocou número**: mensagem enviada ao número antigo falha; UI pede confirmação do número novo.
- **Mensagem muito grande**: chunked automaticamente pelo canal; split natural.
- **Mídia muito grande**: rejeitada no upload com mensagem clara de limite.
- **Múltiplos atendentes na mesma conversa**: sistema não bloqueia, mas exibe "X está digitando..." para evitar sobreposição. Última mensagem enviada vence.
- **Offline**: UI queue mensagens; envia quando reconecta.
- **Mensagem editada no provedor** (quando suportado): atualização reflete em UI (mantém versão antiga em audit).
- **Reactions / emojis do WhatsApp**: capturadas como eventos separados.

## Validações

- Texto: ≤ limite do canal (tipicamente 4000 chars).
- Mídia: tipo MIME e tamanho conforme canal.
- Template: placeholders resolvidos antes do envio.
- Canal ativo para a org.

## Métricas

- Mensagens enviadas/recebidas por período, por canal, por membro/agente.
- Tempo médio de primeira resposta.
- Tempo médio entre mensagens (qualidade de atendimento).
- Takeover rate (quantas conversas exigiram humano).
- Falha de entrega por canal.
- Satisfação (se pesquisa enviada ao lead).

## Performance

- Inbox: paginado; realtime atualiza top automaticamente.
- Thread: infinite scroll para histórico antigo.
- Busca full-text indexada server-side.
- Cache de conversas ativas no cliente.
- Debounce em "digitando..." para evitar flood.

## UX

- Dark-first.
- Distinção clara: humano vs agente vs lead.
- Badges de status (enviado/entregue/lido/falhou) precisos.
- Atalhos de teclado (j/k navegar, r responder, e arquivar).
- Mobile: lista e thread são telas separadas; navegação fluida.
