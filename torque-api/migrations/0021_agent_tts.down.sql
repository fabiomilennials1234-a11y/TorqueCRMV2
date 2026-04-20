-- =====================================================================
-- 0021_agent_tts.down.sql — revert S41 TTS config columns.
-- =====================================================================

ALTER TABLE agents
  DROP COLUMN IF EXISTS tts_voice_id,
  DROP COLUMN IF EXISTS tts_enabled;
