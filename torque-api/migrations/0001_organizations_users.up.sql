-- =====================================================================
-- 0001_organizations_users.up.sql
-- Foundational tables: organizations, users, master users, plans,
-- feature permissions catalog, and a reusable updated_at trigger function.
--
-- Runs in a single transaction by golang-migrate (default).
-- =====================================================================

-- ---------- Extensions ------------------------------------------------

CREATE EXTENSION IF NOT EXISTS "pgcrypto"; -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS "citext";   -- case-insensitive email

-- ---------- Utility: updated_at trigger -------------------------------

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$;

COMMENT ON FUNCTION set_updated_at() IS 'Sets NEW.updated_at = now() on row update. Attached per-table.';

-- ---------- plans (global catalog) ------------------------------------

CREATE TABLE plans (
  id               text        PRIMARY KEY,           -- e.g. 'free', 'growth', 'enterprise'
  name             text        NOT NULL,
  monthly_price    int         NOT NULL DEFAULT 0,    -- cents (BRL)
  yearly_price     int         NOT NULL DEFAULT 0,    -- cents (BRL)
  is_active        bool        NOT NULL DEFAULT true,
  metadata         jsonb       NOT NULL DEFAULT '{}'::jsonb,
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_plans_updated_at
  BEFORE UPDATE ON plans
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE plans IS 'Catalog of subscription plans. Global (not tenant-scoped).';

-- ---------- organizations (tenants) -----------------------------------

CREATE TYPE org_payment_status AS ENUM ('active', 'overdue', 'suspended', 'cancelled');

CREATE TABLE organizations (
  id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  slug             text        NOT NULL UNIQUE
                                   CHECK (slug ~ '^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$'),
  name             text        NOT NULL CHECK (char_length(name) BETWEEN 2 AND 120),
  plan_id          text        NULL REFERENCES plans(id) ON DELETE SET NULL,
  payment_status   org_payment_status NOT NULL DEFAULT 'active',
  logo_url         text        NULL,
  metadata         jsonb       NOT NULL DEFAULT '{}'::jsonb,
  deleted_at       timestamptz NULL,                  -- soft delete
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_organizations_deleted_at ON organizations (deleted_at) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_organizations_updated_at
  BEFORE UPDATE ON organizations
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE organizations IS 'Tenants. Every domain row references organization_id.';

-- ---------- users (global identity) -----------------------------------
--
-- Users are global: the same human can belong to multiple organizations.
-- Organization membership and role live in team_members (migration 0002).

CREATE TABLE users (
  id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  email            citext      NOT NULL UNIQUE CHECK (char_length(email) <= 254),
  password_hash    text        NOT NULL,                       -- bcrypt
  display_name     text        NOT NULL CHECK (char_length(display_name) BETWEEN 1 AND 120),
  avatar_url       text        NULL,
  is_active        bool        NOT NULL DEFAULT true,
  email_verified_at timestamptz NULL,
  last_login_at    timestamptz NULL,
  metadata         jsonb       NOT NULL DEFAULT '{}'::jsonb,
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_users_updated_at
  BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE users IS 'Global user identity. Memberships live in team_members.';

-- ---------- users_master (internal staff) -----------------------------
--
-- Separate table by design: keeps the master role out of the RBAC cascade,
-- prevents accidental promotion, and makes audit logs unambiguous.

CREATE TABLE users_master (
  user_id          uuid        PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  granted_by       uuid        NULL REFERENCES users(id) ON DELETE SET NULL,
  granted_at       timestamptz NOT NULL DEFAULT now(),
  reason           text        NULL
);

COMMENT ON TABLE users_master IS 'Master-admin flag. Never visible to tenants.';

-- ---------- feature_permissions (global catalog) ----------------------
--
-- Declares which permissions exist. RBAC cascade (Autenticacao §RBAC) reads
-- this to resolve effective permission per member.

CREATE TABLE feature_permissions (
  feature_key      text        PRIMARY KEY CHECK (feature_key ~ '^[a-z][a-z0-9_.]{1,80}$'),
  is_admin_only    bool        NOT NULL DEFAULT false,          -- true → only admins can enable
  master_only      bool        NOT NULL DEFAULT false,          -- true → admins cannot bypass
  default_value    bool        NOT NULL DEFAULT true,           -- default when no member override
  description      text        NOT NULL,
  created_at       timestamptz NOT NULL DEFAULT now()
);

COMMENT ON TABLE feature_permissions IS 'Catalog of permission keys referenced by RBAC.';

-- ---------- seeds -----------------------------------------------------

INSERT INTO plans (id, name, monthly_price, yearly_price) VALUES
  ('free',       'Free',       0,     0),
  ('growth',     'Growth',     29900, 299900),
  ('enterprise', 'Enterprise', 99900, 999900)
ON CONFLICT (id) DO NOTHING;

-- Minimal seed for ui.* permissions referenced by ADR-007.
INSERT INTO feature_permissions (feature_key, is_admin_only, master_only, default_value, description) VALUES
  ('ui.view_manager_mode',     false, false, false, 'Pode alternar para modo Gerente.'),
  ('ui.view_salesperson_mode', false, false, true,  'Pode alternar para modo Vendedor.')
ON CONFLICT (feature_key) DO NOTHING;
