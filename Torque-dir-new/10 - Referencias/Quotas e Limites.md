---
tipo: referencia
---

# Quotas e Limites

Limites numéricos enforcing no sistema. Valores abaixo são orientações — implementação ajusta conforme plano real.

## Por Plano (exemplo)

| Recurso | Starter | Pro | Business | Enterprise |
|---|---:|---:|---:|---:|
| Usuários ativos | 3 | 10 | 30 | Ilimitado |
| Leads novos / mês | 500 | 2.000 | 10.000 | Ilimitado |
| Mensagens / mês (outbound) | 1.000 | 5.000 | 30.000 | Ilimitado |
| Agentes IA ativos | 0 | 1 | 3 | 10 |
| FAQs / agente | — | 50 | 200 | Ilimitado |
| Pipelines customizados | 0 | 3 | 15 | Ilimitado |
| Workflows ativos | 5 | 20 | 100 | Ilimitado |
| Campanhas ativas simultâneas | 1 | 3 | 10 | Ilimitado |
| Webhook endpoints | 1 | 5 | 20 | Ilimitado |
| API keys | 1 | 3 | 10 | Ilimitado |
| Storage de mídia | 1 GB | 10 GB | 100 GB | Ilimitado |
| Retenção de mensagens | 30d | 90d | 1 ano | 5 anos |
| Retenção audit | 30d | 1 ano | 1 ano | 7 anos |
| Instâncias de canal | 1 | 3 | 10 | Ilimitado |

## Limites Técnicos (Todas as Orgs)

### Tamanho de campos
- Nome de lead: 200 chars.
- Descrição livre: 2.000 chars.
- Conteúdo de mensagem texto: 4.000 chars (limite do canal WhatsApp ~65k mas recomendado split em 1000).
- Template content: 4.000 chars.
- Business context de agente: 10.000 chars.
- FAQ question + answer: 500 + 2.000 chars.
- Custom field value: conforme type (number: 64-bit, text: 2.000 chars, long_text: 20.000 chars).
- Tag name: 50 chars.
- Bio de membro: 500 chars.

### Tamanho de coleções
- Tags por lead: 50 (soft limit; hard: 100).
- Items por proposta: 100.
- Stages por pipe: 30.
- Nodes por workflow: 200.
- Edges por workflow: 500.
- Members por org: conforme plano.
- Custom fields configurados por org: 50 (soft limit).

### Tamanho de upload
- Avatar: 2 MB.
- Logo: 2 MB.
- Mídia em mensagem: limite do canal (WhatsApp: 16 MB vídeo, 100 MB documento).
- Áudio TTS: 10 MB.

### Volume e Rate
- Ingestão /min por org: 100 req/min default; plano enterprise ajusta.
- Webhook deliveries /s por endpoint: 10 (anti-flood).
- API geral /min por key: 100 default.
- Mensagens outbound /h por instância canal: 100 default (alinhado WhatsApp).

### Tempo
- Token de sessão: 60 min.
- Refresh token: 14 dias.
- Token de reset de senha: 1h, single-use.
- Invite token: 7 dias.
- Master impersonation: 1h.
- Workflow execution max duration: 7 dias.
- Delay máximo em workflow: 30 dias (soft limit).

### Histórico
- Versões de workflow retidas: 20.
- Versões de template retidas: 10.
- Lead history: append-only, limitado por retenção de plano.
- Mensagens de conversa: append-only, limitado por retenção.

## Rate Limits em Integrações

### LLM
- Tokens/mês por plano.
- Budget R$/mês por plano.
- Ao cruzar 80%: alerta.
- Ao cruzar 100%: agentes pausam.

### Canal de Mensagem
- Respeita limite do provider (varia).
- Rate limit interno adicional por reputação.

### Embeddings
- FAQs/mês limite implícito (cada FAQ gera 1-2 embeddings).

## Reset de Quotas

- Diárias: reset a 00:00 no fuso da org.
- Mensais: reset no dia 1 do mês no fuso da org, ou na data do anniversary da assinatura (config).
- Anuais: aniversário da assinatura.

## Over-quota Behavior

Ao atingir 100% de quota:
- Operação bloqueada com erro tipado (`quota_exceeded`).
- UI mostra upsell explícito.
- Admin recebe email.
- Some quotas não bloqueiam, só alertam (ex.: retenção — dados antigos ficam mas são arquivados).

## Upgrade/Downgrade Impact

- Upgrade: quota nova aplicada imediatamente.
- Downgrade: se estado atual excede nova quota, operações de criar bloqueiam; reduções requeridas antes.

## Exceções e Overrides

- Master pode conceder override de quota a org específica (negociação comercial).
- Override tem `granted_by`, `reason`, `expires_at` opcional.

## Auditoria

- Cada quota_exceeded é logada.
- Alertas em volumes anômalos (sinal de abuso ou ataque).

## Monitoramento

- Dashboard: % de orgs em cada plano próximas do limite.
- Proactive upsell em org com uso crescente.
- Alerta interno em orgs que estouram frequentemente.

## Exemplos de Enforcement

### Criar lead com quota estourada
```
POST /webhooks/leads
→ 403 Forbidden
{
  "error": {
    "code": "quota_exceeded",
    "message": "Limite mensal de 2.000 leads atingido no plano Pro. Faça upgrade ou aguarde próximo ciclo.",
    "details": {
      "resource": "leads",
      "current": 2000,
      "limit": 2000,
      "resets_at": "2026-05-01T00:00:00-03:00"
    }
  }
}
```

### Ativar agente IA acima do limite
```
UI: botão "Ativar" exibe tooltip "Seu plano suporta até 1 agente. Faça upgrade."
Backend: mesmo 403 com code="quota_exceeded".
```

## Sintonia Fina

Admin pode ver em `Configurações → Uso`:
- Consumo atual × limite por recurso.
- Projeção (ritmo atual × dias restantes).
- Histórico por mês.
- Comparativo com mês anterior.
- CTA para upgrade quando próximo.
