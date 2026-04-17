---
tipo: feature
dominio: admin
---

# Checkout e Planos

## Propósito

Fluxo de **assinatura e cobrança recorrente** do produto. Lead interessado no Torque passa por checkout, assina plano, provisiona organização, mantém-se pagando mensalmente via provedor de pagamento externo. Admin pode fazer upgrade/downgrade ao longo do tempo.

## Atores e Permissões

- **Lead externo**: acessa landing e checkout.
- **Admin de org existente**: altera plano.
- **Master**: gerencia planos globais.
- **Provedor de pagamento**: envia webhooks de eventos financeiros.

Ações: `plan.view`, `plan.change`, `billing.view`, `billing.update_payment_method`.

## Catálogo de Planos (exemplo)

| Plano | Preço/mês | Usuários | Leads/mês | Agentes IA | FAQs/Agente | Features extras |
|---|---:|---:|---:|---:|---:|---|
| Starter | R$ 99 | 3 | 500 | 0 | — | Pipes estruturais, chat básico |
| Pro | R$ 299 | 10 | 2.000 | 1 | 50 | Workflow builder, analytics avançado |
| Business | R$ 799 | 30 | 10.000 | 3 | 200 | Custom pipes, API pública, integrações |
| Enterprise | Sob consulta | Ilimitado | Ilimitado | 10 | Ilimitado | SLA, dedicated support, multi-org |

## Dados Envolvidos

### Plano (global)
- `id`, `name`, `description`.
- `price_monthly`, `price_yearly` (desconto anual).
- `currency`.
- `features`: array de flags.
- `limits`: JSON com quotas.
- `is_public` (exibe em checkout).
- `is_active`.

### Assinatura
- `id`, `organization_id`, `plan_id`.
- `status`: `active` | `past_due` | `cancelled` | `trialing`.
- `started_at`, `next_billing_at`, `cancelled_at`.
- `payment_method_id` (ref no provedor).
- `trial_until`.
- `external_subscription_id` (no provedor).

### Fatura
- `id`, `subscription_id`.
- `amount`, `currency`.
- `due_at`, `paid_at`.
- `status`: `open` | `paid` | `failed` | `refunded`.
- `provider_invoice_id`.

## Fluxo de Checkout (Novo Cliente)

### 1. Página de Planos
- Cliente navega.
- Compara planos, escolhe Pro.
- Click "Assinar".

### 2. Cadastro
- Form: email, nome empresa, CNPJ (opcional), telefone.
- Validação.

### 3. Método de Pagamento
- Cartão de crédito (tokenização via provedor; Torque nunca vê número).
- Pix recorrente (onde suportado).
- Boleto (para planos anuais).

### 4. Confirmação
- Review do pedido.
- Trial opcional (ex.: 14 dias grátis).
- Aceite de ToS.
- Confirma → processo de cobrança inicia.

### 5. Provisionamento
- Webhook do provedor confirma cobrança (ou autoriza trial).
- Sistema:
  - Cria organização.
  - Cria admin inicial (usuário do email).
  - Envia email com credenciais/link de ativação.
  - Dispara `OrganizationProvisioned`.
- Admin acessa e faz onboarding.

## Fluxo de Cobrança Recorrente

### Renovação Mensal
1. Cron de cobrança no provedor.
2. Provedor tenta cobrar → sucesso → webhook `payment_succeeded`.
3. Sistema atualiza `next_billing_at`.
4. Fatura marcada `paid`.

