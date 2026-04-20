-- =====================================================================
-- seeds/dev_admin.sql
--
-- DEV-ONLY seed. Creates a deterministic organization + admin user so the
-- frontend can log in immediately after `make migrate-up`.
--
--   email:    marcelo@gmail.com
--   password: admin
--
-- Never run this in staging or prod. The Makefile target `make seed-dev`
-- guards against that by refusing if ENV != dev.
--
-- All inserts are ON CONFLICT DO NOTHING so re-running the file is a no-op.
-- =====================================================================

-- --- Organization -----------------------------------------------------
INSERT INTO organizations (id, slug, name, plan_id, payment_status)
VALUES (
  '00000000-0000-0000-0000-0000000000a1',
  'torque-dev',
  'Torque Dev',
  'enterprise',
  'active'
)
ON CONFLICT (slug) DO NOTHING;

-- --- User -------------------------------------------------------------
-- password = 'admin'
-- bcrypt cost 12, generated with node bcryptjs (matches golang.org/x/crypto/bcrypt).
-- Hash is regenerable any time; the value below is committed deliberately
-- because it is a dev credential, not a secret.
INSERT INTO users (id, email, password_hash, display_name, is_active, email_verified_at)
VALUES (
  '00000000-0000-0000-0000-0000000000b1',
  'marcelo@gmail.com',
  '$2b$12$Gufjb5EHyosfJgpez.TpZufev2DvhOXbc9F0/UoQuDT2AhgX1YZK.',
  'Marcelo Montemezzo',
  true,
  now()
)
ON CONFLICT (email) DO UPDATE
  SET password_hash = EXCLUDED.password_hash,
      is_active     = true;

-- --- Membership (admin of Torque Dev) ---------------------------------
INSERT INTO team_members (
  id, organization_id, user_id, role, display_name, is_active, joined_at
)
VALUES (
  '00000000-0000-0000-0000-0000000000c1',
  '00000000-0000-0000-0000-0000000000a1',
  '00000000-0000-0000-0000-0000000000b1',
  'admin',
  'Marcelo Montemezzo',
  true,
  now()
)
ON CONFLICT (organization_id, user_id) DO NOTHING;

-- --- Verify -----------------------------------------------------------
DO $$
DECLARE
  v_found_user  int;
  v_found_team  int;
BEGIN
  SELECT count(*) INTO v_found_user FROM users       WHERE email = 'marcelo@gmail.com';
  SELECT count(*) INTO v_found_team FROM team_members WHERE user_id = '00000000-0000-0000-0000-0000000000b1' AND is_active;
  IF v_found_user = 0 THEN RAISE EXCEPTION 'seed: user marcelo@gmail.com missing'; END IF;
  IF v_found_team = 0 THEN RAISE EXCEPTION 'seed: team_membership for marcelo@gmail.com missing'; END IF;
  RAISE NOTICE 'seed ok: marcelo@gmail.com / admin → Torque Dev (admin role)';
END $$;
