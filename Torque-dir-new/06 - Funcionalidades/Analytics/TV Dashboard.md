---
tipo: feature
dominio: analytics
---

# TV Dashboard

## Propósito

Dashboard **fullscreen** projetado para exibição em TV do escritório. Atualização ao vivo, rotação automática de slides, visual impactante. Motiva o time, celebra conquistas em tempo real, mantém foco coletivo.

## Atores e Permissões

- **Admin**: configura, cria conta de acesso dedicada.
- **TV dedicada**: login com conta read-only amarrada a `organization_id`.

Ações: `tv_dashboard.view`, `tv_dashboard.configure`.

## Modo de Operação

- Acesso em URL especial com token de leitura.
- Modo fullscreen, sem UI de navegação.
- Rotação automática entre slides configurados.
- Resistência a idle (mantém conectado; reconecta automaticamente).

## Slides Configuráveis

### Slide 1: Pulso do Dia
- KPIs do dia: leads criados, reuniões, vendas.
- Metas diárias (progresso visual grande).

### Slide 2: Ranking do Mês
- Top 5 por métrica selecionada.
- Avatares grandes, números destacados.

### Slide 3: Alertas de Conquista
- Feed ao vivo: "João acabou de fechar R$ 5k!", "Ana agendou sua 15ª reunião do mês!".
- Animação de destaque em eventos importantes.

### Slide 4: Meta do Mês (Organização)
- Barra de progresso da meta principal.
- Projeção (estamos on track?).

### Slide 5: Funnel do Mês
- Gráfico grande do funil.

### Slide 6: Mural de Reconhecimento
- Badges recentes recebidos.
- Feedback positivo de leads (se capturado).

### Slide 7 (opcional): Meme/Foto da Empresa
- Conteúdo motivacional/cultural.

## Configuração

### Tempo por Slide
- Default 15s.
- Admin configura.

### Slides Ativos
- Admin escolhe quais exibir.
- Arrasta para reordenar.

### Filtros
- Métricas mostradas.
- Período padrão.

### Integração com Áudio
- Som ao detectar venda nova (som + animação).
- Silenciável.

## Regras de Negócio

1. TV dashboard é **readonly** — não executa ações.
2. Atualização automática a cada N segundos (config).
3. Eventos críticos (venda nova) quebram rotação para exibir imediatamente.
4. Reconexão automática em caso de perda de conexão.
5. Snapshot de métricas acontece no servidor; TV só renderiza.

## Fluxos do Usuário

### Admin Configura
1. `Configurações → TV Dashboard`.
2. Form com slides, tempo, métrica.
3. Preview.
4. Gera token/URL.

### Começar TV
1. Em dispositivo da TV, abre URL.
2. Login com conta TV ou via token.
3. Entra em fullscreen automático.
4. Rota.

### Quebrar Rota (evento)
1. Venda nova detectada.
2. Anima "VENDA!" com detalhes.
3. Volta à rotação após 30s.

## Automações

- Consome eventos realtime.
- Trigger especial para `ProposalWon`, `GoalAchieved`, `MemberMilestone`.

## Edge Cases

- **Sem internet**: últimos dados em cache; aviso discreto.
- **Navegador antigo**: fallback simplificado.
- **TV desligada**: reconecta automaticamente quando liga.
- **Múltiplas TVs**: cada uma com token próprio; admin vê lista.

## Performance

- Subscription direta a eventos (WebSocket).
- Renderização otimizada (animações leves).
- Sem queries pesadas — só consome snapshots.

## UX

- Tipografia grande.
- Cores vibrantes em eventos positivos.
- Animações suaves entre slides.
- Sem scroll (tudo cabe em uma tela).
- Sem UI de controle à vista (controle via admin remoto).

## Segurança

- Token tipicamente permanente (para TV não precisar relogin).
- Mas pode ser revogado em Configurações.
- Conta TV não tem acesso a dados sensíveis além de métricas agregadas.

## Valor

- Engajamento do time.
- Transparência.
- Motivação em tempo real.
- Credibilidade (visitantes veem operação ativa).
