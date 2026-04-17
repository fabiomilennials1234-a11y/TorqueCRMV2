---
tipo: integracao
direcao: out
criticidade: baixa
---

# Text-to-Speech (TTS)

Serviço externo que gera **áudio a partir de texto**. Usado por agentes IA para enviar áudios humanizados aos leads no WhatsApp, trazendo naturalidade à conversa.

## Propósito

- Agente envia áudio gerado (voz humanizada) em vez de só texto.
- Lead percebe atendimento mais humano.
- Útil especialmente em público que prefere áudio (comum no Brasil via WhatsApp).

## Provedor

Atualmente ElevenLabs. Alternativas: Google Cloud TTS, Amazon Polly, Azure Speech, OpenAI TTS.

## Contrato

- Endpoint que recebe texto + voz + opções → retorna áudio (MP3/OGG).
- Suporta múltiplas vozes (características: idade, gênero, tom).
- Parâmetros: stability, similarity_boost, style (depende do provider).

## Autenticação

- API key do provider (por Torque, centralizado).
- Possivelmente por org no plano enterprise (org tem próprias vozes clonadas).

## Endpoints Consumidos

```
POST /text-to-speech/:voice_id
{
  "text": "Olá, tudo bem? Sou a assistente da empresa X...",
  "model_id": "eleven_multilingual_v2",
  "voice_settings": {
    "stability": 0.5,
    "similarity_boost": 0.75,
    "style": 0.3,
    "use_speaker_boost": true
  }
}
```

Resposta: binário MP3 (ou stream).

## Fluxo

1. Agente IA gera resposta e decide enviar áudio (via action `play_audio` com texto).
2. Sistema chama TTS: `generateAudio(text, voice_id, settings)`.
3. Áudio recebido em bytes.
4. Upload para storage de objetos.
5. Mensagem outbound criada com `content_type=audio`, `media_url` da storage.
6. Envio via canal de WhatsApp.

## Biblioteca de Áudios

- Agente pode ter **áudios pré-gerados** armazenados (ex.: cumprimento de abertura).
- Economia de chamadas TTS para frases repetidas.
- Admin gera no wizard + salva.

## Regras de Negócio

1. TTS é opcional — agente funciona só com texto.
2. Feature gated por plano (tier premium).
3. Tamanho de texto limitado (ex.: 2000 chars por chamada).
4. Idioma detectado automaticamente ou configurado na voice.
5. Voice padrão por org; admin pode ter múltiplas.
6. Áudio armazenado em storage com TTL (ex.: 30 dias) ou permanente conforme política.

## Edge Cases

- **Provider down**: fallback para texto (agente envia mensagem sem áudio).
- **Texto com caracteres especiais** (emoji, símbolo): sanitiza antes (TTS pode falhar).
- **Voz indisponível** (removida pelo provider): fallback para voz default.
- **Áudio muito longo**: WhatsApp tem limite (~16 min); split ou resumir.
- **Quota de TTS atingida**: feature desabilitada temporariamente; warning ao admin.

## Performance

- Latência: 1-5s para áudio de 30s (varia por provider).
- Pode adicionar perceptible delay à conversa → dicas: usar áudios pré-gerados para primeira resposta.

## Custo

- Por caractere ou minuto de áudio gerado.
- Mais caro que TTS simples (vozes neurais realistas).
- Tracking por org; alerta em uso anômalo.

## Segurança

- API key em vault.
- HTTPS.
- Áudios em storage com acesso assinado (URL temporária).
- Não armazenar áudios com PII sensível além do necessário.

## Observabilidade

- Log por chamada.
- Métricas: áudios/dia, duração média, custo.

## LGPD

- Consentimento: lead em WhatsApp aceita receber mensagens; áudio é forma de mensagem.
- Vozes clonadas de pessoas reais: requer consentimento formal do dono da voz.

## Evolução

- Voice cloning: org pode ter voz clonada do SDR humano (com consentimento).
- Mais vozes nativas PT-BR.
- Multi-idioma conforme lead.