### Falha de Pagamento
1. Cobrança falhou (cartão expirado, sem saldo).
2. Webhook `payment_failed`.
3. Sistema marca assinatura `past_due`.
4. Email ao admin pedindo atualizar método.
5. Retry automático pelo provedor (tipicamente 3x em 7 dias).
6. Se ainda falha: `cancelled` → org suspensa (ver [[02 - Modelo de Domínio/Organização]]#ciclo-de-vida).

### Reativação
1. Admin atualiza método em Configurações → Billing.
2. Cobrança imediata do pendente.
3. Sucesso → `active` novamente.

## Upgrade/Downgrade

### Upgrade
- Admin clica "Upgrade para Business".
- Confirma pagamento pro-rata da diferença.
- Mudança imediata — features habilitadas, quotas aumentadas.
- Próxima fatura no valor do novo plano.

### Downgrade
- Admin clica "Downgrade para Starter".
- Sistema valida: consumo atual não excede novo limite.
- Se excede: força admin a reduzir (desativar agentes, remover membros).
- Mudança imediata ou no fim do ciclo (config).
- Diferença de valor não reembolsada (regra do provedor).

### Cancelamento
- Admin cancela em Billing.
- Confirmação dupla + pesquisa de motivo.
- `status=cancelled`, `cancelled_at` preenchido.
- Acesso mantido até `next_billing_at` (fim do ciclo pago).
- Após: read-only, depois suspenso, depois carência, depois hard-delete.

## Regras de Negócio

1. Admin só pode mudar o próprio plano da org.
2. Trial: 14 dias padrão; admin pode converter ou deixa expirar.
3. Organização sem método de pagamento após trial → suspende.
4. Refund parcial: política configurável — default sem refund.
5. Plano Enterprise: negociado manualmente; master cria assinatura direto sem passar checkout.
6. Impostos (ISS, IOF): calculados pelo provedor ou agregados ao valor apresentado.
7. Mudança de moeda: só em assinatura nova (não altera existente).

## Fluxos do Usuário

### Ver Plano Atual
- Admin → Billing.
- Mostra: plano, próximo vencimento, forma de pagamento, histórico de faturas.

### Atualizar Método de Pagamento
- Click "Alterar cartão" → formulário seguro via provedor.
- Token persistido; cartão novo usado na próxima cobrança.

### Baixar Fatura
- Lista de faturas → PDF por click.

## Automações

### Emite
- `SubscriptionCreated`, `SubscriptionActivated`, `SubscriptionCancelled`.
- `InvoicePaid`, `InvoiceFailed`.
- `PlanUpgraded`, `PlanDowngraded`.

### Reage
- Webhooks do provedor → atualiza estado.
- Cron diário: avalia assinaturas `past_due` > X dias → suspende.

## Integrações

- **Provedor de pagamento** (ver [[05 - Integrações Externas/Asaas (Provedor de Pagamento)]]).
- **Organização**: estado atualizado conforme assinatura.
- **Email**: notificações de cobrança, lembretes, falhas.
- **Analytics internos** (master): MRR, churn, etc.

## Edge Cases

- **Cartão expirado**: detectado antecipadamente via API do provedor se possível; email de aviso 7 dias antes.
- **Chargeback**: webhook → suspende imediatamente + audit.
- **Dupla cobrança** (bug): provedor tem dedupe; caso passe, refund manual.
- **Assinatura cancelada via telefone/email**: master faz manualmente.
- **Cliente deixou de pagar por motivo técnico**: master pode reativar com confiança e arranjar cobrança posteriormente.
- **Conversão de moeda**: não alterar em assinatura ativa (confusão).

## Validações

- Email válido em checkout.
- Método de pagamento tokenizado com sucesso.
- ToS aceito.
- Plano ativo e `is_public` para checkout self-service.

## Métricas

- MRR (Monthly Recurring Revenue).
- Churn rate.
- Expansion (upgrades).
- Contraction (downgrades).
- Average revenue per account (ARPA).
- Conversion rate (checkout iniciado → completado).
- Tempo médio no trial antes de converter.
- Failed payments (% de assinaturas).

## Segurança

- Nunca armazenar número de cartão localmente.
- PCI compliance via provedor.
- HTTPS obrigatório no checkout.
- Validação de ToS com timestamp.
- Audit de mudanças de plano e método de pagamento.
