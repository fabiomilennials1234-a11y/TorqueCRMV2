-- =====================================================================
-- 0014_products_and_members_perms.up.sql
-- S21 — F10 Equipe (CRUD de membros) + F11 Produtos (catalogo base para
-- Propostas F03/F12).
--
-- F10 nao precisa de tabela nova: team_members + member_feature_permissions
-- ja existem desde 0002. Esta migration apenas seeda as permission keys
-- e adiciona products.
-- =====================================================================

-- ---------- products --------------------------------------------------
--
-- Minimal product model for the catalog that backs Propostas. Prices are
-- in integer cents to avoid any float-money drift. `sku` is optional but
-- unique per tenant when present (ERP integrations rely on it).

CREATE TABLE products (
  id                uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid         NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name              text         NOT NULL CHECK (char_length(name) BETWEEN 2 AND 160),
  description       text         NULL CHECK (description IS NULL OR char_length(description) <= 4000),
  sku               text         NULL CHECK (sku IS NULL OR (char_length(sku) BETWEEN 1 AND 60)),
  price_cents       bigint       NOT NULL CHECK (price_cents >= 0),
  currency          text         NOT NULL DEFAULT 'BRL' CHECK (char_length(currency) = 3),
  is_active         bool         NOT NULL DEFAULT true,
  metadata          jsonb        NOT NULL DEFAULT '{}'::jsonb,
  created_by        uuid         NULL REFERENCES team_members(id) ON DELETE SET NULL,
  created_at        timestamptz  NOT NULL DEFAULT now(),
  updated_at        timestamptz  NOT NULL DEFAULT now()
);

CREATE INDEX idx_products_org_active
  ON products (organization_id, updated_at DESC)
  WHERE is_active = true;

CREATE INDEX idx_products_org_updated
  ON products (organization_id, updated_at DESC, id DESC);

-- SKU is unique per tenant only when present. Partial unique index avoids
-- rejecting tenants that do not maintain SKUs at all.
CREATE UNIQUE INDEX uq_products_org_sku
  ON products (organization_id, sku)
  WHERE sku IS NOT NULL;

CREATE TRIGGER trg_products_updated_at
  BEFORE UPDATE ON products
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE products IS 'F11 catalog — used by Propostas line items. Prices in integer cents.';

-- ---------- seed permission keys -------------------------------------
-- members.view/manage: surface the team tab. Read is member-wide (anyone
-- on the org can see who the rest of the team is); mutations require
-- admin (role change, remove, permission overrides).
--
-- products.view/manage: read is member-wide (salespeople need the
-- catalog to build proposals); mutations are admin-only.

INSERT INTO feature_permissions (feature_key, is_admin_only, master_only, default_value, description) VALUES
  ('members.view',    false, false, true, 'Ver membros da organizacao.'),
  ('members.manage',  true,  false, true, 'Convidar, desativar, alterar papel e permissoes de membros (admin).'),
  ('products.view',   false, false, true, 'Ver catalogo de produtos.'),
  ('products.manage', true,  false, true, 'Criar, editar, arquivar produtos (admin).')
ON CONFLICT (feature_key) DO NOTHING;
