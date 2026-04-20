-- =====================================================================
-- 0018_message_templates.up.sql
-- S35 / Fase B — message templates para o Inbox.
--
-- Cada tenant mantém seu catálogo de templates usados no ComposerBar.
-- Variáveis `{{nome}}`, `{{empresa}}` são preenchidas client-side antes
-- do envio; o servidor não faz merge — a substituição pertence ao
-- momento da digitação para que o usuário veja o que vai mandar.
-- =====================================================================

CREATE TABLE message_templates (
  id               uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id  uuid         NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name             text         NOT NULL CHECK (char_length(name) BETWEEN 2 AND 120),
  body             text         NOT NULL CHECK (char_length(body) BETWEEN 1 AND 16000),
  variables        text[]       NOT NULL DEFAULT ARRAY[]::text[],
  is_active        bool         NOT NULL DEFAULT true,
  created_by       uuid         NULL REFERENCES team_members(id) ON DELETE SET NULL,
  created_at       timestamptz  NOT NULL DEFAULT now(),
  updated_at       timestamptz  NOT NULL DEFAULT now(),

  CONSTRAINT uq_message_templates_org_name UNIQUE (organization_id, name)
);

CREATE INDEX idx_message_templates_org_active
  ON message_templates (organization_id)
  WHERE is_active = true;

CREATE TRIGGER trg_message_templates_updated_at
  BEFORE UPDATE ON message_templates
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE message_templates IS 'S35 — catalog de templates do Inbox com {{variáveis}}.';

-- Permission keys — templates são admin-managed; leitura é member-wide
-- para o ComposerBar mostrar a lista.
INSERT INTO feature_permissions (feature_key, is_admin_only, master_only, default_value, description) VALUES
  ('templates.view',   false, false, true, 'Ver templates de mensagem.'),
  ('templates.manage', true,  false, true, 'Criar, editar, arquivar templates de mensagem (admin).')
ON CONFLICT (feature_key) DO NOTHING;
