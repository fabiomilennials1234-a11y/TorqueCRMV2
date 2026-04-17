---
tipo: feature
dominio: admin
---

# Configurações

## Propósito

Painel único onde o admin da organização ajusta **tudo o que é global da org**: dados da empresa, branding, integrações, canais, janela de negócio, defaults, preferências. Também concentra links para gestão de time, billing, webhooks, API keys.

## Atores e Permissões

- **Admin**: acesso total à configuração da própria org.
- **Membros**: sem acesso.

Ações: `organization.view`, `organization.edit`, `integration.connect`, etc.

## Seções

### 1. Dados da Empresa
- Nome, CNPJ, telefone, email.
- Endereço.
- Setor.
- Tamanho.

### 2. Branding
- Logo (upload).
- Cor primária.
- Cor de destaque.
- Favicon.
- Nome exibido ao lead (em links públicos, se org usa).

### 3. Localização
- Fuso horário.
- Idioma padrão.
- Moeda padrão.
- Formato de data.

### 4. Janela de Negócio
- Dias e horários da semana.
- Exceções (feriados, férias).
- Usado em: mensagens agendadas, workflow wait_business_window, campanha respeita horário.

### 5. Canais Conectados
- Lista de instâncias: WhatsApp (multi-instância), Messenger, etc.
- Conectar novo.
- Desconectar.
- Ver status (connected/disconnected/error).
- Reconectar (trigger QR code, re-OAuth).

### 6. Integrações
- TinyERP, Asaas, Google Calendar, Meta Business, n8n, etc.
- Conectar/desconectar.
- Ver última sync, próxima sync.
- Logs por integração.

### 7. Pipelines
- Atalho para Funis Hub.
- Edit rápido de defaults.

### 8. Lead Score
- Configurar pesos dos sinais.
- Ver simulação.

### 9. Tags
- CRUD de tags da org.
- Categorias.

### 10. Custom Fields
- Schema de campos custom do Lead.
- Tipos, opções, obrigatoriedade.

### 11. Webhooks de Saída
- Atalho para [[Webhooks]].

### 12. API Keys
- Atalho para gestão de keys.

### 13. Time
- Atalho para [[Gestão de Time]].

### 14. Comissões
- Regras de comissão da org.

### 15. Metas
- Metas da org.

### 16. Notificações
- Preferências de notificação em nível de org.
- Defaults para novos membros.

### 17. Billing
- Plano atual, histórico, método de pagamento.

### 18. Ranking / Transparência
- Config de visibilidade de ranking.

### 19. Upsell
- Regras de geração de oportunidades.

### 20. Onboarding (relançar)
- Reabrir wizard.

### 21. Exportar Dados
- Solicitar export completo (LGPD, backup).
- Job assíncrono gera arquivo, envia link por email.

### 22. Deletar Organização
- Soft-delete.
- Confirmação dupla + senha.
- Warn de consequências.

## Regras de Negócio

1. Apenas admin acessa; outros veem 403.
2. Alterações logadas em audit (`OrganizationUpdated` com diff).
3. Configurações críticas (fuso horário, moeda) têm reconfirmação — afetam histórico.
4. Export respeita LGPD: dados pessoais de lead incluídos.

## Fluxos do Usuário

### Conectar Canal WhatsApp
1. Configurações → Canais → Adicionar.
2. Escolhe provedor.
3. Tela com QR code — escaneia com WhatsApp.
4. Valida conexão.
5. Define nome amigável ("Número da Vendas").
6. Testa envio.
7. Ativa.

### Configurar Janela de Negócio
1. Configurações → Janela de Negócio.
2. Matriz dia × hora clicável.
3. Adiciona feriados em lista.
4. Salvar.

### Configurar Custom Fields
1. Configurações → Custom Fields.
2. Adiciona campo: nome (budget), tipo (number), obrigatório?, opções (para enum).
3. Preview em drawer de lead.

## Automações e Eventos

### Emite
- `OrganizationUpdated` (com diff de campos).
- `IntegrationConnected`, `IntegrationDisconnected`.
- `CustomFieldSchemaUpdated`.

### Reage
- Job de sync recarrega config quando muda.

## Integrações

- Tudo na org.

## Edge Cases

- **Mudar fuso horário**: pergunta se quer recalcular métricas históricas ou preservar.
- **Desconectar canal com conversas ativas**: avisa; conversas ficam em "canal desconectado".
- **Schema custom field alterado**: leads existentes mantêm valor; admin pode migrar manualmente.
- **Export muito grande** (> 1GB): divide em múltiplos arquivos; envia por email com links.

## Validações

- Formatos (email, CNPJ, telefone).
- Fuso horário válido.
- Custom field: nome único, tipo suportado.
- Cor: formato válido.
- Logo: tamanho máx (2MB).

## Métricas

- Time spent em Configurações (admin).
- Features mais configuradas.
- Taxa de orgs com branding completo (indicador de "bem configurado").

## UX

- Navegação em árvore / tabs.
- Busca de configuração ("fuso", "meta", "email").
- Autosave em campos longos (com confirmação).
- Desfazer última mudança (30s de janela).
- Dark-first.
